package generator

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/dnitsch/configmanager/internal/testutils"
	vault "github.com/hashicorp/vault/api"
)

func TestMountPathExtract(t *testing.T) {
	ttests := map[string]struct {
		token          string
		tokenSeparator string
		keySeparator   string
		expect         string
	}{
		"without leading slash":               {"VAULT://secret___/demo/configmanager", "://", "|", "secret"},
		"with leading slash":                  {"VAULT:///secret___/demo/configmanager", "://", "|", "secret"},
		"with underscore in path name":        {"VAULT://_secret___/demo/configmanager", "://", "|", "_secret"},
		"with double underscore in path name": {"VAULT://__secret___/demo/configmanager", "://", "|", "__secret"},
		"with multiple paths in mountpath":    {"VAULT://secret/bar/path___/demo/configmanager", "://", "|", "secret/bar/path"},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			strippedToken := stripPrefix(tt.token, HashicorpVaultPrefix, tt.tokenSeparator, tt.keySeparator)
			got := splitToken(strippedToken)
			if got.path != tt.expect {
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
		token      string
		conf       GenVarsConfig
		expect     string
		mockClient func(t *testing.T) hashiVaultApi
		setupEnv   func() func()
	}{
		"happy return": {"VAULT://secret___/foo", GenVarsConfig{tokenSeparator: "://", keySeparator: "|"}, `{"foo":"test2130-9sd-0ds"}`,
			func(t *testing.T) hashiVaultApi {
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
		"incorrect json": {"VAULT://secret___/foo", GenVarsConfig{tokenSeparator: "://", keySeparator: "|"}, `json: unsupported type: func() error`,
			func(t *testing.T) hashiVaultApi {
				mv := mockVaultApi{}
				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
					t.Helper()
					if secretPath != "foo" {
						t.Errorf("got %v; want %s", secretPath, `foo`)
					}
					m := make(map[string]interface{})
					m["error"] = func() error { return fmt.Errorf("ddodod") }
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
		"another return": {
			"VAULT://secret/engine1___/some/other/foo2",
			GenVarsConfig{tokenSeparator: "://", keySeparator: "|"},
			`{"foo1":"test2130-9sd-0ds","foo2":"dsfsdf3454456"}`,
			func(t *testing.T) hashiVaultApi {
				mv := mockVaultApi{}
				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
					t.Helper()
					if secretPath != "some/other/foo2" {
						t.Errorf("got %v; want %s", secretPath, `some/other/foo2`)
					}
					m := make(map[string]interface{})
					m["foo1"] = "test2130-9sd-0ds"
					m["foo2"] = "dsfsdf3454456"
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
		"not found": {"VAULT://secret___/foo", GenVarsConfig{tokenSeparator: "://", keySeparator: "|"}, `secret not found`,
			func(t *testing.T) hashiVaultApi {
				mv := mockVaultApi{}
				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
					t.Helper()
					if secretPath != "foo" {
						t.Errorf("got %v; want %s", secretPath, `foo`)
					}
					return nil, fmt.Errorf("secret not found")
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
		"403": {"VAULT://secret___/some/other/foo2", GenVarsConfig{tokenSeparator: "://", keySeparator: "|"}, `client 403`,
			func(t *testing.T) hashiVaultApi {
				mv := mockVaultApi{}
				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
					t.Helper()
					if secretPath != "some/other/foo2" {
						t.Errorf("got %v; want %s", secretPath, `some/other/foo2`)
					}
					return nil, fmt.Errorf("client 403")
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
		"found but empty": {"VAULT://secret___/some/other/foo2", GenVarsConfig{tokenSeparator: "://", keySeparator: "|"}, `{}`, func(t *testing.T) hashiVaultApi {
			mv := mockVaultApi{}
			mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
				t.Helper()
				if secretPath != "some/other/foo2" {
					t.Errorf("got %v; want %s", secretPath, `some/other/foo2`)
				}
				m := make(map[string]interface{})
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
		"found but nil returned": {"VAULT://secret___/some/other/foo2", GenVarsConfig{tokenSeparator: "://", keySeparator: "|"}, "", func(t *testing.T) hashiVaultApi {
			mv := mockVaultApi{}
			mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
				t.Helper()
				if secretPath != "some/other/foo2" {
					t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
				}
				return &vault.KVSecret{Data: nil}, nil
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
		"version provided correctly": {"VAULT://secret___/some/other/foo2[version:1]", GenVarsConfig{tokenSeparator: "://", keySeparator: "|"}, `{"foo2":"dsfsdf3454456"}`, func(t *testing.T) hashiVaultApi {
			mv := mockVaultApi{}
			mv.gv = func(ctx context.Context, secretPath string, version int) (*vault.KVSecret, error) {
				t.Helper()
				if secretPath != "some/other/foo2" {
					t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
				}
				m := make(map[string]interface{})
				m["foo2"] = "dsfsdf3454456"
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
		"version provided but unable to parse": {"VAULT://secret___/some/other/foo2[version:1a]", GenVarsConfig{tokenSeparator: "://", keySeparator: "|"}, "unable to parse version into an integer: strconv.Atoi: parsing \"1a\": invalid syntax", func(t *testing.T) hashiVaultApi {
			mv := mockVaultApi{}
			mv.gv = func(ctx context.Context, secretPath string, version int) (*vault.KVSecret, error) {
				t.Helper()
				if secretPath != "some/other/foo2" {
					t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
				}
				return nil, nil
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
		"vault rate limit incorrect": {"VAULT://secret___/some/other/foo2", GenVarsConfig{tokenSeparator: "://", keySeparator: "|"}, "unable to initialize Vault client: error encountered setting up default configuration: VAULT_RATE_LIMIT was provided but incorrectly formatted", func(t *testing.T) hashiVaultApi {
			mv := mockVaultApi{}
			mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
				t.Helper()
				if secretPath != "some/other/foo2" {
					t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
				}
				return &vault.KVSecret{Data: nil}, nil
			}
			return mv
		},
			func() func() {
				os.Setenv("VAULT_TOKEN", "")
				os.Setenv("VAULT_RATE_LIMIT", "wrong")
				return func() {
					os.Clearenv()
				}
			},
		},
	}

	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			tearDown := tt.setupEnv()
			defer tearDown()
			impl, err := NewVaultStore(context.TODO(), tt.token, tt.conf)
			if err != nil {
				if err.Error() != tt.expect {
					t.Fatalf("failed to init hashivault, %v", err.Error())
				}
				return
			}

			impl.svc = tt.mockClient(t)
			rs := newRetrieveStrategy(NewDefatultStrategy(), tt.conf)
			rs.setImplementation(impl)
			got, err := rs.getTokenValue()
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

func TestAwsIamAuth(t *testing.T) {
	ttests := map[string]struct {
		token       string
		conf        GenVarsConfig
		expect      string
		mockClient  func(t *testing.T) hashiVaultApi
		mockHanlder func(t *testing.T) http.Handler
		setupEnv    func(addr string) func()
	}{
		"aws_iam auth no role specified": {
			"VAULT://secret___/some/other/foo2[version:1]", GenVarsConfig{tokenSeparator: "://", keySeparator: "|"},
			"role provided is empty, EC2 auth not supported",
			func(t *testing.T) hashiVaultApi {
				mv := mockVaultApi{}
				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
					t.Helper()
					if secretPath != "some/other/foo2" {
						t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
					}
					return &vault.KVSecret{Data: nil}, nil
				}
				return mv
			},
			func(t *testing.T) http.Handler {
				return nil
			},
			func(_ string) func() {
				os.Setenv("VAULT_TOKEN", "aws_iam")
				os.Setenv("AWS_ACCESS_KEY_ID", "1280qwed9u9nsc9fdsbv9gsfrd")
				os.Setenv("AWS_SECRET_ACCESS_KEY", "SED)SDVfdv0jfds08sdfgu09sd943tj4fELH/")
				os.Setenv("AWS_SESSION_TOKEN", "IQoJb3JpZ2luX2VjELH//////////wEaCWV1LXdlc3QtMiJIMEYCIQDPU6UGJ0...df.fdgdfg.dfg.gdf.dgf")
				os.Setenv("AWS_REGION", "eu-west-1")
				return func() {
					os.Clearenv()
				}
			},
		},
		"aws_iam auth incorrectly formatted request": {
			"VAULT://secret___/some/other/foo2[version:1,role:not_a_role]", GenVarsConfig{tokenSeparator: "://", keySeparator: "|"},
			`unable to login to AWS auth method: unable to log in to auth method: unable to log in with AWS auth: Error making API request.

URL: PUT %s/v1/auth/aws/login
Code: 400. Raw Message:

incorrect values supplied`,
			func(t *testing.T) hashiVaultApi {
				mv := mockVaultApi{}
				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
					t.Helper()
					if secretPath != "some/other/foo2" {
						t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
					}
					return &vault.KVSecret{Data: nil}, nil
				}
				return mv
			},
			func(t *testing.T) http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc("/v1/auth/aws/login", func(w http.ResponseWriter, r *http.Request) {

					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(400)
					w.Write([]byte(`incorrect values supplied`))
				})
				return mux
			},
			func(addr string) func() {
				os.Setenv("VAULT_TOKEN", "aws_iam")
				os.Setenv("VAULT_ADDR", addr)
				os.Setenv("AWS_ACCESS_KEY_ID", "1280qwed9u9nsc9fdsbv9gsfrd")
				os.Setenv("AWS_SECRET_ACCESS_KEY", "SED)SDVfdv0jfds08sdfgu09sd943tj4fELH/")
				os.Setenv("AWS_SESSION_TOKEN", "IQoJb3JpZ2luX2VjELH//////////wEaCWV1LXdlc3QtMiJIMEYCIQDPU6UGJ0...df.fdgdfg.dfg.gdf.dgf")
				os.Setenv("AWS_REGION", "eu-west-1")
				return func() {
					os.Clearenv()
				}
			},
		},
		"aws_iam auth success": {
			"VAULT://secret___/some/other/foo2[role:arn:aws:iam::1111111:role/i-orchestration]", GenVarsConfig{tokenSeparator: "://", keySeparator: "|"},
			`{"foo2":"dsfsdf3454456"}`,
			func(t *testing.T) hashiVaultApi {
				mv := mockVaultApi{}
				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
					t.Helper()
					if secretPath != "some/other/foo2" {
						t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
					}
					m := make(map[string]interface{})
					m["foo2"] = "dsfsdf3454456"
					return &vault.KVSecret{Data: m}, nil
				}
				return mv
			},
			func(t *testing.T) http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc("/v1/auth/aws/login", func(w http.ResponseWriter, r *http.Request) {

					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.Write([]byte(`{"auth":{"client_token": "fooresddfasdsasad"}}`))
				})
				return mux
			},
			func(addr string) func() {
				os.Setenv("VAULT_TOKEN", "aws_iam")
				os.Setenv("VAULT_ADDR", addr)
				os.Setenv("AWS_ACCESS_KEY_ID", "1280qwed9u9nsc9fdsbv9gsfrd")
				os.Setenv("AWS_SECRET_ACCESS_KEY", "SED)SDVfdv0jfds08sdfgu09sd943tj4fELH/")
				os.Setenv("AWS_SESSION_TOKEN", "IQoJb3JpZ2luX2VjELH//////////wEaCWV1LXdlc3QtMiJIMEYCIQDPU6UGJ0...df.fdgdfg.dfg.gdf.dgf")
				os.Setenv("AWS_REGION", "eu-west-1")
				return func() {
					os.Clearenv()
				}
			},
		},
		"aws_iam auth no token returned": {
			"VAULT://secret___/some/other/foo2[role:arn:aws:iam::1111111:role/i-orchestration]", GenVarsConfig{tokenSeparator: "://", keySeparator: "|"},
			`unable to login to AWS auth method: response did not return ClientToken, client token not set`,
			func(t *testing.T) hashiVaultApi {
				mv := mockVaultApi{}
				mv.g = func(ctx context.Context, secretPath string) (*vault.KVSecret, error) {
					t.Helper()
					if secretPath != "some/other/foo2" {
						t.Errorf(testutils.TestPhrase, secretPath, `some/other/foo2`)
					}
					m := make(map[string]interface{})
					m["foo2"] = "dsfsdf3454456"
					return &vault.KVSecret{Data: m}, nil
				}
				return mv
			},
			func(t *testing.T) http.Handler {
				mux := http.NewServeMux()
				mux.HandleFunc("/v1/auth/aws/login", func(w http.ResponseWriter, r *http.Request) {

					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.Write([]byte(`{"auth":{}}`))
				})
				return mux
			},
			func(addr string) func() {
				os.Setenv("VAULT_TOKEN", "aws_iam")
				os.Setenv("VAULT_ADDR", addr)
				os.Setenv("AWS_ACCESS_KEY_ID", "1280qwed9u9nsc9fdsbv9gsfrd")
				os.Setenv("AWS_SECRET_ACCESS_KEY", "SED)SDVfdv0jfds08sdfgu09sd943tj4fELH/")
				os.Setenv("AWS_SESSION_TOKEN", "IQoJb3JpZ2luX2VjELH//////////wEaCWV1LXdlc3QtMiJIMEYCIQDPU6UGJ0...df.fdgdfg.dfg.gdf.dgf")
				os.Setenv("AWS_REGION", "eu-west-1")
				return func() {
					os.Clearenv()
				}
			},
		},
	}

	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			//
			ts := httptest.NewServer(tt.mockHanlder(t))
			tearDown := tt.setupEnv(ts.URL)
			defer tearDown()
			impl, err := NewVaultStore(context.TODO(), tt.token, tt.conf)
			if err != nil {
				// WHAT A CRAP way to do this...
				if err.Error() != strings.Split(fmt.Sprintf(tt.expect, ts.URL), `%!`)[0] {
					t.Errorf(testutils.TestPhraseWithContext, "aws iam auth", err.Error(), strings.Split(fmt.Sprintf(tt.expect, ts.URL), `%!`)[0])
					t.Fatalf("failed to init hashivault, %v", err.Error())
				}
				return
			}

			impl.svc = tt.mockClient(t)
			rs := newRetrieveStrategy(NewDefatultStrategy(), tt.conf)
			rs.setImplementation(impl)
			got, err := rs.getTokenValue()
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
