package plugins

import "errors"

// Error codes shared with the host.
const (
	OK                           int32 = 0
	ERR_BUF_TOO_SMALL            int32 = -1
	ERR_INVALID_UTF8             int32 = -2
	ERR_EMPTY_INPUT              int32 = -3
	ERR_INTERNAL                 int32 = -4
	ERR_FAILED_UNMARSHAL_MESSAGE int32 = -5
	ERR_INIT_STORE               int32 = -6
	ERR_FAILED_VALUE_RETRIEVAL   int32 = -7
)

const ImplementationNetworkErr string = "implementation %s error: %v for token: %s"

var (
	ErrRetrieveFailed       = errors.New("failed to retrieve config item")
	ErrClientInitialization = errors.New("failed to initialize the client")
	ErrEmptyResponse        = errors.New("value retrieved but empty for token")
	ErrServiceCallFailed    = errors.New("failed to complete the service call")
)
