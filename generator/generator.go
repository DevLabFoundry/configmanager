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
	"github.com/DevLabFoundry/configmanager/v3/internal/store"
)

// Generator is the main struct holding the
// strategy patterns iface
// any initialised config if overridded with withers
// as well as the final outString and the initial rawMap
// which wil be passed in a loop into a goroutine to perform the
// relevant strategy network calls to the config store implementations
type Generator struct {
	Logger log.ILogger
	// strategy strategy.StrategyFuncMap
	store  *store.Store
	ctx    context.Context
	config config.GenVarsConfig
}

type Opts func(*Generator)

// New returns a new instance of Generator
// with a default strategy pattern wil be overwritten
// during the first run of a found tokens map
func New(ctx context.Context, opts ...Opts) *Generator {
	// defaultStrategy := NewDefatultStrategy()
	return new(ctx, opts...)
}

func new(ctx context.Context, opts ...Opts) *Generator {
	conf := config.NewConfig()
	g := &Generator{
		Logger: log.New(io.Discard),
		ctx:    ctx,
		// return using default config
		config: *conf,
	}
	// g.strategy = nil

	// now apply additional opts
	for _, o := range opts {
		o(g)
	}

	return g
}

// // WithStrategyMap
// //
// // Adds addtional funcs for storageRetrieval used for testing only
// func (c *Generator) WithStrategyMap(sm strategy.StrategyFuncMap) *Generator {
// 	c.strategy = sm
// 	return c
// }

// WithConfig uses custom config
func (c *Generator) WithConfig(cfg *config.GenVarsConfig) *Generator {
	// backwards compatibility
	if cfg != nil {
		c.config = *cfg
	}
	return c
}

// WithContext uses caller passed context
func (c *Generator) WithContext(ctx context.Context) *Generator {
	c.ctx = ctx
	return c
}

// Config gets Config on the GenVars
func (c *Generator) Config() *config.GenVarsConfig {
	return &c.config
}

// Generate generates a k/v map of the tokens with their corresponding secret/paramstore values
// the standard pattern of a token should follow a path like string
//
// Called only from a slice of tokens
func (c *Generator) Generate(tokens []string) (ReplacedToken, error) {

	ntm, err := c.DiscoverTokens(strings.Join(tokens, "\n"))
	if err != nil {
		return nil, err
	}

	// initialise pugins here based on discovered tokens
	//
	s, err := store.Init(c.ctx, ntm.TokenSet())
	if err != nil {
		return nil, err
	}

	c.store = s
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
func (c *Generator) DiscoverTokens(text string) (NormalizedTokenSafe, error) {

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
func (c *Generator) generate(ntm NormalizedTokenSafe) (ReplacedToken, error) {
	if len(ntm.m) < 1 {
		c.Logger.Debug("no replaceable tokens found in input")
		return nil, nil
	}

	wg := &sync.WaitGroup{}

	// initialise the stores here
	// s := strategy.New(c.config, c.Logger, strategy.WithStrategyFuncMap(c.strategy))

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
			prsdTkn.resp = &TokenResponse{}
			prsdTkn.resp.WithKey(token)
			storeStrategy, err := c.store.GetImplementation(token.Prefix())
			if err != nil {
				prsdTkn.resp.Err = err
				return
			}
			// storeStrategy.GetValue(token)
			v, err := storeStrategy.GetValue(token)
			if err != nil {
				prsdTkn.resp.Err = err
				return
			}
			prsdTkn.resp.WithValue(v)
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
// if the token contains metadata then it must be stored uniquely even if the underlying store is the same.
// This is because a token with metadata must be called uniquely
// as it may contain different versions of the same token - hence the value would be different
//
// # Merging strategy
//
// Same Prefix + Same SanitisedToken && No Metadata
type NormalizedToken struct {
	// all the tokens that can be used to do a replacement
	parsedTokens []*config.ParsedTokenConfig
	// will be assigned post generate
	resp *TokenResponse
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
	mu  *sync.Mutex
	m   map[string]*NormalizedToken
	set map[string]struct{}
}

func (n NormalizedTokenSafe) GetMap() map[string]*NormalizedToken {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.m
}

func (n NormalizedTokenSafe) TokenSet() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	ss := []string{}
	for key, _ := range n.set {
		ss = append(ss, strings.ToLower(key))
	}
	return ss
}

func (c *Generator) NormalizeRawToken(rtm *RawTokenConfig) NormalizedTokenSafe {
	ntm := NormalizedTokenSafe{mu: &sync.Mutex{}, m: make(map[string]*NormalizedToken), set: make(map[string]struct{})}

	for _, r := range rtm.RawTokenMap() {
		// if a string contains we need to store it uniquely
		// future improvements might group all the metadata values together
		if len(r.Metadata()) > 0 {
			if n, found := ntm.m[r.String()]; found {
				n.WithParsedToken(r)
				continue
			}
			ntm.m[r.String()] = (&NormalizedToken{}).WithParsedToken(r)
			ntm.set[string(r.Prefix())] = struct{}{}
			continue
		}

		if n, found := ntm.m[r.Keypathless()]; found {
			n.WithParsedToken(r)
			continue
		}
		ntm.m[r.Keypathless()] = (&NormalizedToken{}).WithParsedToken(r)
		ntm.set[string(r.Prefix())] = struct{}{}
		continue
	}
	return ntm
}
