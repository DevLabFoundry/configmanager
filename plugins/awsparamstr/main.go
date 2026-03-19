package main

import (
	"context"

	"github.com/DevLabFoundry/configmanager/plugins/awsparamstr/impl"
	"github.com/DevLabFoundry/configmanager/v3/plugins"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

type TokenStorePlugin struct {
	log hclog.Logger
}

func (ts TokenStorePlugin) Value(key string, metadata []byte) (string, error) {
	srv, err := impl.NewParamStore(context.Background(), ts.log)
	if err != nil {
		return "", err
	}
	return srv.Value(key, metadata)
}

func main() {
	log := hclog.New(hclog.DefaultOptions)
	log.SetLevel(hclog.LevelFromString("error"))

	// if os.Getenv("CONFIGMANAGER_LOG")
	ts := TokenStorePlugin{log: log}
	plugin.Serve(&plugin.ServeConfig{
		// Logger: ,
		HandshakeConfig: plugins.Handshake,
		Plugins: map[string]plugin.Plugin{
			"configmanager_token_store": &plugins.TokenStoreGRPCPlugin{Impl: ts},
		},
		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
