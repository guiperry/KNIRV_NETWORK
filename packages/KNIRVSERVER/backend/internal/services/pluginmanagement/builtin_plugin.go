package pluginmanagement

import "context"

// BuiltinPlugin is implemented by first-party Go services that appear in the plugin registry.
type BuiltinPlugin interface {
	PluginID()    string
	PluginName()  string
	Category()    string
	Description() string
	Version()     string
	IsEnabled()   bool
	IsRunning()   bool
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// ConfigurablePlugin is optionally implemented by BuiltinPlugins that support runtime config patching.
type ConfigurablePlugin interface {
	UpdateRawConfig(patch map[string]interface{}) error
}

// BuiltinPluginInfo is the serializable view returned by the API.
type BuiltinPluginInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Enabled     bool   `json:"enabled"`
	Running     bool   `json:"running"`
}
