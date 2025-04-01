package store

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/DevLabFoundry/configmanager/internal/config"
	"github.com/DevLabFoundry/configmanager/internal/testutils"
	"github.com/DevLabFoundry/configmanager/pkg/log"
)

func Test_azSplitToken(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		expect azServiceHelper
	}{
		{
			name:  "simple_with_preceding_slash",
			token: "/test-vault/somejsontest",
			expect: azServiceHelper{
				serviceUri: "https://test-vault.vault.azure.net",
				token:      "somejsontest",
			},
		},
		{
			name:  "missing_initial_slash",
			token: "test-vault/somejsontest",
			expect: azServiceHelper{
				serviceUri: "https://test-vault.vault.azure.net",
				token:      "somejsontest",
			},
		},
		{
			name:  "missing_initial_slash_multislash_secretname",
			token: "test-vault/some/json/test",
			expect: azServiceHelper{
				serviceUri: "https://test-vault.vault.azure.net",
				token:      "some/json/test",
			},
		},
		{
			name:  "with_initial_slash_multislash_secretname",
			token: "test-vault//some/json/test",
			expect: azServiceHelper{
				serviceUri: "https://test-vault.vault.azure.net",
				token:      "/some/json/test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := azServiceFromToken(tt.token, "https://%s.vault.azure.net", 1)
			if got.token != tt.expect.token {
				t.Errorf(testutils.TestPhrase, tt.expect.token, got.token)
			}
			if got.serviceUri != tt.expect.serviceUri {
				t.Errorf(testutils.TestPhrase, tt.expect.serviceUri, got.serviceUri)
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
	t.Parallel()

	tsuccessParam := "dssdfdweiuyh"
	tests := map[string]struct {
		token      string
		expect     string
		mockClient func(t *testing.T) kvApi
		config     *config.GenVarsConfig
	}{
		"successVal": {"AZKVSECRET#/test-vault//token/1", tsuccessParam, func(t *testing.T) kvApi {
			return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
				t.Helper()
				azKvCommonGetSecretChecker(t, name, "", "/token/1")
				resp := azsecrets.GetSecretResponse{}
				resp.Value = &tsuccessParam
				return resp, nil
			})
		}, config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"),
		},
		"successVal with version": {"AZKVSECRET#/test-vault//token/1[version:123]", tsuccessParam, func(t *testing.T) kvApi {
			return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
				t.Helper()
				azKvCommonGetSecretChecker(t, name, "", "/token/1")
				resp := azsecrets.GetSecretResponse{}
				resp.Value = &tsuccessParam
				return resp, nil
			})
		}, config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"),
		},
		"successVal with keyseparator": {"AZKVSECRET#/test-vault/token/1|somekey", tsuccessParam, func(t *testing.T) kvApi {
			return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
				t.Helper()
				azKvCommonGetSecretChecker(t, name, "", "token/1")

				resp := azsecrets.GetSecretResponse{}
				resp.Value = &tsuccessParam
				return resp, nil
			})
		},
			config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"),
		},
		"errored": {"AZKVSECRET#/test-vault/token/1|somekey", "unable to retrieve secret", func(t *testing.T) kvApi {
			return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
				t.Helper()
				azKvCommonGetSecretChecker(t, name, "", "token/1")

				resp := azsecrets.GetSecretResponse{}
				return resp, fmt.Errorf("unable to retrieve secret")
			})
		},
			config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"),
		},
		"empty": {"AZKVSECRET#/test-vault/token/1|somekey", "", func(t *testing.T) kvApi {
			return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
				t.Helper()
				azKvCommonGetSecretChecker(t, name, "", "token/1")

				resp := azsecrets.GetSecretResponse{}
				return resp, nil
			})
		},
			config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			token, _ := config.NewParsedTokenConfig(tt.token, *tt.config)

			impl, err := NewKvScrtStore(context.TODO(), token, log.New(io.Discard))
			if err != nil {
				t.Errorf("failed to init azkvstore")
			}

			impl.svc = tt.mockClient(t)
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
