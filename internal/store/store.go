package store

import (
	"errors"

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
//
// Defined on the package for easier re-use across the program
type Strategy interface {
	Token() (s string, e error)
	SetToken(s *config.ParsedTokenConfig)
}
