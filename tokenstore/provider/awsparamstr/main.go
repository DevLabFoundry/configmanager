package main

import (
	"context"
	"os"

	"github.com/DevLabFoundry/configmanager-plugin/awsparamstr/impl"
	"github.com/DevLabFoundry/configmanager/v3/tokenstore"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

type implIface interface {
	Value(token string, metadata []byte) (string, error)
}
type TokenStorePlugin struct {
	impl implIface // Value(token string, metadata []byte) (string, error)
}

func (ts TokenStorePlugin) Value(key string, metadata []byte) (string, error) {
	return ts.impl.Value(key, metadata)
}

func main() {
	log := hclog.New(hclog.DefaultOptions)
	log.SetLevel(hclog.LevelFromString("error"))

	i, err := impl.NewParamStore(context.Background(), log)
	if err != nil {
		log.Error("error", err)
		os.Exit(1)
	}

	ts := TokenStorePlugin{impl: i}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: tokenstore.Handshake,
		Plugins: map[string]plugin.Plugin{
			"configmanager_token_store": &tokenstore.GRPCPlugin{Impl: ts},
		},
		VersionedPlugins: map[int]plugin.PluginSet{
			1: {
				"configmanager_token_store": &tokenstore.GRPCPlugin{Impl: ts},
			},
		},
		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
