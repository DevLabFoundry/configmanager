package store

import (
	"context"

	"github.com/DevLabFoundry/configmanager/internal/config"
	"github.com/DevLabFoundry/configmanager/pkg/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConf "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type paramStoreApi interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

type ParamStore struct {
	svc    paramStoreApi
	ctx    context.Context
	logger log.ILogger
	config *ParamStrConfig
	token  *config.ParsedTokenConfig
}

type ParamStrConfig struct {
	// reserved for potential future use
}

func NewParamStore(ctx context.Context, logger log.ILogger) (*ParamStore, error) {
	cfg, err := awsConf.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("unable to load SDK config, %v\n%w", err, ErrClientInitialization)
		return nil, err
	}
	c := ssm.NewFromConfig(cfg)

	return &ParamStore{
		svc:    c,
		logger: logger,
		ctx:    ctx,
	}, nil
}

func (imp *ParamStore) SetToken(token *config.ParsedTokenConfig) {
	storeConf := &ParamStrConfig{}
	token.ParseMetadata(storeConf)
	imp.token = token
	imp.config = storeConf
}

func (imp *ParamStore) Token() (string, error) {
	imp.logger.Info("%s", "Concrete implementation ParameterStore")
	imp.logger.Info("ParamStore Token: %s", imp.token.String())

	input := &ssm.GetParameterInput{
		Name:           aws.String(imp.token.StoreToken()),
		WithDecryption: aws.Bool(true),
	}
	ctx, cancel := context.WithCancel(imp.ctx)
	defer cancel()

	result, err := imp.svc.GetParameter(ctx, input)
	if err != nil {
		imp.logger.Error(implementationNetworkErr, config.ParamStorePrefix, err, imp.token.StoreToken())
		return "", err
	}

	if result.Parameter.Value != nil {
		return *result.Parameter.Value, nil
	}
	imp.logger.Error("value retrieved but empty for token: %v", imp.token)
	return "", nil
}
