package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig"
	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	logger "github.com/DevLabFoundry/configmanager/v2/internal/log"
	"github.com/DevLabFoundry/configmanager/v2/internal/testutils"
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
	t.Parallel()
	tsuccessParam := "somecvla"

	logr := logger.New(&bytes.Buffer{})
	tests := map[string]struct {
		token      string
		expect     string
		mockClient func(t *testing.T) appConfApi
		config     *config.GenVarsConfig
	}{
		"successVal": {
			"AZAPPCONF#/test-app-config-instance/table//token/1",
			tsuccessParam,
			func(t *testing.T) appConfApi {
				return mockAzAppConfApi(func(ctx context.Context, key string, options *azappconfig.GetSettingOptions) (azappconfig.GetSettingResponse, error) {
					azAppConfCommonChecker(t, key, "table//token/1", "", options)
					resp := azappconfig.GetSettingResponse{}
					resp.Value = &tsuccessParam
					return resp, nil
				})
			},
			config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"),
		},
		"successVal with :// token Separator": {
			"AZAPPCONF:///test-app-config-instance/conf_key[label=dev]",
			tsuccessParam,
			func(t *testing.T) appConfApi {
				return mockAzAppConfApi(func(ctx context.Context, key string, options *azappconfig.GetSettingOptions) (azappconfig.GetSettingResponse, error) {
					azAppConfCommonChecker(t, key, "conf_key", "dev", options)
					resp := azappconfig.GetSettingResponse{}
					resp.Value = &tsuccessParam
					return resp, nil
				})
			},
			config.NewConfig().WithKeySeparator("|").WithTokenSeparator("://"),
		},
		"successVal with :// token Separator and etag specified": {
			"AZAPPCONF:///test-app-config-instance/conf_key[label=dev,etag=sometifdsssdsfdi_string01209222]",
			tsuccessParam,
			func(t *testing.T) appConfApi {
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
			config.NewConfig().WithKeySeparator("|").WithTokenSeparator("://"),
		},
		"successVal with keyseparator but no val returned": {
			"AZAPPCONF#/test-app-config-instance/try_to_find|key_separator.lookup",
			"",
			func(t *testing.T) appConfApi {
				return mockAzAppConfApi(func(ctx context.Context, key string, options *azappconfig.GetSettingOptions) (azappconfig.GetSettingResponse, error) {
					azAppConfCommonChecker(t, key, "try_to_find", "", options)
					resp := azappconfig.GetSettingResponse{}
					resp.Value = nil
					return resp, nil
				})
			},
			config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			token, _ := config.NewParsedTokenConfig(tt.token, *tt.config)

			impl, err := NewAzAppConf(context.TODO(), token, logr)
			if err != nil {
				t.Errorf("failed to init AZAPPCONF")
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

func Test_AzAppConf_Error(t *testing.T) {
	t.Parallel()

	logr := logger.New(&bytes.Buffer{})

	tests := map[string]struct {
		token      string
		expect     error
		mockClient func(t *testing.T) appConfApi
		config     *config.GenVarsConfig
	}{
		"errored on service method call": {
			"AZAPPCONF#/test-app-config-instance/table/token/ok",
			ErrRetrieveFailed,
			func(t *testing.T) appConfApi {
				return mockAzAppConfApi(func(ctx context.Context, key string, options *azappconfig.GetSettingOptions) (azappconfig.GetSettingResponse, error) {
					t.Helper()
					resp := azappconfig.GetSettingResponse{}
					return resp, fmt.Errorf("network error")
				})
			},
			config.NewConfig().WithKeySeparator("|").WithTokenSeparator("#"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			token, _ := config.NewParsedTokenConfig(tt.token, *tt.config)
			impl, err := NewAzAppConf(context.TODO(), token, logr)
			if err != nil {
				t.Fatal("failed to init AZAPPCONF")
			}
			impl.svc = tt.mockClient(t)
			if _, err := impl.Token(); !errors.Is(err, tt.expect) {
				t.Errorf(testutils.TestPhrase, err.Error(), tt.expect)
			}
		})
	}
}

func Test_fail_AzAppConf_Client_init(t *testing.T) {
	t.Parallel()

	logr := logger.New(&bytes.Buffer{})

	// this is basically a wrap around test for the url.Parse method in the stdlib
	// as that is what the client uses under the hood
	token, _ := config.NewParsedTokenConfig("AZAPPCONF:///%25%65%6e%301-._~/</partitionKey/rowKey", *config.NewConfig())

	_, err := NewAzAppConf(context.TODO(), token, logr)
	if err == nil {
		t.Fatal("expected err to not be <nil>")
	}
	if !errors.Is(err, ErrClientInitialization) {
		t.Fatalf(testutils.TestPhraseWithContext, "azappconf client init", err.Error(), ErrClientInitialization.Error())
	}
}
