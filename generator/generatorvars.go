package generator

import (
	"sync"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
)

// ParsedMap is the internal working object definition and
// the return type if results are not flushed to file
type ParsedMap map[string]any

func (pm ParsedMap) MapKeys() (keys []string) {
	for k := range pm {
		keys = append(keys, k)
	}
	return
}

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

type tokenMapSafe struct {
	mu       *sync.Mutex
	tokenMap ParsedMap
}

func (tms *tokenMapSafe) getTokenMap() ParsedMap {
	tms.mu.Lock()
	defer tms.mu.Unlock()
	return tms.tokenMap
}

func (tms *tokenMapSafe) addKeyVal(key *config.ParsedTokenConfig, val string) {
	tms.mu.Lock()
	defer tms.mu.Unlock()
	// NOTE: still use the metadata in the key
	// there could be different versions / labels for the same token and hence different values
	// However the JSONpath look up
	tms.tokenMap[key.String()] = keySeparatorLookup(key, val)
}
