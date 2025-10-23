package generator_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/DevLabFoundry/configmanager/v3/internal/store"
	"github.com/DevLabFoundry/configmanager/v3/internal/strategy"
	"github.com/DevLabFoundry/configmanager/v3/internal/testutils"
	"github.com/DevLabFoundry/configmanager/v3/generator"
)

type mockGenerate struct {
	inToken, value string
	err            error
}

func (m *mockGenerate) SetToken(s *config.ParsedTokenConfig) {
}
func (m *mockGenerate) Token() (s string, e error) {
	return m.value, m.err
}

func Test_Generate(t *testing.T) {

	t.Run("succeeds with funcMap", func(t *testing.T) {
		var custFunc = func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
			m := &mockGenerate{"UNKNOWN://mountPath/token", "bar", nil}
			return m, nil
		}

		g := generator.NewGenerator(context.TODO(), func(gv *generator.GenVars) {
			gv.Logger = log.New(&bytes.Buffer{})
		})
		g.WithStrategyMap(strategy.StrategyFuncMap{config.UnknownPrefix: custFunc})
		got, err := g.Generate([]string{"UNKNOWN://mountPath/token"})

		if err != nil {
			t.Fatal("errored on generate")
		}
		if len(got) != 1 {
			t.Errorf(testutils.TestPhraseWithContext, "incorect number in a map", len(got), 1)
		}
	})

	t.Run("errors in retrieval and logs it out", func(t *testing.T) {
		var custFunc = func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
			m := &mockGenerate{"UNKNOWN://mountPath/token", "bar", fmt.Errorf("failed to get value")}
			return m, nil
		}

		g := generator.NewGenerator(context.TODO())
		g.WithStrategyMap(strategy.StrategyFuncMap{config.UnknownPrefix: custFunc})
		got, err := g.Generate([]string{"UNKNOWN://mountPath/token"})

		if err != nil {
			t.Fatal("errored on generate")
		}
		if len(got) != 0 {
			t.Errorf(testutils.TestPhraseWithContext, "incorect number in a map", len(got), 0)
		}
	})

	t.Run("retrieves values correctly from a keylookup inside", func(t *testing.T) {
		var custFunc = func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
			m := &mockGenerate{"token-unused", `{"foo":"bar","key1":{"key2":"val"}}`, nil}
			return m, nil
		}

		g := generator.NewGenerator(context.TODO())
		g.WithStrategyMap(strategy.StrategyFuncMap{config.UnknownPrefix: custFunc})
		got, err := g.Generate([]string{"UNKNOWN://mountPath/token|key1.key2"})

		if err != nil {
			t.Fatal("errored on generate")
		}
		if len(got) != 1 {
			t.Errorf(testutils.TestPhraseWithContext, "incorect number in a map", len(got), 0)
		}
		if got["UNKNOWN://mountPath/token|key1.key2"] != "val" {
			t.Errorf(testutils.TestPhraseWithContext, "incorrect value returned in parsedMap", got["UNKNOWN://mountPath/token|key1.key2"], "val")
		}
	})
}

func Test_generate_withKeys_lookup(t *testing.T) {
	ttests := map[string]struct {
		custFunc  strategy.StrategyFunc
		token     string
		expectVal string
	}{
		"retrieves string value correctly from a keylookup inside": {
			custFunc: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
				m := &mockGenerate{"token", `{"foo":"bar","key1":{"key2":"val"}}`, nil}
				return m, nil
			},
			token:     "UNKNOWN://mountPath/token|key1.key2",
			expectVal: "val",
		},
		"retrieves number value correctly from a keylookup inside": {
			custFunc: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
				m := &mockGenerate{"token", `{"foo":"bar","key1":{"key2":123}}`, nil}
				return m, nil
			},
			token:     "UNKNOWN://mountPath/token|key1.key2",
			expectVal: "123",
		},
		"retrieves nothing as keylookup is incorrect": {
			custFunc: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
				m := &mockGenerate{"token", `{"foo":"bar","key1":{"key2":123}}`, nil}
				return m, nil
			},
			token:     "UNKNOWN://mountPath/token|noprop",
			expectVal: "",
		},
		"retrieves value as is due to incorrectly stored json in backing store": {
			custFunc: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
				m := &mockGenerate{"token", `foo":"bar","key1":{"key2":123}}`, nil}
				return m, nil
			},
			token:     "UNKNOWN://mountPath/token|noprop",
			expectVal: `foo":"bar","key1":{"key2":123}}`,
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			g := generator.NewGenerator(context.TODO())
			g.WithStrategyMap(strategy.StrategyFuncMap{config.UnknownPrefix: tt.custFunc})
			got, err := g.Generate([]string{tt.token})

			if err != nil {
				t.Fatal("errored on generate")
			}
			if len(got) != 1 {
				t.Errorf(testutils.TestPhraseWithContext, "incorect number in a map", len(got), 0)
			}
			if got[tt.token] != tt.expectVal {
				t.Errorf(testutils.TestPhraseWithContext, "incorrect value returned in parsedMap", got[tt.token], tt.expectVal)
			}
		})
	}
}

func Test_IsParsed(t *testing.T) {
	ttests := map[string]struct {
		val      any
		isParsed bool
	}{
		"not parseable": {
			`notparseable`, false,
		},
		"one level parseable": {
			`{"parseable":"foo"}`, true,
		},
		"incorrect JSON": {
			`parseable":"foo"}`, false,
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			typ := generator.ParsedMap{}
			got := generator.IsParsed(tt.val, typ)
			if got != tt.isParsed {
				t.Errorf(testutils.TestPhraseWithContext, "unexpected IsParsed", got, tt.isParsed)
			}
		})
	}
}

// import (
// 	"context"
// 	"fmt"
// 	"strings"
// 	"testing"

// 	"github.com/DevLabFoundry/configmanager/v3/internal/testutils"
// )

// var (
// 	customts   = "___"
// 	customop   = "/foo"
// 	standardop = "./app.env"
// 	standardts = "#"
// )

// type fixture struct {
// 	t  *testing.T
// 	c  *GenVars
// 	rs *retrieveStrategy
// }

// func newFixture(t *testing.T) *fixture {
// 	f := &fixture{}
// 	f.t = t
// 	return f
// }

// func (f *fixture) configGenVars(op, ts string) {
// 	conf := NewConfig().WithOutputPath(op).WithTokenSeparator(ts)
// 	gv := NewGenerator().WithConfig(conf)
// 	f.rs = newRetrieveStrategy(NewDefatultStrategy(), *conf)
// 	f.c = gv
// }

// func TestGenVarsWithConfig(t *testing.T) {

// 	f := newFixture(t)

// 	f.configGenVars(customop, customts)
// 	if f.c.config.outpath != customop {
// 		f.t.Errorf(testutils.TestPhrase, f.c.config.outpath, customop)
// 	}
// 	if f.c.config.tokenSeparator != customts {
// 		f.t.Errorf(testutils.TestPhrase, f.c.config.tokenSeparator, customts)
// 	}
// }

// func TestStripPrefixNormal(t *testing.T) {
// 	ttests := map[string]struct {
// 		prefix         ImplementationPrefix
// 		token          string
// 		keySeparator   string
// 		tokenSeparator string
// 		f              *fixture
// 		expect         string
// 	}{
// 		"standard azkv":               {AzKeyVaultSecretsPrefix, "AZKVSECRET://vault1/secret2", "|", "://", newFixture(t), "vault1/secret2"},
// 		"standard hashivault":         {HashicorpVaultPrefix, "VAULT://vault1/secret2", "|", "://", newFixture(t), "vault1/secret2"},
// 		"custom separator hashivault": {HashicorpVaultPrefix, "VAULT#vault1/secret2", "|", "#", newFixture(t), "vault1/secret2"},
// 	}
// 	for name, tt := range ttests {
// 		t.Run(name, func(t *testing.T) {
// 			tt.f.configGenVars(tt.keySeparator, tt.tokenSeparator)
// 			got := tt.f.rs.stripPrefix(tt.token, tt.prefix)
// 			if got != tt.expect {
// 				t.Errorf(testutils.TestPhrase, got, tt.expect)
// 			}
// 		})
// 	}
// }

// func Test_stripPrefix(t *testing.T) {
// 	f := newFixture(t)
// 	f.configGenVars(standardop, standardts)
// 	tests := []struct {
// 		name   string
// 		token  string
// 		prefix ImplementationPrefix
// 		expect string
// 	}{
// 		{
// 			name:   "simple",
// 			token:  fmt.Sprintf("%s#/test/123", SecretMgrPrefix),
// 			prefix: SecretMgrPrefix,
// 			expect: "/test/123",
// 		},
// 		{
// 			name:   "key appended",
// 			token:  fmt.Sprintf("%s#/test/123|key", ParamStorePrefix),
// 			prefix: ParamStorePrefix,
// 			expect: "/test/123",
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got := f.rs.stripPrefix(tt.token, tt.prefix)
// 			if tt.expect != got {
// 				t.Errorf(testutils.TestPhrase, tt.expect, got)
// 			}
// 		})
// 	}
// }

// func Test_NormaliseMap(t *testing.T) {
// 	f := newFixture(t)
// 	f.configGenVars(standardop, standardts)
// 	tests := []struct {
// 		name     string
// 		gv       *GenVars
// 		input    map[string]any
// 		expected string
// 	}{
// 		{
// 			name:     "foo->FOO",
// 			gv:       f.c,
// 			input:    map[string]any{"foo": "bar"},
// 			expected: "FOO",
// 		},
// 		{
// 			name:     "num->NUM",
// 			gv:       f.c,
// 			input:    map[string]any{"num": 123},
// 			expected: "NUM",
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got := f.c.envVarNormalize(tt.input)
// 			for k := range got {
// 				if k != tt.expected {
// 					t.Errorf(testutils.TestPhrase, tt.expected, k)
// 				}
// 			}
// 		})
// 	}
// }

// func Test_KeyLookup(t *testing.T) {
// 	f := newFixture(t)
// 	f.configGenVars(standardop, standardts)

// 	tests := []struct {
// 		name   string
// 		gv     *GenVars
// 		val    string
// 		key    string
// 		expect string
// 	}{
// 		{
// 			name:   "lowercase key found in str val",
// 			gv:     f.c,
// 			key:    `something|key`,
// 			val:    `{"key": "11235"}`,
// 			expect: "11235",
// 		},
// 		{
// 			name:   "lowercase key found in numeric val",
// 			gv:     f.c,
// 			key:    `something|key`,
// 			val:    `{"key": 11235}`,
// 			expect: "11235",
// 		},
// 		{
// 			name:   "lowercase nested key found in numeric val",
// 			gv:     f.c,
// 			key:    `something|key.test`,
// 			val:    `{"key":{"bar":"foo","test":12345}}`,
// 			expect: "12345",
// 		},
// 		{
// 			name:   "uppercase key found in val",
// 			gv:     f.c,
// 			key:    `something|KEY`,
// 			val:    `{"KEY": "upposeres"}`,
// 			expect: "upposeres",
// 		},
// 		{
// 			name:   "uppercase nested key found in val",
// 			gv:     f.c,
// 			key:    `something|KEY.TEST`,
// 			val:    `{"KEY":{"BAR":"FOO","TEST":"upposeres"}}`,
// 			expect: "upposeres",
// 		},
// 		{
// 			name:   "no key found in val",
// 			gv:     f.c,
// 			key:    `something`,
// 			val:    `{"key": "notfound"}`,
// 			expect: `{"key": "notfound"}`,
// 		},
// 		{
// 			name:   "nested key not found",
// 			gv:     f.c,
// 			key:    `something|KEY.KEY`,
// 			val:    `{"KEY":{"BAR":"FOO","TEST":"upposeres"}}`,
// 			expect: "",
// 		},
// 		{
// 			name:   "incorrect json",
// 			gv:     f.c,
// 			key:    "something|key",
// 			val:    `"KEY":{"BAR":"FOO","TEST":"upposeres"}}`,
// 			expect: `"KEY":{"BAR":"FOO","TEST":"upposeres"}}`,
// 		},
// 		{
// 			name:   "no key provided",
// 			gv:     f.c,
// 			key:    "something",
// 			val:    `{"KEY":{"BAR":"FOO","TEST":"upposeres"}}`,
// 			expect: `{"KEY":{"BAR":"FOO","TEST":"upposeres"}}`,
// 		},
// 		{
// 			name:   "return json object",
// 			gv:     f.c,
// 			key:    "something|key.test",
// 			val:    `{"key":{"bar":"foo","test": {"key": "default"}}}`,
// 			expect: `{"key": "default"}`,
// 		},
// 		{
// 			name:   "unescapable string",
// 			gv:     f.c,
// 			key:    "something|key.test",
// 			val:    `{"key":{"bar":"foo","test":"\\\"upposeres\\\""}}`,
// 			expect: `\"upposeres\"`,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got := f.c.keySeparatorLookup(tt.key, tt.val)
// 			if got != tt.expect {
// 				t.Errorf(testutils.TestPhrase, got, tt.expect)
// 			}
// 		})
// 	}
// }

// type mockRetrieve struct //func(ctx context.Context, impl genVarsStrategy, prefix ImplementationPrefix, in string) chanResp
// {
// 	r func(ctx context.Context, impl genVarsStrategy, prefix ImplementationPrefix, in string) ChanResp
// 	s func(ctx context.Context, prefix ImplementationPrefix, in string, config GenVarsConfig) (genVarsStrategy, error)
// }

// func (m mockRetrieve) RetrieveByToken(ctx context.Context, impl genVarsStrategy, prefix ImplementationPrefix, in string) ChanResp {
// 	return m.r(ctx, impl, prefix, in)
// }
// func (m mockRetrieve) SelectImplementation(ctx context.Context, prefix ImplementationPrefix, in string, config GenVarsConfig) (genVarsStrategy, error) {
// 	return m.s(ctx, prefix, in, config)
// }

// type mockImpl struct {
// 	token, value string
// 	err          error
// }

// func (m *mockImpl) tokenVal(rs *retrieveStrategy) (s string, e error) {
// 	return m.value, m.err
// }
// func (m *mockImpl) setTokenVal(s string) {
// 	m.token = s
// }

// func Test_generate_rawmap_of_tokens_mapped_to_values(t *testing.T) {
// 	ttests := map[string]struct {
// 		rawMap    func(t *testing.T) map[string]string
// 		rs        func(t *testing.T) retrieveIface
// 		expectMap func() map[string]string
// 	}{
// 		"success": {
// 			func(t *testing.T) map[string]string {
// 				rm := make(map[string]string)
// 				rm["foo"] = "bar"
// 				return rm
// 			},
// 			func(t *testing.T) retrieveIface {
// 				return mockRetrieve{
// 					r: func(ctx context.Context, impl genVarsStrategy, prefix ImplementationPrefix, in string) ChanResp {
// 						return ChanResp{
// 							err:   nil,
// 							value: "bar",
// 						}
// 					},
// 					s: func(ctx context.Context, prefix ImplementationPrefix, in string, config GenVarsConfig) (genVarsStrategy, error) {
// 						return &mockImpl{"foo", "bar", nil}, nil
// 					}}
// 			},
// 			func() map[string]string {
// 				rm := make(map[string]string)
// 				rm["foo"] = "bar"
// 				return rm
// 			},
// 		},
// 		// as the method swallows errors at the moment this is not very useful
// 		"error in implementation": {
// 			func(t *testing.T) map[string]string {
// 				rm := make(map[string]string)
// 				rm["foo"] = "bar"
// 				return rm
// 			},
// 			func(t *testing.T) retrieveIface {
// 				return mockRetrieve{
// 					r: func(ctx context.Context, impl genVarsStrategy, prefix ImplementationPrefix, in string) ChanResp {
// 						return ChanResp{
// 							err: fmt.Errorf("unable to retrieve"),
// 						}
// 					},
// 					s: func(ctx context.Context, prefix ImplementationPrefix, in string, config GenVarsConfig) (genVarsStrategy, error) {
// 						return &mockImpl{"foo", "bar", nil}, nil
// 					}}
// 			},
// 			func() map[string]string {
// 				rm := make(map[string]string)
// 				return rm
// 			},
// 		},
// 		"error in imp selection": {
// 			func(t *testing.T) map[string]string {
// 				rm := make(map[string]string)
// 				rm["foo"] = "bar"
// 				return rm
// 			},
// 			func(t *testing.T) retrieveIface {
// 				return mockRetrieve{
// 					r: func(ctx context.Context, impl genVarsStrategy, prefix ImplementationPrefix, in string) ChanResp {
// 						return ChanResp{
// 							err: fmt.Errorf("unable to retrieve"),
// 						}
// 					},
// 					s: func(ctx context.Context, prefix ImplementationPrefix, in string, config GenVarsConfig) (genVarsStrategy, error) {
// 						return nil, fmt.Errorf("implementation not found for input string: %s", in)
// 					}}
// 			},
// 			func() map[string]string {
// 				rm := make(map[string]string)
// 				return rm
// 			},
// 		},
// 	}
// 	for name, tt := range ttests {
// 		t.Run(name, func(t *testing.T) {
// 			generator := newGenVars()
// 			generator.generate(tt.rawMap(t), tt.rs(t))
// 			got := generator.RawMap()
// 			if len(got) != len(tt.expectMap()) {
// 				t.Errorf(testutils.TestPhraseWithContext, "generated raw map did not match", len(got), len(tt.expectMap()))
// 			}
// 		})
// 	}
// }

// func TestGenerate(t *testing.T) {
// 	ttests := map[string]struct {
// 		tokens       func(t *testing.T) []string
// 		expectLength int
// 	}{
// 		"success without correct prefix": {
// 			func(t *testing.T) []string {
// 				return []string{"WRONGIMPL://bar-vault/token1", "AZKVNOTSECRET://bar-vault/token1"}
// 			},
// 			0,
// 		},
// 	}
// 	for name, tt := range ttests {
// 		t.Run(name, func(t *testing.T) {
// 			generator := newGenVars()
// 			pm, err := generator.Generate(tt.tokens(t))
// 			if err != nil {
// 				t.Errorf(testutils.TestPhrase, err.Error(), nil)
// 			}
// 			if len(pm) < tt.expectLength {
// 				t.Errorf(testutils.TestPhrase, len(pm), tt.expectLength)
// 			}
// 		})
// 	}
// }
