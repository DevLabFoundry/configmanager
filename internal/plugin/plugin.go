package plugin

import (
	"net/rpc"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
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
	fallbackPaths   []string
	engineInstance  *Engine
}

// ValueProvider is the interface that we're exposing as a plugin.
type ValueProvider interface {
	Value(token string, metadata string) (string, error)
}

// Here is an implementation that talks over RPC
type StorePluginRPC struct{ client *rpc.Client }

func (g *StorePluginRPC) Greet() string {
	var resp string
	err := g.client.Call("Plugin.Greet", new(interface{}), &resp)
	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		panic(err)
	}

	return resp
}

// Here is the RPC server that GreeterRPC talks to, conforming to
// the requirements of net/rpc
type GreeterRPCServer struct {
	// This is the real implementation
	Impl ValueProvider
}

func (s *GreeterRPCServer) Greet(args interface{}, resp *string) error {
	*resp = s.Impl.Value()
	return nil
}

// This is the implementation of plugin.Plugin so we can serve/consume this
//
// This has two methods: Server must return an RPC server for this plugin
// type. We construct a GreeterRPCServer for this.
//
// Client must return an implementation of our interface that communicates
// over an RPC client. We return GreeterRPC for this.
//
// Ignore MuxBroker. That is used to create more multiplexed streams on our
// plugin connection and is a more advanced use case.
type GreeterPlugin struct {
	// Impl Injection
	Impl ValueProvider
}

func (p *GreeterPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &GreeterRPCServer{Impl: p.Impl}, nil
}

func (GreeterPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &StorePluginRPC{client: c}, nil
}
