package generator

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/dnitsch/configmanager/internal/testutils"
)

var (
	tsuccessParam                   = "someVal"
	tsuccessObj   map[string]string = map[string]string{"AWSPARAMSTR#/token/1": "someVal"}
)

type mockParamApi func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)

func (m mockParamApi) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	return m(ctx, params, optFns...)
}

func awsParamtStoreCommonGetChecker(t *testing.T, params *ssm.GetParameterInput) {
	if params.Name == nil {
		t.Fatal("expect name to not be nil")
	}

	if strings.Contains(*params.Name, "#") {
		t.Errorf("incorrectly stripped token separator")
	}

	if strings.Contains(*params.Name, string(ParamStorePrefix)) {
		t.Errorf("incorrectly stripped prefix")
	}

	if !*params.WithDecryption {
		t.Fatal("expect WithDecryption to not be false")
	}
}

func Test_GetParamStore(t *testing.T) {
	tests := map[string]struct {
		token          string
		keySeparator   string
		tokenSeparator string
		expect         string
		mockClient     func(t *testing.T) paramStoreApi
		config         *GenVarsConfig
	}{
		"successVal": {"AWSPARAMSTR#/token/1", "|", "#", tsuccessParam, func(t *testing.T) paramStoreApi {
			return mockParamApi(func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				t.Helper()
				awsParamtStoreCommonGetChecker(t, params)
				return &ssm.GetParameterOutput{
					Parameter: &types.Parameter{Value: &tsuccessParam},
				}, nil
			})
		}, NewConfig(),
		},
		"successVal with keyseparator": {"AWSPARAMSTR#/token/1|somekey", "|", "#", tsuccessParam, func(t *testing.T) paramStoreApi {
			return mockParamApi(func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				t.Helper()
				awsParamtStoreCommonGetChecker(t, params)

				if strings.Contains(*params.Name, "|somekey") {
					t.Errorf("incorrectly stripped key separator")
				}

				return &ssm.GetParameterOutput{
					Parameter: &types.Parameter{Value: &tsuccessParam},
				}, nil
			})
		}, NewConfig(),
		},
		"errored": {"AWSPARAMSTR#/token/1", "|", "#", "unable to retrieve", func(t *testing.T) paramStoreApi {
			return mockParamApi(func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				t.Helper()
				awsParamtStoreCommonGetChecker(t, params)
				return nil, fmt.Errorf("unable to retrieve")
			})
		}, NewConfig(),
		},
		"nil to empty": {"AWSPARAMSTR#/token/1", "|", "#", "", func(t *testing.T) paramStoreApi {
			return mockParamApi(func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				t.Helper()
				awsParamtStoreCommonGetChecker(t, params)
				return &ssm.GetParameterOutput{
					Parameter: &types.Parameter{Value: nil},
				}, nil
			})
		}, NewConfig(),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.config.WithTokenSeparator(tt.tokenSeparator).WithKeySeparator(tt.keySeparator)
			rs := newRetrieveStrategy(NewDefatultStrategy(), *tt.config)
			impl, err := NewParamStore(context.TODO())
			if err != nil {
				t.Errorf(testutils.TestPhrase, err.Error(), nil)
			}
			impl.svc = tt.mockClient(t)
			rs.setImplementation(impl)
			rs.setTokenVal(tt.token)
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
