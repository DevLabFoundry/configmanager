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
	ErrConfigValidation = errors.New("config validation failed")
)

// GenVarsConfig defines the input config object to be passed
type GenVarsConfig struct {
	outpath        string
	tokenSeparator string
	keySeparator   string
	enableEnvSubst bool
	// parseAdditionalVars func(token string) TokenConfigVars
}

// NewConfig returns a new GenVarsConfig with default values
//
// keySeparator should be only a single character
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

// Config returns the derefed value
func (c *GenVarsConfig) Validate() error {
	if len(c.keySeparator) > 1 {
		return fmt.Errorf("%w, keyseparator can only be 1 character", ErrConfigValidation)
	}
	return nil
}

// Parsed token config section
var ErrInvalidTokenPrefix = errors.New("token prefix has no implementation")

type ParsedTokenConfig struct {
	prefix ImplementationPrefix
	// cofig values
	keySeparator, tokenSeparator string
	// tokenb parts
	metadataStr    string
	keysPath       string
	sanitizedToken string
}

// NewToken initialises a *ParsedTokenConfig
func NewToken(prefix ImplementationPrefix, config GenVarsConfig) (*ParsedTokenConfig, error) {
	tokenConf := &ParsedTokenConfig{}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	tokenConf.keySeparator = config.keySeparator
	tokenConf.tokenSeparator = config.tokenSeparator

	tokenConf.prefix = prefix

	return tokenConf, nil
}

func (ptc *ParsedTokenConfig) WithKeyPath(kp string) {
	ptc.keysPath = kp
}

func (ptc *ParsedTokenConfig) WithMetadata(md string) {
	ptc.metadataStr = md
}

func (ptc *ParsedTokenConfig) WithSanitizedToken(v string) {
	ptc.sanitizedToken = v
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

// StoreToken returns the sanitized token without:
//   - metadata
//   - keySeparator
//   - keys
//   - prefix
func (t *ParsedTokenConfig) StoreToken() string {
	return t.sanitizedToken
}

// Full returns the full Token path.
// Including key separator and metadata values
func (t *ParsedTokenConfig) String() string {
	token := t.Metadaless()
	if len(t.metadataStr) > 0 {
		token += fmt.Sprintf("[%s]", t.metadataStr)
	}
	return token
}

// Keypathless returns the token without the key and metadata attributes
// Token will include the ImplementationPrefix + token separator + path to item
func (t *ParsedTokenConfig) Keypathless() string {
	token := fmt.Sprintf("%s%s%s", t.prefix, t.tokenSeparator, t.sanitizedToken)
	return token
}

func (t *ParsedTokenConfig) Metadaless() string {
	token := fmt.Sprintf("%s%s%s", t.prefix, t.tokenSeparator, t.sanitizedToken)
	if len(t.keysPath) > 0 {
		token += t.keySeparator + t.keysPath
	}
	return token
}

func (t *ParsedTokenConfig) LookupKeys() string {
	return t.keysPath
}

func (t *ParsedTokenConfig) Metadata() string {
	return t.metadataStr
}

func (t *ParsedTokenConfig) Prefix() ImplementationPrefix {
	return t.prefix
}
