package token_test

import (
	"testing"

	"github.com/DevLabFoundry/configmanager/v3/internal/token"
)

func TestLookupIdent(t *testing.T) {
	ttests := map[string]struct {
		char   string
		expect token.TokenType
	}{
		"new line": {"\n", token.NEW_LINE},
		"dash":     {"-", token.TEXT},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			got := token.LookupIdent(tt.char)
			if got != tt.expect {
				t.Errorf("got %v wanted %v", got, tt.expect)
			}
		})
	}
}
