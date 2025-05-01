package configmanager

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/dnitsch/configmanager/internal/testutils"
	"github.com/dnitsch/configmanager/pkg/generator"
)

type mockConfigManageriface interface {
	Retrieve(tokens []string, config generator.GenVarsConfig) (generator.ParsedMap, error)
	RetrieveWithInputReplaced(input string, config generator.GenVarsConfig) (string, error)
	Insert(force bool) error
}

type mockGenVars struct {
	config func() *generator.GenVarsConfig
}

var (
	testKey = "FOO#/test"
	testVal = "val1"
)

func (m *mockGenVars) Generate(tokens []string) (generator.ParsedMap, error) {
	pm := generator.ParsedMap{}
	pm[testKey] = testVal
	return pm, nil
}

func (m *mockGenVars) Config() *generator.GenVarsConfig {
	if m.config != nil {
		return m.config()
	}
	return generator.NewConfig()
}

func (m *mockGenVars) ConvertToExportVar() []string {
	return []string{}
}

func (m *mockGenVars) FlushToFile(w io.Writer, str []string) error {
	return nil
}

func (m *mockGenVars) StrToFile(w io.Writer, str string) error {
	return nil
}

func Test_retrieve(t *testing.T) {
	tests := []struct {
		name      string
		tokens    []string
		genvar    generator.GenVarsiface
		expectKey string
		expectVal string
	}{
		{
			name:      "standard",
			tokens:    []string{"FOO#/test"},
			genvar:    &mockGenVars{},
			expectKey: testKey,
			expectVal: testVal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm, err := retrieve(tt.tokens, tt.genvar)
			if err != nil {
				t.Errorf(testutils.TestPhrase, err, nil)
			}
			for k, v := range pm {
				if k != tt.expectKey {
					t.Errorf(testutils.TestPhrase, k, tt.expectKey)
				}
				if v != tt.expectVal {
					t.Errorf(testutils.TestPhrase, v, tt.expectVal)
				}
			}
		})
	}
}

func Test_retrieveWithInputReplaced(t *testing.T) {
	tests := map[string]struct {
		name   string
		input  string
		genvar *mockGenVars
		expect string
	}{
		"strYaml": {
			input: `
space: preserved
	indents: preserved
	arr: [ "FOO#/test" ]
	// comments preserved
	arr:
		- "FOO#/test"
`,
			genvar: &mockGenVars{},
			expect: `
space: preserved
	indents: preserved
	arr: [ "val1" ]
	// comments preserved
	arr:
		- "val1"
`,
		},
		"strToml": {
			input: `
// TOML
[[somestuff]]
key = "FOO#/test" 
`,
			genvar: &mockGenVars{},
			expect: `
// TOML
[[somestuff]]
key = "val1" 
`,
		},
		"strTomlWithoutQuotes": {
			input: `
// TOML
[[somestuff]]
key = FOO#/test
key2 = FOO#/test
key3 = FOO#/test
key4 = FOO#/test
`,
			genvar: &mockGenVars{},
			expect: `
// TOML
[[somestuff]]
key = val1
key2 = val1
key3 = val1
key4 = val1
`,
		},
		"strTomlWithoutMultiline": {
			input: `
export FOO='FOO#/test'
export FOO1=FOO#/test
export FOO2="FOO#/test"
export FOO3=FOO#/test
export FOO4=FOO#/test

[[section]]

foo23 = FOO#/test
`,
			genvar: &mockGenVars{},
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
			input:  `"{\"patchPayloadTemplate\":\"{\\\"password\\\":\\\"FOO#/test\\\",\\\"passwordConfirm\\\":\\\"FOO#/test\\\"}\\n\"}"`,
			genvar: &mockGenVars{},
			expect: `"{\"patchPayloadTemplate\":\"{\\\"password\\\":\\\"val1\\\",\\\"passwordConfirm\\\":\\\"val1\\\"}\\n\"}"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := retrieveWithInputReplaced(tt.input, tt.genvar)
			if err != nil {
				t.Errorf("failed with %v", err)
			}
			if got != tt.expect {
				t.Errorf(testutils.TestPhrase, got, tt.expect)
			}
		})
	}
}

func Test_replaceString_with_envsubst(t *testing.T) {
	t.Parallel()
	ttests := map[string]struct {
		want   string
		setup  func() func()
		input  string
		genvar *mockGenVars
	}{
		"replaced successfully": {
			input: `{"patchPayloadTemplate":"{"password":"FOO#/${BAR}","passwordConfirm":"FOO#/${BAZ:-test}"}}`,
			want:  `{"patchPayloadTemplate":"{"password":"val1","passwordConfirm":"val1"}}`,
			genvar: &mockGenVars{config: func() *generator.GenVarsConfig {
				return generator.NewConfig().WithEnvSubst(true)
			}},
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

			got, err := retrieveWithInputReplaced(tt.input, tt.genvar)
			if err != nil {
				t.Errorf("failed with %v", err)
			}
			if got != tt.want {
				t.Errorf(testutils.TestPhrase, got, tt.want)
			}
		})
	}
}

func Test_replaceString(t *testing.T) {
	tests := []struct {
		name      string
		parsedMap generator.ParsedMap
		inputStr  string
		expectStr string
	}{
		{
			name: "ordered correctly",
			parsedMap: generator.ParsedMap{
				"AZKVSECRET#/test-vault/db-config|user": "foo",
				"AZKVSECRET#/test-vault/db-config|pass": "bar",
				"AZKVSECRET#/test-vault/db-config":      fmt.Sprintf("%v", "{\"user\": \"foo\", \"pass\": \"bar\"}"),
			},
			inputStr: `app: foo
db2: AZKVSECRET#/test-vault/db-config
db: 
	user: AZKVSECRET#/test-vault/db-config|user
	pass: AZKVSECRET#/test-vault/db-config|pass
`,
			expectStr: `app: foo
db2: {"user": "foo", "pass": "bar"}
db: 
	user: foo
	pass: bar
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceString(tt.parsedMap, tt.inputStr)
			if got != tt.expectStr {
				t.Errorf(testutils.TestPhrase, got, tt.expectStr)
			}
		})
	}
}

type MockCfgMgr struct {
	retrieveInput func(input string, config generator.GenVarsConfig) (string, error)
	// retrieve func(input string, config generator.GenVarsConfig) (string, error)
}

func (m *MockCfgMgr) RetrieveWithInputReplaced(input string, config generator.GenVarsConfig) (string, error) {
	return m.retrieveInput(input, config)
}

func (m *MockCfgMgr) Insert(force bool) error {
	return nil
}

func (m *MockCfgMgr) Retrieve(tokens []string, config generator.GenVarsConfig) (generator.ParsedMap, error) {
	return nil, nil
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

func Test_KubeControllerSpecHelper(t *testing.T) {
	tests := []struct {
		name     string
		testType testSimpleStruct
		expect   testSimpleStruct
		cfmgr    func(t *testing.T) mockConfigManageriface
	}{
		{
			name: "happy path simple struct",
			testType: testSimpleStruct{
				Foo: testTokenAWS,
				Bar: "quz",
			},
			expect: testSimpleStruct{
				Foo: "baz",
				Bar: "quz",
			},
			cfmgr: func(t *testing.T) mockConfigManageriface {
				mcm := &MockCfgMgr{}
				mcm.retrieveInput = func(input string, config generator.GenVarsConfig) (string, error) {
					return `{"foo":"baz","bar":"quz"}`, nil
				}
				return mcm
			},
		},
		{
			name: "happy path simple struct2",
			testType: testSimpleStruct{
				Foo: "AWSSECRETS:///bar/foo2",
				Bar: "quz",
			},
			expect: testSimpleStruct{
				Foo: "baz2",
				Bar: "quz",
			},
			cfmgr: func(t *testing.T) mockConfigManageriface {
				mcm := &MockCfgMgr{}
				mcm.retrieveInput = func(input string, config generator.GenVarsConfig) (string, error) {
					return `{"foo":"baz2","bar":"quz"}`, nil
				}
				return mcm
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			config := generator.NewConfig()
			resp, err := KubeControllerSpecHelper(tt.testType, tt.cfmgr(t), *config)
			if err != nil {
				t.Errorf(testutils.TestPhrase, err.Error(), nil)
			}
			if !reflect.DeepEqual(resp, &tt.expect) {
				t.Error("")
			}
		})
	}
}

func Test_KubeControllerComplex(t *testing.T) {
	tests := []struct {
		name     string
		testType testNestedStruct
		expect   testNestedStruct
		cfmgr    func(t *testing.T) mockConfigManageriface
	}{
		{
			name: "happy path complex struct",
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
			cfmgr: func(t *testing.T) mockConfigManageriface {
				mcm := &MockCfgMgr{}
				mcm.retrieveInput = func(input string, config generator.GenVarsConfig) (string, error) {
					return `{"foo":"baz","bar":"quz", "lol":{"bla":"booo","another":{"number": 1235, "float": 123.09}}}`, nil
				}
				return mcm
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := generator.NewConfig().WithTokenSeparator("://")
			got, err := KubeControllerSpecHelper(tt.testType, tt.cfmgr(t), *config)
			if err != nil {
				t.Errorf(testutils.TestPhrase, err.Error(), nil)
			}
			if !reflect.DeepEqual(got, &tt.expect) {
				t.Errorf(testutils.TestPhraseWithContext, "returned types do not deep equal", got, tt.expect)
			}
		})
	}
}

func Test_YamlRetrieveMarshalled(t *testing.T) {
	tests := []struct {
		name     string
		testType *testNestedStruct
		expect   testNestedStruct
		cfmgr    func(t *testing.T) mockConfigManageriface
	}{
		{
			name: "complex struct - complete",
			testType: &testNestedStruct{
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
			cfmgr: func(t *testing.T) mockConfigManageriface {
				mcm := &MockCfgMgr{}
				mcm.retrieveInput = func(input string, config generator.GenVarsConfig) (string, error) {
					return `{"foo":"baz","bar":"quz", "lol":{"bla":"booo","another":{"number": 1235, "float": 123.09}}}`, nil
				}
				return mcm
			},
		},
		{
			name: "complex struct - missing fields",
			testType: &testNestedStruct{
				Foo: testTokenAWS,
				Bar: "quz",
			},
			expect: testNestedStruct{
				Foo: "baz",
				Bar: "quz",
				Lol: testLol{},
			},
			cfmgr: func(t *testing.T) mockConfigManageriface {
				mcm := &MockCfgMgr{}
				mcm.retrieveInput = func(input string, config generator.GenVarsConfig) (string, error) {
					return `{"foo":"baz","bar":"quz", "lol":{"bla":"","another":{"number": 0, "float": 0}}}`, nil
				}
				return mcm
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := generator.NewConfig().WithTokenSeparator("://")

			got, err := RetrieveMarshalledYaml(tt.testType, tt.cfmgr(t), *config)
			if err != nil {
				t.Errorf(testutils.TestPhrase, err.Error(), nil)
			}
			if !reflect.DeepEqual(got, &tt.expect) {
				t.Errorf(testutils.TestPhraseWithContext, "returned types do not deep equal", got, tt.expect)
			}
		})
	}
}

func Test_YamlRetrieveMarshalled_errored(t *testing.T) {
	tests := []struct {
		name     string
		testType *testNestedStruct
		expect   error
		cfmgr    func(t *testing.T) mockConfigManageriface
	}{
		{
			name: "complex struct - complete",
			testType: &testNestedStruct{
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
			// expect: testNestedStruct{
			// 	Foo: "baz",
			// 	Bar: "quz",
			// 	Lol: testLol{
			// 		Bla: "booo",
			// 		Another: testAnotherNEst{
			// 			Number: 1235,
			// 			Float:  123.09,
			// 		},
			// 	},
			// },
			cfmgr: func(t *testing.T) mockConfigManageriface {
				mcm := &MockCfgMgr{}
				mcm.retrieveInput = func(input string, config generator.GenVarsConfig) (string, error) {
					return ``, fmt.Errorf("%s", "error decoding")
				}
				return mcm
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := generator.NewConfig().WithTokenSeparator("://")

			_, err := RetrieveMarshalledYaml(tt.testType, tt.cfmgr(t), *config)
			if err == nil {
				t.Errorf(testutils.TestPhrase, nil, err.Error())
			}
		})
	}
}

func Test_RetrieveMarshalledJson(t *testing.T) {
	tests := []struct {
		name     string
		testType *testNestedStruct
		expect   testNestedStruct
		cfmgr    func(t *testing.T) mockConfigManageriface
	}{
		{
			name: "happy path complex struct complete",
			testType: &testNestedStruct{
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
			cfmgr: func(t *testing.T) mockConfigManageriface {
				mcm := &MockCfgMgr{}
				mcm.retrieveInput = func(input string, config generator.GenVarsConfig) (string, error) {
					return `{"foo":"baz","bar":"quz", "lol":{"bla":"booo","another":{"number": 1235, "float": 123.09}}}`, nil
				}
				return mcm
			},
		},
		{
			name: "complex struct - missing fields",
			testType: &testNestedStruct{
				Foo: testTokenAWS,
				Bar: "quz",
			},
			expect: testNestedStruct{
				Foo: "baz",
				Bar: "quz",
				Lol: testLol{},
			},
			cfmgr: func(t *testing.T) mockConfigManageriface {
				mcm := &MockCfgMgr{}
				mcm.retrieveInput = func(input string, config generator.GenVarsConfig) (string, error) {
					return `{"foo":"baz","bar":"quz", "lol":{"bla":"","another":{"number": 0, "float": 0}}}`, nil
				}
				return mcm
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := generator.NewConfig().WithTokenSeparator("://")
			got, err := RetrieveMarshalledJson(tt.testType, tt.cfmgr(t), *config)
			if err != nil {
				t.Errorf(testutils.TestPhrase, err.Error(), nil)
			}
			if !reflect.DeepEqual(got, &tt.expect) {
				t.Errorf(testutils.TestPhraseWithContext, "returned types do not deep equal", got, tt.expect)
			}
		})
	}
}

func TestFindTokens(t *testing.T) {
	ttests := map[string]struct {
		input  string
		expect []string
	}{
		"extract from text correctly": {
			`Where does it come from?
			Contrary to popular belief, 
			Lorem Ipsum is AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj <= in middle of sentencenot simply random text.
			It has roots in a piece of classical Latin literature from 45 
			BC, making it over 2000 years old. Richard McClintock, a Latin professor at
			 Hampden-Sydney College in Virginia, looked up one of the more obscure Latin words, c
			 onsectetur, from a Lorem Ipsum passage , at the end of line => AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj
			  and going through the cites of the word in c
			 lassical literature, discovered the undoubtable source. Lorem Ipsum comes from secti
			 ons in singles =>'AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj'1.10.32 and 1.10.33 of "de Finibus Bonorum et Malorum" (The Extremes of Good and Evil)
			 in doubles => "AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj"
			  by Cicero, written in 45 BC. This book is a treatise on the theory of ethics, very popular 
			  during the  :=> embedded in text RenaissanceAWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[] embedded in text <=: 
			  The first line of Lorem Ipsum, "Lorem ipsum dolor sit amet..", comes from a line in section 1.10.32.`,
			[]string{
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[]"},
		},
		"unknown implementation not picked up": {
			`foo: AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj
				bar: AWSPARAMSTR#bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[version:123]
				unknown: GCPPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj`,
			[]string{
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"AWSPARAMSTR#bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[version:123]"},
		},
		"all implementations": {
			`param: AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj
			secretsmgr: AWSSECRETS#bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[version:123]
			gcp: GCPSECRETS:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj
			vault: VAULT:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[]
			som othere strufsd
			azkv: AZKVSECRET:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj`,
			[]string{
				"GCPSECRETS:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"AWSSECRETS#bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[version:123]",
				"AZKVSECRET:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"VAULT:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[]"},
		},
		// "EnvSubst with bash var": {
		// 	`param: AWSPARAMSTR:///${FOO:-bar}
		// 	secretsmgr: AWSSECRETS#bar/${FOO:-bar}[version:123]
		// 	gcp: GCPSECRETS://${FOO:-'bar'}
		// 	gcp: GCPSECRETS://${FOO:-"bar"}`,
		// 	[]string{
		// 		"AWSPARAMSTR:///${FOO:-bar}",
		// 		"AWSSECRETS#bar/${FOO:-bar}[version:123]",
		// 		"GCPSECRETS://${FOO:-'bar'}",
		// 		`GCPSECRETS://${FOO:-"bar"}`},
		// },
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			got := FindTokens(tt.input)
			sort.Strings(got)
			sort.Strings(tt.expect)

			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("input=(%q)\n\ngot: %v\n\nwant: %v", tt.input, got, tt.expect)
			}
		})
	}
}
