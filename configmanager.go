package configmanager

import (
	"context"
	"errors"
	"fmt"
	"io"

	"slices"
	"strings"

	"github.com/DevLabFoundry/configmanager/v3/generator"
	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/a8m/envsubst"
)

const (
	TERMINATING_CHAR string = `[^\'\"\s\n\\\,]` // :\@\?\/
)

// generateAPI
type generateAPI interface {
	Generate(tokens []string) (generator.ReplacedToken, error)
}

type ConfigManager struct {
	Config    *config.GenVarsConfig
	generator generateAPI
	logger    log.ILogger
}

// New returns an initialised instance of ConfigManager
// Uses default config for:
//
//	outputPath = ""
//	keySeparator = "|"
//	tokenSeparator = "://"
//
// # Calling cm.Config.WithXXX() will overwrite the generator config
//
// Default logger will log to io.Discard
// Attach your own if you need via
//
//	WithLogger(l log.ILogger) *ConfigManager
func New(ctx context.Context) *ConfigManager {
	cm := &ConfigManager{}
	cm.Config = config.NewConfig()
	cm.generator = generator.NewGenerator(ctx).WithConfig(cm.Config)
	cm.logger = log.New(io.Discard)
	return cm
}

func (c *ConfigManager) WithLogger(l log.ILogger) *ConfigManager {
	c.logger = l
	return c
}

// GeneratorConfig
// Returns the gettable generator config
func (c *ConfigManager) GeneratorConfig() *config.GenVarsConfig {
	return c.Config
}

// WithGenerator replaces the generator instance
func (c *ConfigManager) WithGenerator(generator generateAPI) *ConfigManager {
	c.generator = generator
	return c
}

// Retrieve gets a rawMap from a set implementation
// will be empty if no matches found
func (c *ConfigManager) Retrieve(tokens []string) (generator.ReplacedToken, error) {
	return c.generator.Generate(tokens)
}

var ErrEnvSubst = errors.New("envsubst enabled and errored on")

// RetrieveReplacedString parses given input against all possible token strings
func (c *ConfigManager) RetrieveReplacedString(input string) (string, error) {
	// replaces all env vars using strict mode of no unset and no empty
	if c.GeneratorConfig().EnvSubstEnabled() {
		var err error
		input, err = envsubst.StringRestrictedNoDigit(input, true, true, false)
		if err != nil {
			return "", fmt.Errorf("%w\n%v", ErrEnvSubst, err)
		}
	}

	// calling the same Generate method with the input as single item in a slice
	m, err := c.generator.Generate([]string{input})

	if err != nil {
		return "", err
	}

	return replaceString(m, input), nil
}

// RetrieveReplacedBytes is functionally identical RetrieveReplacedString
func (c *ConfigManager) RetrieveReplacedBytes(input []byte) ([]byte, error) {
	r, err := c.RetrieveReplacedString(string(input))
	return []byte(r), err
}

// replaceString fills tokens in a provided input with their actual secret/config values
func replaceString(inputMap generator.ReplacedToken, inputString string) string {

	oldNew := []string(nil)
	// ordered values by index
	for _, ov := range orderedKeysList(inputMap) {
		oldNew = append(oldNew, ov, fmt.Sprint(inputMap[ov]))
	}
	replacer := strings.NewReplacer(oldNew...)
	return replacer.Replace(inputString)
}

func orderedKeysList(inputMap generator.ReplacedToken) []string {
	mkeys := inputMap.MapKeys()
	// order map by keys length so that when passed to the
	// replacer it will replace the longest first
	// removing the possibility of partially overwriting
	// another token with same prefix
	// the default sort is ascending
	slices.Sort(mkeys)
	return mkeys
}
