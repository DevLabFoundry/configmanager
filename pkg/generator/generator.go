package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/internal/log"
	"github.com/DevLabFoundry/configmanager/v2/internal/store"
	"github.com/DevLabFoundry/configmanager/v2/internal/strategy"
	"github.com/spyzhov/ajson"
)

type retrieveIface interface {
	WithStrategyFuncMap(funcMap strategy.StrategyFuncMap) *strategy.RetrieveStrategy
	RetrieveByToken(ctx context.Context, impl store.Strategy, in *config.ParsedTokenConfig) *strategy.TokenResponse
	SelectImplementation(ctx context.Context, in *config.ParsedTokenConfig) (store.Strategy, error)
}

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

// ParsedMap is the internal working object definition and
// the return type if results are not flushed to file
type ParsedMap map[string]any

func (pm ParsedMap) MapKeys() (keys []string) {
	for k := range pm {
		keys = append(keys, k)
	}
	return
}

type tokenMapSafe struct {
	mu       *sync.Mutex
	tokenMap ParsedMap
}

func (tms *tokenMapSafe) getTokenMap() ParsedMap {
	tms.mu.Lock()
	defer tms.mu.Unlock()
	return tms.tokenMap
}

func (tms *tokenMapSafe) addKeyVal(key *config.ParsedTokenConfig, val string) {
	tms.mu.Lock()
	defer tms.mu.Unlock()
	// NOTE: still use the metadata in the key
	// there could be different versions / labels for the same token and hence different values
	// However the JSONpath look up
	tms.tokenMap[key.String()] = keySeparatorLookup(key, val)
}

type rawTokenMap struct {
	mu       sync.Mutex
	tokenMap map[string]*config.ParsedTokenConfig
}

func newRawTokenMap() *rawTokenMap {
	return &rawTokenMap{mu: sync.Mutex{}, tokenMap: map[string]*config.ParsedTokenConfig{}}
}

func (rtm *rawTokenMap) addToken(name string, parsedToken *config.ParsedTokenConfig) {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()
	rtm.tokenMap[name] = parsedToken
}

func (rtm *rawTokenMap) mapOfToken() map[string]*config.ParsedTokenConfig {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()
	return rtm.tokenMap
}

// Generate generates a k/v map of the tokens with their corresponding secret/paramstore values
// the standard pattern of a token should follow a path like string
func (c *GenVars) Generate(tokens []string) (ParsedMap, error) {

	rtm := newRawTokenMap()
	for _, token := range tokens {
		// TODO: normalize tokens here potentially
		// merge any tokens that only differ in keys lookup inside the object
		parsedToken, err := config.NewParsedTokenConfig(token, c.config)
		if err != nil {
			c.Logger.Info(err.Error())
			continue
		}
		rtm.addToken(token, parsedToken)
	}
	// pass in default initialised retrieveStrategy
	// input should be
	if err := c.generate(rtm); err != nil {
		return nil, err
	}
	return c.rawMap.getTokenMap(), nil
}

// generate checks if any tokens found
// initiates groutines with fixed size channel map
// to capture responses and errors
// generates ParsedMap which includes
func (c *GenVars) generate(rawMap *rawTokenMap) error {
	rtm := rawMap.mapOfToken()
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

// IsParsed will try to parse the return found string into
// map[string]string
// If found it will convert that to a map with all keys uppercased
// and any characters
func IsParsed(v any, trm ParsedMap) bool {
	str := fmt.Sprint(v)
	err := json.Unmarshal([]byte(str), &trm)
	return err == nil
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
