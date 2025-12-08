package generator

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/spyzhov/ajson"
)

// ReplacedToken is the internal working object definition and
// the return type if results are not flushed to file
type ReplacedToken map[string]any

func (pm ReplacedToken) MapKeys() (keys []string) {
	for k := range pm {
		keys = append(keys, k)
	}
	return
}

// RawTokenConfig represents the map of
// discovered tokens via the lexer/parser
type RawTokenConfig struct {
	mu       *sync.Mutex
	tokenMap map[string]*config.ParsedTokenConfig
}

func NewRawTokenConfig() *RawTokenConfig {
	return &RawTokenConfig{mu: &sync.Mutex{}, tokenMap: map[string]*config.ParsedTokenConfig{}}
}

func (rtm *RawTokenConfig) AddToken(name string, parsedToken *config.ParsedTokenConfig) {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()
	rtm.tokenMap[name] = parsedToken
}

func (rtm *RawTokenConfig) RawTokenMap() map[string]*config.ParsedTokenConfig {
	rtm.mu.Lock()
	defer rtm.mu.Unlock()
	return rtm.tokenMap
}

type TokenResponse struct {
	val string
	key *config.ParsedTokenConfig
	Err error
}

func (tr *TokenResponse) WithKey(key *config.ParsedTokenConfig) {
	tr.key = key
}

func (tr *TokenResponse) WithValue(val string) {
	tr.val = val
}

func (tr *TokenResponse) Key() *config.ParsedTokenConfig {
	return tr.key
}

func (tr *TokenResponse) Value() string {
	return tr.val
}

// keySeparatorLookup checks if the key contains
// keySeparator character
// If it does contain one then it tries to parse
func keySeparatorLookup(token *config.ParsedTokenConfig, val string) string {
	k := token.LookupKeys()
	if k == "" {
		return val
	}

	keys, err := ajson.JSONPath([]byte(val), fmt.Sprintf("$..%s", k))
	if err != nil {
		return val
	}

	if len(keys) == 1 {
		v := keys[0]
		if v.Type() == ajson.String {
			str, err := strconv.Unquote(fmt.Sprintf("%v", v))
			if err != nil {
				return fmt.Sprintf("%v", v)
			}
			return str
		}

		return fmt.Sprintf("%v", v)
	}

	return ""
}
