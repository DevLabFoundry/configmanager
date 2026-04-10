// Package main of empty implementation is used for "unit" testing
//
// The TokenStore Value implementation returns the key and metadata passed
// in the case of key being `err` a simulated error is returned
package main

import (
	"errors"
	"fmt"

	"github.com/DevLabFoundry/configmanager/v3/tokenstore"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// TokenStorePlugin here is a sample plugin we can use in tests
// It handles some basic error scenarios
type TokenStorePlugin struct{}

func (ts TokenStorePlugin) Value(key string, metadata []byte) (string, error) {
	if key == "err" {
		return "", errors.New("token store implementation simulated error")
	}
	return fmt.Sprintf("%s->%s", key, metadata), nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		Logger:          hclog.NewNullLogger(),
		HandshakeConfig: tokenstore.Handshake,
		Plugins: map[string]plugin.Plugin{
			"configmanager_token_store": &tokenstore.GRPCPlugin{Impl: &TokenStorePlugin{}},
		},
		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
