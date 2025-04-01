package store

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/DevLabFoundry/configmanager/internal/config"
	"github.com/DevLabFoundry/configmanager/internal/testutils"
	"github.com/DevLabFoundry/configmanager/pkg/log"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type mockSecretsApi func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)

func (m mockSecretsApi) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return m(ctx, params, optFns...)
}

func awsSecretsMgrGetChecker(t *testing.T, params *secretsmanager.GetSecretValueInput) {
	if params.VersionStage == nil {
		t.Fatal("expect name to not be nil")
	}

	if strings.Contains(*params.SecretId, "#") {
		t.Errorf("incorrectly stripped token separator")
	}

	if strings.Contains(*params.SecretId, string(config.SecretMgrPrefix)) {
		t.Errorf("incorrectly stripped prefix")
	}
}

func Test_GetSecretMgr(t *testing.T) {
	t.Parallel()

	tsuccessSecret := "dsgkbdsf"

	tests := map[string]struct {
		token          string
		keySeparator   string
		tokenSeparator string
		expect         string
		mockClient     func(t *testing.T) secretsMgrApi
		config         *config.GenVarsConfig
	}{
		"success": {"AWSSECRETS#/token/1", "|", "#", tsuccessSecret, func(t *testing.T) secretsMgrApi {
			return mockSecretsApi(func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				t.Helper()
				awsSecretsMgrGetChecker(t, params)
				return &secretsmanager.GetSecretValueOutput{
					SecretString: &tsuccessSecret,
				}, nil
			})
		}, config.NewConfig(),
		},
		"success with version": {"AWSSECRETS#/token/1[version=123]", "|", "#", tsuccessSecret, func(t *testing.T) secretsMgrApi {
			return mockSecretsApi(func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				t.Helper()
				awsSecretsMgrGetChecker(t, params)
				return &secretsmanager.GetSecretValueOutput{
					SecretString: &tsuccessSecret,
				}, nil
			})
		}, config.NewConfig(),
		},
		"success with binary": {"AWSSECRETS#/token/1", "|", "#", tsuccessSecret, func(t *testing.T) secretsMgrApi {
			return mockSecretsApi(func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				t.Helper()
				awsSecretsMgrGetChecker(t, params)
				return &secretsmanager.GetSecretValueOutput{
					SecretBinary: []byte(tsuccessSecret),
				}, nil
			})
		}, config.NewConfig(),
		},
		"errored": {"AWSSECRETS#/token/1", "|", "#", "unable to retrieve secret", func(t *testing.T) secretsMgrApi {
			return mockSecretsApi(func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				t.Helper()
				awsSecretsMgrGetChecker(t, params)
				return nil, fmt.Errorf("unable to retrieve secret")
			})
		}, config.NewConfig(),
		},
		"ok but empty": {"AWSSECRETS#/token/1", "|", "#", "", func(t *testing.T) secretsMgrApi {
			return mockSecretsApi(func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				t.Helper()
				awsSecretsMgrGetChecker(t, params)
				return &secretsmanager.GetSecretValueOutput{
					SecretString: nil,
				}, nil
			})
		}, config.NewConfig(),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			token, _ := config.NewParsedTokenConfig(tt.token, *tt.config.WithTokenSeparator(tt.tokenSeparator).WithKeySeparator(tt.keySeparator))

			impl, _ := NewSecretsMgr(context.TODO(), log.New(io.Discard))
			impl.svc = tt.mockClient(t)

			impl.SetToken(token)
			got, err := impl.Token()
			if err != nil {
				if err.Error() != tt.expect {
					t.Errorf(testutils.TestPhrase, err.Error(), tt.expect)
				}
				return
			}
			if got != tt.expect {
				t.Errorf(testutils.TestPhrase, got, tt.expect)
			}
		})
	}
}
