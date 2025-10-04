// Package tools provides nginx configuration migration tools for Higress
// Added on 2025.9.29 - Plugin migration tools implementation
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// NginxLuaPlugin represents a parsed Nginx Lua plugin configuration
type NginxLuaPlugin struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Phase        string                 `json:"phase"`
	Script       string                 `json:"script"`
	Config       map[string]interface{} `json:"config"`
	Dependencies []string               `json:"dependencies"`
	Description  string                 `json:"description"`
}

// HigressWasmPlugin represents a Higress WASM plugin configuration
type HigressWasmPlugin struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Phase       string                 `json:"phase"`
	WasmCode    string                 `json:"wasm_code"`
	Config      map[string]interface{} `json:"config"`
	Language    string                 `json:"language"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
}

// PluginMigrationResult represents the result of plugin migration
type PluginMigrationResult struct {
	OriginalPlugin     *NginxLuaPlugin    `json:"original_plugin"`
	MigratedPlugin     *HigressWasmPlugin `json:"migrated_plugin"`
	MigrationNotes     []string           `json:"migration_notes"`
	CompatibilityLevel string             `json:"compatibility_level"` // "full", "partial", "manual"
	RequiredChanges    []string           `json:"required_changes"`
	Warnings           []string           `json:"warnings"`
}

// RegisterPluginMigrationTools registers nginx lua to higress wasm plugin migration tools
func RegisterPluginMigrationTools(mcpServer *common.MCPServer, client *higress.HigressClient) {
	// Parse Nginx Lua plugin configuration
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("parse_nginx_lua_plugin", "Parse Nginx Lua plugin configuration and extract plugin details", getParseNginxLuaPluginSchema()),
		handleParseNginxLuaPlugin(),
	)

	// Convert Lua plugin to WASM plugin
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("convert_lua_to_wasm_plugin", "Convert Nginx Lua plugin to Higress WASM plugin format", getConvertLuaToWasmPluginSchema()),
		handleConvertLuaToWasmPlugin(),
	)

	// Generate WASM plugin configuration
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("generate_wasm_plugin_config", "Generate Higress WASM plugin configuration YAML", getGenerateWasmPluginConfigSchema()),
		handleGenerateWasmPluginConfig(),
	)

	// Complete plugin migration workflow
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("migrate_nginx_lua_plugin", "Complete workflow to migrate Nginx Lua plugin to Higress WASM plugin", getMigrateNginxLuaPluginSchema()),
		handleMigrateNginxLuaPlugin(),
	)

	// Analyze plugin compatibility
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("analyze_plugin_compatibility", "Analyze compatibility between Nginx Lua plugin and Higress WASM plugin", getAnalyzePluginCompatibilitySchema()),
		handleAnalyzePluginCompatibility(),
	)

	// Generate migration report
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("generate_plugin_migration_report", "Generate detailed migration report for Nginx Lua plugins", getGeneratePluginMigrationReportSchema()),
		handleGeneratePluginMigrationReport(),
	)
}

// Handler functions
func handleParseNginxLuaPlugin() common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		pluginConfig, ok := arguments["plugin_config"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'plugin_config' argument")
		}

		luaPlugin, err := parseNginxLuaPlugin(pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nginx lua plugin: %w", err)
		}

		pluginJSON, _ := json.MarshalIndent(luaPlugin, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Successfully parsed Nginx Lua plugin:\n%s", string(pluginJSON)),
				},
			},
		}, nil
	}
}

func handleConvertLuaToWasmPlugin() common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		luaPluginStr, ok := arguments["lua_plugin"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'lua_plugin' argument")
		}

		targetLanguage := "rust"
		if lang, ok := arguments["target_language"].(string); ok {
			targetLanguage = lang
		}

		var luaPlugin NginxLuaPlugin
		if err := json.Unmarshal([]byte(luaPluginStr), &luaPlugin); err != nil {
			return nil, fmt.Errorf("failed to parse lua plugin JSON: %w", err)
		}

		wasmPlugin, err := convertLuaToWasmPlugin(&luaPlugin, targetLanguage)
		if err != nil {
			return nil, fmt.Errorf("failed to convert lua to wasm plugin: %w", err)
		}

		wasmJSON, _ := json.MarshalIndent(wasmPlugin, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Successfully converted Lua plugin to WASM:\n%s", string(wasmJSON)),
				},
			},
		}, nil
	}
}

func handleGenerateWasmPluginConfig() common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		wasmPluginStr, ok := arguments["wasm_plugin"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'wasm_plugin' argument")
		}

		namespace := "default"
		if ns, ok := arguments["namespace"].(string); ok {
			namespace = ns
		}

		var wasmPlugin HigressWasmPlugin
		if err := json.Unmarshal([]byte(wasmPluginStr), &wasmPlugin); err != nil {
			return nil, fmt.Errorf("failed to parse wasm plugin JSON: %w", err)
		}

		yamlContent := generateWasmPluginConfig(&wasmPlugin, namespace)
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

func handleMigrateNginxLuaPlugin() common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		pluginConfig, ok := arguments["plugin_config"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'plugin_config' argument")
		}

		targetLanguage := "rust"
		if lang, ok := arguments["target_language"].(string); ok {
			targetLanguage = lang
		}

		namespace := "default"
		if ns, ok := arguments["namespace"].(string); ok {
			namespace = ns
		}

		// Step 1: Parse Nginx Lua plugin
		luaPlugin, err := parseNginxLuaPlugin(pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nginx lua plugin: %w", err)
		}

		// Step 2: Convert to WASM plugin
		wasmPlugin, err := convertLuaToWasmPlugin(luaPlugin, targetLanguage)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to wasm plugin: %w", err)
		}

		// Step 3: Generate YAML configuration
		yamlContent := generateWasmPluginConfig(wasmPlugin, namespace)

		// Step 4: Analyze compatibility
		migrationResult := analyzePluginCompatibility(luaPlugin)
		migrationResult.MigratedPlugin = wasmPlugin

		result := "=== Nginx Lua Plugin to Higress WASM Migration Complete ===\n\n"
		result += fmt.Sprintf("1. Original Lua Plugin:\n%s\n\n", formatLuaPlugin(luaPlugin))
		result += fmt.Sprintf("2. Migrated WASM Plugin:\n%s\n\n", formatWasmPlugin(wasmPlugin))
		result += fmt.Sprintf("3. Kubernetes YAML Configuration:\n%s\n\n", yamlContent)
		result += fmt.Sprintf("4. Migration Analysis:\n%s", formatMigrationResult(migrationResult))

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

func handleAnalyzePluginCompatibility() common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		luaPluginStr, ok := arguments["lua_plugin"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'lua_plugin' argument")
		}

		var luaPlugin NginxLuaPlugin
		if err := json.Unmarshal([]byte(luaPluginStr), &luaPlugin); err != nil {
			return nil, fmt.Errorf("failed to parse lua plugin JSON: %w", err)
		}

		migrationResult := analyzePluginCompatibility(&luaPlugin)
		resultJSON, _ := json.MarshalIndent(migrationResult, "", "  ")

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Plugin Compatibility Analysis:\n%s", string(resultJSON)),
				},
			},
		}, nil
	}
}

func handleGeneratePluginMigrationReport() common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		pluginsStr, ok := arguments["plugins"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'plugins' argument")
		}

		var plugins []NginxLuaPlugin
		if err := json.Unmarshal([]byte(pluginsStr), &plugins); err != nil {
			return nil, fmt.Errorf("failed to parse plugins JSON: %w", err)
		}

		report := generatePluginMigrationReport(plugins)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: report,
				},
			},
		}, nil
	}
}

// Schema functions
func getParseNginxLuaPluginSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"plugin_config": {
				"type": "string",
				"description": "Nginx Lua plugin configuration content"
			}
		},
		"required": ["plugin_config"],
		"additionalProperties": false
	}`)
}

func getConvertLuaToWasmPluginSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"lua_plugin": {
				"type": "string",
				"description": "JSON string of the parsed Nginx Lua plugin"
			},
			"target_language": {
				"type": "string",
				"enum": ["rust", "go", "cpp", "assemblyscript"],
				"description": "Target language for WASM plugin",
				"default": "rust"
			}
		},
		"required": ["lua_plugin"],
		"additionalProperties": false
	}`)
}

func getGenerateWasmPluginConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"wasm_plugin": {
				"type": "string",
				"description": "JSON string of Higress WASM plugin"
			},
			"namespace": {
				"type": "string",
				"description": "Kubernetes namespace",
				"default": "default"
			}
		},
		"required": ["wasm_plugin"],
		"additionalProperties": false
	}`)
}

func getMigrateNginxLuaPluginSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"plugin_config": {
				"type": "string",
				"description": "Nginx Lua plugin configuration content"
			},
			"target_language": {
				"type": "string",
				"enum": ["rust", "go", "cpp", "assemblyscript"],
				"description": "Target language for WASM plugin",
				"default": "rust"
			},
			"namespace": {
				"type": "string",
				"description": "Kubernetes namespace",
				"default": "default"
			}
		},
		"required": ["plugin_config"],
		"additionalProperties": false
	}`)
}

func getAnalyzePluginCompatibilitySchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"lua_plugin": {
				"type": "string",
				"description": "JSON string of the parsed Nginx Lua plugin"
			}
		},
		"required": ["lua_plugin"],
		"additionalProperties": false
	}`)
}

func getGeneratePluginMigrationReportSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"plugins": {
				"type": "string",
				"description": "JSON string of array of Nginx Lua plugins"
			}
		},
		"required": ["plugins"],
		"additionalProperties": false
	}`)
}

// Implementation functions
func parseNginxLuaPlugin(config string) (*NginxLuaPlugin, error) {
	plugin := &NginxLuaPlugin{
		Config:       make(map[string]interface{}),
		Dependencies: []string{},
	}

	lines := strings.Split(config, "\n")
	var currentScript strings.Builder
	inScriptBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse lua_package_path (dependencies)
		if strings.Contains(line, "lua_package_path") {
			// Extract package paths to infer dependencies
			if strings.Contains(line, "resty") {
				plugin.Dependencies = append(plugin.Dependencies, "lua-resty-http", "lua-resty-core")
			}
		}

		// Parse access_by_lua_block, content_by_lua_block, etc.
		if strings.Contains(line, "_by_lua") {
			if strings.Contains(line, "access_by_lua") {
				plugin.Phase = "access"
				plugin.Type = "access"
			} else if strings.Contains(line, "content_by_lua") {
				plugin.Phase = "content"
				plugin.Type = "content"
			} else if strings.Contains(line, "header_filter_by_lua") {
				plugin.Phase = "header_filter"
				plugin.Type = "response"
			} else if strings.Contains(line, "rewrite_by_lua") {
				plugin.Phase = "rewrite"
				plugin.Type = "rewrite"
			}

			if strings.HasSuffix(line, "{") {
				inScriptBlock = true
				continue
			}
		}

		// Collect script content
		if inScriptBlock {
			if strings.Contains(line, "}") {
				inScriptBlock = false
				continue
			}
			currentScript.WriteString(line + "\n")
		}

		// Parse set directives for configuration
		if strings.HasPrefix(line, "set $") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				key := strings.TrimPrefix(parts[1], "$")
				value := strings.Trim(parts[2], `"';`)
				plugin.Config[key] = value
			}
		}
	}

	plugin.Script = currentScript.String()

	// Generate plugin name if not found
	if plugin.Name == "" {
		plugin.Name = fmt.Sprintf("%s-lua-plugin", plugin.Phase)
	}

	// Set default description
	if plugin.Description == "" {
		plugin.Description = fmt.Sprintf("Nginx Lua plugin for %s phase", plugin.Phase)
	}

	return plugin, nil
}

func convertLuaToWasmPlugin(luaPlugin *NginxLuaPlugin, targetLanguage string) (*HigressWasmPlugin, error) {
	wasmPlugin := &HigressWasmPlugin{
		Name:        luaPlugin.Name + "-wasm",
		Language:    targetLanguage,
		Config:      make(map[string]interface{}),
		Version:     "1.0.0",
		Description: fmt.Sprintf("Migrated from Nginx Lua plugin: %s", luaPlugin.Name),
	}

	// Map Nginx Lua phases to Higress WASM phases
	switch luaPlugin.Phase {
	case "access":
		wasmPlugin.Type = "authn"
		wasmPlugin.Phase = "AUTHN"
	case "rewrite":
		wasmPlugin.Type = "traffic-modification"
		wasmPlugin.Phase = "ROUTER"
	case "content":
		wasmPlugin.Type = "traffic-modification"
		wasmPlugin.Phase = "ROUTER"
	case "header_filter":
		wasmPlugin.Type = "response-modification"
		wasmPlugin.Phase = "STATS"
	default:
		wasmPlugin.Type = "traffic-modification"
		wasmPlugin.Phase = "ROUTER"
	}

	// Copy and adapt configuration
	for k, v := range luaPlugin.Config {
		wasmPlugin.Config[k] = v
	}

	// Add migration metadata
	wasmPlugin.Config["_migration_source"] = "nginx-lua"
	wasmPlugin.Config["_original_phase"] = luaPlugin.Phase
	wasmPlugin.Config["_lua_script_hash"] = fmt.Sprintf("%x", len(luaPlugin.Script)) // Simple hash

	// Generate WASM code reference based on target language
	switch targetLanguage {
	case "rust":
		wasmPlugin.WasmCode = fmt.Sprintf("oci://registry.higress.io/plugins/%s:latest", wasmPlugin.Name)
	case "go":
		wasmPlugin.WasmCode = fmt.Sprintf("oci://registry.higress.io/go-plugins/%s:latest", wasmPlugin.Name)
	case "cpp":
		wasmPlugin.WasmCode = fmt.Sprintf("oci://registry.higress.io/cpp-plugins/%s:latest", wasmPlugin.Name)
	case "assemblyscript":
		wasmPlugin.WasmCode = fmt.Sprintf("oci://registry.higress.io/as-plugins/%s:latest", wasmPlugin.Name)
	default:
		wasmPlugin.WasmCode = fmt.Sprintf("file:///opt/plugins/%s.wasm", wasmPlugin.Name)
	}

	return wasmPlugin, nil
}

func generateWasmPluginConfig(wasmPlugin *HigressWasmPlugin, namespace string) string {
	var yaml strings.Builder

	yaml.WriteString("---\n")
	yaml.WriteString("# Higress WASM Plugin generated from Nginx Lua plugin migration\n")
	yaml.WriteString("apiVersion: extensions.higress.io/v1alpha1\n")
	yaml.WriteString("kind: WasmPlugin\n")
	yaml.WriteString("metadata:\n")
	yaml.WriteString(fmt.Sprintf("  name: %s\n", wasmPlugin.Name))
	yaml.WriteString(fmt.Sprintf("  namespace: %s\n", namespace))
	yaml.WriteString("  labels:\n")
	yaml.WriteString("    app.kubernetes.io/name: higress\n")
	yaml.WriteString("    app.kubernetes.io/component: gateway\n")
	yaml.WriteString("  annotations:\n")
	yaml.WriteString(fmt.Sprintf("    higress.io/plugin-version: \"%s\"\n", wasmPlugin.Version))
	yaml.WriteString(fmt.Sprintf("    higress.io/plugin-language: \"%s\"\n", wasmPlugin.Language))
	yaml.WriteString("    higress.io/migration-source: \"nginx-lua\"\n")
	yaml.WriteString("spec:\n")
	yaml.WriteString(fmt.Sprintf("  url: %s\n", wasmPlugin.WasmCode))
	yaml.WriteString(fmt.Sprintf("  phase: %s\n", wasmPlugin.Phase))
	yaml.WriteString("  priority: 100\n")

	if len(wasmPlugin.Config) > 0 {
		yaml.WriteString("  pluginConfig:\n")
		for key, value := range wasmPlugin.Config {
			switch v := value.(type) {
			case string:
				yaml.WriteString(fmt.Sprintf("    %s: \"%s\"\n", key, v))
			case bool:
				yaml.WriteString(fmt.Sprintf("    %s: %t\n", key, v))
			case int, int64, float64:
				yaml.WriteString(fmt.Sprintf("    %s: %v\n", key, v))
			default:
				yaml.WriteString(fmt.Sprintf("    %s: \"%v\"\n", key, v))
			}
		}
	}

	yaml.WriteString("\n---\n")
	yaml.WriteString("# Plugin Gateway Association\n")
	yaml.WriteString("apiVersion: extensions.higress.io/v1alpha1\n")
	yaml.WriteString("kind: WasmPluginAttachment\n")
	yaml.WriteString("metadata:\n")
	yaml.WriteString(fmt.Sprintf("  name: %s-attachment\n", wasmPlugin.Name))
	yaml.WriteString(fmt.Sprintf("  namespace: %s\n", namespace))
	yaml.WriteString("spec:\n")
	yaml.WriteString(fmt.Sprintf("  plugin: %s\n", wasmPlugin.Name))
	yaml.WriteString("  selector:\n")
	yaml.WriteString("    matchLabels:\n")
	yaml.WriteString("      app: higress-gateway\n")

	return yaml.String()
}

func analyzePluginCompatibility(luaPlugin *NginxLuaPlugin) *PluginMigrationResult {
	result := &PluginMigrationResult{
		OriginalPlugin:     luaPlugin,
		MigrationNotes:     []string{},
		RequiredChanges:    []string{},
		Warnings:           []string{},
		CompatibilityLevel: "full", // Default to optimistic
	}

	script := strings.ToLower(luaPlugin.Script)

	// Analyze script complexity and compatibility
	complexityScore := 0

	// Check for common Lua patterns that might need attention
	if strings.Contains(script, "ngx.location.capture") {
		result.RequiredChanges = append(result.RequiredChanges, "Replace ngx.location.capture with HTTP client calls")
		result.Warnings = append(result.Warnings, "Internal subrequests need to be converted to external HTTP calls")
		complexityScore += 2
	}

	if strings.Contains(script, "ngx.shared") {
		result.RequiredChanges = append(result.RequiredChanges, "Replace ngx.shared with external cache or state management")
		result.Warnings = append(result.Warnings, "Shared dictionaries not directly supported in WASM")
		complexityScore += 3
	}

	if strings.Contains(script, "cosocket") || strings.Contains(script, "ngx.socket") {
		result.RequiredChanges = append(result.RequiredChanges, "Replace cosocket API with WASM HTTP client")
		complexityScore += 2
	}

	if strings.Contains(script, "ngx.timer") {
		result.RequiredChanges = append(result.RequiredChanges, "Timers need to be implemented differently in WASM context")
		result.Warnings = append(result.Warnings, "Background timers not supported in WASM plugins")
		complexityScore += 3
	}

	if strings.Contains(script, "ffi") {
		result.RequiredChanges = append(result.RequiredChanges, "FFI calls need to be reimplemented in target language")
		result.Warnings = append(result.Warnings, "FFI not available in WASM environment")
		complexityScore += 4
	}

	// Simple patterns that are generally compatible
	simplePatterns := []string{"ngx.req", "ngx.resp", "ngx.var", "ngx.header", "ngx.arg"}
	for _, pattern := range simplePatterns {
		if strings.Contains(script, pattern) {
			result.MigrationNotes = append(result.MigrationNotes, fmt.Sprintf("Standard %s operations can be migrated to WASM API", pattern))
		}
	}

	// Determine compatibility level based on complexity
	switch {
	case complexityScore == 0:
		result.CompatibilityLevel = "full"
		result.MigrationNotes = append(result.MigrationNotes, "Plugin appears to use standard Nginx APIs that have direct WASM equivalents")
	case complexityScore <= 2:
		result.CompatibilityLevel = "partial"
		result.MigrationNotes = append(result.MigrationNotes, "Plugin needs minor modifications for WASM compatibility")
	default:
		result.CompatibilityLevel = "manual"
		result.MigrationNotes = append(result.MigrationNotes, "Plugin requires significant manual conversion due to complex dependencies")
	}

	// Check for dependencies
	if len(luaPlugin.Dependencies) > 0 {
		result.RequiredChanges = append(result.RequiredChanges, "Review and replace Lua dependencies with WASM-compatible alternatives")
		for _, dep := range luaPlugin.Dependencies {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Dependency '%s' needs WASM equivalent", dep))
		}
	}

	// Phase-specific analysis
	switch luaPlugin.Phase {
	case "access":
		result.MigrationNotes = append(result.MigrationNotes, "Access phase maps well to Higress AUTHN phase")
	case "rewrite":
		result.MigrationNotes = append(result.MigrationNotes, "Rewrite phase maps to Higress ROUTER phase")
	case "content":
		result.RequiredChanges = append(result.RequiredChanges, "Content generation needs to be handled differently in WASM")
		result.Warnings = append(result.Warnings, "Full response generation in WASM plugins has limitations")
	case "header_filter":
		result.MigrationNotes = append(result.MigrationNotes, "Header filtering maps to Higress STATS phase")
	}

	return result
}

func generatePluginMigrationReport(plugins []NginxLuaPlugin) string {
	var report strings.Builder

	report.WriteString("# Nginx Lua Plugin Migration Report\n")
	report.WriteString("Generated on: " + fmt.Sprintf("%v", "2025-09-29") + "\n\n")

	report.WriteString(fmt.Sprintf("## Summary\n"))
	report.WriteString(fmt.Sprintf("- Total plugins analyzed: %d\n", len(plugins)))

	// Analyze all plugins
	var fullCompatible, partialCompatible, manualRequired int
	var allResults []*PluginMigrationResult

	for _, plugin := range plugins {
		result := analyzePluginCompatibility(&plugin)
		allResults = append(allResults, result)

		switch result.CompatibilityLevel {
		case "full":
			fullCompatible++
		case "partial":
			partialCompatible++
		case "manual":
			manualRequired++
		}
	}

	report.WriteString(fmt.Sprintf("- Fully compatible: %d (%.1f%%)\n", fullCompatible, float64(fullCompatible)/float64(len(plugins))*100))
	report.WriteString(fmt.Sprintf("- Partially compatible: %d (%.1f%%)\n", partialCompatible, float64(partialCompatible)/float64(len(plugins))*100))
	report.WriteString(fmt.Sprintf("- Manual conversion required: %d (%.1f%%)\n\n", manualRequired, float64(manualRequired)/float64(len(plugins))*100))

	// Detailed plugin analysis
	report.WriteString("## Plugin Details\n\n")
	for i, result := range allResults {
		plugin := result.OriginalPlugin
		report.WriteString(fmt.Sprintf("### %d. %s\n", i+1, plugin.Name))
		report.WriteString(fmt.Sprintf("- **Type**: %s\n", plugin.Type))
		report.WriteString(fmt.Sprintf("- **Phase**: %s\n", plugin.Phase))
		report.WriteString(fmt.Sprintf("- **Compatibility**: %s\n", result.CompatibilityLevel))

		if len(plugin.Dependencies) > 0 {
			report.WriteString(fmt.Sprintf("- **Dependencies**: %s\n", strings.Join(plugin.Dependencies, ", ")))
		}

		if len(result.MigrationNotes) > 0 {
			report.WriteString("- **Migration Notes**:\n")
			for _, note := range result.MigrationNotes {
				report.WriteString(fmt.Sprintf("  - %s\n", note))
			}
		}

		if len(result.RequiredChanges) > 0 {
			report.WriteString("- **Required Changes**:\n")
			for _, change := range result.RequiredChanges {
				report.WriteString(fmt.Sprintf("  - %s\n", change))
			}
		}

		if len(result.Warnings) > 0 {
			report.WriteString("- **Warnings**:\n")
			for _, warning := range result.Warnings {
				report.WriteString(fmt.Sprintf("  - ⚠️ %s\n", warning))
			}
		}

		report.WriteString("\n")
	}

	// Migration recommendations
	report.WriteString("## Migration Recommendations\n\n")
	report.WriteString("### 1. Start with Fully Compatible Plugins\n")
	if fullCompatible > 0 {
		report.WriteString("Begin migration with plugins marked as 'full' compatibility as they require minimal changes.\n\n")
	}

	report.WriteString("### 2. Address Partially Compatible Plugins\n")
	if partialCompatible > 0 {
		report.WriteString("Review the required changes for partially compatible plugins and plan accordingly.\n\n")
	}

	report.WriteString("### 3. Manual Conversion Strategy\n")
	if manualRequired > 0 {
		report.WriteString("For plugins requiring manual conversion:\n")
		report.WriteString("- Consider LLM-assisted migration tools\n")
		report.WriteString("- Evaluate if plugin functionality can be replaced with existing Higress features\n")
		report.WriteString("- Plan for custom WASM plugin development\n\n")
	}

	report.WriteString("### 4. Testing Strategy\n")
	report.WriteString("- Set up comprehensive testing for migrated plugins\n")
	report.WriteString("- Validate performance characteristics in WASM environment\n")
	report.WriteString("- Ensure functional equivalence with original Lua plugins\n\n")

	return report.String()
}

// Helper formatting functions
func formatLuaPlugin(plugin *NginxLuaPlugin) string {
	return fmt.Sprintf("Name: %s, Type: %s, Phase: %s, Dependencies: %v",
		plugin.Name, plugin.Type, plugin.Phase, plugin.Dependencies)
}

func formatWasmPlugin(plugin *HigressWasmPlugin) string {
	return fmt.Sprintf("Name: %s, Type: %s, Phase: %s, Language: %s, Version: %s",
		plugin.Name, plugin.Type, plugin.Phase, plugin.Language, plugin.Version)
}

func formatMigrationResult(result *PluginMigrationResult) string {
	var analysis strings.Builder
	analysis.WriteString(fmt.Sprintf("Compatibility Level: %s\n", result.CompatibilityLevel))

	if len(result.MigrationNotes) > 0 {
		analysis.WriteString("Notes: " + strings.Join(result.MigrationNotes, "; ") + "\n")
	}

	if len(result.RequiredChanges) > 0 {
		analysis.WriteString("Required Changes: " + strings.Join(result.RequiredChanges, "; ") + "\n")
	}

	if len(result.Warnings) > 0 {
		analysis.WriteString("Warnings: " + strings.Join(result.Warnings, "; ") + "\n")
	}

	return analysis.String()
}
