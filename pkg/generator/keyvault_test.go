package generator

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/dnitsch/configmanager/internal/testutils"
)

func Test_azSplitToken(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		expect azVaultHelper
	}{
		{
			name:  "simple_with_preceding_slash",
			token: "/test-vault/somejsontest",
			expect: azVaultHelper{
				vaultUri: "https://test-vault.vault.azure.net",
				token:    "somejsontest",
			},
		},
		{
			name:  "missing_initial_slash",
			token: "test-vault/somejsontest",
			expect: azVaultHelper{
				vaultUri: "https://test-vault.vault.azure.net",
				token:    "somejsontest",
			},
		},
		{
			name:  "missing_initial_slash_multislash_secretname",
			token: "test-vault/some/json/test",
			expect: azVaultHelper{
				vaultUri: "https://test-vault.vault.azure.net",
				token:    "some/json/test",
			},
		},
		{
			name:  "with_initial_slash_multislash_secretname",
			token: "test-vault//some/json/test",
			expect: azVaultHelper{
				vaultUri: "https://test-vault.vault.azure.net",
				token:    "/some/json/test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := azSplitToken(tt.token)
			if got.token != tt.expect.token {
				t.Errorf(testutils.TestPhrase, tt.expect.token, got.token)
			}
			if got.vaultUri != tt.expect.vaultUri {
				t.Errorf(testutils.TestPhrase, tt.expect.vaultUri, got.vaultUri)
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

	if strings.Contains(name, string(AzKeyVaultSecretsPrefix)) {
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

func Test_GetAzKeyVaultSecretVarHappy(t *testing.T) {

	tests := map[string]struct {
		token          string
		keySeparator   string
		tokenSeparator string
		expect         string
		mockClient     func(t *testing.T) kvApi
		config         *GenVarsConfig
	}{
		"successVal": {"AZKVSECRET#/test-vault//token/1", "|", "#", tsuccessParam, func(t *testing.T) kvApi {
			return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
				t.Helper()
				azKvCommonGetSecretChecker(t, name, "", "/token/1")
				resp := azsecrets.GetSecretResponse{}
				resp.Value = &tsuccessParam
				return resp, nil
			})
		}, NewConfig(),
		},
		"successVal with keyseparator": {"AZKVSECRET#/test-vault/token/1|somekey", "|", "#", tsuccessParam, func(t *testing.T) kvApi {
			return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
				t.Helper()
				azKvCommonGetSecretChecker(t, name, "", "token/1")

				resp := azsecrets.GetSecretResponse{}
				resp.Value = &tsuccessParam
				return resp, nil
			})
		},
			NewConfig(),
		},
		"errored": {"AZKVSECRET#/test-vault/token/1|somekey", "|", "#", "unable to retrieve secret", func(t *testing.T) kvApi {
			return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
				t.Helper()
				azKvCommonGetSecretChecker(t, name, "", "token/1")

				resp := azsecrets.GetSecretResponse{}
				return resp, fmt.Errorf("unable to retrieve secret")
			})
		},
			NewConfig(),
		},
		"empty": {"AZKVSECRET#/test-vault/token/1|somekey", "|", "#", "", func(t *testing.T) kvApi {
			return mockAzKvSecretApi(func(ctx context.Context, name string, version string, options *azsecrets.GetSecretOptions) (azsecrets.GetSecretResponse, error) {
				t.Helper()
				azKvCommonGetSecretChecker(t, name, "", "token/1")

				resp := azsecrets.GetSecretResponse{}
				return resp, nil
			})
		},
			NewConfig(),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			impl, err := NewKvScrtStore(context.TODO(), tt.token, tt.tokenSeparator, tt.keySeparator)
			if err != nil {
				t.Errorf("failed to init azkvstore")
			}
			tt.config.WithKeySeparator(tt.keySeparator).WithTokenSeparator(tt.tokenSeparator)
			impl.svc = tt.mockClient(t)
			rs := newRetrieveStrategy(NewDefatultStrategy(), *tt.config)
			rs.setImplementation(impl)
			got, err := rs.getTokenValue()
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
