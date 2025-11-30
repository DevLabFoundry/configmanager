package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/store"
)

func TestPlugin_GetValue_integration(t *testing.T) {
	// as the plugin is technically a subprocess
	// setting env vars at this level will affect the loaded plugin
	os.Setenv("AWS_REGION", "eu-west-1")
	os.Setenv("AWS_PROFILE", "PROFILE_TO_USE")
	np, err := store.New(context.TODO(), "../../plugins/awsparamstr/bin/awsparamstr", config.ParamStorePrefix)
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
	if got == "" {
		t.Error("empty...")
	}
}
