package store

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/DevLabFoundry/configmanager/v3/config"
	"github.com/DevLabFoundry/configmanager/v3/tokenstore"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

var ErrTokenRetrieval = errors.New("failed to exchange token for value")

// Plugin is responsible for managing the plugin lifecycle
// within the configmanager flow. Each Implementation will initialise exactly one instance of the plugin
type Plugin struct {
	Implementations config.ImplementationPrefix
	SourcePath      string
	Version         string
	ClientCleanUp   func()
	tokenStore      tokenstore.TokenStore
}

// NewPlugin Plugin gets called once per implementation
func NewPlugin(ctx context.Context, path string) (*Plugin, error) {
	// We're a host. Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  tokenstore.Handshake,
		Plugins:          plugin.PluginSet{"configmanager_token_store": &tokenstore.GRPCPlugin{}},
		Cmd:              exec.Command(path),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           hclog.NewNullLogger(),
	})
	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, err
	}

	// ensure the loaded plugin can dispense the required prefix implementation
	raw, err := rpcClient.Dispense("configmanager_token_store")
	if err != nil {
		client.Kill()
		return nil, err
	}

	ts := raw.(tokenstore.TokenStore)

	p := &Plugin{
		ClientCleanUp: client.Kill,
		tokenStore:    ts,
	}
	return p, nil
}

func (p *Plugin) WithTokenStore(ts tokenstore.TokenStore) {
	p.tokenStore = ts
}

func (p *Plugin) GetValue(token *config.ParsedTokenConfig) (string, error) {
	result, err := p.tokenStore.Value(token.StoreToken(), []byte(token.Metadata()))
	if err != nil {
		return "", fmt.Errorf("%w - (%s), %v", ErrRetrieveFailed, token.String(), err)
	}
	return result, nil
}

type PluginDownloadInfo struct {
	BaseUrl string
	Name    string
}

const corePluginBaseUrl = "https://github.com/DevLabFoundry/configmanager/releases"

type PluginDownloadInfoMap map[string]*PluginDownloadInfo

// corePluginMap are the configmanager maintained plugins
var corePluginMap PluginDownloadInfoMap = map[string]*PluginDownloadInfo{
	"empty": {
		BaseUrl: corePluginBaseUrl,
		Name:    "",
	},
	"awsparamstr": {
		BaseUrl: corePluginBaseUrl,
		Name:    "",
	},
	"awssecrets": {
		BaseUrl: corePluginBaseUrl,
		Name:    "",
	},
	// ...
}
