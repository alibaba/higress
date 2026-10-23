package plugins

// PluginTargets represents the targets for different scopes
type PluginTargets struct {
	Domain  string `json:"DOMAIN,omitempty"`
	Service string `json:"SERVICE,omitempty"`
	Route   string `json:"ROUTE,omitempty"`
}

// PluginInstance represents a plugin instance configuration
type PluginInstance[T any] struct {
	Version           string        `json:"version,omitempty"`
	Scope             string        `json:"scope"`
	Target            string        `json:"target,omitempty"`
	Targets           PluginTargets `json:"targets,omitempty"`
	PluginName        string        `json:"pluginName,omitempty"`
	PluginVersion     string        `json:"pluginVersion,omitempty"`
	Internal          bool          `json:"internal,omitempty"`
	Enabled           bool          `json:"enabled"`
	RawConfigurations string        `json:"rawConfigurations,omitempty"`
	Configurations    T             `json:"configurations,omitempty"`
}
