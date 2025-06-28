package plugins

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
		return "/v1/plugins/" + pluginName
	case ScopeDomain:
		return "/v1/domains/" + resourceName + "/plugins/" + pluginName
	case ScopeService:
		return "/v1/services/" + resourceName + "/plugins/" + pluginName
	case ScopeRoute:
		return "/v1/routes/" + resourceName + "/plugins/" + pluginName
	default:
		return "/v1/plugins/" + pluginName
	}
}
