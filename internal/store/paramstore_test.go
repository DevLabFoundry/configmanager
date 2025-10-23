package store_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/DevLabFoundry/configmanager/v3/internal/store"
	"github.com/DevLabFoundry/configmanager/v3/internal/testutils"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// var (
// 	tsuccessParam                   = "someVal"
// 	tsuccessObj   map[string]string = map[string]string{"AWSPARAMSTR#/token/1": "someVal"}
// )

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

	if strings.Contains(*params.Name, string(config.ParamStorePrefix)) {
		t.Errorf("incorrectly stripped prefix")
	}

	if !*params.WithDecryption {
		t.Fatal("expect WithDecryption to not be false")
	}
}

func Test_GetParamStore(t *testing.T) {
	var (
		tsuccessParam = "someVal"
		// tsuccessObj   map[string]string = map[string]string{"AWSPARAMSTR#/token/1": "someVal"}
	)
	tests := map[string]struct {
		token      func() *config.ParsedTokenConfig
		expect     string
		mockClient func(t *testing.T) mockParamApi
	}{
		"successVal": {
			func() *config.ParsedTokenConfig {
				// "VAULT://secret___/demo/configmanager"
				tkn, _ := config.NewToken(config.ParamStorePrefix, *config.NewConfig())
				tkn.WithSanitizedToken("/token/1")
				tkn.WithKeyPath("")
				tkn.WithMetadata("")
				return tkn
			},
			tsuccessParam, func(t *testing.T) mockParamApi {
				return mockParamApi(func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
					t.Helper()
					awsParamtStoreCommonGetChecker(t, params)
					return &ssm.GetParameterOutput{
						Parameter: &types.Parameter{Value: &tsuccessParam},
					}, nil
				})
			},
		},
		// "successVal with keyseparator": {"AWSPARAMSTR#/token/1|somekey", "|", "#", tsuccessParam, func(t *testing.T) paramStoreApi {
		// 	return mockParamApi(func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
		// 		t.Helper()
		// 		awsParamtStoreCommonGetChecker(t, params)

		// 		if strings.Contains(*params.Name, "|somekey") {
		// 			t.Errorf("incorrectly stripped key separator")
		// 		}

		// 		return &ssm.GetParameterOutput{
		// 			Parameter: &types.Parameter{Value: &tsuccessParam},
		// 		}, nil
		// 	})
		// }, config.NewConfig(),
		// },
		// "errored": {"AWSPARAMSTR#/token/1", "|", "#", "unable to retrieve", func(t *testing.T) paramStoreApi {
		// 	return mockParamApi(func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
		// 		t.Helper()
		// 		awsParamtStoreCommonGetChecker(t, params)
		// 		return nil, fmt.Errorf("unable to retrieve")
		// 	})
		// }, config.NewConfig(),
		// },
		// "nil to empty": {"AWSPARAMSTR#/token/1", "|", "#", "", func(t *testing.T) paramStoreApi {
		// 	return mockParamApi(func(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
		// 		t.Helper()
		// 		awsParamtStoreCommonGetChecker(t, params)
		// 		return &ssm.GetParameterOutput{
		// 			Parameter: &types.Parameter{Value: nil},
		// 		}, nil
		// 	})
		// }, config.NewConfig(),
		// },
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			// token, _ := config.NewToken(tt.token, *tt.config.WithTokenSeparator(tt.tokenSeparator).WithKeySeparator(tt.keySeparator))

			impl, err := store.NewParamStore(context.TODO(), log.New(io.Discard))
			if err != nil {
				t.Errorf(testutils.TestPhrase, err.Error(), nil)
			}
			impl.WithSvc(tt.mockClient(t))
			impl.SetToken(tt.token())
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
