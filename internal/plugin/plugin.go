package plugin

import "github.com/DevLabFoundry/configmanager/v3/internal/config"

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
