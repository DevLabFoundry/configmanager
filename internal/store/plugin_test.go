package store_test

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/store"
)

// Note: this step depends on pre-built empty tester plugin provider
// Running the
func Test_Plugin_GetValue(t *testing.T) {
	tp := fmt.Sprintf("../../tokenstore/provider/empty/bin/empty-%s-%s", runtime.GOOS, runtime.GOARCH)
	np, err := store.NewPlugin(context.TODO(), tp)
	if err != nil {
		t.Fatal(err)
	}

	defer np.ClientCleanUp()
	token, err := config.NewParsedToken(config.ParamStorePrefix, *config.NewConfig())
	if err != nil {
		t.Fatal(err)
	}

	token.WithSanitizedToken("/int-test/pocketbase/admin-pwd")
	got, err := np.GetValue(token)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) < 1 {
		t.Fatal("empty...")
	}
	if got != "/int-test/pocketbase/admin-pwd->" {
		t.Errorf("")
	}
}
