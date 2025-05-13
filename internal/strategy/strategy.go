package strategy

import (
	"context"
	"errors"
	"fmt"

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

type RetrieveStrategy struct {
	implementation  store.Strategy
	config          config.GenVarsConfig
	strategyFuncMap StrategyFuncMap
	token           string
}

// New
func New(s store.Strategy, config config.GenVarsConfig, logger log.ILogger) *RetrieveStrategy {
	rs := &RetrieveStrategy{
		implementation:  s,
		config:          config,
		strategyFuncMap: defaultStrategyFuncMap(logger),
	}
	return rs
}

// WithStrategyFuncMap Adds custom implementations for prefix
//
// Mainly used for testing
// NOTE: this may lead to eventual optional configurations by users
func (rs *RetrieveStrategy) WithStrategyFuncMap(funcMap StrategyFuncMap) *RetrieveStrategy {
	for prefix, implementation := range funcMap {
		rs.strategyFuncMap[config.ImplementationPrefix(prefix)] = implementation
	}
	return rs
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

	if store, found := rs.strategyFuncMap[token.Prefix()]; found {
		return store(ctx, token)
	}

	return nil, fmt.Errorf("implementation not found for input string: %s", token)
}
