/**
 * Azure TableStore implementation
**/
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/log"
)

var ErrIncorrectlyStructuredToken = errors.New("incorrectly structured token")

// tableStoreApi
// uses this package https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/data/aztables
type tableStoreApi interface {
	GetEntity(ctx context.Context, partitionKey string, rowKey string, options *aztables.GetEntityOptions) (aztables.GetEntityResponse, error)
}

type AzTableStore struct {
	svc    tableStoreApi
	ctx    context.Context
	logger log.ILogger
	config *AzTableStrgConfig
	token  *config.ParsedTokenConfig
	// token only without table indicators
	// key only
	strippedToken string
}

type AzTableStrgConfig struct {
	Format string `json:"format"`
}

// NewAzTableStore
func NewAzTableStore(ctx context.Context, token *config.ParsedTokenConfig, logger log.ILogger) (*AzTableStore, error) {

	storeConf := &AzTableStrgConfig{}
	_ = token.ParseMetadata(storeConf)
	// initialToken := config.ParseMetadata(token, storeConf)
	backingStore := &AzTableStore{
		ctx:    ctx,
		logger: logger,
		config: storeConf,
		token:  token,
	}

	srvInit := AzServiceFromToken(token.StoreToken(), "https://%s.table.core.windows.net/%s", 2)
	backingStore.strippedToken = srvInit.Token

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error("failed to get credentials: %v", err)
		return nil, err
	}

	c, err := aztables.NewClient(srvInit.ServiceUri, cred, nil)
	if err != nil {
		logger.Error("failed to init the client: %v", err)
		return nil, fmt.Errorf("%v\n%w", err, ErrClientInitialization)
	}

	backingStore.svc = c
	return backingStore, nil
}

func (s *AzTableStore) WithSvc(svc tableStoreApi) {
	s.svc = svc
}

// setToken already happens in the constructor
func (implmt *AzTableStore) SetToken(token *config.ParsedTokenConfig) {}

// tokenVal in AZ table storage if an Entity contains the `value` property
// we attempt to extract it and return.
//
// From this point then normal rules of configmanager apply,
// including keySeperator and lookup.
func (imp *AzTableStore) Value() (string, error) {
	imp.logger.Info("AzTableSTore Token: %s", imp.token.String())
	imp.logger.Info("Concrete implementation AzTableSTore")

	ctx, cancel := context.WithCancel(imp.ctx)
	defer cancel()

	// split the token for partition and rowKey
	pKey, rKey, err := azTableStoreTokenSplitter(imp.strippedToken)
	if err != nil {
		return "", err
	}

	s, err := imp.svc.GetEntity(ctx, pKey, rKey, &aztables.GetEntityOptions{})
	if err != nil {
		imp.logger.Error(implementationNetworkErr, config.AzTableStorePrefix, err, imp.strippedToken)
		return "", fmt.Errorf(implementationNetworkErr+" %w", config.AzTableStorePrefix, err, imp.token.StoreToken(), ErrRetrieveFailed)
	}
	if len(s.Value) > 0 {
		// check for `value` property in entity
		checkVal := make(map[string]interface{})
		_ = json.Unmarshal(s.Value, &checkVal)
		if checkVal["value"] != nil {
			return fmt.Sprintf("%v", checkVal["value"]), nil
		}
		return string(s.Value), nil
	}
	imp.logger.Error("value retrieved but empty for token: %v", imp.token)
	return "", nil
}

func azTableStoreTokenSplitter(token string) (partitionKey, rowKey string, err error) {
	splitToken := strings.Split(strings.TrimPrefix(token, "/"), "/")
	if len(splitToken) < 2 {
		return "", "", fmt.Errorf("token: %s - could not be correctly destructured to pluck the partition and row keys\n%w", token, ErrIncorrectlyStructuredToken)
	}
	partitionKey = splitToken[0]
	rowKey = splitToken[1]
	// naked return to save having to define another struct
	return
}
