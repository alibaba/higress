package tools

import (
	"strings"
	"testing"
)

// contains checks if a string contains a substring (helper function for tests)
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestParseNginxLuaPlugin(t *testing.T) {
	pluginConfig := `
access_by_lua_block {
    -- Custom authentication plugin
    local http = require "resty.http"
    local cjson = require "cjson"
    
    local function authenticate()
        local httpc = http.new()
        local res, err = httpc:request_uri("http://auth-service:8080/validate", {
            method = "POST",
            headers = {
                ["Content-Type"] = "application/json",
                ["Authorization"] = ngx.var.http_authorization
            }
        })
        
        if not res or res.status ~= 200 then
            ngx.status = 401
            ngx.say(cjson.encode({error = "Unauthorized"}))
            ngx.exit(401)
        end
        
        local user_data = cjson.decode(res.body)
        ngx.req.set_header("X-User-ID", user_data.user_id)
        ngx.req.set_header("X-User-Role", user_data.role)
    end
    
    authenticate()
}
`

	plugin, err := parseNginxLuaPlugin(pluginConfig)
	if err != nil {
		t.Fatalf("Failed to parse nginx lua plugin: %v", err)
	}

	// The parser generates a name based on the phase when no explicit name is found
	if plugin.Name != "access-lua-plugin" {
		t.Errorf("Expected plugin name 'access-lua-plugin', got %s", plugin.Name)
	}

	if plugin.Type != "access" {
		t.Errorf("Expected plugin type 'access', got %s", plugin.Type)
	}

	if plugin.Phase != "access" {
		t.Errorf("Expected plugin phase 'access', got %s", plugin.Phase)
	}
}

func TestConvertLuaToWasmPlugin(t *testing.T) {
	luaPlugin := &NginxLuaPlugin{
		Name:         "auth-plugin",
		Type:         "access",
		Phase:        "access",
		Script:       "-- Lua script content",
		Config:       map[string]interface{}{"auth_service_url": "http://auth-service:8080/validate"},
		Dependencies: []string{"lua-resty-http", "cjson"},
		Description:  "Custom authentication plugin",
	}

	wasmPlugin, err := convertLuaToWasmPlugin(luaPlugin, "rust")
	if err != nil {
		t.Fatalf("Failed to convert lua to wasm plugin: %v", err)
	}

	if wasmPlugin.Name != "auth-plugin-wasm" {
		t.Errorf("Expected wasm plugin name 'auth-plugin-wasm', got %s", wasmPlugin.Name)
	}

	// Nginx "access" phase maps to Higress "authn" type and "AUTHN" phase
	if wasmPlugin.Type != "authn" {
		t.Errorf("Expected wasm plugin type 'authn', got %s", wasmPlugin.Type)
	}

	if wasmPlugin.Language != "rust" {
		t.Errorf("Expected wasm plugin language 'rust', got %s", wasmPlugin.Language)
	}

	if wasmPlugin.Phase != "AUTHN" {
		t.Errorf("Expected wasm plugin phase 'AUTHN', got %s", wasmPlugin.Phase)
	}
}

func TestGenerateWasmPluginConfig(t *testing.T) {
	wasmPlugin := &HigressWasmPlugin{
		Name:        "auth-plugin-wasm",
		Type:        "http",
		Phase:       "access",
		WasmCode:    "base64-encoded-wasm-binary",
		Config:      map[string]interface{}{"auth_service_url": "http://auth-service:8080/validate"},
		Language:    "rust",
		Description: "Migrated authentication plugin",
		Version:     "1.0.0",
	}

	yaml := generateWasmPluginConfig(wasmPlugin, "production")

	// Basic checks for YAML content
	if !contains(yaml, "apiVersion: extensions.higress.io/v1alpha1") {
		t.Error("YAML should contain apiVersion")
	}
	if !contains(yaml, "kind: WasmPlugin") {
		t.Error("YAML should contain kind: WasmPlugin")
	}
	if !contains(yaml, "name: auth-plugin-wasm") {
		t.Error("YAML should contain plugin name")
	}
	if !contains(yaml, "namespace: production") {
		t.Error("YAML should contain namespace")
	}
	if !contains(yaml, "phase: access") {
		t.Error("YAML should contain plugin phase")
	}
	if !contains(yaml, "higress.io/plugin-language: \"rust\"") {
		t.Error("YAML should contain language annotation")
	}
}

func TestAnalyzePluginCompatibility(t *testing.T) {
	luaPlugin := &NginxLuaPlugin{
		Name:         "auth-plugin",
		Type:         "access",
		Phase:        "access",
		Script:       "-- Lua script content",
		Config:       map[string]interface{}{"auth_service_url": "http://auth-service:8080/validate"},
		Dependencies: []string{"lua-resty-http", "cjson"},
		Description:  "Custom authentication plugin",
	}

	compatibility := analyzePluginCompatibility(luaPlugin)

	if compatibility.OriginalPlugin == nil {
		t.Error("Compatibility analysis should include original plugin")
	}

	if compatibility.CompatibilityLevel == "" {
		t.Error("Compatibility analysis should include compatibility level")
	}

	if len(compatibility.MigrationNotes) == 0 {
		t.Error("Compatibility analysis should include migration notes")
	}
}

func TestGeneratePluginMigrationReport(t *testing.T) {
	plugins := []NginxLuaPlugin{
		{
			Name:        "auth-plugin",
			Type:        "access",
			Phase:       "access",
			Description: "Authentication plugin",
		},
		{
			Name:        "log-plugin",
			Type:        "log",
			Phase:       "log",
			Description: "Logging plugin",
		},
		{
			Name:        "rate-limit-plugin",
			Type:        "access",
			Phase:       "access",
			Description: "Rate limiting plugin",
		},
	}

	report := generatePluginMigrationReport(plugins)

	if !contains(report, "Plugin Migration Report") {
		t.Error("Report should contain title")
	}

	if !contains(report, "Total plugins analyzed: 3") {
		t.Error("Report should contain plugin count")
	}
}

func TestCompletePluginMigrationWorkflow(t *testing.T) {
	pluginConfig := `
access_by_lua_block {
    -- Simple authentication plugin
    local function authenticate()
        if ngx.var.http_authorization == "" then
            ngx.status = 401
            ngx.say("Unauthorized")
            ngx.exit(401)
        end
    end
    
    authenticate()
}
`

	// Test parsing
	plugin, err := parseNginxLuaPlugin(pluginConfig)
	if err != nil {
		t.Fatalf("Failed to parse nginx lua plugin: %v", err)
	}

	// Test conversion
	wasmPlugin, err := convertLuaToWasmPlugin(plugin, "rust")
	if err != nil {
		t.Fatalf("Failed to convert lua to wasm plugin: %v", err)
	}

	// Test config generation
	yaml := generateWasmPluginConfig(wasmPlugin, "test")

	// Verify the complete workflow
	if plugin.Name == "" {
		t.Error("Plugin should have a name")
	}

	if wasmPlugin.Name == "" {
		t.Error("WASM plugin should have a name")
	}

	if wasmPlugin.Language != "rust" {
		t.Error("WASM plugin should have correct language")
	}

	// Verify YAML contains expected content
	if !contains(yaml, "WasmPlugin") {
		t.Error("YAML should contain WasmPlugin kind")
	}

	if !contains(yaml, "test") {
		t.Error("YAML should contain test namespace")
	}
}
