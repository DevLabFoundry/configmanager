package generator

import (
	"regexp"
	"strings"
)

// TokenConfigVars
type TokenConfigVars struct {
	Token string
	// AWS IAM Role for Vault AWS IAM Auth
	Role string
	// where supported a version of the secret can be specified
	//
	// e.g. HashiVault or AWS SecretsManager
	//
	Version string
}

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

func (c GenVarsConfig) ParseTokenVars(token string) TokenConfigVars {
	tc := TokenConfigVars{}
	// strip anything in []
	vars := regexp.MustCompile(`\[.*\]`)
	rawVars := vars.FindString(token)
	// extract [role:,version:]
	if rawVars != "" {
		role := regexp.MustCompile(`role:(.*?)(?:,|])`).FindStringSubmatch(rawVars)
		if len(role) > 0 {
			tc.Role = role[1]
		}
		version := regexp.MustCompile(`version:(.*?)(?:,|])`).FindStringSubmatch(rawVars)
		if len(version) > 0 {
			tc.Version = version[1]
		}
		tc.Token = strings.ReplaceAll(token, rawVars, "")
		// tc.Role =
		return tc
	}
	tc.Token = token
	return tc
}
