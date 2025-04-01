package store_test

import (
	"testing"

	"github.com/DevLabFoundry/configmanager/internal/store"
)

func Test_StoreDefault(t *testing.T) {

	t.Run("Default Shoudl not errror", func(t *testing.T) {
		rs := store.NewDefatultStrategy()
		if rs == nil {
			t.Fatal("unable to init default strategy")
		}
	})
	t.Run("Token method should error", func(t *testing.T) {
		rs := store.NewDefatultStrategy()
		if _, err := rs.Token(); err == nil {
			t.Fatal("Token should return not implemented error")
		}
	})

}
