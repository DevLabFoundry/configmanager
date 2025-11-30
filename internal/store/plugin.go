package store

import (
	"context"
	"os/exec"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/plugins"
	"github.com/hashicorp/go-plugin"
)

// Plugin is responsible for managing plugins within configmanager
//
// It includes the following methods
//   - fetch plugins from known sources
//   - maintains a list of tokens answerable by a specified pluginEngine
type Plugin struct {
	Implementations config.ImplementationPrefix
	SourcePath      string
	Version         string
	ClientCleanUp   func()
	tokenStore      plugins.TokenStore
}

// New Plugin gets called once per implementation
func New(ctx context.Context, path string, prefix config.ImplementationPrefix) (*Plugin, error) {
	// We're a host. Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  plugins.Handshake,
		Plugins:          plugin.PluginSet{"configmanager_token_store": &plugins.TokenStoreGRPCPlugin{}},
		Cmd:              exec.Command(path),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
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

	ts := raw.(plugins.TokenStore)

	p := &Plugin{
		ClientCleanUp: client.Kill,
		tokenStore:    ts,
	}
	return p, nil
}

func (p *Plugin) GetValue(token *config.ParsedTokenConfig) (string, error) {
	result, err := p.tokenStore.Value(token.StoreToken(), []byte(token.Metadata()))
	if err != nil {
		return "", err
	}
	return result, nil
}
