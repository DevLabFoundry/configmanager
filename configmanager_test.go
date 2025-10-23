package configmanager_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"slices"
	"sort"
	"testing"

	"github.com/DevLabFoundry/configmanager/v2"
	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/internal/testutils"
	"github.com/DevLabFoundry/configmanager/v2/generator"
	"github.com/go-test/deep"
)

type mockGenerator struct {
	generate func(tokens []string) (generator.ParsedMap, error)
}

func (m *mockGenerator) Generate(tokens []string) (generator.ParsedMap, error) {
	if m.generate != nil {
		return m.generate(tokens)
	}
	pm := generator.ParsedMap{}
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

func Test_retrieveWithInputReplaced(t *testing.T) {
	tests := map[string]struct {
		name   string
		input  string
		genvar *mockGenerator
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
		- ANOTHER://bar/quz
`,
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
			input: `
// TOML
[[somestuff]]
key = "FOO#/test"
`,
			genvar: &mockGenerator{},
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
key = FOO#/test,FOO#/test-FOO#/test
key2 = FOO#/test
key3 = FOO#/test
key4 = FOO#/test
`,
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
			input: `
export FOO='FOO#/test'
export FOO1=FOO#/test
export FOO2="FOO#/test"
export FOO3=FOO#/test
export FOO4=FOO#/test

[[section]]

foo23 = FOO#/test
`,
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
			input:  `"{\"patchPayloadTemplate\":\"{\\\"password\\\":\\\"FOO#/test\\\",\\\"passwordConfirm\\\":\\\"FOO#/test\\\"}\\n\"}"`,
			genvar: &mockGenerator{},
			expect: `"{\"patchPayloadTemplate\":\"{\\\"password\\\":\\\"val1\\\",\\\"passwordConfirm\\\":\\\"val1\\\"}\\n\"}"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := configmanager.New(context.TODO())
			cm.WithGenerator(tt.genvar)
			got, err := cm.RetrieveWithInputReplaced(tt.input)
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
			got, err := cm.RetrieveWithInputReplaced(tt.input)
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
			m.generate = func(tokens []string) (generator.ParsedMap, error) {
				pm := make(generator.ParsedMap)
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
			m.generate = func(tokens []string) (generator.ParsedMap, error) {
				pm := make(generator.ParsedMap)
				pm[testTokenAWS] = "baz"
				return pm, nil
			}
			return m
		},
	},
}

func Test_RetrieveMarshalledJson(t *testing.T) {
	for name, tt := range marshallTests {
		t.Run(name, func(t *testing.T) {
			c := configmanager.New(context.TODO())
			c.Config.WithTokenSeparator("://")
			c.WithGenerator(tt.generator(t))

			input := &tt.testType
			err := c.RetrieveMarshalledJson(input)
			MarhsalledHelper(t, err, input, &tt.expect)
		})
	}
}

func Test_YamlRetrieveMarshalled(t *testing.T) {
	for name, tt := range marshallTests {
		t.Run(name, func(t *testing.T) {
			c := configmanager.New(context.TODO())
			c.Config.WithTokenSeparator("://")
			c.WithGenerator(tt.generator(t))

			input := &tt.testType
			err := c.RetrieveMarshalledYaml(input)
			MarhsalledHelper(t, err, input, &tt.expect)
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

func Test_YamlRetrieveUnmarshalled(t *testing.T) {
	ttests := map[string]struct {
		input     []byte
		expect    testNestedStruct
		generator func(t *testing.T) *mockGenerator
	}{
		"happy path complex struct complete": {
			input: []byte(`foo: AWSSECRETS:///bar/foo
bar: quz
lol:
  bla: booo
  another:
    number: 1235
    float: 123.09`),
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
				m.generate = func(tokens []string) (generator.ParsedMap, error) {
					pm := make(generator.ParsedMap)
					pm[testTokenAWS] = "baz"
					return pm, nil
				}
				return m
			},
		},
		"complex struct - missing fields": {
			input: []byte(`foo: AWSSECRETS:///bar/foo
bar: quz`),
			expect: testNestedStruct{
				Foo: "baz",
				Bar: "quz",
				Lol: testLol{},
			},
			generator: func(t *testing.T) *mockGenerator {
				m := &mockGenerator{}
				m.generate = func(tokens []string) (generator.ParsedMap, error) {
					pm := make(generator.ParsedMap)
					pm[testTokenAWS] = "baz"
					return pm, nil
				}
				return m
			},
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			c := configmanager.New(context.TODO())
			c.Config.WithTokenSeparator("://")
			c.WithGenerator(tt.generator(t))
			output := &testNestedStruct{}
			err := c.RetrieveUnmarshalledFromYaml(tt.input, output)
			MarhsalledHelper(t, err, output, &tt.expect)
		})
	}
}

func Test_JsonRetrieveUnmarshalled(t *testing.T) {
	tests := map[string]struct {
		input     []byte
		expect    testNestedStruct
		generator func(t *testing.T) *mockGenerator
	}{
		"happy path complex struct complete": {
			input: []byte(`{"foo":"AWSSECRETS:///bar/foo","bar":"quz","lol":{"bla":"booo","another":{"number":1235,"float":123.09}}}`),
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
				m.generate = func(tokens []string) (generator.ParsedMap, error) {
					pm := make(generator.ParsedMap)
					pm[testTokenAWS] = "baz"
					return pm, nil
				}
				return m
			},
		},
		"complex struct - missing fields": {
			input: []byte(`{"foo":"AWSSECRETS:///bar/foo","bar":"quz"}`),
			expect: testNestedStruct{
				Foo: "baz",
				Bar: "quz",
				Lol: testLol{},
			},
			generator: func(t *testing.T) *mockGenerator {
				m := &mockGenerator{}
				m.generate = func(tokens []string) (generator.ParsedMap, error) {
					pm := make(generator.ParsedMap)
					pm[testTokenAWS] = "baz"
					return pm, nil
				}
				return m
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := configmanager.New(context.TODO())
			c.Config.WithTokenSeparator("://")
			c.WithGenerator(tt.generator(t))
			output := &testNestedStruct{}
			err := c.RetrieveUnmarshalledFromJson(tt.input, output)
			MarhsalledHelper(t, err, output, &tt.expect)
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
			secretsmgr: AWSSECRETS://bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[version:123]
			gcp: GCPSECRETS:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj
			vault: VAULT:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[]
			som othere strufsd
			azkv: AZKVSECRET:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj`,
			[]string{
				"GCPSECRETS:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"AWSSECRETS://bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[version:123]",
				"AZKVSECRET:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"VAULT:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[]"},
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			got := configmanager.FindTokens(tt.input)
			sort.Strings(got)
			sort.Strings(tt.expect)

			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("input=(%q)\n\ngot: %v\n\nwant: %v", tt.input, got, tt.expect)
			}
		})
	}
}

func Test_ConfigManager_DiscoverTokens(t *testing.T) {
	ttests := map[string]struct {
		input     string
		separator string
		expect    []string
	}{
		"multiple tokens in single string": {
			`Lorem_Ipsum: AWSPARAMSTR:///path/config|foo.user:AWSPARAMSTR:///path/config|password@AWSPARAMSTR:///path/config|foo.endpoint:AWSPARAMSTR:///path/config|foo.port/?someQ=AWSPARAMSTR:///path/queryparam|p1[version=123]&anotherQ=false`,
			"://",
			[]string{
				"AWSPARAMSTR:///path/config|foo.user",
				"AWSPARAMSTR:///path/config|password",
				"AWSPARAMSTR:///path/config|foo.endpoint",
				"AWSPARAMSTR:///path/config|foo.port",
				"AWSPARAMSTR:///path/queryparam|p1[version=123]"},
		},
		"# tokens in single string": {
			`Lorem_Ipsum: AWSPARAMSTR#/path/config|foo.user:AWSPARAMSTR#/path/config|password@AWSPARAMSTR#/path/config|foo.endpoint:AWSPARAMSTR#/path/config|foo.port/?someQ=AWSPARAMSTR#/path/queryparam|p1[version=123]&anotherQ=false`,
			"#",
			[]string{
				"AWSPARAMSTR#/path/config|foo.user",
				"AWSPARAMSTR#/path/config|password",
				"AWSPARAMSTR#/path/config|foo.endpoint",
				"AWSPARAMSTR#/path/config|foo.port",
				"AWSPARAMSTR#/path/queryparam|p1[version=123]"},
		},
		"without leading slash and path like name # tokens in single string": {
			`Lorem_Ipsum: AWSPARAMSTR#path_config|foo.user:AWSPARAMSTR#path_config|password@AWSPARAMSTR#path_config|foo.endpoint:AWSPARAMSTR#path_config|foo.port/?someQ=AWSPARAMSTR#path_queryparam|p1[version=123]&anotherQ=false`,
			"#",
			[]string{
				"AWSPARAMSTR#path_config|foo.user",
				"AWSPARAMSTR#path_config|password",
				"AWSPARAMSTR#path_config|foo.endpoint",
				"AWSPARAMSTR#path_config|foo.port",
				"AWSPARAMSTR#path_queryparam|p1[version=123]"},
		},
		// Ensures all previous test cases pass as well
		"extract from text correctly": {
			`Where does it come from?
			Contrary to popular belief,
			Lorem Ipsum is AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsfl1 <= in middle of sentencenot simply random text.
			It has roots in a piece of classical Latin literature from 45
			BC, making it over 2000 years old. Richard McClintock, a Latin professor at
			 Hampden-Sydney College in Virginia, looked up one of the more obscure Latin words, c
			 onsectetur, from a Lorem Ipsum passage , at the end of line => AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsfl4
			  and going through the cites of the word in c
			 lassical literature, discovered the undoubtable source. Lorem Ipsum comes from secti
			 ons in singles =>'AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsfl2'1.10.32 and 1.10.33 of "de Finibus Bonorum et Malorum" (The Extremes of Good and Evil)
			 in doubles => "AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsfl3"
			  by Cicero, written in 45 BC. This book is a treatise on the theory of ethics, very popular
			  during the  :=> embedded in text RenaissanceAWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsfl5 embedded in text <=:
			  The first line of Lorem Ipsum, "Lorem ipsum dolor sit amet..", comes from a line in section 1.10.32.`,
			"://",
			[]string{
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsfl1",
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsfl2",
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsfl3",
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsfl4",
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsfl5",
			},
		},
		"unknown implementation not picked up": {
			`foo: AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj
				bar: AWSPARAMSTR://bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[version=123]
				unknown: GCPPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj
				unknown: GCPSECRETS#/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj`,
			"://",
			[]string{
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"AWSPARAMSTR://bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[version=123]"},
		},
		"all implementations": {
			`param: AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj
			secretsmgr: AWSSECRETS://bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[version=123]
			gcp: GCPSECRETS:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj
			vault: VAULT:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj
			som othere strufsd
			azkv: AZKVSECRET:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj`,
			"://",
			[]string{
				"GCPSECRETS:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"AWSSECRETS://bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[version=123]",
				"AZKVSECRET:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj",
				"VAULT:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj"},
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			config.VarPrefix = map[config.ImplementationPrefix]bool{"AWSPARAMSTR": true}
			c := configmanager.New(context.TODO())
			c.Config.WithTokenSeparator(tt.separator)
			got, err := c.DiscoverTokens(tt.input)
			if err != nil {
				t.Fatal(err)
			}

			if len(got) != len(tt.expect) {
				t.Errorf("wrong nmber of tokens resolved\ngot (%d) want (%d)", len(got), len(tt.expect))
			}
			for _, v := range got {
				if !slices.Contains(tt.expect, v.String()) {
					t.Errorf("got (%s) not found in expected slice (%v)", v, tt.expect)
				}
			}
		})
	}
}

func Test_YamlRetrieveMarshalled_errored_in_generator(t *testing.T) {
	m := &mockGenerator{}
	m.generate = func(tokens []string) (generator.ParsedMap, error) {
		return nil, fmt.Errorf("failed to generate a parsedMap")
	}
	c := configmanager.New(context.TODO())
	c.Config.WithTokenSeparator("://")
	c.WithGenerator(m)
	input := &testNestedStruct{}
	err := c.RetrieveMarshalledYaml(input)
	if err != nil {
	} else {
		t.Errorf(testutils.TestPhrase, nil, "err")
	}
}

func Test_YamlRetrieveMarshalled_errored_in_marshal(t *testing.T) {
	t.Skip()
	m := &mockGenerator{}
	m.generate = func(tokens []string) (generator.ParsedMap, error) {
		return generator.ParsedMap{}, nil
	}
	c := configmanager.New(context.TODO())
	c.Config.WithTokenSeparator("://")
	c.WithGenerator(m)
	err := c.RetrieveMarshalledYaml(&struct {
		A int
		B map[string]int `yaml:",inline"`
	}{1, map[string]int{"a": 2}})
	if err != nil {
	} else {
		t.Errorf(testutils.TestPhrase, nil, "err")
	}
}

func Test_JsonRetrieveMarshalled_errored_in_generator(t *testing.T) {
	m := &mockGenerator{}
	m.generate = func(tokens []string) (generator.ParsedMap, error) {
		return nil, fmt.Errorf("failed to generate a parsedMap")
	}
	c := configmanager.New(context.TODO())
	c.Config.WithTokenSeparator("://")
	c.WithGenerator(m)
	input := &testNestedStruct{}
	err := c.RetrieveMarshalledJson(input)
	if err != nil {
	} else {
		t.Errorf(testutils.TestPhrase, nil, "err")
	}
}

func Test_JsonRetrieveMarshalled_errored_in_marshal(t *testing.T) {
	m := &mockGenerator{}
	m.generate = func(tokens []string) (generator.ParsedMap, error) {
		return generator.ParsedMap{}, nil
	}
	c := configmanager.New(context.TODO())
	c.Config.WithTokenSeparator("://")
	c.WithGenerator(m)
	// input := &testNestedStruct{}
	err := c.RetrieveMarshalledJson(nil)
	if err != nil {
	} else {
		t.Errorf(testutils.TestPhrase, nil, "err")
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
