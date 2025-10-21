package store_test

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/internal/log"
	"github.com/DevLabFoundry/configmanager/v2/internal/store"
	"github.com/DevLabFoundry/configmanager/v2/internal/testutils"
	vault "github.com/hashicorp/vault/api"
)

func TestMountPathExtract(t *testing.T) {
	ttests := map[string]struct {
		token  func() *config.ParsedTokenConfig
		expect string
	}{
		"without leading slash": {
			func() *config.ParsedTokenConfig {
				// "VAULT://secret___/demo/configmanager"
				tkn, _ := config.NewToken(config.HashicorpVaultPrefix, *config.NewConfig())
				tkn.WithSanitizedToken("secret___/demo/configmanager")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			}, "secret"},
		"with leading slash": {
			func() *config.ParsedTokenConfig {
				// "VAULT:///secret___/demo/configmanager",
				tkn, _ := config.NewToken(config.HashicorpVaultPrefix, *config.NewConfig())
				tkn.WithSanitizedToken("/secret___/demo/configmanager")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			}, "secret"},
		"with underscore in path name": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.HashicorpVaultPrefix, *config.NewConfig())
				tkn.WithSanitizedToken("_secret___/demo/configmanager")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			}, "_secret"},
		"with double underscore in path name": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.HashicorpVaultPrefix, *config.NewConfig())
				tkn.WithSanitizedToken("__secret___/demo/configmanager")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			}, "__secret"},
		"with multiple paths in mountpath": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.HashicorpVaultPrefix, *config.NewConfig())
				tkn.WithSanitizedToken("secret/bar/path___/demo/configmanager")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			}, "secret/bar/path"},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			got := store.SplitHashiVaultToken(tt.token().StoreToken())
			if got.Path != tt.expect {
				t.Errorf("got %q, expected %q", got, tt.expect)
			}
		})
	}
}

type mockVaultApi struct {
	g  func(ctx context.Context, secretPath string) (*vault.KVSecret, error)
	gv func(ctx context.Context, secretPath string, version int) (*vault.KVSecret, error)
}

func (m mockVaultApi) Get(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
	return m.g(ctx, secretPath)
}

func (m mockVaultApi) GetVersion(ctx context.Context, secretPath string, version int) (*vault.KVSecret, error) {
	return m.gv(ctx, secretPath, version)
}

func TestVaultScenarios(t *testing.T) {

	ttests := map[string]struct {
		token      func() *config.ParsedTokenConfig
		expect     string
		mockClient func(t *testing.T) mockVaultApi
		setupEnv   func() func()
	}{
		"happy return": {
			func() *config.ParsedTokenConfig {
				tkn, _ := config.NewToken(config.HashicorpVaultPrefix, *config.NewConfig())
				tkn.WithSanitizedToken("secret___/foo")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			}, `{"foo":"test2130-9sd-0ds"}`,
			func(t *testing.T) mockVaultApi {
				mv := mockVaultApi{}
				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
					t.Helper()
					if secretPath != "foo" {
						t.Errorf("got %v; want %s", secretPath, `foo`)
					}
					m := make(map[string]interface{})
					m["foo"] = "test2130-9sd-0ds"
					return &vault.KVSecret{Data: m}, nil
				}
				return mv
			},
			func() func() {
				os.Setenv("VAULT_TOKEN", "129378y1231283")
				return func() {
					os.Clearenv()
				}
			},
		},
		// 		"incorrect json": {"VAULT://secret___/foo", config.NewConfig(), `json: unsupported type: func() error`,
		// 			func(t *testing.T) hashiVaultApi {
		// 				mv := mockVaultApi{}
		// 				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
		// 					t.Helper()
		// 					if secretPath != "foo" {
		// 						t.Errorf("got %v; want %s", secretPath, `foo`)
		// 					}
		// 					m := make(map[string]interface{})
		// 					m["error"] = func() error { return fmt.Errorf("ddodod") }
		// 					return &vault.KVSecret{Data: m}, nil
		// 				}
		// 				return mv
		// 			},
		// 			func() func() {
		// 				os.Setenv("VAULT_TOKEN", "129378y1231283")
		// 				return func() {
		// 					os.Clearenv()
		// 				}
		// 			},
		// 		},
		// 		"another return": {
		// 			"VAULT://secret/engine1___/some/other/foo2",
		// 			config.NewConfig(),
		// 			`{"foo1":"test2130-9sd-0ds","foo2":"dsfsdf3454456"}`,
		// 			func(t *testing.T) hashiVaultApi {
		// 				mv := mockVaultApi{}
		// 				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
		// 					t.Helper()
		// 					if secretPath != "some/other/foo2" {
		// 						t.Errorf("got %v; want %s", secretPath, `some/other/foo2`)
		// 					}
		// 					m := make(map[string]interface{})
		// 					m["foo1"] = "test2130-9sd-0ds"
		// 					m["foo2"] = "dsfsdf3454456"
		// 					return &vault.KVSecret{Data: m}, nil
		// 				}
		// 				return mv
		// 			},
		// 			func() func() {
		// 				os.Setenv("VAULT_TOKEN", "129378y1231283")
		// 				return func() {
		// 					os.Clearenv()
		// 				}
		// 			},
		// 		},
		// 		"not found": {"VAULT://secret___/foo", config.NewConfig(), `secret not found`,
		// 			func(t *testing.T) hashiVaultApi {
		// 				mv := mockVaultApi{}
		// 				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
		// 					t.Helper()
		// 					if secretPath != "foo" {
		// 						t.Errorf("got %v; want %s", secretPath, `foo`)
		// 					}
		// 					return nil, fmt.Errorf("secret not found")
		// 				}
		// 				return mv
		// 			},
		// 			func() func() {
		// 				os.Setenv("VAULT_TOKEN", "129378y1231283")
		// 				return func() {
		// 					os.Clearenv()
		// 				}
		// 			},
		// 		},
		// 		"403": {"VAULT://secret___/some/other/foo2", config.NewConfig(), `client 403`,
		// 			func(t *testing.T) hashiVaultApi {
		// 				mv := mockVaultApi{}
		// 				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
		// 					t.Helper()
		// 					if secretPath != "some/other/foo2" {
		// 						t.Errorf("got %v; want %s", secretPath, `some/other/foo2`)
		// 					}
		// 					return nil, fmt.Errorf("client 403")
		// 				}
		// 				return mv
		// 			},
		// 			func() func() {
		// 				os.Setenv("VAULT_TOKEN", "129378y1231283")
		// 				return func() {
		// 					os.Clearenv()
		// 				}
		// 			},
		// 		},
		// 		"found but empty": {"VAULT://secret___/some/other/foo2", config.NewConfig(), `{}`, func(t *testing.T) hashiVaultApi {
		// 			mv := mockVaultApi{}
		// 			mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
		// 				t.Helper()
		// 				if secretPath != "some/other/foo2" {
		// 					t.Errorf("got %v; want %s", secretPath, `some/other/foo2`)
		// 				}
		// 				m := make(map[string]interface{})
		// 				return &vault.KVSecret{Data: m}, nil
		// 			}
		// 			return mv
		// 		},
		// 			func() func() {
		// 				os.Setenv("VAULT_TOKEN", "129378y1231283")
		// 				return func() {
		// 					os.Clearenv()
		// 				}
		// 			},
		// 		},
		// 		"found but nil returned": {"VAULT://secret___/some/other/foo2", config.NewConfig(), "", func(t *testing.T) hashiVaultApi {
		// 			mv := mockVaultApi{}
		// 			mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
		// 				t.Helper()
		// 				if secretPath != "some/other/foo2" {
		// 					t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
		// 				}
		// 				return &vault.KVSecret{Data: nil}, nil
		// 			}
		// 			return mv
		// 		},
		// 			func() func() {
		// 				os.Setenv("VAULT_TOKEN", "129378y1231283")
		// 				return func() {
		// 					os.Clearenv()
		// 				}
		// 			},
		// 		},
		// 		"version provided correctly": {"VAULT://secret___/some/other/foo2[version=1]", config.NewConfig(), `{"foo2":"dsfsdf3454456"}`, func(t *testing.T) hashiVaultApi {
		// 			mv := mockVaultApi{}
		// 			mv.gv = func(ctx context.Context, secretPath string, version int) (*vault.KVSecret, error) {
		// 				t.Helper()
		// 				if secretPath != "some/other/foo2" {
		// 					t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
		// 				}
		// 				m := make(map[string]interface{})
		// 				m["foo2"] = "dsfsdf3454456"
		// 				return &vault.KVSecret{Data: m}, nil
		// 			}
		// 			return mv
		// 		},
		// 			func() func() {
		// 				os.Setenv("VAULT_TOKEN", "129378y1231283")
		// 				return func() {
		// 					os.Clearenv()
		// 				}
		// 			},
		// 		},
		// 		"version provided but unable to parse": {"VAULT://secret___/some/other/foo2[version=1a]", config.NewConfig(), "unable to parse version into an integer: strconv.Atoi: parsing \"1a\": invalid syntax", func(t *testing.T) hashiVaultApi {
		// 			mv := mockVaultApi{}
		// 			mv.gv = func(ctx context.Context, secretPath string, version int) (*vault.KVSecret, error) {
		// 				t.Helper()
		// 				if secretPath != "some/other/foo2" {
		// 					t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
		// 				}
		// 				return nil, nil
		// 			}
		// 			return mv
		// 		},
		// 			func() func() {
		// 				os.Setenv("VAULT_TOKEN", "129378y1231283")
		// 				return func() {
		// 					os.Clearenv()
		// 				}
		// 			},
		// 		},
		// 		"vault rate limit incorrect": {
		// 			"VAULT://secret___/some/other/foo2",
		// 			config.NewConfig(),
		// 			`error encountered setting up default configuration: VAULT_RATE_LIMIT was provided but incorrectly formatted
		// failed to initialize the client`,
		// 			func(t *testing.T) hashiVaultApi {
		// 				mv := mockVaultApi{}
		// 				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
		// 					t.Helper()
		// 					if secretPath != "some/other/foo2" {
		// 						t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
		// 					}
		// 					return &vault.KVSecret{Data: nil}, nil
		// 				}
		// 				return mv
		// 			},
		// 			func() func() {
		// 				os.Setenv("VAULT_TOKEN", "")
		// 				os.Setenv("VAULT_RATE_LIMIT", "wrong")
		// 				return func() {
		// 					os.Clearenv()
		// 				}
		// 			},
		// 		},
	}

	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			tearDown := tt.setupEnv()
			defer tearDown()

			impl, err := store.NewVaultStore(context.TODO(), tt.token(), log.New(io.Discard))
			if err != nil {
				if err.Error() != tt.expect {
					t.Fatalf("failed to init hashivault, %v", err.Error())
				}
				return
			}

			impl.WithSvc(tt.mockClient(t))
			got, err := impl.Token()
			if err != nil {
				if err.Error() != tt.expect {
					t.Errorf(testutils.TestPhrase, err.Error(), tt.expect)
				}
				return
			}
			if got != tt.expect {
				t.Errorf(testutils.TestPhrase, got, tt.expect)
			}
		})
	}
}

// func TestAwsIamAuth(t *testing.T) {
// 	ttests := map[string]struct {
// 		token       string
// 		conf        *config.GenVarsConfig
// 		expect      string
// 		mockClient  func(t *testing.T) hashiVaultApi
// 		mockHanlder func(t *testing.T) http.Handler
// 		setupEnv    func(addr string) func()
// 	}{
// 		"aws_iam auth no role specified": {
// 			"VAULT://secret___/some/other/foo2[version:1]", config.NewConfig(),
// 			"role provided is empty, EC2 auth not supported",
// 			func(t *testing.T) hashiVaultApi {
// 				mv := mockVaultApi{}
// 				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
// 					t.Helper()
// 					if secretPath != "some/other/foo2" {
// 						t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
// 					}
// 					return &vault.KVSecret{Data: nil}, nil
// 				}
// 				return mv
// 			},
// 			func(t *testing.T) http.Handler {
// 				return nil
// 			},
// 			func(_ string) func() {
// 				os.Setenv("VAULT_TOKEN", "aws_iam")
// 				os.Setenv("AWS_ACCESS_KEY_ID", "1280qwed9u9nsc9fdsbv9gsfrd")
// 				os.Setenv("AWS_SECRET_ACCESS_KEY", "SED)SDVfdv0jfds08sdfgu09sd943tj4fELH/")
// 				os.Setenv("AWS_SESSION_TOKEN", "IQoJb3JpZ2luX2VjELH//////////wEaCWV1LXdlc3QtMiJIMEYCIQDPU6UGJ0...df.fdgdfg.dfg.gdf.dgf")
// 				os.Setenv("AWS_REGION", "eu-west-1")
// 				return func() {
// 					os.Clearenv()
// 				}
// 			},
// 		},
// 		"aws_iam auth incorrectly formatted request": {
// 			"VAULT://secret___/some/other/foo2[version=1,iam_role=not_a_role]", config.NewConfig(),
// 			`unable to login to AWS auth method: unable to log in to auth method: unable to log in with AWS auth: Error making API request.

// URL: PUT %s/v1/auth/aws/login
// Code: 400. Raw Message:

// incorrect values supplied. failed to initialize the client`,
// 			func(t *testing.T) hashiVaultApi {
// 				mv := mockVaultApi{}
// 				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
// 					t.Helper()
// 					if secretPath != "some/other/foo2" {
// 						t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
// 					}
// 					return &vault.KVSecret{Data: nil}, nil
// 				}
// 				return mv
// 			},
// 			func(t *testing.T) http.Handler {
// 				mux := http.NewServeMux()
// 				mux.HandleFunc("/v1/auth/aws/login", func(w http.ResponseWriter, r *http.Request) {

// 					w.Header().Set("Content-Type", "application/json; charset=utf-8")
// 					w.WriteHeader(400)
// 					w.Write([]byte(`incorrect values supplied`))
// 				})
// 				return mux
// 			},
// 			func(addr string) func() {
// 				os.Setenv("VAULT_TOKEN", "aws_iam")
// 				os.Setenv("VAULT_ADDR", addr)
// 				os.Setenv("AWS_ACCESS_KEY_ID", "1280qwed9u9nsc9fdsbv9gsfrd")
// 				os.Setenv("AWS_SECRET_ACCESS_KEY", "SED)SDVfdv0jfds08sdfgu09sd943tj4fELH/")
// 				os.Setenv("AWS_SESSION_TOKEN", "IQoJb3JpZ2luX2VjELH//////////wEaCWV1LXdlc3QtMiJIMEYCIQDPU6UGJ0...df.fdgdfg.dfg.gdf.dgf")
// 				os.Setenv("AWS_REGION", "eu-west-1")
// 				return func() {
// 					os.Clearenv()
// 				}
// 			},
// 		},
// 		"aws_iam auth success": {
// 			"VAULT://secret___/some/other/foo2[iam_role=arn:aws:iam::1111111:role/i-orchestration]", config.NewConfig(),
// 			`{"foo2":"dsfsdf3454456"}`,
// 			func(t *testing.T) hashiVaultApi {
// 				mv := mockVaultApi{}
// 				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
// 					t.Helper()
// 					if secretPath != "some/other/foo2" {
// 						t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
// 					}
// 					m := make(map[string]interface{})
// 					m["foo2"] = "dsfsdf3454456"
// 					return &vault.KVSecret{Data: m}, nil
// 				}
// 				return mv
// 			},
// 			func(t *testing.T) http.Handler {
// 				mux := http.NewServeMux()
// 				mux.HandleFunc("/v1/auth/aws/login", func(w http.ResponseWriter, r *http.Request) {

// 					w.Header().Set("Content-Type", "application/json; charset=utf-8")
// 					w.Write([]byte(`{"auth":{"client_token": "fooresddfasdsasad"}}`))
// 				})
// 				return mux
// 			},
// 			func(addr string) func() {
// 				os.Setenv("VAULT_TOKEN", "aws_iam")
// 				os.Setenv("VAULT_ADDR", addr)
// 				os.Setenv("AWS_ACCESS_KEY_ID", "1280qwed9u9nsc9fdsbv9gsfrd")
// 				os.Setenv("AWS_SECRET_ACCESS_KEY", "SED)SDVfdv0jfds08sdfgu09sd943tj4fELH/")
// 				os.Setenv("AWS_SESSION_TOKEN", "IQoJb3JpZ2luX2VjELH//////////wEaCWV1LXdlc3QtMiJIMEYCIQDPU6UGJ0...df.fdgdfg.dfg.gdf.dgf")
// 				os.Setenv("AWS_REGION", "eu-west-1")
// 				return func() {
// 					os.Clearenv()
// 				}
// 			},
// 		},
// 		"aws_iam auth no token returned": {
// 			"VAULT://secret___/some/other/foo2[iam_role=arn:aws:iam::1111111:role/i-orchestration]", config.NewConfig(),
// 			`unable to login to AWS auth method: response did not return ClientToken, client token not set. failed to initialize the client`,
// 			func(t *testing.T) hashiVaultApi {
// 				mv := mockVaultApi{}
// 				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
// 					t.Helper()
// 					if secretPath != "some/other/foo2" {
// 						t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
// 					}
// 					m := make(map[string]interface{})
// 					m["foo2"] = "dsfsdf3454456"
// 					return &vault.KVSecret{Data: m}, nil
// 				}
// 				return mv
// 			},
// 			func(t *testing.T) http.Handler {
// 				mux := http.NewServeMux()
// 				mux.HandleFunc("/v1/auth/aws/login", func(w http.ResponseWriter, r *http.Request) {

// 					w.Header().Set("Content-Type", "application/json; charset=utf-8")
// 					w.Write([]byte(`{"auth":{}}`))
// 				})
// 				return mux
// 			},
// 			func(addr string) func() {
// 				os.Setenv("VAULT_TOKEN", "aws_iam")
// 				os.Setenv("VAULT_ADDR", addr)
// 				os.Setenv("AWS_ACCESS_KEY_ID", "1280qwed9u9nsc9fdsbv9gsfrd")
// 				os.Setenv("AWS_SECRET_ACCESS_KEY", "SED)SDVfdv0jfds08sdfgu09sd943tj4fELH/")
// 				os.Setenv("AWS_SESSION_TOKEN", "IQoJb3JpZ2luX2VjELH//////////wEaCWV1LXdlc3QtMiJIMEYCIQDPU6UGJ0...df.fdgdfg.dfg.gdf.dgf")
// 				os.Setenv("AWS_REGION", "eu-west-1")
// 				return func() {
// 					os.Clearenv()
// 				}
// 			},
// 		},
// 	}

// 	for name, tt := range ttests {
// 		t.Run(name, func(t *testing.T) {
// 			//
// 			ts := httptest.NewServer(tt.mockHanlder(t))
// 			tearDown := tt.setupEnv(ts.URL)
// 			defer tearDown()
// 			token, _ := config.NewToken(tt.token, *tt.conf)

// 			impl, err := NewVaultStore(context.TODO(), token, log.New(io.Discard))
// 			if err != nil {
// 				// WHAT A CRAP way to do this...
// 				if err.Error() != strings.Split(fmt.Sprintf(tt.expect, ts.URL), `%!`)[0] {
// 					t.Errorf(testutils.TestPhraseWithContext, "aws iam auth", err.Error(), strings.Split(fmt.Sprintf(tt.expect, ts.URL), `%!`)[0])
// 					t.Fatalf("failed to init hashivault, %v", err.Error())
// 				}
// 				return
// 			}

// 			impl.svc = tt.mockClient(t)
// 			got, err := impl.Token()
// 			if err != nil {
// 				if err.Error() != tt.expect {
// 					t.Errorf(testutils.TestPhrase, err.Error(), tt.expect)
// 				}
// 				return
// 			}
// 			if got != tt.expect {
// 				t.Errorf(testutils.TestPhrase, got, tt.expect)
// 			}
// 		})
// 	}
// }
