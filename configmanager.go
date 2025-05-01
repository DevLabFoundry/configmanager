package configmanager

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v3"

	"github.com/a8m/envsubst"
	"github.com/dnitsch/configmanager/pkg/generator"
)

const (
	TERMINATING_CHAR string = `[^\'\"\s\n\\]`
)

type ConfigManager struct{}

// Retrieve gets a rawMap from a set implementation
// will be empty if no matches found
func (c *ConfigManager) Retrieve(tokens []string, config generator.GenVarsConfig) (generator.ParsedMap, error) {
	gv := generator.NewGenerator().WithConfig(&config)
	return retrieve(tokens, gv)
}

// GenerateAPI
type GenerateAPI interface {
	Generate(tokens []string) (generator.ParsedMap, error)
	Config() *generator.GenVarsConfig
}

func retrieve(tokens []string, gv GenerateAPI) (generator.ParsedMap, error) {
	return gv.Generate(tokens)
}

// RetrieveWithInputReplaced parses given input against all possible token strings
// using regex to grab a list of found tokens in the given string and returns the replaced string
func (c *ConfigManager) RetrieveWithInputReplaced(input string, config generator.GenVarsConfig) (string, error) {
	gv := generator.NewGenerator().WithConfig(&config)
	return retrieveWithInputReplaced(input, gv)
}

func retrieveWithInputReplaced(input string, gv GenerateAPI) (string, error) {
	// replaces all env vars using strict mode of no unset and no empty
	//
	// NOTE: this happens before the FindTokens is called
	// currently it uses a regex, and envsubst uses a more robust lexer => parser mechanism
	//
	// TODO: configmanager needs an own lexer => parser to allow for easier modification extension in the future
	if gv.Config().EnvSubstEnabled() {
		var err error
		input, err = envsubst.StringRestrictedNoDigit(input, true, true, false)
		if err != nil {
			return "", err
		}
	}
	m, err := retrieve(FindTokens(input), gv)
	if err != nil {
		return "", err
	}

	return replaceString(m, input), nil
}

// FindTokens extracts all replaceable tokens
// from a given input string
func FindTokens(input string) []string {
	tokens := []string{}
	for k := range generator.VarPrefix {
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
	mkeys := make([]string, 0, len(inputMap))
	for k := range inputMap {
		mkeys = append(mkeys, k)
	}

	// order map by keys length so that when passed to the
	// replacer it will replace the longest first
	// removing the possibility of partially overwriting
	// another token with same prefix
	sort.SliceStable(mkeys, func(i, j int) bool {
		l1, l2 := len(mkeys[i]), len(mkeys[j])
		if l1 != l2 {
			return l1 > l2
		}
		return mkeys[i] > mkeys[j]
	})
	return mkeys
}

type CMRetrieveWithInputReplacediface interface {
	RetrieveWithInputReplaced(input string, config generator.GenVarsConfig) (string, error)
}

// KubeControllerSpecHelper is a helper method, it marshalls an input value of that type into a string and passes it into the relevant configmanger retrieve method
// and returns the unmarshalled object back.
//
// # It accepts a DI of configmanager and the config (for testability) to replace all occurences of replaceable tokens inside a Marshalled string of that type
//
// Deprecated: Left for compatibility reasons
func KubeControllerSpecHelper[T any](inputType T, cm CMRetrieveWithInputReplacediface, config generator.GenVarsConfig) (*T, error) {
	outType := new(T)
	rawBytes, err := json.Marshal(inputType)
	if err != nil {
		return nil, err
	}

	replaced, err := cm.RetrieveWithInputReplaced(string(rawBytes), config)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(replaced), outType); err != nil {
		return nil, err
	}
	return outType, nil
}

// RetrieveMarshalledJson is a helper method.
//
// It marshalls an input value of that type into a []byte and passes it into the relevant configmanger retrieve method
// returns the unmarshalled object back with all tokens replaced IF found for their specific vault implementation values.
// Type must contain all public members with a JSON tag on the struct
func RetrieveMarshalledJson[T any](input *T, cm CMRetrieveWithInputReplacediface, config generator.GenVarsConfig) (*T, error) {
	outType := new(T)
	rawBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	outVal, err := RetrieveUnmarshalledFromJson(rawBytes, outType, cm, config)
	if err != nil {
		return outType, err
	}

	return outVal, nil
}

// RetrieveUnmarshalledFromJson is a helper method.
// Same as RetrieveMarshalledJson but it accepts an already marshalled byte slice
func RetrieveUnmarshalledFromJson[T any](input []byte, output *T, cm CMRetrieveWithInputReplacediface, config generator.GenVarsConfig) (*T, error) {
	replaced, err := cm.RetrieveWithInputReplaced(string(input), config)
	if err != nil {
		return output, err
	}
	if err := json.Unmarshal([]byte(replaced), output); err != nil {
		return output, err
	}
	return output, nil
}

// RetrieveMarshalledYaml is a helper method.
//
// It marshalls an input value of that type into a []byte and passes it into the relevant configmanger retrieve method
// returns the unmarshalled object back with all tokens replaced IF found for their specific vault implementation values.
// Type must contain all public members with a YAML tag on the struct
func RetrieveMarshalledYaml[T any](input *T, cm CMRetrieveWithInputReplacediface, config generator.GenVarsConfig) (*T, error) {
	outType := new(T)

	rawBytes, err := yaml.Marshal(input)
	if err != nil {
		return outType, err
	}
	outVal, err := RetrieveUnmarshalledFromYaml(rawBytes, outType, cm, config)
	if err != nil {
		return outType, err
	}

	return outVal, nil
}

// RetrieveUnmarshalledFromYaml is a helper method.
//
// Same as RetrieveMarshalledYaml but it accepts an already marshalled byte slice
func RetrieveUnmarshalledFromYaml[T any](input []byte, output *T, cm CMRetrieveWithInputReplacediface, config generator.GenVarsConfig) (*T, error) {
	replaced, err := cm.RetrieveWithInputReplaced(string(input), config)
	if err != nil {
		return output, err
	}
	if err := yaml.Unmarshal([]byte(replaced), output); err != nil {
		return output, err
	}
	return output, nil
}
