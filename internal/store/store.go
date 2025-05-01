package store

import (
	"errors"
	"fmt"

	"github.com/DevLabFoundry/configmanager/v2/internal/config"
)

const implementationNetworkErr string = "implementation %s error: %v for token: %s"

var (
	ErrRetrieveFailed       = errors.New("failed to retrieve config item")
	ErrClientInitialization = errors.New("failed to initialize the client")
	ErrEmptyResponse        = errors.New("value retrieved but empty for token")
	ErrServiceCallFailed    = errors.New("failed to complete the service call")
)

// Strategy iface that all store implementations
// must conform to, in order to be be used by the retrieval implementation
type Strategy interface {
	Token() (s string, e error)
	SetToken(s *config.ParsedTokenConfig)
}

type DefaultStrategy struct {
}

func NewDefatultStrategy() *DefaultStrategy {
	return &DefaultStrategy{}
}

// SetToken on default strategy
func (implmt *DefaultStrategy) SetToken(token *config.ParsedTokenConfig) {}

// Token
func (implmt *DefaultStrategy) Token() (string, error) {
	return "", fmt.Errorf("default strategy does not implement token retrieval")
}
