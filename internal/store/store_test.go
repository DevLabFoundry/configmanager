package store_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/config"
	"github.com/DevLabFoundry/configmanager/v3/internal/store"
)

// These tests are more of an integration test as they rely on
func Test_Store(t *testing.T) {

	// Setup test store
	os.Setenv(config.CONFIGMANAGER_DIR, "../../tokenstore/provider/empty")

	defer os.Unsetenv(config.CONFIGMANAGER_DIR)

	s := store.New(context.TODO())

	if err := s.Init(context.TODO(), []string{"empty"}); err != nil {
		t.Fatal(err)
	}
	token, err := config.NewParsedToken("empty", *config.NewConfig())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("success no metadata", func(t *testing.T) {
		token.WithSanitizedToken("/my/token")
		got, err := s.GetValue(token)
		assertStoreResp(t, got, "/my/token->", err, nil)
	})

	t.Run("succeds with metadata", func(t *testing.T) {
		token.WithSanitizedToken("/my/token")
		token.WithMetadata(`[version=123]`)
		got, err := s.GetValue(token)
		assertStoreResp(t, got, "/my/token->[version=123]", err, nil)

	})

	t.Run("errors on retrieve", func(t *testing.T) {
		token.WithSanitizedToken("err")
		got, err := s.GetValue(token)
		assertStoreResp(t, got, "", err, store.ErrRetrieveFailed)
	})
}

func assertStoreResp(t *testing.T, got, want string, err, wantErr error) {
	t.Helper()

	if err != nil && wantErr == nil {
		t.Fatal(err)
	}

	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Errorf("errors don't match, got %v, wanted %v", err, wantErr)
		}
	}

	if got != want {
		t.Errorf("got %s, wanted %s", got, want)
	}

}
