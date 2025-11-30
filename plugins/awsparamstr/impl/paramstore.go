package impl

import (
	"context"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/DevLabFoundry/configmanager/v3/plugins"
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
	config *ParamStrConfig
	token  *config.ParsedTokenConfig
	logger log.ILogger
}

type ParamStrConfig struct {
	// reserved for potential future use
}

func NewParamStore(ctx context.Context, logger log.ILogger) (*ParamStore, error) {
	cfg, err := awsConf.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	c := ssm.NewFromConfig(cfg)

	return &ParamStore{
		svc:    c,
		logger: logger,
		ctx:    ctx,
	}, nil
}

func (s *ParamStore) WithSvc(svc paramStoreApi) {
	s.svc = svc
}

func (imp *ParamStore) Value(token string, metadata []byte) (string, error) {
	imp.logger.Info("Concrete implementation ParameterStore")
	imp.logger.Info("ParamStore Token: %s", token)

	input := &ssm.GetParameterInput{
		Name:           aws.String(token),
		WithDecryption: aws.Bool(true),
	}
	ctx, cancel := context.WithCancel(imp.ctx)
	defer cancel()

	result, err := imp.svc.GetParameter(ctx, input)
	if err != nil {
		imp.logger.Error(plugins.ImplementationNetworkErr, config.ParamStorePrefix, err, token)
		return "", err
	}

	if result.Parameter.Value != nil {
		return *result.Parameter.Value, nil
	}
	imp.logger.Error("value retrieved but empty for token: %v", imp.token)
	return "", nil
}
