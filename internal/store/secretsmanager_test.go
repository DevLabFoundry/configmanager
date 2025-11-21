package store_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/DevLabFoundry/configmanager/v3/internal/store"
	"github.com/DevLabFoundry/configmanager/v3/internal/testutils"
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

	tsuccessSecret := "dsgkbdsf"

	tests := map[string]struct {
		token      func() *config.ParsedTokenConfig
		expect     string
		mockClient func(t *testing.T) mockSecretsApi
	}{
		"success": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.SecretMgrPrefix, *config.NewConfig())
				tkn.WithSanitizedToken("/token/1")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			}, tsuccessSecret, func(t *testing.T) mockSecretsApi {
				return mockSecretsApi(func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
					t.Helper()
					awsSecretsMgrGetChecker(t, params)
					return &secretsmanager.GetSecretValueOutput{
						SecretString: &tsuccessSecret,
					}, nil
				})
			},
		},
		"success with version": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.SecretMgrPrefix, *config.NewConfig().WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/token/1")
				tkn.WithKeyPath("")
				tkn.WithMetadata("version=123")
				return tkn
			},
			tsuccessSecret, func(t *testing.T) mockSecretsApi {
				return mockSecretsApi(func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
					t.Helper()
					awsSecretsMgrGetChecker(t, params)
					return &secretsmanager.GetSecretValueOutput{
						SecretString: &tsuccessSecret,
					}, nil
				})
			},
		},
		"success with binary": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.SecretMgrPrefix, *config.NewConfig().WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/token/1")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			},
			tsuccessSecret, func(t *testing.T) mockSecretsApi {
				return mockSecretsApi(func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
					t.Helper()
					awsSecretsMgrGetChecker(t, params)
					return &secretsmanager.GetSecretValueOutput{
						SecretBinary: []byte(tsuccessSecret),
					}, nil
				})
			},
		},
		"errored": {
			func() *config.ParsedTokenConfig {
				// "AWSSECRETS#/token/1", "|", "#",
				tkn, _ := config.NewToken(config.SecretMgrPrefix, *config.NewConfig().WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/token/1")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			},
			"unable to retrieve secret", func(t *testing.T) mockSecretsApi {
				return mockSecretsApi(func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
					t.Helper()
					awsSecretsMgrGetChecker(t, params)
					return nil, fmt.Errorf("unable to retrieve secret")
				})
			},
		},
		"ok but empty": {
			func() *config.ParsedTokenConfig {
				// "AWSSECRETS#/token/1", "|", "#",
				tkn, _ := config.NewToken(config.SecretMgrPrefix, *config.NewConfig().WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/token/1")
				tkn.WithKeyPath("")
				tkn.WithMetadata("version=123")
				return tkn
			},
			"", func(t *testing.T) mockSecretsApi {
				return mockSecretsApi(func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
					t.Helper()
					awsSecretsMgrGetChecker(t, params)
					return &secretsmanager.GetSecretValueOutput{
						SecretString: nil,
					}, nil
				})
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			impl, _ := store.NewSecretsMgr(context.TODO(), log.New(io.Discard))
			impl.WithSvc(tt.mockClient(t))

			impl.SetToken(tt.token())
			got, err := impl.Value()
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
