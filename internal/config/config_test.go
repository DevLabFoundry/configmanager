package config_test

import (
	"testing"

	"github.com/DevLabFoundry/configmanager/internal/config"
	"github.com/DevLabFoundry/configmanager/internal/testutils"
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
		config                *config.GenVarsConfig
		rawToken              string
		wantLabel             string
		wantMetaStrippedToken string
	}{
		"when provider expects label on token and label exists": {
			config.NewConfig().WithTokenSeparator("://"),
			`AZTABLESTORE://basjh/dskjuds/123|d88[label=dev]`,
			"dev",
			"basjh/dskjuds/123",
		},
		"when provider expects label on token and label does not exist": {
			config.NewConfig().WithTokenSeparator("://"),
			`AZTABLESTORE://basjh/dskjuds/123|d88[someother=dev]`,
			"",
			"basjh/dskjuds/123",
		},
		"no metadata found": {
			config.NewConfig().WithTokenSeparator("://"),
			`AZTABLESTORE://basjh/dskjuds/123|d88`,
			"",
			"basjh/dskjuds/123",
		},
		"no metadata found incorrect marker placement": {
			config.NewConfig().WithTokenSeparator("://"),
			`AZTABLESTORE://basjh/dskjuds/123|d88]asdas=bar[`,
			"",
			"basjh/dskjuds/123",
		},
		"no metadata found incorrect marker placement and no key separator": {
			config.NewConfig().WithTokenSeparator("://"),
			`AZTABLESTORE://basjh/dskjuds/123]asdas=bar[`,
			"",
			"basjh/dskjuds/123]asdas=bar[",
		},
		"no end found incorrect marker placement and no key separator": {
			config.NewConfig().WithTokenSeparator("://"),
			`AZTABLESTORE://basjh/dskjuds/123[asdas=bar`,
			"",
			"basjh/dskjuds/123[asdas=bar",
		},
		"no start found incorrect marker placement and no key separator": {
			config.NewConfig().WithTokenSeparator("://"),
			`AZTABLESTORE://basjh/dskjuds/123]asdas=bar]`,
			"",
			"basjh/dskjuds/123]asdas=bar]",
		},
		"metadata is in the middle of path lookup": {
			config.NewConfig().WithTokenSeparator("://"),
			`AZTABLESTORE://basjh/dskjuds/123[label=bar]|lookup`,
			"bar",
			"basjh/dskjuds/123",
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			inputTyp := &labelMeta{}
			got, err := config.NewParsedTokenConfig(tt.rawToken, *tt.config)

			if err != nil {
				t.Fatalf("got an error on NewParsedTokenconfig (%s)\n", tt.rawToken)
			}

			if got == nil {
				t.Errorf(testutils.TestPhraseWithContext, "Unable to parse token", nil, config.ParsedTokenConfig{})
			}

			got.ParseMetadata(inputTyp)

			if got.StoreToken() != tt.wantMetaStrippedToken {
				t.Errorf(testutils.TestPhraseWithContext, "Token does not match", got.StripMetadata(), tt.wantMetaStrippedToken)
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
		input              string
		expPrefix          config.ImplementationPrefix
		expLookupKeys      string
		expStoreToken      string
		expString          string // fullToken
		expMetadataVersion string
	}{
		"bare":                              {"AWSSECRETS://foo/bar", config.SecretMgrPrefix, "", "foo/bar", "AWSSECRETS://foo/bar", ""},
		"with metadata version":             {"AWSSECRETS://foo/bar[version=123]", config.SecretMgrPrefix, "", "foo/bar", "AWSSECRETS://foo/bar[version=123]", "123"},
		"with keys lookup and label":        {"AWSSECRETS://foo/bar|key1.key2[version=123]", config.SecretMgrPrefix, "key1.key2", "foo/bar", "AWSSECRETS://foo/bar|key1.key2[version=123]", "123"},
		"with keys lookup and longer token": {"AWSSECRETS://foo/bar|key1.key2]version=123]", config.SecretMgrPrefix, "key1.key2]version=123]", "foo/bar", "AWSSECRETS://foo/bar|key1.key2]version=123]", ""},
		"with keys lookup but no keys":      {"AWSSECRETS://foo/bar/sdf/sddd.90dsfsd|[version=123]", config.SecretMgrPrefix, "", "foo/bar/sdf/sddd.90dsfsd", "AWSSECRETS://foo/bar/sdf/sddd.90dsfsd|[version=123]", "123"},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			conf := &mockConfAwsSecrMgr{}
			got, _ := config.NewParsedTokenConfig(tt.input, *config.NewConfig())
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
