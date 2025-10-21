/**
 * Azure App Config implementation
**/
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig"
	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/internal/log"
)

// appConfApi
// uses this package https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig
type appConfApi interface {
	GetSetting(ctx context.Context, key string, options *azappconfig.GetSettingOptions) (azappconfig.GetSettingResponse, error)
}

type AzAppConf struct {
	svc           appConfApi
	ctx           context.Context
	config        *AzAppConfConfig
	token         *config.ParsedTokenConfig
	strippedToken string
	logger        log.ILogger
}

// AzAppConfConfig is the azure conf service specific config
// and it is parsed from the token metadata
type AzAppConfConfig struct {
	Label          string       `json:"label"`
	Etag           *azcore.ETag `json:"etag"`
	AcceptDateTime *time.Time   `json:"acceptedDateTime"`
}

// NewAzAppConf
func NewAzAppConf(ctx context.Context, token *config.ParsedTokenConfig, logger log.ILogger) (*AzAppConf, error) {
	storeConf := &AzAppConfConfig{}
	if err := token.ParseMetadata(storeConf); err != nil {
		return nil, err
	}

	backingStore := &AzAppConf{
		ctx:    ctx,
		config: storeConf,
		token:  token,
		logger: logger,
	}
	srvInit := AzServiceFromToken(token.StoreToken(), "https://%s.azconfig.io", 1)
	backingStore.strippedToken = srvInit.Token

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error("failed to get credentials: %v", err)
		return nil, err
	}

	c, err := azappconfig.NewClient(srvInit.ServiceUri, cred, nil)
	if err != nil {
		logger.Error("failed to init the client: %v", err)
		return nil, fmt.Errorf("%v\n%w", err, ErrClientInitialization)
	}

	backingStore.svc = c
	return backingStore, nil

}

func (s *AzAppConf) WithSvc(svc appConfApi) {
	s.svc = svc
}

// setTokenVal sets the token
func (implmt *AzAppConf) SetToken(token *config.ParsedTokenConfig) {}

// tokenVal in AZ App Config
// label can be specified
// From this point then normal rules of configmanager apply,
// including keySeperator and lookup.
func (imp *AzAppConf) Token() (string, error) {
	imp.logger.Info("Concrete implementation AzAppConf")
	imp.logger.Info("AzAppConf Token: %s", imp.token.String())

	ctx, cancel := context.WithCancel(imp.ctx)
	defer cancel()
	opts := &azappconfig.GetSettingOptions{}

	// assign any metadatas from the token
	if imp.config.Label != "" {
		opts.Label = &imp.config.Label
	}

	if imp.config.Etag != nil {
		opts.OnlyIfChanged = imp.config.Etag
	}

	s, err := imp.svc.GetSetting(ctx, imp.strippedToken, opts)
	if err != nil {
		imp.logger.Error(implementationNetworkErr, config.AzAppConfigPrefix, err, imp.strippedToken)
		return "", fmt.Errorf("token: %s, error: %v. %w", imp.strippedToken, err, ErrRetrieveFailed)
	}
	if s.Value != nil {
		return *s.Value, nil
	}
	imp.logger.Error("token: %v, %w", imp.token.String(), ErrEmptyResponse)
	return "", nil
}
