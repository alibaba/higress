package higress_ops

// APIResponse represents the standard Higress Console API response format
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

// Route represents a Higress route configuration
type Route struct {
	Name          string            `json:"name"`
	Domains       []string          `json:"domains,omitempty"`
	Path          *PathConfig       `json:"path"`
	Methods       []string          `json:"methods,omitempty"`
	Headers       []HeaderConfig    `json:"headers,omitempty"`
	URLParams     []URLParamConfig  `json:"urlParams,omitempty"`
	Services      []ServiceConfig   `json:"services"`
	CustomConfigs map[string]string `json:"customConfigs,omitempty"`
}

// PathConfig represents path matching configuration
type PathConfig struct {
	MatchType     string `json:"matchType"` // PRE, EQUAL, REGULAR
	MatchValue    string `json:"matchValue"`
	CaseSensitive *bool  `json:"caseSensitive,omitempty"`
}

// HeaderConfig represents header matching configuration
type HeaderConfig struct {
	Key           string `json:"key"`
	MatchType     string `json:"matchType"` // PRE, EQUAL, REGULAR
	MatchValue    string `json:"matchValue"`
	CaseSensitive *bool  `json:"caseSensitive,omitempty"`
}

// URLParamConfig represents URL parameter matching configuration
type URLParamConfig struct {
	Key           string `json:"key"`
	MatchType     string `json:"matchType"` // PRE, EQUAL, REGULAR
	MatchValue    string `json:"matchValue"`
	CaseSensitive *bool  `json:"caseSensitive,omitempty"`
}

// ServiceConfig represents service configuration in a route
type ServiceConfig struct {
	Name   string `json:"name"`
	Port   int    `json:"port"`
	Weight int    `json:"weight"`
}

// ServiceSource represents a Higress service source
type ServiceSource struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // static, dns
	Domain   string `json:"domain"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol,omitempty"` // http, https
	SNI      string `json:"sni,omitempty"`
}

// PluginInstance represents a plugin instance configuration
type PluginInstance struct {
	Enabled        bool                   `json:"enabled"`
	Configurations map[string]interface{} `json:"configurations"`
}

// RequestBlockConfig represents the request-block plugin configuration
type RequestBlockConfig struct {
	BlockBodies   []string `json:"block_bodies,omitempty"`
	BlockHeaders  []string `json:"block_headers,omitempty"`
	BlockURLs     []string `json:"block_urls,omitempty"`
	BlockedCode   int      `json:"blocked_code,omitempty"`
	CaseSensitive bool     `json:"case_sensitive,omitempty"`
}
