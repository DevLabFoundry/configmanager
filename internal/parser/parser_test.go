package parser_test

import (
	"errors"
	"os"
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/lexer"
	"github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/DevLabFoundry/configmanager/v3/internal/parser"
	"github.com/DevLabFoundry/configmanager/v3/internal/store"
)

var lexerSource = lexer.Source{FileName: "bar", FullPath: "/foo/bar"}

func Test_ParserBlocks(t *testing.T) {
	ttests := map[string]struct {
		input string
		// prefix,path,keyLookup
		expected [][3]string
	}{
		"tokens touching each other in source": {
			`foo stuyfsdfsf
		foo=AWSPARAMSTR:///path|keyAWSSECRETS:///foo
		other text her
		BAR=something
				`, [][3]string{
				{string(config.ParamStorePrefix), "/path", "key"},
				{string(config.SecretMgrPrefix), "/foo", ""},
			}},
		"full URL of tokens": {
			`foo stuyfsdfsf
		foo=proto://AWSPARAMSTR:///config|user:AWSSECRETS:///creds|password@AWSPARAMSTR:///config|endpoint:AWSPARAMSTR:///config|port/?queryParam1=123&queryParam2=AWSPARAMSTR:///config|qp2
		# some comment
		BAR=something
				`, [][3]string{
				{string(config.ParamStorePrefix), "/config", "user"},
				{string(config.SecretMgrPrefix), "/creds", "password"},
				{string(config.ParamStorePrefix), "/config", "endpoint"},
				{string(config.ParamStorePrefix), "/config", "port"},
				{string(config.ParamStorePrefix), "/config", "qp2"},
			},
		},
		"touching EOF single token": {
			`AWSPARAMSTR:///config|qp2`,
			[][3]string{
				{string(config.ParamStorePrefix), "/config", "qp2"},
			},
		},
		"touching EOF multi token": {
			`proto://AWSPARAMSTR:///config|user:AWSSECRETS:///creds|password@AWSPARAMSTR:///config|endpoint:AWSPARAMSTR:///config|port/?queryParam1=123&queryParam2=AWSPARAMSTR:///config|qp2`,
			[][3]string{
				{string(config.ParamStorePrefix), "/config", "user"},
				{string(config.SecretMgrPrefix), "/creds", "password"},
				{string(config.ParamStorePrefix), "/config", "endpoint"},
				{string(config.ParamStorePrefix), "/config", "port"},
				{string(config.ParamStorePrefix), "/config", "qp2"},
			},
		},
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
				if !testHelperGenDocBlock(t, stmt, config.ImplementationPrefix(tt.expected[idx][0]), tt.expected[idx][1], tt.expected[idx][2]) {
					return
				}
			}
		})
	}
}

func Test_Parse_should_fail_on_metadata(t *testing.T) {
	ttests := map[string]struct {
		input  string
		errTyp error
	}{
		"when _end_tag_found without keysPath": {
			`AWSSECRETS:///foo[version=1.2.3`,
			parser.ErrNoEndTagFound,
		},
		"when _end_tag_found with keysPath": {
			`AWSSECRETS:///foo|path.one[version=1.2.3`,
			parser.ErrNoEndTagFound,
		},
		"when no metadata has been supplied": {
			`AWSSECRETS:///foo|path.one[]`,
			parser.ErrMetadataEmpty,
		},
		"when no metadata has been supplied - without key path": {
			`AWSSECRETS:///foo[]`,
			parser.ErrMetadataEmpty,
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			lexerSource.Input = tt.input
			cfg := config.NewConfig()
			l := lexer.New(lexerSource, *cfg)
			p := parser.New(l, cfg).WithLogger(log.New(os.Stderr))
			_, errs := p.Parse()
			if len(errs) != 1 {
				t.Fatalf("unexpected number of errors\n got: %v, wanted: 1", errs)
			}
			if !errors.Is(errs[0], tt.errTyp) {
				t.Errorf("unexpected error type\n got: %T, wanted: %T", errs, parser.ErrNoEndTagFound)
			}
		})
	}
}

func Test_Parse_should_pass_with_metadata_end_tag(t *testing.T) {
	ttests := map[string]struct {
		input      string
		metdataStr string
	}{
		"without keysPath": {
			`AWSSECRETS:///foo[version=1.2.3]`,
			`version=1.2.3`,
		},
		"with keysPath": {
			`AWSSECRETS:///foo|path.one[version=1.2.3]`,
			`version=1.2.3`,
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			lexerSource.Input = tt.input
			cfg := config.NewConfig()
			l := lexer.New(lexerSource, *cfg)
			p := parser.New(l, cfg).WithLogger(log.New(os.Stderr))
			parsed, errs := p.Parse()
			if len(errs) > 0 {
				t.Fatalf("unexpected number of errors\n got: %v, wanted: 0", errs)
			}
			for _, prsd := range parsed {
				prsd.ParsedToken.LookupKeys()

			}
		})
	}
}

func Test_Parse_ParseMetadata(t *testing.T) {

	ttests := map[string]struct {
		input string
		typ   *store.SecretsMgrConfig
	}{
		"without keysPath": {
			`AWSSECRETS:///foo[version=1.2.3]`,
			&store.SecretsMgrConfig{},
		},
		"with keysPath": {
			`AWSSECRETS:///foo|path.one[version=1.2.3]`,
			&store.SecretsMgrConfig{},
		},
		"nestled in text": {
			`someQ=AWSPARAMSTR:///path/queryparam|p1[version=1.2.3]&anotherQ`,
			&store.SecretsMgrConfig{},
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			lexerSource.Input = tt.input
			cfg := config.NewConfig()
			l := lexer.New(lexerSource, *cfg)
			p := parser.New(l, cfg).WithLogger(log.New(os.Stderr))
			parsed, errs := p.Parse()
			if len(errs) > 0 {
				t.Fatalf("%v", errs)
			}

			for _, p := range parsed {
				if err := p.ParsedToken.ParseMetadata(tt.typ); err != nil {
					t.Fatal(err)
				}
				if tt.typ.Version != "1.2.3" {
					t.Errorf("got %v wanted 1.2.3", tt.typ.Version)
				}
			}
		})
	}
}

func Test_Parse_Path_Keys_WithParsedMetadat(t *testing.T) {

	ttests := map[string]struct {
		input             string
		typ               *store.SecretsMgrConfig
		wantSanitizedPath string
		wantKeyPath       string
	}{
		"without keysPath": {
			`AWSSECRETS:///foo[version=1.2.3]`,
			&store.SecretsMgrConfig{},
			"/foo", "",
		},
		"with keysPath": {
			`AWSSECRETS:///foo|path.one[version=1.2.3]`,
			&store.SecretsMgrConfig{},
			"/foo", "path.one",
		},
		"nestled in text": {
			`someQ=AWSPARAMSTR:///path/queryparam|p1[version=1.2.3]&anotherQ`,
			&store.SecretsMgrConfig{},
			"/path/queryparam", "p1",
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			lexerSource.Input = tt.input
			cfg := config.NewConfig()
			l := lexer.New(lexerSource, *cfg)
			p := parser.New(l, cfg).WithLogger(log.New(os.Stderr))
			parsed, errs := p.Parse()
			if len(errs) > 0 {
				t.Fatalf("%v", errs)
			}

			for _, p := range parsed {
				if p.ParsedToken.StoreToken() != tt.wantSanitizedPath {
					t.Errorf("got %s want %s", p.ParsedToken.StoreToken(), tt.wantSanitizedPath)
				}
				if p.ParsedToken.LookupKeys() != tt.wantKeyPath {
					t.Errorf("got %s want %s", p.ParsedToken.LookupKeys(), tt.wantKeyPath)
				}
				if err := p.ParsedToken.ParseMetadata(tt.typ); err != nil {
					t.Fatal(err)
				}
				if tt.typ.Version != "1.2.3" {
					t.Errorf("got %v wanted 1.2.3", tt.typ.Version)
				}
			}
		})
	}
}

func testHelperGenDocBlock(t *testing.T, stmtBlock parser.ConfigManagerTokenBlock, tokenType config.ImplementationPrefix, tokenValue, keysLookupPath string) bool {
	t.Helper()
	if stmtBlock.ParsedToken.Prefix() != tokenType {
		t.Errorf("got=%q, wanted stmtBlock.ImpPrefix = '%v'.", stmtBlock.ParsedToken.Prefix(), tokenType)
		return false
	}

	if stmtBlock.ParsedToken.StoreToken() != tokenValue {
		t.Errorf("token StoreToken got=%s, wanted=%s", stmtBlock.ParsedToken.StoreToken(), tokenValue)
		return false
	}

	if stmtBlock.ParsedToken.LookupKeys() != keysLookupPath {
		t.Errorf("token LookupKeys. got=%s, wanted=%s", stmtBlock.ParsedToken.LookupKeys(), keysLookupPath)
		return false
	}

	return true
}
