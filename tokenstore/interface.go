package tokenstore

import (
	"context"

	"google.golang.org/grpc"

	"github.com/DevLabFoundry/configmanager/v3/tokenstore/proto"
	"github.com/hashicorp/go-plugin"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	// This isn't required when using VersionedPlugins
	ProtocolVersion:  1,
	MagicCookieKey:   "CONFIGMANAGER_PLUGIN",
	MagicCookieValue: "configmanager-plugin-hello",
}

// // PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	"configmanager_token_store": &GRPCPlugin{},
}

// TokenStore is the interface that we're exposing as a plugin.
type TokenStore interface {
	Value(token string, metadata []byte) (string, error)
}

// This is the implementation of plugin.GRPCPlugin so we can serve/consume this.
type GRPCPlugin struct {
	// GRPCPlugin must still implement the Plugin interface
	plugin.Plugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl TokenStore
}

func (p *GRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterTokenStoreServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

func (p *GRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: proto.NewTokenStoreClient(c)}, nil
}
