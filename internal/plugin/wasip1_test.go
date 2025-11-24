package plugin_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/plugin"
)

func Test_FullFlow(t *testing.T) {
	inputReader, err := os.Open("/Users/dusannitschneider/git/dnitsch/configmanager/plugins/awsparams/awsparams.wasm")
	if err != nil {
		t.Fatal(fmt.Errorf("open plugin.wasm: %w", err))
	}
	ctx := context.Background()

	// Load the compiled WASI plugin.
	engine, err := plugin.NewEngine(ctx, inputReader)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close(ctx)

	inst, err := engine.NewApiInstance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close(ctx)

	os.Setenv("AWS_PROFILE", "anabode_terraform_dev")
	os.Setenv("AWS_REGION", "eu-west-1")
	t1, _ := config.NewToken(config.ParamStorePrefix, *config.NewConfig())
	t1.WithSanitizedToken("/int-test/pocketbase/admin-pwd")
	val, err := inst.TokenValue(ctx, t1)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("TokenValue(\"foo\") => %q\n", string(val))

	// Zero-length test (should error)
	t2, _ := config.NewToken(config.ParamStorePrefix, *config.NewConfig())
	_, err = inst.TokenValue(ctx, t2)
	fmt.Printf("TokenValue(\"\") error: %v\n", err)
}
