package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/DevLabFoundry/configmanager/v3/config"
	"github.com/schollz/progressbar/v3"
)

var (
	ErrRetrieveFailed       = errors.New("failed to retrieve config item")
	ErrClientInitialization = errors.New("failed to initialize the client")
	ErrEmptyResponse        = errors.New("value retrieved but empty for token")
	ErrServiceCallFailed    = errors.New("failed to complete the service call")
	ErrPluginIssue          = errors.New("plugin init failed")
	ErrMkdirAllFail         = errors.New("unable to create the required directory")
)

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
	pluginsLocation string = "." + config.SELF_NAME + "/plugins"
	namePattern     string = "%s-%s-%s"
)

type osOps struct {
	UserHomeDir func() (string, error)
	Getwd       func() (dir string, err error)
	MkdirAll    func(path string, perm os.FileMode) error
	Create      func(name string) (io.WriteCloser, error)
}

type downloadClient interface {
	Do(*http.Request) (*http.Response, error)
}
type Store struct {
	plugin          pluginMap
	osOps           osOps
	downloadInfoMap PluginDownloadInfoMap
	downloadClient  downloadClient
}

type StoreOpts func(s *Store)

func New(ctx context.Context, opts ...StoreOpts) *Store {
	pm := pluginMap{
		mu: &sync.Mutex{},
		m:  make(map[string]*Plugin),
	}
	os.Getwd()
	s := &Store{
		plugin: pm,
		osOps: osOps{
			UserHomeDir: os.UserHomeDir,
			Getwd:       os.Getwd,
			MkdirAll:    os.MkdirAll,
			Create: func(name string) (io.WriteCloser, error) {
				return os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0700)
			},
		},
		downloadInfoMap: corePluginMap,
		downloadClient:  &http.Client{},
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

func WithOsOps(v osOps) StoreOpts {
	return func(s *Store) {
		s.osOps = v
	}
}

func WithDownloadClient(v downloadClient) StoreOpts {
	return func(s *Store) {
		s.downloadClient = v
	}
}

func WithAdditionalPluginInfo(v PluginDownloadInfoMap) StoreOpts {
	return func(s *Store) {
		maps.Copy(s.downloadInfoMap, v)
	}
}

// Init ensures all the discovered tokens have their implementations initialised
//
// NOTE: it is important to package the providers at a build stage
// if your target deployment environment does not support open network outbound connections
func (s *Store) Init(ctx context.Context, implt []string) error {
	dir, err := s.configManagerDir()
	if err != nil {
		return err
	}

	for _, plugin := range implt {
		// we first look for existing plugins
		plpath, err := s.findPlugin(dir, plugin)
		if err != nil {
			// try to retrieve from remote source
			return fmt.Errorf("configmanager provider: ( %s ) %w\n%v", plugin, ErrPluginIssue, err)
		}
		// Initialising the plugins will ensure the client (configmanager-core) and the server (token-store-plugin-provider) are at expected versions of each other
		p, err := NewPlugin(ctx, plpath)
		if err != nil {
			// wrap in init error
			return err
		}
		s.plugin.Add(plugin, p)
	}
	return nil
}

func (s *Store) GetValue(implemenation *config.ParsedTokenConfig) (string, error) {
	plugin, exists := s.plugin.m[strings.ToLower(string(implemenation.Prefix()))]
	if !exists {
		return "", ErrPluginIssue
	}
	return plugin.GetValue(implemenation)
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
func (s *Store) findPlugin(pluginDir, plugin string) (string, error) {
	fullPluginPath := path.Join(pluginDir, plugin, fmt.Sprintf(namePattern, plugin, runtime.GOOS, runtime.GOARCH))
	if _, err := os.Stat(fullPluginPath); err == nil {
		// plugin exists let's use it
		return fullPluginPath, nil
	}

	// Plugin is not found - will attempt to download and write them to the first specified directory
	return s.downloadPlugin(fullPluginPath, plugin)
}

// downloadPlugin deals with a specific plugin and only gets the current architecture
//
// it will create the required directory
func (s *Store) downloadPlugin(pluginFullPath, plugin string) (string, error) {

	releaseName := filepath.Base(pluginFullPath)
	// This is opinionated and all windows plugins must be stored with the `.exe` extension
	if runtime.GOOS == "windows" {
		releaseName = releaseName + ".exe"
		pluginFullPath = pluginFullPath + ".exe"
	}

	// as an example
	// https://github.com/DevLabFoundry/configmanager/releases/download/v3.0.0/awsparamstr-linux-amd64
	if err := s.osOps.MkdirAll(filepath.Dir(pluginFullPath), 0777); err != nil {
		return "", fmt.Errorf("%w, %v", ErrMkdirAllFail, err)
	}

	w, err := s.osOps.Create(pluginFullPath)
	if err != nil {
		return "", err
	}
	defer w.Close()

	specific := "download/%s"
	// latest := "latest/download"
	// TODO: need to think about providing these in a more configurable way
	//
	releasePath := path.Join(fmt.Sprintf(specific, "v3.0.0"), releaseName)

	pluginInfo, found := s.downloadInfoMap[plugin]
	if !found {
		return "", fmt.Errorf("download info not found for plugin ( %s )", plugin)
	}

	link, err := url.Parse(fmt.Sprintf("%s/%s", pluginInfo.BaseUrl, releasePath))
	if err != nil {
		return "", err
	}

	req := &http.Request{
		URL:    link,
		Method: http.MethodGet,
	}
	resp, err := s.downloadClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"downloading", releaseName,
	)

	if _, err = io.Copy(io.MultiWriter(w, bar), resp.Body); err != nil {
		return "", err
	}

	return pluginFullPath, nil

}

// configManagerDir ensures the directory exists and returns the correct one based on config
//
// It will return the full path to /some/dir/.configmanager and and ensures it exists
func (s *Store) configManagerDir() (string, error) {
	var err error

	// if env var provided - it takes precendence over current directory
	val, exists := os.LookupEnv(config.CONFIGMANAGER_DIR)
	if !exists {
		// otherwise we default to the current directory - i.e. from which the command is being run
		val, err = s.osOps.Getwd()
		if err != nil {
			return "", err
		}
	}

	initialConfigPath := filepath.Join(val, pluginsLocation)
	if err := s.osOps.MkdirAll(initialConfigPath, 0777); err != nil {
		return "", err
	}
	return initialConfigPath, nil
}
