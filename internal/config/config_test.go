package config_test

import (
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/testutils"
)

func Test_SelfName(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "configmanager",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name != config.SELF_NAME {
				t.Error("self name does not match")
			}
		})
	}
}

func Test_MarshalMetadata_with_label_struct_succeeds(t *testing.T) {
	type labelMeta struct {
		Label string `json:"label"`
	}

	ttests := map[string]struct {
		token                 func() *config.ParsedTokenConfig
		wantLabel             string
		wantMetaStrippedToken string
	}{
		"when provider expects label on token and label exists": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.AzTableStorePrefix, *config.NewConfig().WithTokenSeparator("://"))
				tkn.WithKeyPath("d88")
				tkn.WithMetadata("label=dev")
				tkn.WithSanitizedToken("basjh/dskjuds/123")
				return tkn
			},
			"dev",
			"basjh/dskjuds/123",
		},
		"when provider expects label on token and label does not exist": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.AzTableStorePrefix, *config.NewConfig().WithTokenSeparator("://"))
				tkn.WithKeyPath("d88")
				tkn.WithMetadata("someother=dev")
				tkn.WithSanitizedToken("basjh/dskjuds/123")
				return tkn
			},
			"",
			"basjh/dskjuds/123",
		},
		"no metadata found": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.AzTableStorePrefix, *config.NewConfig().WithTokenSeparator("://"))
				tkn.WithKeyPath("d88")
				tkn.WithSanitizedToken("basjh/dskjuds/123")
				return tkn
			},
			"",
			"basjh/dskjuds/123",
		},
		"no metadata found incorrect marker placement": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.AzTableStorePrefix, *config.NewConfig().WithTokenSeparator("://"))
				tkn.WithKeyPath("d88]asdas=bar[")
				tkn.WithSanitizedToken("basjh/dskjuds/123")
				return tkn
			},
			"",
			"basjh/dskjuds/123",
		},
		"no metadata found incorrect marker placement and no key separator": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.AzTableStorePrefix, *config.NewConfig().WithTokenSeparator("://"))
				tkn.WithSanitizedToken("basjh/dskjuds/123]asdas=bar[")
				return tkn
			},
			"",
			"basjh/dskjuds/123]asdas=bar[",
		},
		"no start found incorrect marker placement and no key separator": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.AzTableStorePrefix, *config.NewConfig().WithTokenSeparator("://"))
				tkn.WithKeyPath("d88")
				tkn.WithMetadata("someother=dev")
				tkn.WithSanitizedToken("basjh/dskjuds/123]asdas=bar]")
				return tkn
			},
			"",
			"basjh/dskjuds/123]asdas=bar]",
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			inputTyp := &labelMeta{}
			got := tt.token()
			if got == nil {
				t.Errorf(testutils.TestPhraseWithContext, "Unable to parse token", nil, config.ParsedTokenConfig{})
			}

			got.ParseMetadata(inputTyp)

			if got.StoreToken() != tt.wantMetaStrippedToken {
				t.Errorf(testutils.TestPhraseWithContext, "Token does not match", got.StoreToken(), tt.wantMetaStrippedToken)
			}

			if inputTyp.Label != tt.wantLabel {
				t.Errorf(testutils.TestPhraseWithContext, "Metadata Label does not match", inputTyp.Label, tt.wantLabel)
			}
		})
	}
}

func Test_TokenParser_config(t *testing.T) {
	type mockConfAwsSecrMgr struct {
		Version string `json:"version"`
	}
	ttests := map[string]struct {
		rawToken, keyPath, metadataStr string
		expPrefix                      config.ImplementationPrefix
		expLookupKeys                  string
		expStoreToken                  string // sanitised
		expString                      string // fullToken
		expMetadataVersion             string
	}{
		"bare":                              {"foo/bar", "", "", config.SecretMgrPrefix, "", "foo/bar", "AWSSECRETS://foo/bar", ""},
		"with metadata version":             {"foo/bar", "", "version=123", config.SecretMgrPrefix, "", "foo/bar", "AWSSECRETS://foo/bar[version=123]", "123"},
		"with keys lookup and label":        {"foo/bar", "key1.key2", "version=123", config.SecretMgrPrefix, "key1.key2", "foo/bar", "AWSSECRETS://foo/bar|key1.key2[version=123]", "123"},
		"with keys lookup and longer token": {"foo/bar", "key1.key2]version=123]", "", config.SecretMgrPrefix, "key1.key2]version=123]", "foo/bar", "AWSSECRETS://foo/bar|key1.key2]version=123]", ""},
		"with keys lookup but no keys":      {"foo/bar/sdf/sddd.90dsfsd", "", "version=123", config.SecretMgrPrefix, "", "foo/bar/sdf/sddd.90dsfsd", "AWSSECRETS://foo/bar/sdf/sddd.90dsfsd[version=123]", "123"},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			conf := &mockConfAwsSecrMgr{}
			got, _ := config.NewToken(tt.expPrefix, *config.NewConfig())
			got.WithSanitizedToken(tt.rawToken)
			got.WithKeyPath(tt.keyPath)
			got.WithMetadata(tt.metadataStr)

			got.ParseMetadata(conf)

			if got.LookupKeys() != tt.expLookupKeys {
				t.Errorf(testutils.TestPhrase, got.LookupKeys(), tt.expLookupKeys)
			}
			if got.StoreToken() != tt.expStoreToken {
				t.Errorf(testutils.TestPhrase, got.StoreToken(), tt.expLookupKeys)
			}
			if got.String() != tt.expString {
				t.Errorf(testutils.TestPhrase, got.String(), tt.expString)
			}
			if got.Prefix() != tt.expPrefix {
				t.Errorf(testutils.TestPhrase, got.Prefix(), tt.expPrefix)
			}
			if conf.Version != tt.expMetadataVersion {
				t.Errorf(testutils.TestPhrase, conf.Version, tt.expMetadataVersion)
			}
		})
	}
}
