package main

import (
	"github.com/DevLabFoundry/configmanager/v3/tokenstore"
	"github.com/hashicorp/go-plugin"
)

type TokenStorePlugin struct{}

func (ts TokenStorePlugin) Value(key string, metadata []byte) (string, error) {
	// srv, err := impl.NewParamStore(context.Background(), log.New(os.Stderr))
	// if err != nil {
	// 	return "", err
	// }
	// return srv.Value(key, metadata)
	return "", nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: tokenstore.Handshake,
		Plugins: map[string]plugin.Plugin{
			"configmanager_token_store": &tokenstore.GRPCPlugin{Impl: &TokenStorePlugin{}},
		},
		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
