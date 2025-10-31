package configmanager_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/DevLabFoundry/configmanager/v3"
	"github.com/DevLabFoundry/configmanager/v3/generator"
	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/testutils"
	"github.com/go-test/deep"
	"gopkg.in/yaml.v3"
)

type mockGenerator struct {
	generate func(tokens []string) (generator.ReplacedToken, error)
}

func (m *mockGenerator) Generate(tokens []string) (generator.ReplacedToken, error) {
	if m.generate != nil {
		return m.generate(tokens)
	}
	pm := generator.ReplacedToken{}
	pm["FOO#/test"] = "val1"
	pm["ANOTHER://bar/quz"] = "fux"
	pm["ZODTHER://bar/quz"] = "xuf"
	return pm, nil
}

func Test_Retrieve_from_token_list(t *testing.T) {
	tests := map[string]struct {
		tokens    []string
		genvar    *mockGenerator
		expectKey string
		expectVal string
	}{
		"standard": {
			tokens:    []string{"FOO#/test", "ANOTHER://bar/quz", "ZODTHER://bar/quz"},
			genvar:    &mockGenerator{},
			expectKey: "FOO#/test",
			expectVal: "val1",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cm := configmanager.New(context.TODO())
			cm.WithGenerator(tt.genvar)
			pm, err := cm.Retrieve(tt.tokens)
			if err != nil {
				t.Errorf(testutils.TestPhrase, err, nil)
			}
			if val, found := pm[tt.expectKey]; found {
				if val != pm[tt.expectKey] {
					t.Errorf(testutils.TestPhrase, val, tt.expectVal)
				}
			} else {
				t.Errorf(testutils.TestPhrase, "nil", tt.expectKey)
			}
		})
	}
}

func Test_retrieveReplacedBytes(t *testing.T) {
	tests := map[string]struct {
		name   string
		input  []byte
		genvar *mockGenerator
		expect string
	}{
		"strYaml": {
			input: []byte(`
space: preserved
	indents: preserved
	arr: [ "FOO#/test" ]
	// comments preserved
	arr:
		- "FOO#/test"
		- ANOTHER://bar/quz
`),
			genvar: &mockGenerator{},
			expect: `
space: preserved
	indents: preserved
	arr: [ "val1" ]
	// comments preserved
	arr:
		- "val1"
		- fux
`,
		},
		"strToml": {
			input: []byte(`
// TOML
[[somestuff]]
key = "FOO#/test"
`),
			genvar: &mockGenerator{},
			expect: `
// TOML
[[somestuff]]
key = "val1"
`,
		},
		"strTomlWithoutQuotes": {
			input: []byte(`
// TOML
[[somestuff]]
key = FOO#/test,FOO#/test-FOO#/test
key2 = FOO#/test
key3 = FOO#/test
key4 = FOO#/test
`),
			genvar: &mockGenerator{},
			expect: `
// TOML
[[somestuff]]
key = val1,val1-val1
key2 = val1
key3 = val1
key4 = val1
`,
		},
		"strTomlWithoutMultiline": {
			input: []byte(`
export FOO='FOO#/test'
export FOO1=FOO#/test
export FOO2="FOO#/test"
export FOO3=FOO#/test
export FOO4=FOO#/test

[[section]]

foo23 = FOO#/test
`),
			genvar: &mockGenerator{},
			expect: `
export FOO='val1'
export FOO1=val1
export FOO2="val1"
export FOO3=val1
export FOO4=val1

[[section]]

foo23 = val1
`,
		},
		"escaped input": {
			input:  []byte(`"{\"patchPayloadTemplate\":\"{\\\"password\\\":\\\"FOO#/test\\\",\\\"passwordConfirm\\\":\\\"FOO#/test\\\"}\\n\"}"`),
			genvar: &mockGenerator{},
			expect: `"{\"patchPayloadTemplate\":\"{\\\"password\\\":\\\"val1\\\",\\\"passwordConfirm\\\":\\\"val1\\\"}\\n\"}"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := configmanager.New(context.TODO())
			cm.WithGenerator(tt.genvar)
			got, err := cm.RetrieveReplacedBytes([]byte(tt.input))
			if err != nil {
				t.Errorf("failed with %v", err)
			}
			if string(got) != string(tt.expect) {
				t.Errorf(testutils.TestPhrase, got, tt.expect)
			}
		})
	}
}

func Test_replaceString_with_envsubst(t *testing.T) {
	ttests := map[string]struct {
		expect string
		setup  func() func()
		input  string
		genvar *mockGenerator
	}{
		"replaced successfully": {
			input:  `{"patchPayloadTemplate":"{"password":"FOO#/${BAR}","passwordConfirm":"FOO#/${BAZ:-test}"}}`,
			expect: `{"patchPayloadTemplate":"{"password":"val1","passwordConfirm":"val1"}}`,
			genvar: &mockGenerator{},
			setup: func() func() {
				os.Setenv("BAR", "test")
				return func() {
					os.Unsetenv("BAR")
				}
			},
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			tearDown := tt.setup()
			defer tearDown()

			cm := configmanager.New(context.TODO())
			cm.WithGenerator(tt.genvar)
			cm.Config.WithEnvSubst(true)
			got, err := cm.RetrieveReplacedString(tt.input)
			if err != nil {
				t.Errorf("failed with %v", err)
			}
			if got != tt.expect {
				t.Errorf(testutils.TestPhrase, got, tt.expect)
			}
		})
	}
}

type testSimpleStruct struct {
	Foo string `json:"foo" yaml:"foo"`
	Bar string `json:"bar" yaml:"bar"`
}

type testAnotherNEst struct {
	Number int     `json:"number,omitempty" yaml:"number"`
	Float  float32 `json:"float,omitempty" yaml:"float"`
}

type testLol struct {
	Bla     string          `json:"bla,omitempty" yaml:"bla"`
	Another testAnotherNEst `json:"another,omitempty" yaml:"another"`
}

type testNestedStruct struct {
	Foo string  `json:"foo" yaml:"foo"`
	Bar string  `json:"bar" yaml:"bar"`
	Lol testLol `json:"lol,omitempty" yaml:"lol"`
}

const (
	testTokenAWS = "AWSSECRETS:///bar/foo"
)

var marshallTests = map[string]struct {
	testType  testNestedStruct
	expect    testNestedStruct
	generator func(t *testing.T) *mockGenerator
}{
	"happy path complex struct complete": {
		testType: testNestedStruct{
			Foo: testTokenAWS,
			Bar: "quz",
			Lol: testLol{
				Bla: "booo",
				Another: testAnotherNEst{
					Number: 1235,
					Float:  123.09,
				},
			},
		},
		expect: testNestedStruct{
			Foo: "baz",
			Bar: "quz",
			Lol: testLol{
				Bla: "booo",
				Another: testAnotherNEst{
					Number: 1235,
					Float:  123.09,
				},
			},
		},
		generator: func(t *testing.T) *mockGenerator {
			m := &mockGenerator{}
			m.generate = func(tokens []string) (generator.ReplacedToken, error) {
				pm := make(generator.ReplacedToken)
				pm[testTokenAWS] = "baz"
				return pm, nil
			}
			return m
		},
	},
	"complex struct - missing fields": {
		testType: testNestedStruct{
			Foo: testTokenAWS,
			Bar: "quz",
		},
		expect: testNestedStruct{
			Foo: "baz",
			Bar: "quz",
			Lol: testLol{},
		},
		generator: func(t *testing.T) *mockGenerator {
			m := &mockGenerator{}
			m.generate = func(tokens []string) (generator.ReplacedToken, error) {
				pm := make(generator.ReplacedToken)
				pm[testTokenAWS] = "baz"
				return pm, nil
			}
			return m
		},
	},
}

func Test_RetrieveBytes_MarshalledJson(t *testing.T) {
	for name, tt := range marshallTests {
		t.Run(name, func(t *testing.T) {
			c := configmanager.New(context.TODO())
			c.Config.WithTokenSeparator("://")
			c.WithGenerator(tt.generator(t))

			b, err := json.Marshal(tt.testType)
			if err != nil {
				t.Fatal(err)
			}
			got, err := c.RetrieveReplacedBytes(b)
			output := testNestedStruct{}
			json.Unmarshal(got, &output)
			MarhsalledHelper(t, err, &output, &tt.expect)
		})
	}
}

// func Example_RetrieveReplacedBytesMarshalledJSON(t *testing.T) {
// 	return
// }

func Test_RetrieveBytes_MarshalledYaml(t *testing.T) {
	for name, tt := range marshallTests {
		t.Run(name, func(t *testing.T) {
			c := configmanager.New(context.TODO())
			c.Config.WithTokenSeparator("://")
			c.WithGenerator(tt.generator(t))

			b, err := yaml.Marshal(tt.testType)
			if err != nil {
				t.Fatal(err)
			}
			got, err := c.RetrieveReplacedBytes(b)
			output := testNestedStruct{}
			yaml.Unmarshal(got, &output)
			MarhsalledHelper(t, err, &output, &tt.expect)
		})
	}
}

func MarhsalledHelper(t *testing.T, err error, input, expectOut any) {
	t.Helper()
	if err != nil {
		t.Errorf(testutils.TestPhrase, err.Error(), nil)
	}
	if !reflect.DeepEqual(input, expectOut) {
		t.Errorf(testutils.TestPhraseWithContext, "returned types do not deep equal", input, expectOut)
	}
}

// config tests
func Test_Generator_Config_(t *testing.T) {
	ttests := map[string]struct {
		expect                             config.GenVarsConfig
		keySeparator, tokenSep, outputPath string
	}{
		"default config": {
			expect: config.NewConfig().Config(),
			// keySeparator: "|", tokenSep: "://",outputPath:"",
		},
		"outputPath overwritten only": {
			expect: (config.NewConfig()).WithOutputPath("baresd").Config(),
			// keySeparator: "|", tokenSep: "://",
			outputPath: "baresd",
		},
		"outputPath and keysep overwritten": {
			expect:       (config.NewConfig()).WithOutputPath("baresd").WithKeySeparator("##").Config(),
			keySeparator: "##",
			outputPath:   "baresd",
			// tokenSep: "://",
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			cm := configmanager.New(context.TODO())
			if tt.keySeparator != "" {
				cm.Config.WithKeySeparator(tt.keySeparator)
			}
			if tt.tokenSep != "" {
				cm.Config.WithTokenSeparator(tt.tokenSep)
			}
			if tt.outputPath != "" {
				cm.Config.WithOutputPath(tt.outputPath)
			}
			got := cm.GeneratorConfig()
			if diff := deep.Equal(got, &tt.expect); diff != nil {
				t.Errorf(testutils.TestPhraseWithContext, "generator config", fmt.Sprintf("%v", got), fmt.Sprintf("%v", tt.expect))
			}
		})
	}
}
