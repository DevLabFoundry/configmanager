/**
 * Azure KeyVault implementation
**/
package generator

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"

	"github.com/dnitsch/configmanager/pkg/log"
)

type kvApi interface {
	GetSecret(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error)
}

type KvScrtStore struct {
	svc   kvApi
	ctx   context.Context
	token string
}

// azVaultHelper provides a broken up string
type azVaultHelper struct {
	vaultUri string
	token    string
}

// NewKvScrtStore returns a KvScrtStore
// requires `AZURE_SUBSCRIPTION_ID` environment variable to be present to successfully work
func NewKvScrtStore(ctx context.Context) (*KvScrtStore, error) {
	return &KvScrtStore{
		ctx: ctx,
	}, nil
}

// NewKvScrtStoreWithToken returns a KvScrtStore
// requires `AZURE_SUBSCRIPTION_ID` environment variable to be present to successfully work
func NewKvScrtStoreWithToken(ctx context.Context, token, tokenSeparator, keySeparator string) (*KvScrtStore, error) {

	//
	vc := azSplitToken(stripPrefix(token, AzKeyVaultSecretsPrefix, tokenSeparator, keySeparator))

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	c, err := azsecrets.NewClient(vc.vaultUri, cred, nil)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &KvScrtStore{
		svc:   c,
		ctx:   ctx,
		token: vc.token,
	}, nil
}

// setToken already happens in AzureKVClient in the constructor
// no need to re-set it here
func (implmt *KvScrtStore) setToken(token string) {
}

func (implmt *KvScrtStore) setValue(val string) {
}

func (imp *KvScrtStore) getTokenValue(v *retrieveStrategy) (string, error) {
	log.Infof("%s", "Concrete implementation AzKeyVault Secret")
	log.Infof("AzKeyVault Token: %s", imp.token)

	ctx, cancel := context.WithCancel(imp.ctx)
	defer cancel()

	// secretVersion as "" => latest
	s, err := imp.svc.GetSecret(ctx, imp.token, "", nil)
	if err != nil {
		log.Errorf("AzKeyVault: %v", err)
		return "", err
	}
	if s.Value != nil {
		return *s.Value, nil
	}
	log.Errorf("value retrieved but empty for token: %v", imp.token)
	return "", nil
}

func azSplitToken(token string) azVaultHelper {
	// ensure preceding slash is trimmed
	splitToken := strings.Split(strings.TrimPrefix(token, "/"), "/")
	vaultUri := fmt.Sprintf("https://%s.vault.azure.net", splitToken[0])
	return azVaultHelper{vaultUri: vaultUri, token: strings.Join(splitToken[1:], "/")}
}
