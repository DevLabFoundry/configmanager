package store

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConf "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/dnitsch/configmanager/internal/config"
	"github.com/dnitsch/configmanager/pkg/log"
)

type secretsMgrApi interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

type SecretsMgr struct {
	svc    secretsMgrApi
	ctx    context.Context
	logger log.ILogger
	config *SecretsMgrConfig
	token  *config.ParsedTokenConfig
}

type SecretsMgrConfig struct {
	Version string `json:"version"`
}

func NewSecretsMgr(ctx context.Context, logger log.ILogger) (*SecretsMgr, error) {
	cfg, err := awsConf.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("unable to load SDK config, %v\n%w", err, ErrClientInitialization)
		return nil, err
	}
	c := secretsmanager.NewFromConfig(cfg)

	return &SecretsMgr{
		svc:    c,
		logger: logger,
		ctx:    ctx,
	}, nil

}

func (imp *SecretsMgr) SetToken(token *config.ParsedTokenConfig) {
	storeConf := &SecretsMgrConfig{}
	token.ParseMetadata(storeConf)
	imp.token = token
	imp.config = storeConf
}

func (imp *SecretsMgr) Token() (string, error) {
	imp.logger.Info("Concrete implementation SecretsManager")
	imp.logger.Info("SecretsManager Token: %s", imp.token.String())

	version := "AWSCURRENT"
	if imp.config.Version != "" {
		version = imp.config.Version
	}

	imp.logger.Info("Getting Secret: %s @version: %s", imp.token, version)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(imp.token.StoreToken()),
		VersionStage: aws.String(version),
	}

	ctx, cancel := context.WithCancel(imp.ctx)
	defer cancel()

	result, err := imp.svc.GetSecretValue(ctx, input)
	if err != nil {
		imp.logger.Error(implementationNetworkErr, imp.token.Prefix(), err, imp.token.String())
		return "", err
	}

	if result.SecretString != nil {
		return *result.SecretString, nil
	}

	if len(result.SecretBinary) > 0 {
		return string(result.SecretBinary), nil
	}

	imp.logger.Error("value retrieved but empty for token: %v", imp.token)
	return "", nil
}
