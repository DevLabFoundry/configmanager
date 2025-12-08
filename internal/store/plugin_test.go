package store_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/store"
)

// TODO: make the implementation of the plugin system more testable
func TestPlugin_GetValue_integration(t *testing.T) {
	t.Skip()
	// as the plugin is technically a subprocess
	// setting env vars at this level will affect the loaded plugin
	os.Setenv("AWS_REGION", "eu-west-1")
	os.Setenv("AWS_PROFILE", "FOO")
	tp := fmt.Sprintf("../../.configmanager/plugins/awsparamstr/awsparamstr-%s-%s", runtime.GOOS, runtime.GOARCH)
	np, err := store.NewPlugin(context.TODO(), tp)
	if err != nil {
		t.Fatal(err)
	}

	defer np.ClientCleanUp()
	token, err := config.NewToken(config.ParamStorePrefix, *config.NewConfig())
	if err != nil {
		t.Fatal(err)
	}

	token.WithSanitizedToken("/int-test/pocketbase/admin-pwd")
	got, err := np.GetValue(token)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) < 1 {
		t.Error("empty...")
	}
}
