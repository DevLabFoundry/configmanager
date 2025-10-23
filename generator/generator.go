package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/lexer"
	"github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/DevLabFoundry/configmanager/v3/internal/parser"
	"github.com/DevLabFoundry/configmanager/v3/internal/strategy"
	"github.com/spyzhov/ajson"
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
	// rawMap is the internal object that holds the values
	// of original token => retrieved value - decrypted in plain text
	// with a mutex RW locker
	rawMap tokenMapSafe //ParsedMap
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
	m := make(ParsedMap)
	conf := config.NewConfig()
	g := &GenVars{
		Logger: log.New(io.Discard),
		rawMap: tokenMapSafe{
			tokenMap: m,
			mu:       &sync.Mutex{},
		},
		ctx: ctx,
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
func (c *GenVars) Generate(tokens []string) (ParsedMap, error) {

	rtm := NewRawTokenConfig()
	for _, token := range tokens {
		lexerSource := lexer.Source{FileName: token, FullPath: "", Input: token}
		l := lexer.New(lexerSource, c.config)
		p := parser.New(l, &c.config).WithLogger(log.New(os.Stderr))
		parsed, errs := p.Parse()
		if len(errs) > 0 {
			c.Logger.Info(fmt.Sprintf("%v", errs))
			continue
		}
		for _, prsdToken := range parsed {
			rtm.AddToken(token, &prsdToken.ParsedToken)
		}
	}
	// pass in default initialised retrieveStrategy
	// input should be
	if err := c.generate(rtm); err != nil {
		return nil, err
	}
	return c.rawMap.getTokenMap(), nil
}

// IsParsed will try to parse the return found string into
// map[string]string
// If found it will convert that to a map with all keys uppercased
// and any characters
func IsParsed(v any, trm ParsedMap) bool {
	str := fmt.Sprint(v)
	err := json.Unmarshal([]byte(str), &trm)
	return err == nil
}

// generate checks if any tokens found
// initiates groutines with fixed size channel map
// to capture responses and errors
// generates ParsedMap which includes
//
// TODO: change this slightly
func (c *GenVars) generate(rawMap *RawTokenConfig) error {
	rtm := rawMap.RawTokenMap()
	if len(rtm) < 1 {
		c.Logger.Debug("no replaceable tokens found in input")
		return nil
	}

	tokenCount := len(rtm)
	outCh := make(chan *strategy.TokenResponse, tokenCount)

	// TODO: initialise the singleton serviceContainer
	// pass into each goroutine
	for _, parsedToken := range rtm {
		token := parsedToken // safe closure capture
		// take value from config allocation on a per iteration basis
		go func() {
			s := strategy.New(c.config, c.Logger, strategy.WithStrategyFuncMap(c.strategy))
			storeStrategy, err := s.SelectImplementation(c.ctx, token)
			if err != nil {
				outCh <- &strategy.TokenResponse{Err: err}
				return
			}
			outCh <- s.RetrieveByToken(c.ctx, storeStrategy, token)
		}()
	}

	// Fan-in: receive results with pure select
	received := 0
	for received < tokenCount {
		select {
		case cr := <-outCh:
			if cr == nil {
				continue // defensive (shouldn't happen)
			}
			c.Logger.Debug("cro: %+v", cr)
			if cr.Err != nil {
				c.Logger.Debug("cr.err %v, for token: %s", cr.Err, cr.Key())
			} else {
				c.rawMap.addKeyVal(cr.Key(), cr.Value())
			}
			received++
		case <-c.ctx.Done():
			c.Logger.Debug("context done: %v", c.ctx.Err())
			return c.ctx.Err() // propagate context error (cancel/timeout)
		}
	}
	return nil
}

// keySeparatorLookup checks if the key contains
// keySeparator character
// If it does contain one then it tries to parse
func keySeparatorLookup(key *config.ParsedTokenConfig, val string) string {
	// key has separator
	k := key.LookupKeys()
	if k == "" {
		// c.logger.Info("no keyseparator found")
		return val
	}

	keys, err := ajson.JSONPath([]byte(val), fmt.Sprintf("$..%s", k))
	if err != nil {
		// c.logger.Debug("unable to parse as json object %v", err.Error())
		return val
	}

	if len(keys) == 1 {
		v := keys[0]
		if v.Type() == ajson.String {
			str, err := strconv.Unquote(fmt.Sprintf("%v", v))
			if err != nil {
				// c.logger.Debug("unable to unquote value: %v returning as is", v)
				return fmt.Sprintf("%v", v)
			}
			return str
		}

		return fmt.Sprintf("%v", v)
	}

	// c.logger.Info("no value found in json using path expression")
	return ""
}
