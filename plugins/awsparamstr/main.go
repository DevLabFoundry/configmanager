package main

import (
	"context"
	"os"

	"github.com/DevLabFoundry/configmanager/v3/internal/log"
	"github.com/DevLabFoundry/configmanager/v3/plugins"
	"github.com/DevLabFoundry/configmanager/v3/plugins/awsparamstr/impl"
	"github.com/hashicorp/go-plugin"
)

// Here is a real implementation of KV that writes to a local file with
// the key name and the contents are the value of the key.
type TokenStorePlugin struct{}

func (ts TokenStorePlugin) Value(key string, metadata []byte) (string, error) {
	srv, err := impl.NewParamStore(context.Background(), log.New(os.Stderr))
	if err != nil {
		return "", err
	}
	return srv.Value(key, metadata)
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugins.Handshake,
		Plugins: map[string]plugin.Plugin{
			"configmanager_token_store": &plugins.TokenStoreGRPCPlugin{Impl: &TokenStorePlugin{}},
		},
		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
