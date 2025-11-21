package store_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig"
	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	logger "github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/DevLabFoundry/configmanager/v3/internal/store"
	"github.com/DevLabFoundry/configmanager/v3/internal/testutils"
)

func azAppConfCommonChecker(t *testing.T, key string, expectedKey string, expectLabel string, opts *azappconfig.GetSettingOptions) {
	t.Helper()
	if key != expectedKey {
		t.Errorf(testutils.TestPhrase, key, expectedKey)
	}

	if expectLabel != "" {
		if opts == nil {
			t.Errorf(testutils.TestPhrase, nil, expectLabel)
		}
		if *opts.Label != expectLabel {
			t.Errorf(testutils.TestPhrase, opts.Label, expectLabel)
		}
	}
}

type mockAzAppConfApi func(ctx context.Context, key string, options *azappconfig.GetSettingOptions) (azappconfig.GetSettingResponse, error)

func (m mockAzAppConfApi) GetSetting(ctx context.Context, key string, options *azappconfig.GetSettingOptions) (azappconfig.GetSettingResponse, error) {
	return m(ctx, key, options)
}

func Test_AzAppConf_Success(t *testing.T) {
	tsuccessParam := "somecvla"

	logr := logger.New(&bytes.Buffer{})
	tests := map[string]struct {
		token      func() *config.ParsedTokenConfig
		expect     string
		mockClient func(t *testing.T) mockAzAppConfApi
	}{
		"successVal": {
			func() *config.ParsedTokenConfig {
				// "AZAPPCONF#/test-app-config-instance/table//token/1",
				tkn, _ := config.NewToken(config.AzAppConfigPrefix, *config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/test-app-config-instance/table//token/1")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			},
			tsuccessParam,
			func(t *testing.T) mockAzAppConfApi {
				return mockAzAppConfApi(func(ctx context.Context, key string, options *azappconfig.GetSettingOptions) (azappconfig.GetSettingResponse, error) {
					azAppConfCommonChecker(t, key, "table//token/1", "", options)
					resp := azappconfig.GetSettingResponse{}
					resp.Value = &tsuccessParam
					return resp, nil
				})
			},
		},
		"successVal with :// token Separator": {
			func() *config.ParsedTokenConfig {
				// "AZAPPCONF:///test-app-config-instance/conf_key[label=dev]",
				tkn, _ := config.NewToken(config.AzAppConfigPrefix, *config.NewConfig().WithKeySeparator("|").WithTokenSeparator("://"))
				tkn.WithSanitizedToken("/test-app-config-instance/conf_key")
				tkn.WithKeyPath("")
				tkn.WithMetadata("label=dev")
				return tkn
			},
			tsuccessParam,
			func(t *testing.T) mockAzAppConfApi {
				return mockAzAppConfApi(func(ctx context.Context, key string, options *azappconfig.GetSettingOptions) (azappconfig.GetSettingResponse, error) {
					azAppConfCommonChecker(t, key, "conf_key", "dev", options)
					resp := azappconfig.GetSettingResponse{}
					resp.Value = &tsuccessParam
					return resp, nil
				})
			},
		},
		"successVal with :// token Separator and etag specified": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.AzAppConfigPrefix, *config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/test-app-config-instance/conf_key")
				tkn.WithKeyPath("")
				tkn.WithMetadata("label=dev,etag=sometifdsssdsfdi_string01209222")
				return tkn
			},
			tsuccessParam,
			func(t *testing.T) mockAzAppConfApi {
				return mockAzAppConfApi(func(ctx context.Context, key string, options *azappconfig.GetSettingOptions) (azappconfig.GetSettingResponse, error) {
					azAppConfCommonChecker(t, key, "conf_key", "dev", options)
					if !options.OnlyIfChanged.Equals("sometifdsssdsfdi_string01209222") {
						t.Errorf(testutils.TestPhraseWithContext, "Etag not correctly set", options.OnlyIfChanged, "sometifdsssdsfdi_string01209222")
					}
					resp := azappconfig.GetSettingResponse{}
					resp.Value = &tsuccessParam
					return resp, nil
				})
			},
		},
		"successVal with keyseparator but no val returned": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.AzAppConfigPrefix, *config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/test-app-config-instance/try_to_find")
				tkn.WithKeyPath("key_separator.lookup")
				tkn.WithMetadata("")
				return tkn
			},
			"",
			func(t *testing.T) mockAzAppConfApi {
				return mockAzAppConfApi(func(ctx context.Context, key string, options *azappconfig.GetSettingOptions) (azappconfig.GetSettingResponse, error) {
					azAppConfCommonChecker(t, key, "try_to_find", "", options)
					resp := azappconfig.GetSettingResponse{}
					resp.Value = nil
					return resp, nil
				})
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			impl, err := store.NewAzAppConf(context.TODO(), tt.token(), logr)
			if err != nil {
				t.Errorf("failed to init AZAPPCONF")
			}

			impl.WithSvc(tt.mockClient(t))
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

func Test_AzAppConf_Error(t *testing.T) {
	logr := logger.New(&bytes.Buffer{})

	tests := map[string]struct {
		token      func() *config.ParsedTokenConfig
		expect     error
		mockClient func(t *testing.T) mockAzAppConfApi
	}{
		"errored on service method call": {
			func() *config.ParsedTokenConfig {
				// "AZAPPCONF#/test-app-config-instance/table/token/ok",
				tkn, _ := config.NewToken(config.AzAppConfigPrefix, *config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"))
				tkn.WithSanitizedToken("/test-app-config-instance/table/token/ok")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			},
			store.ErrRetrieveFailed,
			func(t *testing.T) mockAzAppConfApi {
				return mockAzAppConfApi(func(ctx context.Context, key string, options *azappconfig.GetSettingOptions) (azappconfig.GetSettingResponse, error) {
					t.Helper()
					resp := azappconfig.GetSettingResponse{}
					return resp, fmt.Errorf("network error")
				})
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			impl, err := store.NewAzAppConf(context.TODO(), tt.token(), logr)
			if err != nil {
				t.Fatal("failed to init AZAPPCONF")
			}
			impl.WithSvc(tt.mockClient(t))
			if _, err := impl.Value(); !errors.Is(err, tt.expect) {
				t.Errorf(testutils.TestPhrase, err.Error(), tt.expect)
			}
		})
	}
}

func Test_fail_AzAppConf_Client_init(t *testing.T) {

	logr := logger.New(&bytes.Buffer{})

	// this is basically a wrap around test for the url.Parse method in the stdlib
	// as that is what the client uses under the hood
	token, _ := config.NewToken(config.AzAppConfigPrefix, *config.NewConfig())
	token.WithSanitizedToken("/%25%65%6e%301-._~/</partitionKey/rowKey")

	_, err := store.NewAzAppConf(context.TODO(), token, logr)
	if err == nil {
		t.Fatal("expected err to not be <nil>")
	}
	if !errors.Is(err, store.ErrClientInitialization) {
		t.Fatalf(testutils.TestPhraseWithContext, "azappconf client init", err.Error(), store.ErrClientInitialization.Error())
	}
}
