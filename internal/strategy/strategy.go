// Package strategy is a factory method wrapper around the backing store implementations
package strategy

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"sync"

// 	"github.com/DevLabFoundry/configmanager/v3/internal/config"
// 	"github.com/DevLabFoundry/configmanager/v3/internal/log"
// 	"github.com/DevLabFoundry/configmanager/v3/internal/store"
// )

// var ErrTokenInvalid = errors.New("invalid token - cannot get prefix")

// // StrategyFunc
// type StrategyFunc func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error)

// // StrategyFuncMap
// type StrategyFuncMap map[config.ImplementationPrefix]StrategyFunc

// type Strategy struct {
// 	config          config.GenVarsConfig
// 	strategyFuncMap strategyFnMap
// }

// type Opts func(*Strategy)

// // New
// func New(config config.GenVarsConfig, logger log.ILogger, opts ...Opts) *Strategy {
// 	rs := &Strategy{
// 		config:          config,
// 		strategyFuncMap: strategyFnMap{mu: sync.Mutex{}, funcMap: defaultStrategyFuncMap(logger)},
// 	}
// 	// overwrite or add any options/defaults set above
// 	for _, o := range opts {
// 		o(rs)
// 	}

// 	return rs
// }

// // WithStrategyFuncMap Adds custom implementations for prefix
// //
// // Mainly used for testing
// // NOTE: this may lead to eventual optional configurations by users
// func WithStrategyFuncMap(funcMap StrategyFuncMap) Opts {
// 	return func(rs *Strategy) {
// 		rs.strategyFuncMap.mu.Lock()
// 		defer rs.strategyFuncMap.mu.Unlock()
// 		for prefix, implementation := range funcMap {
// 			rs.strategyFuncMap.funcMap[config.ImplementationPrefix(prefix)] = implementation
// 		}
// 	}
// }

// // GetImplementation is a factory method returning the concrete implementation for the retrieval of the token value
// // i.e. facilitating the exchange of the supplied token for the underlying value
// func (rs *Strategy) GetImplementation(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
// 	if token == nil {
// 		return nil, fmt.Errorf("unable to get prefix, %w", ErrTokenInvalid)
// 	}

// 	if store, found := rs.strategyFuncMap.funcMap[token.Prefix()]; found {
// 		return store(ctx, token)
// 	}

// 	return nil, fmt.Errorf("implementation not found for input string: %s", token)
// }

// func ExchangeToken(s store.Strategy, token *config.ParsedTokenConfig) *TokenResponse {
// 	cr := &TokenResponse{}
// 	cr.Err = nil
// 	cr.key = token
// 	s.SetToken(token)
// 	cr.val, cr.Err = s.Value()
// 	return cr
// }

// type TokenResponse struct {
// 	val string
// 	key *config.ParsedTokenConfig
// 	Err error
// }

// func (tr *TokenResponse) WithKey(key *config.ParsedTokenConfig) {
// 	tr.key = key
// }

// func (tr *TokenResponse) WithValue(val string) {
// 	tr.val = val
// }

// func (tr *TokenResponse) Key() *config.ParsedTokenConfig {
// 	return tr.key
// }

// func (tr *TokenResponse) Value() string {
// 	return tr.val
// }

// func defaultStrategyFuncMap(logger log.ILogger) StrategyFuncMap {
// 	return map[config.ImplementationPrefix]StrategyFunc{
// 		// config.AzTableStorePrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
// 		// 	return store.NewAzTableStore(ctx, token, logger)
// 		// },
// 		// config.AzAppConfigPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
// 		// 	return store.NewAzAppConf(ctx, token, logger)
// 		// },
// 		// config.GcpSecretsPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
// 		// 	return store.NewGcpSecrets(ctx, logger)
// 		// },
// 		// config.SecretMgrPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
// 		// 	return store.NewSecretsMgr(ctx, logger)
// 		// },
// 		// config.ParamStorePrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
// 		// 	return store.NewParamStore(ctx, logger)
// 		// },
// 		// config.AzKeyVaultSecretsPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
// 		// 	return store.NewKvScrtStore(ctx, token, logger)
// 		// },
// 		// config.HashicorpVaultPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
// 		// 	return store.NewVaultStore(ctx, token, logger)
// 		// },
// 	}
// }

// type strategyFnMap struct {
// 	mu      sync.Mutex
// 	funcMap StrategyFuncMap
// }
