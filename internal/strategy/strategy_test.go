package strategy_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	log "github.com/DevLabFoundry/configmanager/v2/internal/log"
	"github.com/DevLabFoundry/configmanager/v2/internal/store"
	"github.com/DevLabFoundry/configmanager/v2/internal/strategy"
	"github.com/DevLabFoundry/configmanager/v2/internal/testutils"
	"github.com/go-test/deep"
)

type mockGenerate struct {
	inToken, value string
	err            error
}

func (m mockGenerate) SetToken(s *config.ParsedTokenConfig) {
}

func (m mockGenerate) Token() (s string, e error) {
	return m.value, m.err
}

var TEST_GCP_CREDS = []byte(`{
	"type": "service_account",
	"project_id": "xxxxx",
	"private_key_id": "yyyyyyyyyyyy",
	"private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDf842hcn5Nvp6e\n7yKARaCVIDfLXpKDhRwUOvHMzJ1ioRgQo/kbv1n4yHGCSUFyY6hKGj0HBjaGj5kE\n79H/6Y3dJNGhnsMnxBhHdo+3FI8QF0CHZh460NMZSAJ41UMQSBGssGVsNfyUzXGH\nLc45sIx/Twx3yr1k2GD3E8FlDcKlZqa3xGHf+aipg2X3NxbYi+Sz7Yed+SOMhNHl\ncX6E/TqG9n1aTyIwjMIHscCYarJqURkJxr24ukDroCeMxAfxYTdMvRU2e8pFEdoY\nrgUC88fYfaVI5txJ6j/ZKauKQX9Pa8tSyXJeGva3JYp4VC7V4IyoVviCUgEGWZDN\n6/i3zoF/AgMBAAECggEAcVBCcVYFIkE48SH+Svjv74SFtpj7eSB4vKO2hPFjEOyB\nyKmu+aMwWvjQtiNqwf46wIPWLR+vpxYxTpYpo1sBNMvUZfp2tEA8KKyMuw3j9ThO\npjO9R/UxWrFcztbZP/u3NbFrH/2Q95mbv9IlbnsuG5xbqqEig0wYg+uzBvaXbig3\n/Jr0vLT2BkRCBKQkYGjVZcHlHVLoF7/J8cghFgkV1PGvknOv6/q7qzn9L4TjQIet\nfhrhN8Z1vgFiSYtpjP6YQEUEPSHmCQeD3WzJcnASPpU2uCUwd/z65ltKPnn+rqMt\n6jt9R1S1Ju2ZSjv+kR5fIXzihdOzncyzDDm33c/QwQKBgQD2QDZuzLjTxnhsfGii\nKJDAts+Jqfs/6SeEJcJKtEngj4m7rgzyEjbKVp8qtRHIzglKRWAe62/qzzy2BkKi\nvAd4+ZzmG2SkgypGsKVfjGXVFixz2gtUdmBOmK/TnYsxNT9yTt+rX9IGqKK60q73\nOWl8VsliLIsfvSH7+bqi7sRcXQKBgQDo0VUebyQHoTAXPdzGy2ysrVPDiHcldH0Y\n/hvhQTZwxYaJr3HpOCGol2Xl6zyawuudEQsoQwJ3Li6yeb0YMGiWX77/t+qX3pSn\nkGuoftGaNDV7sLn9UV2y+InF8EL1CasrhG1k5RIuxyfV0w+QUo+E7LpVR5XkbJqT\n9QNKnDQXiwKBgQDvvEYCCqbp7e/xVhEbxbhfFdro4Cat6tRAz+3egrTlvXhO0jzi\nMp9Kz5f3oP5ma0gaGX5hu75icE1fvKqE+d+ghAqe7w5FJzkyRulJI0tEb2jphN7A\n5NoPypBqyZboWjmhlG4mzouPVf/POCuEnk028truDAWJ6by7Lj3oP+HFNQKBgQCc\n5BQ8QiFBkvnZb7LLtGIzq0n7RockEnAK25LmJRAOxs13E2fsBguIlR3x5qgckqY8\nXjPqmd2bet+1HhyzpEuWqkcIBGRum2wJz2T9UxjklbJE/D8Z2i8OYDZX0SUOA8n5\ntXASwduS8lqB2Y1vcHOO3AhlV6xHFnjEpCPnr4PbKQKBgAhQ9D9MPeuz+5yw3yHg\nkvULZRtud+uuaKrOayprN25RTxr9c0erxqnvM7KHeo6/urOXeEa7x2n21kAT0Nch\nkF2RtWBLZKXGZEVBtw1Fw0UKNh4IDgM26dwlzRfTVHCiw6M6dCiTNk9KkP2vlkim\n3QFDSSUp+eBTXA17WkDAQf7w\n-----END PRIVATE KEY-----\n",
	"client_email": "foo@project.iam.gserviceaccount.com",
	"client_id": "99999911111111",
	"auth_uri": "https://accounts.google.com/o/oauth2/auth",
	"token_uri": "https://oauth2.googleapis.com/token",
	"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
	"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/bla"
  }`)

func Test_Strategy_Retrieve_succeeds(t *testing.T) {
	t.Parallel()
	ttests := map[string]struct {
		impl   func(t *testing.T) store.Strategy
		config *config.GenVarsConfig
		token  string
		expect string
	}{
		"with mocked implementation AZTABLESTORAGE": {
			func(t *testing.T) store.Strategy {
				return &mockGenerate{"AZTABLESTORE://mountPath/token", "bar", nil}
			},
			config.NewConfig().WithOutputPath("stdout").WithTokenSeparator("://"),
			"AZTABLESTORE://mountPath/token",
			"bar",
		},
		// "error in retrieval": {
		// 	func(t *testing.T) store.Strategy {
		// 		return &mockGenerate{"SOME://mountPath/token", "bar", fmt.Errorf("unable to perform getTokenValue")}
		// 	},
		// 	config.NewConfig().WithOutputPath("stdout").WithTokenSeparator("://"),
		// 	[]string{"SOME://token"},
		// 	config.AzAppConfigPrefix,
		// 	"unable to perform getTokenValue",
		// },
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			rs := strategy.New(store.NewDefatultStrategy(), *tt.config, log.New(io.Discard))
			token, _ := config.NewParsedTokenConfig(tt.token, *tt.config)
			got := rs.RetrieveByToken(context.TODO(), tt.impl(t), token)
			if got.Err != nil {
				t.Errorf(testutils.TestPhraseWithContext, "Token response errored", got.Err.Error(), tt.expect)
			}
			if got.Value() != tt.expect {
				t.Errorf(testutils.TestPhraseWithContext, "Value not correct", got.Value(), tt.expect)
			}
			if got.Key().String() != tt.token {
				t.Errorf(testutils.TestPhraseWithContext, "INcorrect Token returned in Key", got.Key().String(), tt.token)
			}
		})
	}
}

func Test_CustomStrategyFuncMap_add_own(t *testing.T) {
	t.Parallel()

	ttests := map[string]struct {
	}{
		"default": {},
	}
	for name, _ := range ttests {
		t.Run(name, func(t *testing.T) {
			called := 0
			genVarsConf := config.NewConfig()
			token, _ := config.NewParsedTokenConfig("AZTABLESTORE://mountPath/token", *genVarsConf)

			var custFunc = func(ctx context.Context, token *config.ParsedTokenConfig) (store.Strategy, error) {
				m := &mockGenerate{"AZTABLESTORE://mountPath/token", "bar", nil}
				called++
				return m, nil
			}

			s := strategy.New(store.NewDefatultStrategy(), *genVarsConf, log.New(io.Discard))
			s.WithStrategyFuncMap(strategy.StrategyFuncMap{config.AzTableStorePrefix: custFunc})

			store, _ := s.SelectImplementation(context.TODO(), token)
			_ = s.RetrieveByToken(context.TODO(), store, token)

			if called != 1 {
				t.Errorf(testutils.TestPhraseWithContext, "custom func not called", called, 1)
			}
		})
	}
}

func Test_SelectImpl_With(t *testing.T) {

	ttests := map[string]struct {
		setUpTearDown func() func()
		token         string
		config        *config.GenVarsConfig
		expect        func() store.Strategy
		expErr        error
	}{
		// "unknown": {
		// 	func() func() {
		// 		return func() {
		// 		}
		// 	},
		// 	"UNKNOWN#foo/bar",
		// 	config.NewConfig().WithTokenSeparator("#"),
		// 	func() store.Strategy { return nil },
		// 	fmt.Errorf("implementation not found for input string: UNKNOWN#foo/bar"),
		// },
		// "success AZTABLESTORE": {
		// 	func() func() {
		// 		os.Setenv("AZURE_stuff", "foo")
		// 		return func() {
		// 			os.Clearenv()
		// 		}
		// 	},
		// 	"AZTABLESTORE#foo/bar1",
		// 	config.NewConfig().WithTokenSeparator("#"),
		// 	func() store.Strategy {
		// 		token, _ := config.NewParsedTokenConfig("AZTABLESTORE#foo/bar1", *config.NewConfig().WithTokenSeparator("#"))
		// 		s, _ := store.NewAzTableStore(context.TODO(), token, log.New(io.Discard))
		// 		return s
		// 	},
		// 	nil,
		// },
		// "success AWSPARAMSTR": {
		// 	func() func() {
		// 		os.Setenv("AWS_ACCESS_KEY", "AAAAAAAAAAAAAAA")
		// 		os.Setenv("AWS_SECRET_ACCESS_KEY", "00000000000000000000111111111")
		// 		return func() {
		// 			os.Clearenv()
		// 		}
		// 	},
		// 	"AWSPARAMSTR#foo/bar1",
		// 	config.NewConfig().WithTokenSeparator("#"),
		// 	func() store.Strategy {
		// 		s, _ := store.NewParamStore(context.TODO(), log.New(io.Discard))
		// 		return s
		// 	},
		// 	nil,
		// },
		"success AWSSECRETS": {
			func() func() {
				os.Setenv("AWS_ACCESS_KEY", "AAAAAAAAAAAAAAA")
				os.Setenv("AWS_SECRET_ACCESS_KEY", "00000000000000000000111111111")
				return func() {
					os.Clearenv()
				}
			},
			"AWSSECRETS#foo/bar1",
			config.NewConfig().WithTokenSeparator("#"),
			func() store.Strategy {
				s, _ := store.NewSecretsMgr(context.TODO(), log.New(io.Discard))
				return s
			},
			nil,
		},
		"success AZKVSECRET": {
			func() func() {
				os.Setenv("AWS_ACCESS_KEY", "AAAAAAAAAAAAAAA")
				os.Setenv("AWS_SECRET_ACCESS_KEY", "00000000000000000000111111111")
				return func() {
					os.Clearenv()
				}
			},
			"AZKVSECRET#foo/bar1",
			config.NewConfig().WithTokenSeparator("#"),
			func() store.Strategy {
				token, _ := config.NewParsedTokenConfig("AZKVSECRET#foo/bar1", *config.NewConfig().WithTokenSeparator("#"))
				s, _ := store.NewKvScrtStore(context.TODO(), token, log.New(io.Discard))
				return s
			},
			nil,
		},
		"success AZAPPCONF": {
			func() func() {
				return func() {
					os.Clearenv()
				}
			},
			"AZAPPCONF#foo/bar1",
			config.NewConfig().WithTokenSeparator("#"),
			func() store.Strategy {
				token, _ := config.NewParsedTokenConfig("AZAPPCONF#foo/bar1", *config.NewConfig().WithTokenSeparator("#"))
				s, _ := store.NewAzAppConf(context.TODO(), token, log.New(io.Discard))
				return s
			},
			nil,
		},
		"success VAULT": {
			func() func() {
				os.Setenv("VAULT_", "AAAAAAAAAAAAAAA")
				return func() {
					os.Clearenv()
				}
			},
			"VAULT#foo/bar1",
			config.NewConfig().WithTokenSeparator("#"),
			func() store.Strategy {
				token, _ := config.NewParsedTokenConfig("VAULT#foo/bar1", *config.NewConfig().WithTokenSeparator("#"))
				s, _ := store.NewVaultStore(context.TODO(), token, log.New(io.Discard))
				return s
			},
			nil,
		},
		"success GCPSECRETS": {
			func() func() {
				cf, _ := os.CreateTemp(".", "*")
				cf.Write(TEST_GCP_CREDS)
				os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", cf.Name())
				return func() {
					os.Remove(cf.Name())
					os.Clearenv()
				}
			},
			"GCPSECRETS#foo/bar1",
			config.NewConfig().WithTokenSeparator("#"),
			func() store.Strategy {
				s, _ := store.NewGcpSecrets(context.TODO(), log.New(io.Discard))
				return s
			},
			nil,
		},
		// "default Error": {
		// 	func() func() {
		// 		os.Setenv("AWS_ACCESS_KEY", "AAAAAAAAAAAAAAA")
		// 		os.Setenv("AWS_SECRET_ACCESS_KEY", "00000000000000000000111111111")
		// 		return func() {
		// 			os.Clearenv()
		// 		}
		// 	},
		// 	context.TODO(),
		// 	UnknownPrefix, "AWSPARAMSTR://foo/bar", (&GenVarsConfig{}).WithKeySeparator("|").WithTokenSeparator("://"),
		// 	func(t *testing.T, ctx context.Context, conf GenVarsConfig) genVarsStrategy {
		// 		imp, err := NewParamStore(ctx)
		// 		if err != nil {
		// 			t.Errorf(testutils.TestPhraseWithContext, "init impl error", err.Error(), nil)
		// 		}
		// 		return imp
		// 	},
		// },
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			tearDown := tt.setUpTearDown()
			defer tearDown()
			want := tt.expect()
			rs := strategy.New(store.NewDefatultStrategy(), *tt.config, log.New(io.Discard))
			token, _ := config.NewParsedTokenConfig(tt.token, *tt.config)
			got, err := rs.SelectImplementation(context.TODO(), token)

			if err != nil {
				if err.Error() != tt.expErr.Error() {
					t.Errorf(testutils.TestPhraseWithContext, "uncaught error", err.Error(), tt.expErr.Error())
				}
				return
			}

			diff := deep.Equal(got, want)
			if diff != nil {
				t.Errorf(testutils.TestPhraseWithContext, "reflection of initialised implentations", fmt.Sprintf("%q", got), fmt.Sprintf("%q", want))
			}
		})
	}
}
