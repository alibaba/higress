// Configuration management for nginx migration MCP server - Standalone Mode
package standalone

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ServerConfig holds all configurable values
type ServerConfig struct {
	Server   ServerSettings  `json:"server"`
	Gateway  GatewaySettings `json:"gateway"`
	Service  ServiceSettings `json:"service"`
	Defaults DefaultSettings `json:"defaults"`
}

type ServerSettings struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Port       string `json:"port"`
	APIBaseURL string `json:"api_base_url"`
}

type GatewaySettings struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ServiceSettings struct {
	DefaultName   string `json:"default_name"`
	DefaultPort   int    `json:"default_port"`
	DefaultTarget int    `json:"default_target_port"`
}

type DefaultSettings struct {
	Hostname    string `json:"hostname"`
	Namespace   string `json:"namespace"`
	PathPrefix  string `json:"path_prefix"`
	RoutePrefix string `json:"route_prefix"`
}

// LoadConfig loads configuration from environment variables and files
func LoadConfig() *ServerConfig {
	config := &ServerConfig{
		Server: ServerSettings{
			Name:       getEnvOrDefault("NGINX_MCP_SERVER_NAME", "nginx-migration-mcp"),
			Version:    getEnvOrDefault("NGINX_MCP_VERSION", "1.0.0"),
			Port:       getEnvOrDefault("NGINX_MCP_PORT", "8080"),
			APIBaseURL: getEnvOrDefault("NGINX_MIGRATION_API_URL", "http://localhost:8080"),
		},
		Gateway: GatewaySettings{
			Name:      getEnvOrDefault("HIGRESS_GATEWAY_NAME", "higress-gateway"),
			Namespace: getEnvOrDefault("HIGRESS_GATEWAY_NAMESPACE", "higress-system"),
		},
		Service: ServiceSettings{
			DefaultName:   getEnvOrDefault("DEFAULT_SERVICE_NAME", "backend-service"),
			DefaultPort:   getIntEnvOrDefault("DEFAULT_SERVICE_PORT", 80),
			DefaultTarget: getIntEnvOrDefault("DEFAULT_TARGET_PORT", 8080),
		},
		Defaults: DefaultSettings{
			Hostname:    getEnvOrDefault("DEFAULT_HOSTNAME", "example.com"),
			Namespace:   getEnvOrDefault("DEFAULT_NAMESPACE", "default"),
			PathPrefix:  getEnvOrDefault("DEFAULT_PATH_PREFIX", "/"),
			RoutePrefix: getEnvOrDefault("ROUTE_NAME_PREFIX", "nginx-migrated"),
		},
	}

	// Try to load from config file if exists
	if configFile := os.Getenv("NGINX_MCP_CONFIG_FILE"); configFile != "" {
		if err := loadConfigFromFile(config, configFile); err != nil {
			fmt.Printf("Warning: Failed to load config from %s: %v\n", configFile, err)
		}
	}

	return config
}

// loadConfigFromFile loads configuration from JSON file
func loadConfigFromFile(config *ServerConfig, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, config)
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnvOrDefault returns environment variable as int or default
func getIntEnvOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GenerateRouteName generates a unique route name
func (c *ServerConfig) GenerateRouteName(hostname string) string {
	if hostname == "" || hostname == c.Defaults.Hostname {
		return fmt.Sprintf("%s-route", c.Defaults.RoutePrefix)
	}
	// Replace dots and special characters for valid k8s name
	safeName := hostname
	for _, char := range []string{".", "_", ":"} {
		safeName = strings.ReplaceAll(safeName, char, "-")
	}
	return fmt.Sprintf("%s-%s", c.Defaults.RoutePrefix, safeName)
}

// GenerateIngressName generates a unique ingress name
func (c *ServerConfig) GenerateIngressName(hostname string) string {
	if hostname == "" || hostname == c.Defaults.Hostname {
		return fmt.Sprintf("%s-ingress", c.Defaults.RoutePrefix)
	}
	// Replace dots and special characters for valid k8s name
	safeName := hostname
	for _, char := range []string{".", "_", ":"} {
		safeName = strings.ReplaceAll(safeName, char, "-")
	}
	return fmt.Sprintf("%s-%s", c.Defaults.RoutePrefix, safeName)
}

// GenerateServiceName generates service name based on hostname
func (c *ServerConfig) GenerateServiceName(hostname string) string {
	if hostname == "" || hostname == c.Defaults.Hostname {
		return c.Service.DefaultName
	}
	return fmt.Sprintf("%s-service", hostname)
}
