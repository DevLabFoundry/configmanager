package configmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/pkg/generator"
	"github.com/a8m/envsubst"
	"gopkg.in/yaml.v3"
)

const (
	TERMINATING_CHAR string = `[^\'\"\s\n\\\,]`
)

// generateAPI
type generateAPI interface {
	Generate(tokens []string) (generator.ParsedMap, error)
}

type ConfigManager struct {
	Config    *config.GenVarsConfig
	generator generateAPI
}

// New returns an initialised instance of ConfigManager
// Uses default config for:
//
// ```
// outputPath = ""
// keySeparator = "|"
// tokenSeparator = "://"
// ```
//
// Calling cm.Config.WithXXX() will overwrite the generator config
func New(ctx context.Context) *ConfigManager {
	cm := &ConfigManager{}
	cm.Config = config.NewConfig()
	cm.generator = generator.NewGenerator(ctx).WithConfig(cm.Config)
	return cm
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
func (c *ConfigManager) Retrieve(tokens []string) (generator.ParsedMap, error) {
	return c.retrieve(tokens)
}

func (c *ConfigManager) retrieve(tokens []string) (generator.ParsedMap, error) {
	return c.generator.Generate(tokens)
}

var ErrEnvSubst = errors.New("envsubst enabled and errored on")

// RetrieveWithInputReplaced parses given input against all possible token strings
// using regex to grab a list of found tokens in the given string and returns the replaced string
func (c *ConfigManager) RetrieveWithInputReplaced(input string) (string, error) {
	// replaces all env vars using strict mode of no unset and no empty
	//
	// NOTE: this happens before the FindTokens is called
	// currently it uses a regex, and envsubst uses a more robust lexer => parser mechanism
	//
	// NOTE: configmanager needs an own lexer => parser to allow for easier modification extension in the future
	if c.GeneratorConfig().EnvSubstEnabled() {
		var err error
		input, err = envsubst.StringRestrictedNoDigit(input, true, true, false)
		if err != nil {
			return "", fmt.Errorf("%w\n%v", ErrEnvSubst, err)
		}
	}
	m, err := c.retrieve(FindTokens(input))

	if err != nil {
		return "", err
	}

	return replaceString(m, input), nil
}

// FindTokens extracts all replaceable tokens
// from a given input string
func FindTokens(input string) []string {
	tokens := []string{}
	for k := range config.VarPrefix {
		matches := regexp.MustCompile(regexp.QuoteMeta(string(k))+`.(`+TERMINATING_CHAR+`+)`).FindAllString(input, -1)
		tokens = append(tokens, matches...)
	}
	return tokens
}

// replaceString fills tokens in a provided input with their actual secret/config values
func replaceString(inputMap generator.ParsedMap, inputString string) string {

	oldNew := []string(nil)
	// ordered values by index
	for _, ov := range orderedKeysList(inputMap) {
		oldNew = append(oldNew, ov, fmt.Sprint(inputMap[ov]))
	}
	replacer := strings.NewReplacer(oldNew...)
	return replacer.Replace(inputString)
}

func orderedKeysList(inputMap generator.ParsedMap) []string {
	mkeys := inputMap.MapKeys()
	// order map by keys length so that when passed to the
	// replacer it will replace the longest first
	// removing the possibility of partially overwriting
	// another token with same prefix
	// the default sort is ascending
	slices.Sort(mkeys)
	return mkeys
}

// RetrieveMarshalledJson
//
// It marshalls an input pointer value of a type with appropriate struct tags in JSON
// marshalls it into a string and runs the appropriate token replacement.
// and fills the same pointer value with the replaced fields.
//
// This is useful for when you have another tool or framework already passing you a known type.
// e.g. a CRD Spec in kubernetes - where you POSTed the json/yaml spec with tokens in it
// but now want to use them with tokens replaced for values in a stateless way.
//
// Enables you to store secrets in CRD Specs and other metadata your controller can use
func (cm *ConfigManager) RetrieveMarshalledJson(input any) error {

	// marshall type into a []byte
	// with tokens in a string like object
	rawBytes, err := json.Marshal(input)
	if err != nil {
		return err
	}
	// run the replacement of tokens for values
	replacedString, err := cm.RetrieveWithInputReplaced(string(rawBytes))
	if err != nil {
		return err
	}
	// replace the original pointer value with replaced tokens
	if err := json.Unmarshal([]byte(replacedString), input); err != nil {
		return err
	}
	return nil
}

// RetrieveUnmarshalledFromJson
// It accepts an already marshalled byte slice and pointer to the value type.
// It fills the type with the replaced
func (c *ConfigManager) RetrieveUnmarshalledFromJson(input []byte, output any) error {
	replaced, err := c.RetrieveWithInputReplaced(string(input))
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(replaced), output); err != nil {
		return err
	}
	return nil
}

// RetrieveMarshalledYaml
//
// Same as RetrieveMarshalledJson
func (cm *ConfigManager) RetrieveMarshalledYaml(input any) error {

	// marshall type into a []byte
	// with tokens in a string like object
	rawBytes, err := yaml.Marshal(input)
	if err != nil {
		return err
	}
	// run the replacement of tokens for values
	replacedString, err := cm.RetrieveWithInputReplaced(string(rawBytes))
	if err != nil {
		return err
	}
	// replace the original pointer value with replaced tokens
	if err := yaml.Unmarshal([]byte(replacedString), input); err != nil {
		return err
	}
	return nil
}

// RetrieveUnmarshalledFromYaml
//
// Same as RetrieveUnmarshalledFromJson
func (c *ConfigManager) RetrieveUnmarshalledFromYaml(input []byte, output any) error {
	replaced, err := c.RetrieveWithInputReplaced(string(input))
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal([]byte(replaced), output); err != nil {
		return err
	}
	return nil
}
