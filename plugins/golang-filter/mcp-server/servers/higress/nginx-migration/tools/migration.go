// Package tools provides nginx configuration migration tools for Higress
// Added on 2025.9.29 - Core migration tools implementation
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// NginxConfig represents a parsed Nginx configuration
type NginxConfig struct {
	ServerBlocks []ServerBlock `json:"server_blocks"`
}

// ServerBlock represents an Nginx server block
type ServerBlock struct {
	Listen     []string          `json:"listen"`
	ServerName []string          `json:"server_name"`
	Location   []LocationBlock   `json:"location"`
	Root       string            `json:"root,omitempty"`
	Index      string            `json:"index,omitempty"`
	ProxyPass  string            `json:"proxy_pass,omitempty"`
	Rewrite    []RewriteRule     `json:"rewrite,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// LocationBlock represents an Nginx location block
type LocationBlock struct {
	Path      string            `json:"path"`
	ProxyPass string            `json:"proxy_pass,omitempty"`
	Rewrite   []RewriteRule     `json:"rewrite,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Root      string            `json:"root,omitempty"`
	Index     string            `json:"index,omitempty"`
}

// RewriteRule represents an Nginx rewrite rule
type RewriteRule struct {
	Pattern string `json:"pattern"`
	Replace string `json:"replace"`
	Flag    string `json:"flag,omitempty"`
}

// HigressRoute represents a Higress route configuration
type HigressRoute struct {
	Name        string            `json:"name"`
	Host        string            `json:"host"`
	Path        string            `json:"path"`
	Service     string            `json:"service"`
	Port        int               `json:"port"`
	Rewrite     *RewriteConfig    `json:"rewrite,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// RewriteConfig represents Higress rewrite configuration
type RewriteConfig struct {
	Path string `json:"path"`
}

// RegisterMigrationTools registers all nginx migration tools
func RegisterMigrationTools(mcpServer *common.MCPServer, client *higress.HigressClient) {
	// Parse Nginx configuration tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("parse_nginx_config", "Parse Nginx configuration file and extract server blocks, locations, and routing rules", getParseNginxConfigSchema()),
		handleParseNginxConfig(),
	)

	// Convert to Higress format tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("convert_to_higress", "Convert parsed Nginx configuration to Higress route format", getConvertToHigressSchema()),
		handleConvertToHigress(),
	)

	// Generate Kubernetes YAML tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("generate_k8s_yaml", "Generate Kubernetes YAML manifests for Higress routes", getGenerateK8sYamlSchema()),
		handleGenerateK8sYaml(),
	)

	// Complete migration workflow tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("migrate_nginx_to_higress", "Complete workflow to migrate Nginx configuration to Higress - parses config, converts to Higress format, and generates Kubernetes YAML", getMigrateNginxToHigressSchema()),
		handleMigrateNginxToHigress(),
	)
}

// Handler functions
func handleParseNginxConfig() common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		configContent, ok := arguments["config_content"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'config_content' argument")
		}

		nginxConfig, err := parseNginxConfig(configContent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nginx config: %w", err)
		}

		configJSON, _ := json.MarshalIndent(nginxConfig, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Successfully parsed Nginx configuration:\n%s", string(configJSON)),
				},
			},
		}, nil
	}
}

func handleConvertToHigress() common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		nginxConfigStr, ok := arguments["nginx_config"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'nginx_config' argument")
		}

		namespace := "default"
		if ns, ok := arguments["namespace"].(string); ok {
			namespace = ns
		}

		var nginxConfig NginxConfig
		if err := json.Unmarshal([]byte(nginxConfigStr), &nginxConfig); err != nil {
			return nil, fmt.Errorf("failed to parse nginx config JSON: %w", err)
		}

		routes, err := convertToHigressRoutes(nginxConfig, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to higress format: %w", err)
		}

		routesJSON, _ := json.MarshalIndent(routes, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Successfully converted to Higress routes:\n%s", string(routesJSON)),
				},
			},
		}, nil
	}
}

func handleGenerateK8sYaml() common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		routesStr, ok := arguments["routes"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'routes' argument")
		}

		namespace := "default"
		if ns, ok := arguments["namespace"].(string); ok {
			namespace = ns
		}

		var routes []HigressRoute
		if err := json.Unmarshal([]byte(routesStr), &routes); err != nil {
			return nil, fmt.Errorf("failed to parse routes JSON: %w", err)
		}

		yamlContent := generateK8sYAML(routes, namespace)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: yamlContent,
				},
			},
		}, nil
	}
}

func handleMigrateNginxToHigress() common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		configContent, ok := arguments["nginx_config"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'nginx_config' argument")
		}

		namespace := "default"
		if ns, ok := arguments["namespace"].(string); ok {
			namespace = ns
		}

		// Step 1: Parse Nginx config
		nginxConfig, err := parseNginxConfig(configContent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nginx config: %w", err)
		}

		// Step 2: Convert to Higress format
		routes, err := convertToHigressRoutes(*nginxConfig, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to higress format: %w", err)
		}

		// Step 3: Generate YAML
		yamlContent := generateK8sYAML(routes, namespace)

		result := "=== Nginx to Higress Migration Complete ===\n\n"
		result += fmt.Sprintf("1. Parsed Nginx Configuration:\n%s\n\n", formatNginxConfig(nginxConfig))
		result += fmt.Sprintf("2. Generated Higress Routes:\n%s\n\n", formatHigressRoutes(routes))
		result += fmt.Sprintf("3. Kubernetes YAML Manifests:\n%s", yamlContent)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: result,
				},
			},
		}, nil
	}
}

// Schema functions
func getParseNginxConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"config_content": {
				"type": "string",
				"description": "The content of the Nginx configuration file"
			}
		},
		"required": ["config_content"],
		"additionalProperties": false
	}`)
}

func getConvertToHigressSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"nginx_config": {
				"type": "string",
				"description": "JSON string of the parsed Nginx configuration"
			},
			"namespace": {
				"type": "string",
				"description": "Kubernetes namespace for the routes",
				"default": "default"
			}
		},
		"required": ["nginx_config"],
		"additionalProperties": false
	}`)
}

func getGenerateK8sYamlSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"routes": {
				"type": "string",
				"description": "JSON string of Higress routes"
			},
			"namespace": {
				"type": "string",
				"description": "Kubernetes namespace",
				"default": "default"
			}
		},
		"required": ["routes"],
		"additionalProperties": false
	}`)
}

func getMigrateNginxToHigressSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"nginx_config": {
				"type": "string",
				"description": "The content of the Nginx configuration file"
			},
			"namespace": {
				"type": "string",
				"description": "Kubernetes namespace for the routes",
				"default": "default"
			}
		},
		"required": ["nginx_config"],
		"additionalProperties": false
	}`)
}

// parseNginxConfig parses Nginx configuration content
func parseNginxConfig(content string) (*NginxConfig, error) {
	config := &NginxConfig{
		ServerBlocks: []ServerBlock{},
	}

	// Simple regex-based parser for basic Nginx configurations
	// Use a more sophisticated approach to handle nested braces
	serverBlockRegex := regexp.MustCompile(`server\s*\{`)
	serverMatches := serverBlockRegex.FindAllStringIndex(content, -1)

	var serverBlocks [][]string
	for _, match := range serverMatches {
		start := match[1] // After the opening brace
		braceCount := 1
		end := start

		for i := start; i < len(content) && braceCount > 0; i++ {
			if content[i] == '{' {
				braceCount++
			} else if content[i] == '}' {
				braceCount--
			}
			end = i
		}

		if braceCount == 0 {
			blockContent := content[start:end]
			serverBlocks = append(serverBlocks, []string{"", blockContent})
		}
	}

	for _, block := range serverBlocks {
		serverBlock := ServerBlock{
			Location: []LocationBlock{},
			Headers:  make(map[string]string),
		}

		blockContent := block[1]

		// Parse listen directives
		listenRegex := regexp.MustCompile(`listen\s+([^;]+);`)
		listenMatches := listenRegex.FindAllStringSubmatch(blockContent, -1)
		for _, match := range listenMatches {
			serverBlock.Listen = append(serverBlock.Listen, strings.TrimSpace(match[1]))
		}

		// Parse server_name directives
		serverNameRegex := regexp.MustCompile(`server_name\s+([^;]+);`)
		serverNameMatches := serverNameRegex.FindAllStringSubmatch(blockContent, -1)
		for _, match := range serverNameMatches {
			names := strings.Fields(match[1])
			serverBlock.ServerName = append(serverBlock.ServerName, names...)
		}

		// Parse location blocks
		locationRegex := regexp.MustCompile(`location\s+([^{]+)\{`)
		locationMatches := locationRegex.FindAllStringIndex(blockContent, -1)

		for _, match := range locationMatches {
			pathStart := match[0]
			pathEnd := match[1] - 1 // Before the opening brace
			path := strings.TrimSpace(blockContent[pathStart:pathEnd])
			path = strings.TrimPrefix(path, "location")
			path = strings.TrimSpace(path)

			start := match[1] // After the opening brace
			braceCount := 1
			end := start

			for i := start; i < len(blockContent) && braceCount > 0; i++ {
				if blockContent[i] == '{' {
					braceCount++
				} else if blockContent[i] == '}' {
					braceCount--
				}
				end = i
			}

			if braceCount == 0 {
				location := LocationBlock{
					Path:    path,
					Headers: make(map[string]string),
				}

				locationContent := blockContent[start:end]

				// Parse proxy_pass
				proxyPassRegex := regexp.MustCompile(`proxy_pass\s+([^;]+);`)
				if proxyPassMatch := proxyPassRegex.FindStringSubmatch(locationContent); len(proxyPassMatch) > 1 {
					location.ProxyPass = strings.TrimSpace(proxyPassMatch[1])
				}

				// Parse rewrite rules
				rewriteRegex := regexp.MustCompile(`rewrite\s+([^;]+);`)
				rewriteMatches := rewriteRegex.FindAllStringSubmatch(locationContent, -1)
				for _, rewriteMatch := range rewriteMatches {
					parts := strings.Fields(rewriteMatch[1])
					if len(parts) >= 2 {
						rewrite := RewriteRule{
							Pattern: parts[0],
							Replace: parts[1],
						}
						if len(parts) > 2 {
							rewrite.Flag = parts[2]
						}
						location.Rewrite = append(location.Rewrite, rewrite)
					}
				}

				serverBlock.Location = append(serverBlock.Location, location)
			}
		}

		// Parse root directive
		rootRegex := regexp.MustCompile(`root\s+([^;]+);`)
		if rootMatch := rootRegex.FindStringSubmatch(blockContent); len(rootMatch) > 1 {
			serverBlock.Root = strings.TrimSpace(rootMatch[1])
		}

		// Parse index directive
		indexRegex := regexp.MustCompile(`index\s+([^;]+);`)
		if indexMatch := indexRegex.FindStringSubmatch(blockContent); len(indexMatch) > 1 {
			serverBlock.Index = strings.TrimSpace(indexMatch[1])
		}

		config.ServerBlocks = append(config.ServerBlocks, serverBlock)
	}

	return config, nil
}

// convertToHigressRoutes converts Nginx configuration to Higress routes
func convertToHigressRoutes(nginxConfig NginxConfig, namespace string) ([]HigressRoute, error) {
	var routes []HigressRoute

	for _, serverBlock := range nginxConfig.ServerBlocks {
		// Use first server_name as host, or default to "*"
		host := "*"
		if len(serverBlock.ServerName) > 0 {
			host = serverBlock.ServerName[0]
		}

		for _, location := range serverBlock.Location {
			route := HigressRoute{
				Name:        fmt.Sprintf("%s-%s", host, strings.ReplaceAll(location.Path, "/", "-")),
				Host:        host,
				Path:        location.Path,
				Headers:     make(map[string]string),
				Annotations: make(map[string]string),
			}

			// Convert proxy_pass to service and port
			if location.ProxyPass != "" {
				service, port := parseProxyPass(location.ProxyPass)
				route.Service = service
				route.Port = port
			}

			// Convert rewrite rules
			if len(location.Rewrite) > 0 {
				// Use the first rewrite rule for simplicity
				rewrite := location.Rewrite[0]
				route.Rewrite = &RewriteConfig{
					Path: rewrite.Replace,
				}
			}

			// Copy headers
			for k, v := range location.Headers {
				route.Headers[k] = v
			}

			// Add Higress-specific annotations
			route.Annotations["higress.io/route-type"] = "http"
			if len(serverBlock.Listen) > 0 {
				route.Annotations["higress.io/listen-port"] = serverBlock.Listen[0]
			}

			routes = append(routes, route)
		}
	}

	return routes, nil
}

// parseProxyPass extracts service name and port from proxy_pass directive
func parseProxyPass(proxyPass string) (string, int) {
	// Remove protocol prefix if present
	proxyPass = strings.TrimPrefix(proxyPass, "http://")
	proxyPass = strings.TrimPrefix(proxyPass, "https://")

	// Split by colon to get host and port
	parts := strings.Split(proxyPass, ":")
	if len(parts) == 2 {
		// Extract port
		portStr := strings.TrimSuffix(parts[1], "/")
		var port int
		if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil {
			return parts[0], port
		}
	}

	// Default values
	return "backend-service", 80
}

// generateK8sYAML generates Kubernetes YAML manifests
func generateK8sYAML(routes []HigressRoute, namespace string) string {
	var yaml strings.Builder

	yaml.WriteString("---\n")
	yaml.WriteString("# Higress Routes generated from Nginx configuration\n")
	yaml.WriteString("apiVersion: networking.k8s.io/v1\n")
	yaml.WriteString("kind: Ingress\n")
	yaml.WriteString("metadata:\n")
	yaml.WriteString("  name: nginx-migrated-routes\n")
	yaml.WriteString(fmt.Sprintf("  namespace: %s\n", namespace))
	yaml.WriteString("  annotations:\n")
	yaml.WriteString("    kubernetes.io/ingress.class: higress\n")

	// Group routes by host
	hostRoutes := make(map[string][]HigressRoute)
	for _, route := range routes {
		hostRoutes[route.Host] = append(hostRoutes[route.Host], route)
	}

	yaml.WriteString("spec:\n")
	yaml.WriteString("  rules:\n")

	for host, hostRoutes := range hostRoutes {
		yaml.WriteString(fmt.Sprintf("  - host: %s\n", host))
		yaml.WriteString("    http:\n")
		yaml.WriteString("      paths:\n")

		for _, route := range hostRoutes {
			yaml.WriteString("      - path: " + route.Path + "\n")
			yaml.WriteString("        pathType: Prefix\n")
			yaml.WriteString("        backend:\n")
			yaml.WriteString("          service:\n")
			yaml.WriteString("            name: " + route.Service + "\n")
			yaml.WriteString("            port:\n")
			yaml.WriteString(fmt.Sprintf("              number: %d\n", route.Port))
		}
	}

	return yaml.String()
}

// formatNginxConfig formats Nginx configuration for display
func formatNginxConfig(config *NginxConfig) string {
	var result strings.Builder
	for i, server := range config.ServerBlocks {
		result.WriteString(fmt.Sprintf("Server Block %d:\n", i+1))
		result.WriteString(fmt.Sprintf("  Listen: %v\n", server.Listen))
		result.WriteString(fmt.Sprintf("  Server Name: %v\n", server.ServerName))
		if server.Root != "" {
			result.WriteString("  Root: " + server.Root + "\n")
		}
		if server.Index != "" {
			result.WriteString("  Index: " + server.Index + "\n")
		}
		for j, location := range server.Location {
			result.WriteString(fmt.Sprintf("  Location %d: %s\n", j+1, location.Path))
			if location.ProxyPass != "" {
				result.WriteString("    Proxy Pass: " + location.ProxyPass + "\n")
			}
			for _, rewrite := range location.Rewrite {
				result.WriteString("    Rewrite: " + rewrite.Pattern + " -> " + rewrite.Replace + "\n")
			}
		}
		result.WriteString("\n")
	}
	return result.String()
}

// formatHigressRoutes formats Higress routes for display
func formatHigressRoutes(routes []HigressRoute) string {
	var result strings.Builder
	for i, route := range routes {
		result.WriteString(fmt.Sprintf("Route %d: %s\n", i+1, route.Name))
		result.WriteString("  Host: " + route.Host + "\n")
		result.WriteString("  Path: " + route.Path + "\n")
		result.WriteString(fmt.Sprintf("  Service: %s:%d\n", route.Service, route.Port))
		if route.Rewrite != nil {
			result.WriteString("  Rewrite: " + route.Rewrite.Path + "\n")
		}
		result.WriteString("\n")
	}
	return result.String()
}
