package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/pkg/log"

	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/aws"
)

// vaultHelper provides a broken up string
type vaultHelper struct {
	path  string
	token string
}

type hashiVaultApi interface {
	Get(ctx context.Context, secretPath string) (*vault.KVSecret, error)
	GetVersion(ctx context.Context, secretPath string, version int) (*vault.KVSecret, error)
}

type VaultStore struct {
	svc           hashiVaultApi
	ctx           context.Context
	logger        log.ILogger
	config        *VaultConfig
	token         *config.ParsedTokenConfig
	strippedToken string
}

// VaultConfig holds the parseable metadata struct
type VaultConfig struct {
	Version string `json:"version"`
	Role    string `json:"iam_role"`
}

func NewVaultStore(ctx context.Context, token *config.ParsedTokenConfig, logger log.ILogger) (*VaultStore, error) {
	storeConf := &VaultConfig{}
	token.ParseMetadata(storeConf)
	imp := &VaultStore{
		ctx:    ctx,
		logger: logger,
		config: storeConf,
		token:  token,
	}

	config := vault.DefaultConfig()
	vt := splitToken(token.StoreToken())
	imp.strippedToken = vt.token
	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("%v\n%w", err, ErrClientInitialization)
	}

	if strings.HasPrefix(os.Getenv("VAULT_TOKEN"), "aws_iam") {
		awsclient, err := newVaultStoreWithAWSAuthIAM(client, storeConf.Role)
		if err != nil {
			return nil, err
		}
		client = awsclient
	}
	imp.svc = client.KVv2(vt.path)
	return imp, nil
}

// newVaultStoreWithAWSAuthIAM returns an initialised client with AWSIAMAuth
// EC2 auth type is not supported currently
func newVaultStoreWithAWSAuthIAM(client *vault.Client, role string) (*vault.Client, error) {
	if len(role) < 1 {
		return nil, fmt.Errorf("role provided is empty, EC2 auth not supported")
	}
	awsAuth, err := auth.NewAWSAuth(
		auth.WithRole(role),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize AWS auth method: %s. %w", err, ErrClientInitialization)
	}

	authInfo, err := client.Auth().Login(context.Background(), awsAuth)

	if err != nil {
		return nil, fmt.Errorf("unable to login to AWS auth method: %s. %w", err, ErrClientInitialization)
	}
	if authInfo == nil {
		return nil, fmt.Errorf("no auth info was returned after login")
	}

	return client, nil
}

// setTokenVal
// imp.token is already set in the Vault constructor
//
// This happens inside the New func call
// due to the way the client needs to be
// initialised with a mountpath
// and mountpath is part of the token so it is set then
func (imp *VaultStore) SetToken(token *config.ParsedTokenConfig) {}

// getTokenValue implements the underlying techonology
// token retrieval and returns a stringified version
// of the secret
func (imp *VaultStore) Token() (string, error) {
	imp.logger.Info("%s", "Concrete implementation HashiVault")
	imp.logger.Info("Getting Secret: %s", imp.token)

	ctx, cancel := context.WithCancel(imp.ctx)
	defer cancel()

	secret, err := imp.getSecret(ctx, imp.strippedToken, imp.config.Version)
	if err != nil {
		imp.logger.Error(implementationNetworkErr, imp.token.Prefix(), err, imp.token.String())
		return "", err
	}

	if secret.Data != nil {
		resp, err := marshall(secret.Data)
		if err != nil {
			imp.logger.Error("marshalling error: %s", err.Error())
			return "", err
		}
		imp.logger.Debug("marhalled kvv2: %s", resp)
		return resp, nil
	}

	imp.logger.Error("value retrieved but empty for token: %v", imp.token)
	return "", nil
}

func (imp *VaultStore) getSecret(ctx context.Context, token string, version string) (*vault.KVSecret, error) {
	if version != "" {
		v, err := strconv.Atoi(version)
		if err != nil {
			return nil, fmt.Errorf("unable to parse version into an integer: %s", err.Error())
		}
		return imp.svc.GetVersion(ctx, token, v)
	}
	return imp.svc.Get(ctx, token)
}

func splitToken(token string) vaultHelper {
	vh := vaultHelper{}
	// split token to extract the mount path
	s := strings.Split(strings.TrimPrefix(token, "/"), "___")
	// grab token and trim prefix if slash
	vh.token = strings.TrimPrefix(strings.Join(s[1:], ""), "/")
	// assign mount path as extracted from input token
	vh.path = s[0]
	return vh
}

// marshall converts map[string]any into a JSON
// object. Secrets should only be a single level
// deep.
func marshall(secret map[string]any) (string, error) {
	b, err := json.Marshal(secret)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
