package store_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/DevLabFoundry/configmanager/v3/internal/store"
	"github.com/DevLabFoundry/configmanager/v3/internal/testutils"
)

func Test_azSplitToken(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		expect store.AzServiceHelper
	}{
		{
			name:  "simple_with_preceding_slash",
			token: "/test-vault/somejsontest",
			expect: store.AzServiceHelper{
				ServiceUri: "https://test-vault.vault.azure.net",
				Token:      "somejsontest",
			},
		},
		{
			name:  "missing_initial_slash",
			token: "test-vault/somejsontest",
			expect: store.AzServiceHelper{
				ServiceUri: "https://test-vault.vault.azure.net",
				Token:      "somejsontest",
			},
		},
		{
			name:  "missing_initial_slash_multislash_secretname",
			token: "test-vault/some/json/test",
			expect: store.AzServiceHelper{
				ServiceUri: "https://test-vault.vault.azure.net",
				Token:      "some/json/test",
			},
		},
		{
			name:  "with_initial_slash_multislash_secretname",
			token: "test-vault//some/json/test",
			expect: store.AzServiceHelper{
				ServiceUri: "https://test-vault.vault.azure.net",
				Token:      "/some/json/test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := store.AzServiceFromToken(tt.token, "https://%s.vault.azure.net", 1)
			if got.Token != tt.expect.Token {
				t.Errorf(testutils.TestPhrase, tt.expect.Token, got.Token)
			}
			if got.ServiceUri != tt.expect.ServiceUri {
				t.Errorf(testutils.TestPhrase, tt.expect.ServiceUri, got.ServiceUri)
			}
		})
	}
}

func azKvCommonGetSecretChecker(t *testing.T, name, version, expectedName string) {
	if name == "" {
		t.Errorf("expect name to not be nil")
	}
	if name != expectedName {
		t.Errorf(testutils.TestPhrase, name, expectedName)
	}

	if strings.Contains(name, "#") {
		t.Errorf("incorrectly stripped token separator")
	}

	if strings.Contains(name, string(config.AzKeyVaultSecretsPrefix)) {
		t.Errorf("incorrectly stripped prefix")
	}

	if version != "" {
		t.Fatal("expect version to be \"\" an empty string ")
	}
}

type mockAzKvSecretApi func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error)

func (m mockAzKvSecretApi) GetSecret(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
	return m(ctx, name, version, options)
}

func TestAzKeyVault(t *testing.T) {
	tsuccessParam := "dssdfdweiuyh"
	tests := map[string]struct {
		token      func() *config.ParsedTokenConfig
		expect     string
		mockClient func(t *testing.T) mockAzKvSecretApi
	}{
		"successVal": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.AzKeyVaultSecretsPrefix, *config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/test-vault//token/1")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			},
			tsuccessParam, func(t *testing.T) mockAzKvSecretApi {
				return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
					t.Helper()
					azKvCommonGetSecretChecker(t, name, "", "/token/1")
					resp := azsecrets.GetSecretResponse{}
					resp.Value = &tsuccessParam
					return resp, nil
				})
			},
		},
		"successVal with version": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.AzKeyVaultSecretsPrefix, *config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/test-vault//token/1")
				tkn.WithKeyPath("")
				tkn.WithMetadata("version:123")
				return tkn
			}, tsuccessParam, func(t *testing.T) mockAzKvSecretApi {
				return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
					t.Helper()
					azKvCommonGetSecretChecker(t, name, "", "/token/1")
					resp := azsecrets.GetSecretResponse{}
					resp.Value = &tsuccessParam
					return resp, nil
				})
			},
		},
		"successVal with keyseparator": {
			func() *config.ParsedTokenConfig {
				// "AZKVSECRET#/test-vault/token/1|somekey"
				tkn, _ := config.NewToken(config.AzKeyVaultSecretsPrefix, *config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/test-vault/token/1")
				tkn.WithKeyPath("somekey")
				tkn.WithMetadata("")
				return tkn
			}, tsuccessParam, func(t *testing.T) mockAzKvSecretApi {
				return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
					t.Helper()
					azKvCommonGetSecretChecker(t, name, "", "token/1")

					resp := azsecrets.GetSecretResponse{}
					resp.Value = &tsuccessParam
					return resp, nil
				})
			},
		},
		"errored": {
			func() *config.ParsedTokenConfig {
				// "AZKVSECRET#/test-vault/token/1|somekey"
				tkn, _ := config.NewToken(config.AzKeyVaultSecretsPrefix, *config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/test-vault/token/1")
				tkn.WithKeyPath("somekey")
				tkn.WithMetadata("")
				return tkn
			},
			"unable to retrieve secret",
			func(t *testing.T) mockAzKvSecretApi {
				return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
					t.Helper()
					azKvCommonGetSecretChecker(t, name, "", "token/1")

					resp := azsecrets.GetSecretResponse{}
					return resp, fmt.Errorf("unable to retrieve secret")
				})
			},
		},
		"empty": {
			func() *config.ParsedTokenConfig {
				// "AZKVSECRET#/test-vault/token/1|somekey"
				tkn, _ := config.NewToken(config.AzKeyVaultSecretsPrefix, *config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/test-vault/token/1")
				tkn.WithKeyPath("somekey")
				tkn.WithMetadata("")
				return tkn
			}, "", func(t *testing.T) mockAzKvSecretApi {
				return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
					t.Helper()
					azKvCommonGetSecretChecker(t, name, "", "token/1")

					resp := azsecrets.GetSecretResponse{}
					return resp, nil
				})
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			impl, err := store.NewKvScrtStore(context.TODO(), tt.token(), log.New(io.Discard))
			if err != nil {
				t.Errorf("failed to init azkvstore")
			}

			impl.WithSvc(tt.mockClient(t))
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
