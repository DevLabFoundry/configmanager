package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	SELF_NAME = "configmanager"
)

const (
	// tokenSeparator used for identifying the end of a prefix and beginning of token
	// see notes about special consideration for AZKVSECRET tokens
	tokenSeparator = "://"
	// keySeparator used for accessing nested objects within the retrieved map
	keySeparator = "|"
)

type ImplementationPrefix string

const (
	// AWS SecretsManager prefix
	SecretMgrPrefix ImplementationPrefix = "AWSSECRETS"
	// AWS Parameter Store prefix
	ParamStorePrefix ImplementationPrefix = "AWSPARAMSTR"
	// Azure Key Vault Secrets prefix
	AzKeyVaultSecretsPrefix ImplementationPrefix = "AZKVSECRET"
	// Azure Key Vault Secrets prefix
	AzTableStorePrefix ImplementationPrefix = "AZTABLESTORE"
	// Azure App Config prefix
	AzAppConfigPrefix ImplementationPrefix = "AZAPPCONF"
	// Hashicorp Vault prefix
	HashicorpVaultPrefix ImplementationPrefix = "VAULT"
	// GcpSecrets
	GcpSecretsPrefix ImplementationPrefix = "GCPSECRETS"
	// Unknown
	UnknownPrefix ImplementationPrefix = "UNKNOWN"
)

var (
	// default varPrefix used by the replacer function
	// any token must beging with one of these else
	// it will be skipped as not a replaceable token
	VarPrefix = map[ImplementationPrefix]bool{
		SecretMgrPrefix: true, ParamStorePrefix: true, AzKeyVaultSecretsPrefix: true,
		GcpSecretsPrefix: true, HashicorpVaultPrefix: true, AzTableStorePrefix: true,
		AzAppConfigPrefix: true, UnknownPrefix: true,
	}
)

// GenVarsConfig defines the input config object to be passed
type GenVarsConfig struct {
	outpath        string
	tokenSeparator string
	keySeparator   string
	enableEnvSubst bool
	// parseAdditionalVars func(token string) TokenConfigVars
}

// NewConfig
func NewConfig() *GenVarsConfig {
	return &GenVarsConfig{
		tokenSeparator: tokenSeparator,
		keySeparator:   keySeparator,
	}
}

// WithOutputPath
func (c *GenVarsConfig) WithOutputPath(out string) *GenVarsConfig {
	c.outpath = out
	return c
}

// WithTokenSeparator adds a custom token separator
// token is the actual value of the parameter/secret in the
// provider store
func (c *GenVarsConfig) WithTokenSeparator(tokenSeparator string) *GenVarsConfig {
	c.tokenSeparator = tokenSeparator
	return c
}

// WithKeySeparator adds a custom key separotor
func (c *GenVarsConfig) WithKeySeparator(keySeparator string) *GenVarsConfig {
	c.keySeparator = keySeparator
	return c
}

// WithKeySeparator adds a custom key separotor
func (c *GenVarsConfig) WithEnvSubst(enabled bool) *GenVarsConfig {
	c.enableEnvSubst = enabled
	return c
}

// OutputPath returns the outpath set in the config
func (c *GenVarsConfig) OutputPath() string {
	return c.outpath
}

// TokenSeparator returns the tokenSeparator set in the config
func (c *GenVarsConfig) TokenSeparator() string {
	return c.tokenSeparator
}

// KeySeparator returns the keySeparator set in the config
func (c *GenVarsConfig) KeySeparator() string {
	return c.keySeparator
}

// EnvSubstEnabled returns whether or not envsubst is enabled
func (c *GenVarsConfig) EnvSubstEnabled() bool {
	return c.enableEnvSubst
}

// Config returns the derefed value
func (c *GenVarsConfig) Config() GenVarsConfig {
	cc := *c
	return cc
}

// Parsed token config section

var ErrInvalidTokenPrefix = errors.New("token prefix has no implementation")

type ParsedTokenConfig struct {
	prefix                       ImplementationPrefix
	keySeparator, tokenSeparator string
	prefixLessToken, fullToken   string
	metadataStr, keysPath        string
	storeToken, metadataLess     string
}

// NewParsedTokenConfig returns a pointer to a new TokenConfig struct
// returns nil if current prefix does not correspond to an Implementation
//
// The caller needs to make sure it is not nil
// TODO: a custom parser would be best here
func NewParsedTokenConfig(token string, config GenVarsConfig) (*ParsedTokenConfig, error) {
	ptc := &ParsedTokenConfig{}
	prfx := strings.Split(token, config.TokenSeparator())[0]

	// This should already only be a list of properly supported tokens but just in case
	if found := VarPrefix[ImplementationPrefix(prfx)]; !found {
		return nil, fmt.Errorf("prefix: %s\n%w", prfx, ErrInvalidTokenPrefix)
	}

	ptc.keySeparator = config.keySeparator
	ptc.tokenSeparator = config.tokenSeparator
	ptc.prefix = ImplementationPrefix(prfx)
	ptc.fullToken = token
	return ptc.new(), nil
}

func (ptc *ParsedTokenConfig) new() *ParsedTokenConfig {
	// order must be respected here
	//
	ptc.prefixLessToken = strings.Replace(ptc.fullToken, fmt.Sprintf("%s%s", ptc.prefix, ptc.tokenSeparator), "", 1)

	// token without metadata and the string itself
	ptc.extractMetadataStr()
	// token without keys
	ptc.keysLookup()
	return ptc
}

func (t *ParsedTokenConfig) ParseMetadata(metadataTyp any) error {
	// crude json like builder from key/val tags
	// since we are only ever dealing with a string input
	// extracted from the token there is little chance panic would occur here
	// WATCH THIS SPACE "¯\_(ツ)_/¯"
	metaMap := []string{}
	for keyVal := range strings.SplitSeq(t.metadataStr, ",") {
		mapKeyVal := strings.Split(keyVal, "=")
		if len(mapKeyVal) == 2 {
			metaMap = append(metaMap, fmt.Sprintf(`"%s":"%s"`, mapKeyVal[0], mapKeyVal[1]))
		}
	}

	// empty map will be parsed as `{}` still resulting in a valid json
	// and successful unmarshalling but default value pointer struct
	if err := json.Unmarshal(fmt.Appendf(nil, `{%s}`, strings.Join(metaMap, ",")), metadataTyp); err != nil {
		// It would very hard to test this since
		// we are forcing the key and value to be strings
		// return non-filled pointer
		return err
	}
	return nil
}

func (t *ParsedTokenConfig) StripPrefix() string {
	return t.prefixLessToken
}

// StripMetadata returns the fullToken without the
// metadata
func (t *ParsedTokenConfig) StripMetadata() string {
	return t.metadataLess
}

// Strip
//
// returns the only the store indicator string
// without any of the configmanager token enrichment:
//
// - metadata
//
// - keySeparator
//
// - keys
//
// - prefix
func (t *ParsedTokenConfig) StoreToken() string {
	return t.storeToken
}

// Full returns the full Token path.
// Including key separator and metadata values
func (t *ParsedTokenConfig) String() string {
	return t.fullToken
}

func (t *ParsedTokenConfig) LookupKeys() string {
	return t.keysPath
}

func (t *ParsedTokenConfig) Prefix() ImplementationPrefix {
	return t.prefix
}

const (
	startMetaStr string = `[`
	endMetaStr   string = `]`
)

// extractMetadataStr returns anything between the start and end
// metadata markers in the token string itself
// returns the token without meta
func (t *ParsedTokenConfig) extractMetadataStr() {
	token := t.prefixLessToken
	t.metadataLess = token
	startIndex := strings.Index(token, startMetaStr)
	// token has no startMetaStr
	if startIndex == -1 {
		return
	}
	newS := token[startIndex+len(startMetaStr):]

	endIndex := strings.Index(newS, endMetaStr)
	// token has no meta end
	if endIndex == -1 {
		return
	}
	// metastring extracted
	// complete [key=value] has been found
	metaString := newS[:endIndex]
	t.metadataStr = metaString
	// Set Metadataless token
	t.metadataLess = strings.ReplaceAll(token, startMetaStr+metaString+endMetaStr, "")
}

// keysLookup returns the keysLookup path and the string without it
//
// NOTE: metadata was already stripped at this point
func (t *ParsedTokenConfig) keysLookup() {
	keysIndex := strings.Index(t.metadataLess, t.keySeparator)
	if keysIndex >= 0 {
		t.keysPath = t.metadataLess[keysIndex+len(t.keySeparator):]
		t.storeToken = t.metadataLess[:keysIndex]
		return
	}
	t.storeToken = t.metadataLess
}
