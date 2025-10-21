package store_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/internal/log"
	"github.com/DevLabFoundry/configmanager/v2/internal/store"
	"github.com/DevLabFoundry/configmanager/v2/internal/testutils"
)

func azTableStoreCommonChecker(t *testing.T, partitionKey, rowKey, expectedPartitionKey, expectedRowKey string) {
	t.Helper()
	if partitionKey == "" {
		t.Errorf("expect name to not be nil")
	}
	if partitionKey != expectedPartitionKey {
		t.Errorf(testutils.TestPhrase, partitionKey, expectedPartitionKey)
	}

	if strings.Contains(partitionKey, string(config.AzTableStorePrefix)) {
		t.Errorf("incorrectly stripped prefix")
	}

	if rowKey != expectedRowKey {
		t.Errorf(testutils.TestPhrase, rowKey, expectedPartitionKey)
	}
}

type mockAzTableStoreApi func(ctx context.Context, partitionKey string, rowKey string, options *aztables.GetEntityOptions) (aztables.GetEntityResponse, error)

func (m mockAzTableStoreApi) GetEntity(ctx context.Context, partitionKey string, rowKey string, options *aztables.GetEntityOptions) (aztables.GetEntityResponse, error) {
	return m(ctx, partitionKey, rowKey, options)
}

func Test_AzTableStore_Success(t *testing.T) {

	tests := map[string]struct {
		token      func() *config.ParsedTokenConfig
		expect     string
		mockClient func(t *testing.T) mockAzTableStoreApi
	}{
		"successVal": {
			func() *config.ParsedTokenConfig {
				// "AZTABLESTORE#/test-account/table//token/1"
				tkn, _ := config.NewToken(config.AzTableStorePrefix, *config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/test-account/table//token/1")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			}, "tsuccessParam", func(t *testing.T) mockAzTableStoreApi {
				return mockAzTableStoreApi(func(ctx context.Context, partitionKey string, rowKey string, options *aztables.GetEntityOptions) (aztables.GetEntityResponse, error) {
					t.Helper()
					azTableStoreCommonChecker(t, partitionKey, rowKey, "token", "1")
					resp := aztables.GetEntityResponse{}
					resp.Value = []byte("tsuccessParam")
					return resp, nil
				})
			},
		},
		// "successVal with :// token Separator": {"AZTABLESTORE:///test-account/table//token/1", "tsuccessParam", func(t *testing.T) tableStoreApi {
		// 	return mockAzTableStoreApi(func(ctx context.Context, partitionKey string, rowKey string, options *aztables.GetEntityOptions) (aztables.GetEntityResponse, error) {
		// 		t.Helper()
		// 		azTableStoreCommonChecker(t, partitionKey, rowKey, "token", "1")
		// 		resp := aztables.GetEntityResponse{}
		// 		resp.Value = []byte("tsuccessParam")
		// 		return resp, nil
		// 	})
		// }, config.NewConfig().WithKeySeparator("|").WithTokenSeparator("://"),
		// },
		// "successVal with keyseparator but no val returned": {"AZTABLESTORE#/test-account/table/token/1|somekey", "", func(t *testing.T) tableStoreApi {
		// 	return mockAzTableStoreApi(func(ctx context.Context, partitionKey string, rowKey string, options *aztables.GetEntityOptions) (aztables.GetEntityResponse, error) {
		// 		t.Helper()
		// 		azTableStoreCommonChecker(t, partitionKey, rowKey, "token", "1")

		// 		resp := aztables.GetEntityResponse{}
		// 		resp.Value = nil
		// 		return resp, nil
		// 	})
		// },
		// 	config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"),
		// },
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			impl, err := store.NewAzTableStore(context.TODO(), tt.token(), log.New(io.Discard))
			if err != nil {
				t.Errorf("failed to init aztablestore")
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

func Test_azstorage_with_value_property(t *testing.T) {

	conf := config.NewConfig().WithKeySeparator("|").WithTokenSeparator("://")
	ttests := map[string]struct {
		token      func() *config.ParsedTokenConfig
		expect     string
		mockClient func(t *testing.T) mockAzTableStoreApi
	}{
		"return value property with json like object": {
			func() *config.ParsedTokenConfig {
				// "AZTABLESTORE:///test-account/table/partitionkey/rowKey|host",
				tkn, _ := config.NewToken(config.AzKeyVaultSecretsPrefix, *conf)
				tkn.WithSanitizedToken("/test-account/table/partitionkey/rowKey")
				tkn.WithKeyPath("host")
				tkn.WithMetadata("version:123]")
				return tkn
			},
			"map[bool:true host:foo port:1234]",
			func(t *testing.T) mockAzTableStoreApi {
				return mockAzTableStoreApi(func(ctx context.Context, partitionKey string, rowKey string, options *aztables.GetEntityOptions) (aztables.GetEntityResponse, error) {
					t.Helper()
					resp := aztables.GetEntityResponse{Value: []byte(`{"value":{"host":"foo","port":1234,"bool":true}}`)}
					return resp, nil
				})
			},
		},
		// "return value property with string only": {
		// 	"AZTABLESTORE:///test-account/table/partitionkey/rowKey",
		// 	"foo.bar.com",
		// 	func(t *testing.T) tableStoreApi {
		// 		return mockAzTableStoreApi(func(ctx context.Context, partitionKey string, rowKey string, options *aztables.GetEntityOptions) (aztables.GetEntityResponse, error) {
		// 			t.Helper()
		// 			resp := aztables.GetEntityResponse{Value: []byte(`{"value":"foo.bar.com"}`)}
		// 			return resp, nil
		// 		})
		// 	},
		// 	conf,
		// },
		// "return value property with numeric only": {
		// 	"AZTABLESTORE:///test-account/table/partitionkey/rowKey",
		// 	"1234",
		// 	func(t *testing.T) tableStoreApi {
		// 		return mockAzTableStoreApi(func(ctx context.Context, partitionKey string, rowKey string, options *aztables.GetEntityOptions) (aztables.GetEntityResponse, error) {
		// 			t.Helper()
		// 			resp := aztables.GetEntityResponse{Value: []byte(`{"value":1234}`)}
		// 			return resp, nil
		// 		})
		// 	},
		// 	conf,
		// },
		// "return value property with boolean only": {
		// 	"AZTABLESTORE:///test-account/table/partitionkey/rowKey",
		// 	"false",
		// 	func(t *testing.T) tableStoreApi {
		// 		return mockAzTableStoreApi(func(ctx context.Context, partitionKey string, rowKey string, options *aztables.GetEntityOptions) (aztables.GetEntityResponse, error) {
		// 			t.Helper()
		// 			resp := aztables.GetEntityResponse{Value: []byte(`{"value":false}`)}
		// 			return resp, nil
		// 		})
		// 	},
		// 	conf,
		// },
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			// token, _ := config.NewToken(tt.token(), *tt.config)

			impl, err := store.NewAzTableStore(context.TODO(), tt.token(), log.New(io.Discard))
			if err != nil {
				t.Fatal("failed to init aztablestore")
			}

			impl.WithSvc(tt.mockClient(t))

			got, err := impl.Token()
			if err != nil {
				t.Fatalf(testutils.TestPhrase, err.Error(), nil)
			}

			if got != tt.expect {
				t.Errorf(testutils.TestPhraseWithContext, "AZ Table storage with value property inside entity", fmt.Sprintf("%q", got), fmt.Sprintf("%q", tt.expect))
			}
		})
	}
}

func Test_AzTableStore_Error(t *testing.T) {

	tests := map[string]struct {
		token      func() *config.ParsedTokenConfig
		expect     error
		mockClient func(t *testing.T) mockAzTableStoreApi
	}{
		"errored on token parsing to partiationKey": {
			func() *config.ParsedTokenConfig {
				// "AZTABLESTORE#/test-vault/token/1|somekey"
				tkn, _ := config.NewToken(config.AzTableStorePrefix, *config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/test-vault/token/1")
				tkn.WithKeyPath("somekey")
				tkn.WithMetadata("")
				return tkn
			}, store.ErrIncorrectlyStructuredToken, func(t *testing.T) mockAzTableStoreApi {
				return mockAzTableStoreApi(func(ctx context.Context, partitionKey string, rowKey string, options *aztables.GetEntityOptions) (aztables.GetEntityResponse, error) {
					t.Helper()
					resp := aztables.GetEntityResponse{}
					return resp, nil
				})
			},
		},
		// "errored on service method call": {"AZTABLESTORE#/test-account/table/token/ok", ErrRetrieveFailed, func(t *testing.T) tableStoreApi {
		// 	return mockAzTableStoreApi(func(ctx context.Context, partitionKey string, rowKey string, options *aztables.GetEntityOptions) (aztables.GetEntityResponse, error) {
		// 		t.Helper()
		// 		resp := aztables.GetEntityResponse{}
		// 		return resp, fmt.Errorf("network error")
		// 	})
		// },
		// 	config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"),
		// },

		// "empty": {"AZTABLESTORE#/test-vault/token/1|somekey", ErrIncorrectlyStructuredToken, func(t *testing.T) tableStoreApi {
		// 	return mockAzTableStoreApi(func(ctx context.Context, partitionKey string, rowKey string, options *aztables.GetEntityOptions) (aztables.GetEntityResponse, error) {
		// 		t.Helper()
		// 		resp := aztables.GetEntityResponse{}
		// 		return resp, nil
		// 	})
		// },
		// 	config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"),
		// },
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			impl, err := store.NewAzTableStore(context.TODO(), tt.token(), log.New(io.Discard))
			if err != nil {
				t.Fatal("failed to init aztablestore")
			}

			impl.WithSvc(tt.mockClient(t))
			if _, err := impl.Token(); !errors.Is(err, tt.expect) {
				t.Errorf(testutils.TestPhrase, err.Error(), tt.expect)
			}
		})
	}
}

func Test_fail_AzTable_Client_init(t *testing.T) {
	// this is basically a wrap around test for the url.Parse method in the stdlib
	// as that is what the client uses under the hood
	token, _ := config.NewToken(config.AzTableStorePrefix, *config.NewConfig())
	// "AZTABLESTORE:///%25%65%6e%301-._~/</partitionKey/rowKey"
	token.WithSanitizedToken("/%25%65%6e%301-._~/</partitionKey/rowKey")
	_, err := store.NewAzTableStore(context.TODO(), token, log.New(io.Discard))
	if err == nil {
		t.Fatal("expected err to not be <nil>")
	}
	if !errors.Is(err, store.ErrClientInitialization) {
		t.Fatalf(testutils.TestPhraseWithContext, "aztables client init", err.Error(), store.ErrClientInitialization.Error())
	}
}

func Test_azSplitTokenTableStore(t *testing.T) {

	tests := []struct {
		name   string
		token  string
		expect store.AzServiceHelper
	}{
		{
			name:  "simple_with_preceding_slash",
			token: "/test-account/tablename/somejsontest",
			expect: store.AzServiceHelper{
				ServiceUri: "https://test-account.table.core.windows.net/tablename",
				Token:      "somejsontest",
			},
		},
		{
			name:  "missing_initial_slash",
			token: "test-account/tablename/somejsontest",
			expect: store.AzServiceHelper{
				ServiceUri: "https://test-account.table.core.windows.net/tablename",
				Token:      "somejsontest",
			},
		},
		{
			name:  "missing_initial_slash_multislash_secretname",
			token: "test-account/tablename/some/json/test",
			expect: store.AzServiceHelper{
				ServiceUri: "https://test-account.table.core.windows.net/tablename",
				Token:      "some/json/test",
			},
		},
		{
			name:  "with_initial_slash_multislash_secretname",
			token: "test-account/tablename//some/json/test",
			expect: store.AzServiceHelper{
				ServiceUri: "https://test-account.table.core.windows.net/tablename",
				Token:      "/some/json/test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := store.AzServiceFromToken(tt.token, "https://%s.table.core.windows.net/%s", 2)
			if got.Token != tt.expect.Token {
				t.Errorf(testutils.TestPhrase, tt.expect.Token, got.Token)
			}
			if got.ServiceUri != tt.expect.ServiceUri {
				t.Errorf(testutils.TestPhrase, tt.expect.ServiceUri, got.ServiceUri)
			}
		})
	}
}
