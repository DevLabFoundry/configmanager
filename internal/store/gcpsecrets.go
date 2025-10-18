package store

import (
	"context"
	"fmt"

	gcpsecrets "cloud.google.com/go/secretmanager/apiv1"
	gcpsecretspb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/internal/log"
	"github.com/googleapis/gax-go/v2"
)

type gcpSecretsApi interface {
	AccessSecretVersion(ctx context.Context, req *gcpsecretspb.AccessSecretVersionRequest, opts ...gax.CallOption) (*gcpsecretspb.AccessSecretVersionResponse, error)
}

type GcpSecrets struct {
	svc    gcpSecretsApi
	logger log.ILogger
	ctx    context.Context
	config *GcpSecretsConfig
	close  func() error
	token  *config.ParsedTokenConfig
}

type GcpSecretsConfig struct {
	Version string `json:"version"`
}

func NewGcpSecrets(ctx context.Context, logger log.ILogger) (*GcpSecrets, error) {

	c, err := gcpsecrets.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GcpSecrets{
		svc:    c,
		logger: logger,
		ctx:    ctx,
		close:  c.Close,
	}, nil
}

func (imp *GcpSecrets) SetToken(token *config.ParsedTokenConfig) {
	storeConf := &GcpSecretsConfig{}
	_ = token.ParseMetadata(storeConf)
	imp.token = token
	imp.config = storeConf
}

func (imp *GcpSecrets) Token() (string, error) {
	// Close client currently as new one would be created per iteration
	defer func() {
		_ = imp.close()
	}()

	imp.logger.Info("Concrete implementation GcpSecrets")
	imp.logger.Info("GcpSecrets Token: %s", imp.token.String())

	version := "latest"
	if imp.config.Version != "" {
		version = imp.config.Version
	}

	imp.logger.Info("Getting Secret: %s @version: %s", imp.token, version)

	input := &gcpsecretspb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("%s/versions/%s", imp.token.StoreToken(), version),
	}

	ctx, cancel := context.WithCancel(imp.ctx)
	defer cancel()

	result, err := imp.svc.AccessSecretVersion(ctx, input)

	if err != nil {
		imp.logger.Error(implementationNetworkErr, imp.token.Prefix(), err, imp.token.String())
		return "", err
	}
	if result.Payload != nil {
		return string(result.Payload.Data), nil
	}

	imp.logger.Error("value retrieved but empty for token: %v", imp.token)
	return "", nil
}
