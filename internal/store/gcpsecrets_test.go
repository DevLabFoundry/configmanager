package store

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	gcpsecretspb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/DevLabFoundry/configmanager/v2/internal/config"
	"github.com/DevLabFoundry/configmanager/v2/internal/testutils"
	"github.com/DevLabFoundry/configmanager/v2/pkg/log"
	"github.com/googleapis/gax-go/v2"
)

type mockGcpSecretsApi func(ctx context.Context, req *gcpsecretspb.AccessSecretVersionRequest, opts ...gax.CallOption) (*gcpsecretspb.AccessSecretVersionResponse, error)

func (m mockGcpSecretsApi) AccessSecretVersion(ctx context.Context, req *gcpsecretspb.AccessSecretVersionRequest, opts ...gax.CallOption) (*gcpsecretspb.AccessSecretVersionResponse, error) {
	return m(ctx, req, opts...)
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

func fixtureInitMockClient(t *testing.T) struct {
	name   string
	close  func() error
	delete func(name string) error
} {

	cf, err := os.CreateTemp("", "gcp-creds*")
	if err != nil {
		t.Fatalf(testutils.TestPhraseWithContext, "unable to set up creds file", err.Error(), nil)
	}
	if _, err := cf.Write(TEST_GCP_CREDS); err != nil {
		t.Fatalf(testutils.TestPhraseWithContext, "unable to write mock creds into file", err.Error(), nil)
	}

	resp := struct {
		name   string
		close  func() error
		delete func(name string) error
	}{
		name:   cf.Name(),
		close:  cf.Close,
		delete: os.Remove,
	}
	return resp
}

func gcpSecretsGetChecker(t *testing.T, req *gcpsecretspb.AccessSecretVersionRequest) {
	t.Helper()
	if req.Name == "" {
		t.Fatal("expect name to not be nil")
	}
	if strings.Contains(req.Name, "#") {
		t.Errorf("incorrectly stripped token separator")
	}
	if strings.Contains(req.Name, string(config.GcpSecretsPrefix)) {
		t.Errorf("incorrectly stripped prefix")
	}
}

func Test_GetGcpSecretVarHappy(t *testing.T) {
	// t.Parallel()

	tests := map[string]struct {
		token      string
		expect     string
		mockClient func(t *testing.T) gcpSecretsApi
		config     *config.GenVarsConfig
	}{
		"success": {"GCPSECRETS#/token/1", "someValue", func(t *testing.T) gcpSecretsApi {
			return mockGcpSecretsApi(func(ctx context.Context, req *gcpsecretspb.AccessSecretVersionRequest, opts ...gax.CallOption) (*gcpsecretspb.AccessSecretVersionResponse, error) {
				gcpSecretsGetChecker(t, req)
				return &gcpsecretspb.AccessSecretVersionResponse{
					Payload: &gcpsecretspb.SecretPayload{Data: []byte("someValue")},
				}, nil
			})
		}, config.NewConfig().WithTokenSeparator("#").WithKeySeparator("|"),
		},
		"success with version": {"GCPSECRETS#/token/1[version=123]", "someValue", func(t *testing.T) gcpSecretsApi {
			return mockGcpSecretsApi(func(ctx context.Context, req *gcpsecretspb.AccessSecretVersionRequest, opts ...gax.CallOption) (*gcpsecretspb.AccessSecretVersionResponse, error) {
				gcpSecretsGetChecker(t, req)
				return &gcpsecretspb.AccessSecretVersionResponse{
					Payload: &gcpsecretspb.SecretPayload{Data: []byte("someValue")},
				}, nil
			})
		}, config.NewConfig().WithTokenSeparator("#").WithKeySeparator("|"),
		},
		"error": {"GCPSECRETS#/token/1", "unable to retrieve secret", func(t *testing.T) gcpSecretsApi {
			return mockGcpSecretsApi(func(ctx context.Context, req *gcpsecretspb.AccessSecretVersionRequest, opts ...gax.CallOption) (*gcpsecretspb.AccessSecretVersionResponse, error) {
				gcpSecretsGetChecker(t, req)
				return nil, fmt.Errorf("unable to retrieve secret")
			})
		}, config.NewConfig().WithTokenSeparator("#").WithKeySeparator("|"),
		},
		"found but empty": {
			"GCPSECRETS#/token/1",
			"",
			func(t *testing.T) gcpSecretsApi {
				return mockGcpSecretsApi(func(ctx context.Context, req *gcpsecretspb.AccessSecretVersionRequest, opts ...gax.CallOption) (*gcpsecretspb.AccessSecretVersionResponse, error) {
					gcpSecretsGetChecker(t, req)
					return &gcpsecretspb.AccessSecretVersionResponse{}, nil
				})
			},
			config.NewConfig().WithTokenSeparator("#").WithKeySeparator("|"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := fixtureInitMockClient(t)
			defer fixture.close()
			defer fixture.delete(fixture.name)

			os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", fixture.name)
			token, _ := config.NewParsedTokenConfig(tt.token, *tt.config)

			impl, err := NewGcpSecrets(context.TODO(), log.New(io.Discard))

			if err != nil {
				t.Errorf(testutils.TestPhrase, err.Error(), nil)
			}

			impl.svc = tt.mockClient(t)
			impl.close = func() error { return nil }
			impl.SetToken(token)
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
