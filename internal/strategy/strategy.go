// Package strategy is a strategy pattern wrapper around the store implementations
//
// NOTE: this may be refactored out into the store package directly
package strategy

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/internal/log"
	"github.com/DevLabFoundry/configmanager/v2/internal/store"
)

var ErrTokenInvalid = errors.New("invalid token - cannot get prefix")

// StrategyFunc
type StrategyFunc func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error)

// StrategyFuncMap
type StrategyFuncMap map[config.ImplementationPrefix]StrategyFunc

func defaultStrategyFuncMap(logger log.ILogger) map[config.ImplementationPrefix]StrategyFunc {
	return map[config.ImplementationPrefix]StrategyFunc{
		config.AzTableStorePrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
			return store.NewAzTableStore(ctx, token, logger)
		},
		config.AzAppConfigPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
			return store.NewAzAppConf(ctx, token, logger)
		},
		config.GcpSecretsPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
			return store.NewGcpSecrets(ctx, logger)
		},
		config.SecretMgrPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
			return store.NewSecretsMgr(ctx, logger)
		},
		config.ParamStorePrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
			return store.NewParamStore(ctx, logger)
		},
		config.AzKeyVaultSecretsPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
			return store.NewKvScrtStore(ctx, token, logger)
		},
		config.HashicorpVaultPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
			return store.NewVaultStore(ctx, token, logger)
		},
	}
}

type strategyFnMap struct {
	mu      sync.Mutex
	funcMap StrategyFuncMap
}
type RetrieveStrategy struct {
	implementation  store.Strategy
	config          config.GenVarsConfig
	strategyFuncMap strategyFnMap
}
type Opts func(*RetrieveStrategy)

// New
func New(config config.GenVarsConfig, logger log.ILogger, opts ...Opts) *RetrieveStrategy {
	rs := &RetrieveStrategy{
		config:          config,
		strategyFuncMap: strategyFnMap{mu: sync.Mutex{}, funcMap: defaultStrategyFuncMap(logger)},
	}
	// overwrite or add any options/defaults set above
	for _, o := range opts {
		o(rs)
	}

	return rs
}

// WithStrategyFuncMap Adds custom implementations for prefix
//
// Mainly used for testing
// NOTE: this may lead to eventual optional configurations by users
func WithStrategyFuncMap(funcMap StrategyFuncMap) Opts {
	return func(rs *RetrieveStrategy) {
		for prefix, implementation := range funcMap {
			rs.strategyFuncMap.mu.Lock()
			defer rs.strategyFuncMap.mu.Unlock()
			rs.strategyFuncMap.funcMap[config.ImplementationPrefix(prefix)] = implementation
		}
	}
}

func (rs *RetrieveStrategy) setImplementation(strategy store.Strategy) {
	rs.implementation = strategy
}

func (rs *RetrieveStrategy) setTokenVal(s *config.ParsedTokenConfig) {
	rs.implementation.SetToken(s)
}

func (rs *RetrieveStrategy) getTokenValue() (string, error) {
	return rs.implementation.Token()
}

type TokenResponse struct {
	value string
	key   *config.ParsedTokenConfig
	Err   error
}

func (tr *TokenResponse) Key() *config.ParsedTokenConfig {
	return tr.key
}

func (tr *TokenResponse) Value() string {
	return tr.value
}

// retrieveSpecificCh wraps around the specific strategy implementation
// and publishes results to a channel
func (rs *RetrieveStrategy) RetrieveByToken(ctx context.Context, impl store.Strategy, tokenConf *config.ParsedTokenConfig) *TokenResponse {
	cr := &TokenResponse{}
	cr.Err = nil
	cr.key = tokenConf
	rs.setImplementation(impl)
	rs.setTokenVal(tokenConf)
	s, err := rs.getTokenValue()
	if err != nil {
		cr.Err = err
		return cr
	}
	cr.value = s
	return cr
}

func (rs *RetrieveStrategy) SelectImplementation(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
	if token == nil {
		return nil, fmt.Errorf("unable to get prefix, %w", ErrTokenInvalid)
	}

	if store, found := rs.strategyFuncMap.funcMap[token.Prefix()]; found {
		return store(ctx, token)
	}

	return nil, fmt.Errorf("implementation not found for input string: %s", token)
}
