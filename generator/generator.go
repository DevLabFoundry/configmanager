package generator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/lexer"
	"github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/DevLabFoundry/configmanager/v3/internal/parser"
	"github.com/DevLabFoundry/configmanager/v3/internal/strategy"
)

// GenVars is the main struct holding the
// strategy patterns iface
// any initialised config if overridded with withers
// as well as the final outString and the initial rawMap
// which wil be passed in a loop into a goroutine to perform the
// relevant strategy network calls to the config store implementations
type GenVars struct {
	Logger   log.ILogger
	strategy strategy.StrategyFuncMap
	ctx      context.Context
	config   config.GenVarsConfig
}

type Opts func(*GenVars)

// NewGenerator returns a new instance of Generator
// with a default strategy pattern wil be overwritten
// during the first run of a found tokens map
func NewGenerator(ctx context.Context, opts ...Opts) *GenVars {
	// defaultStrategy := NewDefatultStrategy()
	return newGenVars(ctx, opts...)
}

func newGenVars(ctx context.Context, opts ...Opts) *GenVars {
	conf := config.NewConfig()
	g := &GenVars{
		Logger: log.New(io.Discard),
		ctx:    ctx,
		// return using default config
		config: *conf,
	}
	g.strategy = nil

	// now apply additional opts
	for _, o := range opts {
		o(g)
	}

	return g
}

// WithStrategyMap
//
// Adds addtional funcs for storageRetrieval used for testing only
func (c *GenVars) WithStrategyMap(sm strategy.StrategyFuncMap) *GenVars {
	c.strategy = sm
	return c
}

// WithConfig uses custom config
func (c *GenVars) WithConfig(cfg *config.GenVarsConfig) *GenVars {
	// backwards compatibility
	if cfg != nil {
		c.config = *cfg
	}
	return c
}

// WithContext uses caller passed context
func (c *GenVars) WithContext(ctx context.Context) *GenVars {
	c.ctx = ctx
	return c
}

// Config gets Config on the GenVars
func (c *GenVars) Config() *config.GenVarsConfig {
	return &c.config
}

// Generate generates a k/v map of the tokens with their corresponding secret/paramstore values
// the standard pattern of a token should follow a path like string
//
// Called only from a slice of tokens
func (c *GenVars) Generate(tokens []string) (ReplacedToken, error) {

	ntm, err := c.DiscoverTokens(strings.Join(tokens, "\n"))
	if err != nil {
		return nil, err
	}

	// pass in default initialised retrieveStrategy
	// input should be
	rt, err := c.generate(ntm)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

var ErrTokenDiscovery = errors.New("failed to discover tokens")

// DiscoverToken generates a k/v map of the tokens with their corresponding secret/paramstore values
// the standard pattern of a token should follow a path like string
//
// Called only from a slice of tokens
func (c *GenVars) DiscoverTokens(text string) (NormalizedTokenSafe, error) {

	rtm := NewRawTokenConfig()

	lexerSource := lexer.Source{FileName: text[0:min(len(text), 20)], FullPath: "", Input: text}
	l := lexer.New(lexerSource, c.config)
	p := parser.New(l, &c.config).WithLogger(log.New(os.Stderr))
	parsed, errs := p.Parse()
	if len(errs) > 0 {
		return NormalizedTokenSafe{}, fmt.Errorf("%w in input (%s) with errors: %q", ErrTokenDiscovery, text[0:min(len(text), 25)], errs)
	}
	for _, prsdToken := range parsed {
		rtm.AddToken(prsdToken.ParsedToken.String(), &prsdToken.ParsedToken)
	}
	return c.NormalizeRawToken(rtm), nil
}

// IsParsed will try to parse the return found string into
// map[string]string
// If found it will convert that to a map with all keys uppercased
// and any characters
func IsParsed(v any, trm ReplacedToken) bool {
	str := fmt.Sprint(v)
	err := json.Unmarshal([]byte(str), &trm)
	return err == nil
}

// generate initiates waitGroup to handle 1 or more normalized network calls concurrently to the underlying stores
//
// Captures the response/error in TokenResponse struct
// It then denormalizes the NormalizedTokenSafe back to a ReplacedToken map
// which stores the values for each token to be returned to the caller
func (c *GenVars) generate(ntm NormalizedTokenSafe) (ReplacedToken, error) {
	if len(ntm.normalizedTokenMap) < 1 {
		c.Logger.Debug("no replaceable tokens found in input")
		return nil, nil
	}

	wg := &sync.WaitGroup{}

	s := strategy.New(c.config, c.Logger, strategy.WithStrategyFuncMap(c.strategy))

	// safe read of normalized token map
	// this will ensure that we are minimizing
	// the number of network calls to each underlying store
	for _, prsdTkn := range ntm.GetMap() {
		if len(prsdTkn.parsedTokens) == 0 {
			// TODO: err type this
			return nil, fmt.Errorf("no tokens assigned to parsedTokens slice")
		}
		token := prsdTkn.parsedTokens[0]
		wg.Go(func() {
			prsdTkn.resp = &strategy.TokenResponse{}
			storeStrategy, err := s.GetImplementation(c.ctx, token)
			if err != nil {
				prsdTkn.resp.Err = err
				return
			}
			prsdTkn.resp = strategy.ExchangeToken(storeStrategy, token)
		})
	}

	wg.Wait()

	// now we fan out the normalized value to ReplacedToken map
	// this will ensure all found tokens will have a value assigned to them
	replacedToken := make(ReplacedToken)
	for _, r := range ntm.GetMap() {
		if r == nil {
			// defensive as this shouldn't happen
			continue
		}
		if r.resp.Err != nil {
			c.Logger.Debug("cr.err %v, for token: %s", r.resp.Err, r.resp.Key().String())
			continue
		}
		for _, originalToken := range r.parsedTokens {
			replacedToken[originalToken.String()] = keySeparatorLookup(originalToken, r.resp.Value())
		}
	}
	return replacedToken, nil
}

// NormalizedToken represents the struct after all the possible tokens
// were merged into the lowest commmon denominator.
// The idea is to minimize the number of networks calls to the underlying `store` Implementations
//
// The merging is based on the implemenentation and sanitized token being the same,
// if the token contains metadata then it must be
//
// # Merging strategy
//
// Same Prefix + Same SanitisedToken && No Metadata
type NormalizedToken struct {
	// all the tokens that can be used to do a replacement
	parsedTokens []*config.ParsedTokenConfig
	// will be assigned post generate
	resp *strategy.TokenResponse
	// // configToken is the last assigned full config in the loop if multip
	// configToken *config.ParsedTokenConfig
}

func (n *NormalizedToken) WithParsedToken(v *config.ParsedTokenConfig) *NormalizedToken {
	n.parsedTokens = append(n.parsedTokens, v)
	return n
}

// NormalizedTokenSafe is the map of lowest common denominators
// by token.Keypathless or token.String (full token) if metadata is included
type NormalizedTokenSafe struct {
	mu                 *sync.Mutex
	normalizedTokenMap map[string]*NormalizedToken
}

func (n NormalizedTokenSafe) GetMap() map[string]*NormalizedToken {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.normalizedTokenMap
}

func (c *GenVars) NormalizeRawToken(rtm *RawTokenConfig) NormalizedTokenSafe {
	ntm := NormalizedTokenSafe{mu: &sync.Mutex{}, normalizedTokenMap: make(map[string]*NormalizedToken)}

	for _, r := range rtm.RawTokenMap() {
		// if a string contains we need to store it uniquely
		// future improvements might group all the metadata values together
		if len(r.Metadata()) > 0 {
			if n, found := ntm.normalizedTokenMap[r.String()]; found {
				n.WithParsedToken(r)
				continue
			}
			ntm.normalizedTokenMap[r.String()] = (&NormalizedToken{}).WithParsedToken(r)
			continue
		}

		if n, found := ntm.normalizedTokenMap[r.Keypathless()]; found {
			n.WithParsedToken(r)
			continue
		}
		ntm.normalizedTokenMap[r.Keypathless()] = (&NormalizedToken{}).WithParsedToken(r)
		continue
	}
	return ntm
}
