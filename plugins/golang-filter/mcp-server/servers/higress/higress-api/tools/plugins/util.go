package plugins

import "fmt"

// Plugin scope constants
const (
	ScopeGlobal  = "global"
	ScopeDomain  = "domain"
	ScopeService = "service"
	ScopeRoute   = "route"
)

// ValidScopes contains all valid plugin scopes
var ValidScopes = []string{ScopeGlobal, ScopeDomain, ScopeService, ScopeRoute}

// IsValidScope checks if the given scope is valid
func IsValidScope(scope string) bool {
	for _, validScope := range ValidScopes {
		if scope == validScope {
			return true
		}
	}
	return false
}

// BuildPluginPath builds the API path for plugin operations based on scope and resource
func BuildPluginPath(pluginName, scope, resourceName string) string {
	switch scope {
	case ScopeGlobal:
		return fmt.Sprintf("/v1/global/plugin-instances/%s", pluginName)
	case ScopeDomain:
		return fmt.Sprintf("/v1/domains/%s/plugin-instances/%s", resourceName, pluginName)
	case ScopeService:
		return fmt.Sprintf("/v1/services/%s/plugin-instances/%s", resourceName, pluginName)
	case ScopeRoute:
		return fmt.Sprintf("/v1/routes/%s/plugin-instances/%s", resourceName, pluginName)
	default:
		return fmt.Sprintf("/v1/global/plugin-instances/%s", pluginName)
	}
}
