package parser_test

import (
	"errors"
	"os"
	"testing"

	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/internal/lexer"
	"github.com/DevLabFoundry/configmanager/v2/internal/log"
	"github.com/DevLabFoundry/configmanager/v2/internal/parser"
)

var lexerSource = lexer.Source{FileName: "bar", FullPath: "/foo/bar"}

func Test_ParserBlocks(t *testing.T) {
	ttests := map[string]struct {
		input    string
		expected [][]string
	}{
		"tokens touching each other in source": {`foo stuyfsdfsf
foo=AWSPARAMSTR:///path|keyAWSSECRETS:///foo
other text her
BAR=something
		`, [][]string{
			{string(config.ParamStorePrefix), "/path|key"},
			{string(config.SecretMgrPrefix), "/foo"},
		}},
		"full URL of tokens": {`foo stuyfsdfsf
foo=proto://AWSPARAMSTR:///config|user:AWSSECRETS:///creds|password@AWSPARAMSTR:///config|endpoint:AWSPARAMSTR:///config|port/?queryParam1=123&queryParam2=AWSPARAMSTR:///config|qp2
# some comment
BAR=something
`, [][]string{
			{string(config.ParamStorePrefix), "/config|user"},
			{string(config.SecretMgrPrefix), "/creds|password"},
			{string(config.ParamStorePrefix), "/config|endpoint"},
			{string(config.ParamStorePrefix), "/config|port"},
			{string(config.ParamStorePrefix), "/config|qp2"},
		}},
	}

	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			lexerSource.Input = tt.input
			l := lexer.New(lexerSource, *config.NewConfig())
			p := parser.New(l, config.NewConfig()).WithLogger(log.New(os.Stderr))
			parsed, errs := p.Parse()
			if len(errs) > 0 {
				t.Fatalf("parser had errors, expected <nil>\nerror: %v", errs)
			}

			if len(parsed) != len(tt.expected) {
				t.Fatalf("parsed statements count does not match\ngot=%d want=%d\nparsed %q",
					len(parsed),
					len(tt.expected),
					parsed)
			}

			for idx, stmt := range parsed {
				if !testHelperGenDocBlock(t, stmt, config.ImplementationPrefix(tt.expected[idx][0]), tt.expected[idx][1]) {
					return
				}
			}
		})
	}
}

func Test_ShouldError_when_no_End_tag_found(t *testing.T) {
	input := `let x = 5;
	//+gendoc category=message type=nameId parent=id1 id=id
	`

	lexerSource.Input = input
	l := lexer.New(lexerSource, *config.NewConfig())
	p := parser.New(l, &config.GenVarsConfig{}).WithLogger(log.New(os.Stderr))
	_, errs := p.Parse()
	if len(errs) != 1 {
		t.Errorf("unexpected number of errors\n got: %v, wanted: 1", errs)
	}
	if !errors.Is(errs[0], parser.ErrNoEndTagFound) {
		t.Errorf("unexpected error type\n got: %T, wanted: %T", errs, parser.ErrNoEndTagFound)
	}
}

func testHelperGenDocBlock(t *testing.T, stmtBlock parser.ConfigManagerTokenBlock, tokenType config.ImplementationPrefix, tokenValue string) bool {
	t.Helper()
	if stmtBlock.BeginToken.ImpPrefix != tokenType {
		t.Errorf("got=%q, wanted stmtBlock.ImpPrefix = '%v'.", stmtBlock.BeginToken.Literal, tokenType)
		return false
	}

	if stmtBlock.ParsedToken.StripPrefix() != tokenValue {
		t.Errorf("stmtBlock.Value. got=%s, wanted=%s", stmtBlock.ParsedToken.StripPrefix(), tokenValue)
		return false
	}

	return true
}

func Test_ExpandEnvVariables_succeeds(t *testing.T) {
	ttests := map[string]struct {
		input  string
		expect string
		envVar []string
	}{
		"with single var": {
			"some var is $var",
			"some var is foo",
			[]string{"var=foo"},
		},
		"with multiple var": {
			"some var is $var and docs go [here]($DOC_LINK/stuff)",
			"some var is foo and docs go [here](https://somestuff.com/stuff)",
			[]string{"var=foo", "DOC_LINK=https://somestuff.com"},
		},
		"with no vars in content": {
			"some var is foo and docs go [here](foo.com/stuff)",
			"some var is foo and docs go [here](foo.com/stuff)",
			[]string{"var=foo", "DOC_LINK=https://somestuff.com"},
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			defer os.Clearenv()
			got, err := parser.ExpandEnvVariables(tt.input, tt.envVar)
			if err != nil {
				t.Errorf("expected %v to be <nil>", err)
			}
			if got != tt.expect {
				t.Errorf("want: %s, got: %s", got, tt.expect)
			}
		})
	}
}

func Test_ExpandEnvVariables_fails(t *testing.T) {

	ttests := map[string]struct {
		input  string
		setup  func() func()
		envVar []string
	}{
		"with single var": {
			"some var is $var",
			func() func() {
				return func() {
					os.Clearenv()
				}
			},
			[]string{"v=foo"},
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			clear := tt.setup()
			defer clear()
			_, err := parser.ExpandEnvVariables(tt.input, tt.envVar)
			if err == nil {
				t.Errorf("wanted error, got <nil>")
			}
		})
	}
}

func Test_Parse_WithOwnEnviron_passed_in_succeeds(t *testing.T) {
	ttests := map[string]struct {
		input   string
		expect  string
		environ []string
	}{
		"test1": {
			input: `let x = 42;
//+gendoc category=message type=description id=foo
this is some description with $foo
//-gendoc`,
			environ: []string{"foo=bar"},
			expect:  "this is some description with bar",
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			defer os.Clearenv()
			lexerSource.Input = tt.input
			l := lexer.New(lexerSource, *config.NewConfig())
			p := parser.New(l, &config.GenVarsConfig{}).WithLogger(log.New(os.Stderr)).WithEnvironment(tt.environ)
			got, errs := p.Parse()
			if len(errs) > 0 {
				t.Error(errs)
			}
			if got[0].Value != tt.expect {
				t.Error("")
			}
		})
	}
}

func Test_Parse_WithOwnEnviron_passed_in_fails(t *testing.T) {
	ttests := map[string]struct {
		input   string
		expect  error
		environ []string
	}{
		"if variable is not set": {
			input: `let x = 42;
		//+gendoc category=message type=description id=foo
		this is some description with $foo
		//-gendoc`,
			expect:  parser.ErrUnableToReplaceVarPlaceholder,
			environ: []string{"notfoo=bar"},
		},
		"if variable is not set but empty": {
			input: `let x = 42;
//+gendoc category=message type=description id=foo
this is some description with $foo
//-gendoc`,
			expect:  parser.ErrUnableToReplaceVarPlaceholder,
			environ: []string{"foo="},
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			defer os.Clearenv()
			lexerSource.Input = tt.input
			l := lexer.New(lexerSource, *config.NewConfig())
			p := parser.New(l, &config.GenVarsConfig{}).WithLogger(log.New(os.Stderr)).WithEnvironment(tt.environ)
			_, errs := p.Parse()

			if len(errs) < 1 {
				t.Error("expected errors to occur")
				t.Fail()
			}
			if !errors.Is(errs[0], tt.expect) {
				t.Errorf("unexpected error type\n got: %T, wanted: %T", errs, tt.expect)
			}
		})
	}
}
