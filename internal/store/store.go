package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
)

var (
	ErrRetrieveFailed       = errors.New("failed to retrieve config item")
	ErrClientInitialization = errors.New("failed to initialize the client")
	ErrEmptyResponse        = errors.New("value retrieved but empty for token")
	ErrServiceCallFailed    = errors.New("failed to complete the service call")
	ErrPluginNotFound       = errors.New("plugin does not exist")
)

// Strategy iface that all store implementations
// must conform to, in order to be be used by the retrieval implementation
//
// Defined on the package for easier re-use across the program
type Strategy interface {
	// Value retrieves the underlying value for the token
	Value() (s string, e error)
	// SetToken
	SetToken(s *config.ParsedTokenConfig)
}

//
// It includes the following methods
//   - fetch plugins from known sources
//   - maintains a list of tokens answerable by a specified pluginEngine

type pluginMap struct {
	mu *sync.Mutex
	// m holds the map of plugins where the key is the lowercased implementation prefix
	// 	e.g. `AWSPARAMSTR://` => `awsparamstr`
	m map[string]*Plugin
}

func (p pluginMap) Add(key string, pl *Plugin) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.m[key] = pl
}

const (
	loc         string = ".configmanager/plugins"
	namePattern string = "%s-%s-%s"
)

type Store struct {
	pluginLocation []string
	plugin         pluginMap
	// PluginCleanUp  func()
}

func Init(ctx context.Context, implt []string) (*Store, error) {
	pm := pluginMap{mu: &sync.Mutex{}, m: make(map[string]*Plugin)}

	// l := []string{""}
	//
	for _, plugin := range implt {
		plpath, err := findPlugin(plugin)
		if err != nil {
			return nil, err
		}
		p, err := NewPlugin(ctx, plpath)
		pm.Add(plugin, p)
	}
	return &Store{plugin: pm}, nil
}

func (s *Store) GetImplementation(implemenation config.ImplementationPrefix) (plugin *Plugin, err error) {
	var exists bool
	if plugin, exists = s.plugin.m[strings.ToLower(string(implemenation))]; exists {
		return plugin, nil
	}
	return nil, ErrPluginNotFound
}

// PluginCleanUp ensures the plugins are properly shut down
func (s *Store) PluginCleanUp() {
	s.plugin.mu.Lock()
	defer s.plugin.mu.Unlock()
	for _, plugin := range s.plugin.m {
		plugin.ClientCleanUp()
	}
}

// findPlugin ensures the path exists and search the following locations
//
//	current dir
//	home dir
func findPlugin(plugin string) (string, error) {
	// fallback locations
	// current dir
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	hd, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	for _, p := range []string{cwd, hd} {
		ff := path.Join(p, loc, plugin, fmt.Sprintf(namePattern, plugin, runtime.GOOS, runtime.GOARCH))
		if _, err := os.Stat(ff); err == nil {
			// break on first non nil error
			return ff, nil
		}
	}
	return "", ErrPluginNotFound
}
