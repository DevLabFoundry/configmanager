package generator_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/config"
	"github.com/DevLabFoundry/configmanager/v3/generator"
	"github.com/DevLabFoundry/configmanager/v3/internal/testutils"
)

type mockStore struct {
	getVal func(implemenation *config.ParsedTokenConfig) (string, error)
	init   func(ctx context.Context, implt []string) error
}

func (m mockStore) GetValue(implemenation *config.ParsedTokenConfig) (string, error) {
	return m.getVal(implemenation)
}

func (m mockStore) Init(ctx context.Context, implt []string) error {
	if m.init != nil {
		return m.init(ctx, implt)
	}
	return nil
}

func (m mockStore) PluginCleanUp() {}

func Test_Generate(t *testing.T) {

	t.Run("succeeds", func(t *testing.T) {
		m := &mockStore{}
		m.getVal = func(implemenation *config.ParsedTokenConfig) (string, error) {
			return `{"foo":"bar","key1":{"key2":"val"}}`, nil
		}
		g := generator.New(context.TODO())
		g.WithStores(m)

		got, err := g.Generate([]string{"AWSPARAMSTR://mountPath/token"})

		if err != nil {
			t.Fatal("errored on generate")
		}
		if len(got) != 1 {
			t.Errorf(testutils.TestPhraseWithContext, "incorect number in a map", len(got), 1)
		}
	})

	t.Run("fails to init providers", func(t *testing.T) {
		m := &mockStore{}
		m.init = func(ctx context.Context, implt []string) error {
			return fmt.Errorf("failed to find providers")
		}

		g := generator.New(context.TODO())
		g.WithStores(m)

		_, err := g.Generate([]string{"AWSPARAMSTR://mountPath/token"})

		if err == nil {
			t.Fatal("got nil, wanted err")
		}
		if !errors.Is(err, generator.ErrProvidersNotFound) {
			t.Errorf("got %v, wanted %v", err, generator.ErrProvidersNotFound)
		}
	})
	t.Run("ignores no tokens", func(t *testing.T) {
		m := &mockStore{}
		g := generator.New(context.TODO())
		g.WithStores(m)

		_, err := g.Generate([]string{})

		if err != nil {
			t.Fatal("got nil, wanted err")
		}
	})
	t.Run("lax mode enabled - maintains v2 behaviour no rerrors in retrieval", func(t *testing.T) {

		m := &mockStore{}
		m.getVal = func(implemenation *config.ParsedTokenConfig) (string, error) {
			return ``, fmt.Errorf("failed to get value")
		}

		g := generator.New(context.TODO())
		g.Config().WithLaxMode(true)
		g.WithStores(m)

		got, err := g.Generate([]string{"AWSPARAMSTR://mountPath/token"})

		if err != nil {
			t.Fatal("errored on generate")
		}
		if len(got) != 0 {
			t.Errorf(testutils.TestPhraseWithContext, "incorect number in a map", len(got), 0)
		}
	})
	t.Run("errors in retrieval and logs it out", func(t *testing.T) {
		m := &mockStore{}
		m.getVal = func(implemenation *config.ParsedTokenConfig) (string, error) {
			return ``, fmt.Errorf("failed to get value")
		}
		g := generator.New(context.TODO())
		g.WithStores(m)

		_, err := g.Generate([]string{"AWSPARAMSTR://mountPath/token"})

		if err == nil {
			t.Fatal("got nil, wanted err")
		}
		if !errors.Is(err, generator.ErrTokenNotFound) {
			t.Errorf("got %v, wanted %v", err, generator.ErrTokenNotFound)
		}
	})
}

func TestGenerate_withKeys_lookup(t *testing.T) {

	ttests := map[string]struct {
		store     func(t *testing.T) *mockStore
		token     string
		expectVal string
	}{
		"retrieves string value correctly from a keylookup inside": {
			store: func(t *testing.T) *mockStore {
				m := &mockStore{}
				m.getVal = func(implemenation *config.ParsedTokenConfig) (string, error) {
					return `{"foo":"bar","key1":{"key2":"val"}}`, nil
				}
				return m
			},
			token:     "AWSPARAMSTR://mountPath/token|key1.key2",
			expectVal: "val",
		},
		"retrieves number value correctly from a keylookup inside": {

			store: func(t *testing.T) *mockStore {
				m := &mockStore{}
				m.getVal = func(implemenation *config.ParsedTokenConfig) (string, error) {
					return `{"foo":"bar","key1":{"key2":123}}`, nil
				}
				return m
			},
			token:     "AWSPARAMSTR://mountPath/token|key1.key2",
			expectVal: "123",
		},
		"retrieves nothing as keylookup is incorrect": {
			store: func(t *testing.T) *mockStore {
				m := &mockStore{}
				m.getVal = func(implemenation *config.ParsedTokenConfig) (string, error) {
					return `{"foo":"bar","key1":{"key2":123}}`, nil
				}
				return m
			},
			token:     "AWSPARAMSTR://mountPath/token|noprop",
			expectVal: "",
		},
		"retrieves value as is due to incorrectly stored json in backing store": {
			store: func(t *testing.T) *mockStore {
				m := &mockStore{}
				m.getVal = func(implemenation *config.ParsedTokenConfig) (string, error) {
					return `foo":"bar","key1":{"key2":123}}`, nil
				}
				return m
			},
			token:     "AWSPARAMSTR://mountPath/token|noprop",
			expectVal: `foo":"bar","key1":{"key2":123}}`,
		},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			g := generator.New(context.TODO())
			g.WithStores(tt.store(t))
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
			typ := generator.ReplacedToken{}
			got := generator.IsParsed(tt.val, typ)
			if got != tt.isParsed {
				t.Errorf(testutils.TestPhraseWithContext, "unexpected IsParsed", got, tt.isParsed)
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
				"AWSPARAMSTR:///path/config",
				// "AWSPARAMSTR:///path/config|password",
				// "AWSPARAMSTR:///path/config|foo.endpoint",
				// "AWSPARAMSTR:///path/config|foo.port",
				"AWSPARAMSTR:///path/queryparam|p1[version=123]"},
		},
		"# tokens in single string": {
			`Lorem_Ipsum: AWSPARAMSTR#/path/config|foo.user:AWSPARAMSTR#/path/config|password@AWSPARAMSTR#/path/config|foo.endpoint:AWSPARAMSTR#/path/config|foo.port/?someQ=AWSPARAMSTR#/path/queryparam|p1[version=123]&anotherQ=false`,
			"#",
			[]string{
				"AWSPARAMSTR#/path/config",
				// "AWSPARAMSTR#/path/config|password",
				// "AWSPARAMSTR#/path/config|foo.endpoint",
				// "AWSPARAMSTR#/path/config|foo.port",
				"AWSPARAMSTR#/path/queryparam|p1[version=123]"},
		},
		"without leading slash and path like name # tokens in single string": {
			`Lorem_Ipsum: AWSPARAMSTR#path_config|foo.user:AWSPARAMSTR#path_config|password@AWSPARAMSTR#path_config|foo.endpoint:AWSPARAMSTR#path_config|foo.port/?someQ=AWSPARAMSTR#path_queryparam|p1[version=123]&anotherQ=false`,
			"#",
			[]string{
				"AWSPARAMSTR#path_config",
				// "AWSPARAMSTR#path_config|password",
				// "AWSPARAMSTR#path_config|foo.endpoint",
				// "AWSPARAMSTR#path_config|foo.port",
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
			// config.VarPrefix = map[config.ImplementationPrefix]bool{"AWSPARAMSTR": true}
			g := generator.New(context.TODO())
			g.Config().WithTokenSeparator(tt.separator)
			gdt, err := g.DiscoverTokens(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			got := gdt.GetMap()

			if len(got) != len(tt.expect) {
				t.Errorf("wrong nmber of tokens resolved\ngot (%d) want (%d)", len(got), len(tt.expect))
			}
			for key := range got {
				if !slices.Contains(tt.expect, key) {
					t.Errorf("got (%s) not found in expected slice (%v)", key, tt.expect)
				}
			}
		})
	}
}

// func Test_Generate_EnsureRaceFree(t *testing.T) {
// 	g := generator.New(context.TODO())

// 	input := `
// fg
// dfg gdfgfdGCPSECRETS:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj
// GCPSECRETS:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj|a
// GCPSECRETS:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj|b
// GCPSECRETS:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj|c
// ddsffds			AWSPARAMSTR:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj
// 			'AWSSECRETS://bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj[version=123]'
// 			AWSSECRETS://bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj|key1
// 			AWSSECRETS://bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj|key2
// 			AZKVSECRET:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj gdf gdfgdf
//  dfg gdf gdf gdf
// 			fdg dgf dgf
// 			VAULT:///djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj . dfg dfgdf dfg fddf`

// 	g.WithStrategyMap(strategy.StrategyFuncMap{
// 		config.GcpSecretsPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
// 			m := &mockGenerate{"/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj", `{"a":"bar","b":{"key2":"val"},"c":123}`, nil}
// 			return m, nil
// 		},
// 		config.ParamStorePrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
// 			m := &mockGenerate{"/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj", `{"a":"bar","b":{"key2":"val"},"c":123}`, nil}
// 			return m, nil
// 		},
// 		config.SecretMgrPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
// 			m := &mockGenerate{"bar/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj", `{"key1":"bar","key2":"val","c":123}`, nil}
// 			return m, nil
// 		},
// 		config.AzKeyVaultSecretsPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
// 			m := &mockGenerate{"/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj", `{"key1":"bar","key2":"val","c":123}`, nil}
// 			return m, nil
// 		},
// 		config.HashicorpVaultPrefix: func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
// 			m := &mockGenerate{"/djsfsdkjvfjkhfdvibdfinjdsfnjvdsflj", `{"key1":"bar","key2":"val","c":123}`, nil}
// 			return m, nil
// 		},
// 	})

// 	got, err := g.Generate([]string{input})
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if len(got) != 10 {
// 		t.Errorf("got %v wanted %d", len(got), 10)
// 	}

// }
