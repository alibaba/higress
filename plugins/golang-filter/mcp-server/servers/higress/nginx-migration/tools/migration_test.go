// Package tools provides real test cases for nginx migration functionality
// Added on 2025.9.30 - Real business logic tests
package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// --- Test Data ---

const testNginxConfig = `
server {
    listen 80;
    server_name api.example.com;
    
    location /users {
        proxy_pass http://user-service:8080;
        rewrite ^/users/(.*) /api/v1/users/$1 break;
    }
    
    location /orders {
        proxy_pass http://order-service:8080/api/orders;
    }
}
`

const testLuaPlugin = `
# Simple authentication plugin
access_by_lua_block {
    local auth_header = ngx.var.http_authorization
    if not auth_header or auth_header == "" then
        ngx.status = 401
        ngx.say('{"error": "unauthorized"}')
        return ngx.exit(401)
    end
    
    -- Forward the header to upstream
    ngx.req.set_header("X-Auth-Token", auth_header)
}

set $auth_token "";
`

const testComplexLuaPlugin = `
# Shared dictionary and HTTP call plugin
lua_shared_dict user_cache 10m;

access_by_lua_block {
    local http = require "resty.http"
    local cjson = require "cjson"
    local user_cache = ngx.shared.user_cache
    
    local api_key = ngx.var.http_x_api_key
    if not api_key then
        return ngx.exit(401)
    end
    
    -- Check cache
    local user_id = user_cache:get(api_key)
    if user_id then
        ngx.var.user_id = user_id
        return
    end
    
    -- Validate via external service
    local httpc = http.new()
    local res, err = httpc:request_uri("http://auth-service/validate", {
        method = "POST",
        body = cjson.encode({api_key = api_key})
    })
    
    if not res or res.status ~= 200 then
        return ngx.exit(403)
    end
    
    local body = cjson.decode(res.body)
    user_cache:set(api_key, body.user_id, 300)
    ngx.var.user_id = body.user_id
}
`

// --- Test Functions ---

func TestRealNginxParsing(t *testing.T) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🧪 测试 1: 真实 Nginx 配置解析")
	fmt.Println(strings.Repeat("-", 80))

	fmt.Println("📥 输入的 Nginx 配置:")
	fmt.Printf("```nginx\n%s\n```\n", testNginxConfig)

	// REAL FUNCTION CALL
	config, err := parseNginxConfig(testNginxConfig)
	if err != nil {
		t.Fatalf("❌ parseNginxConfig FAILED: %v", err)
	}

	output, _ := json.MarshalIndent(config, "", "  ")

	fmt.Println("📤 真实的解析输出:")
	fmt.Printf("```json\n%s\n```\n", string(output))

	if len(config.ServerBlocks) != 1 {
		t.Errorf("Expected 1 server block, but got %d", len(config.ServerBlocks))
	}
	if len(config.ServerBlocks[0].Location) != 2 {
		t.Errorf("Expected 2 location blocks, but got %d", len(config.ServerBlocks[0].Location))
	}

	fmt.Println("✅ 真实解析成功！")
}

func TestRealNginxToHigressConversion(t *testing.T) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🧪 测试 2: 真实 Nginx 到 Higress 路由转换")
	fmt.Println(strings.Repeat("-", 80))

	// REAL FUNCTION CALL (parse)
	config, err := parseNginxConfig(testNginxConfig)
	if err != nil {
		t.Fatalf("❌ parseNginxConfig FAILED: %v", err)
	}

	// REAL FUNCTION CALL (convert)
	routes, err := convertToHigressRoutes(*config, "production")
	if err != nil {
		t.Fatalf("❌ convertToHigressRoutes FAILED: %v", err)
	}

	output, _ := json.MarshalIndent(routes, "", "  ")

	fmt.Println("📤 真实的 Higress 路由输出:")
	fmt.Printf("```json\n%s\n```\n", string(output))

	if len(routes) != 2 {
		t.Errorf("Expected 2 Higress routes, but got %d", len(routes))
	}
	if routes[0].Service != "user-service" {
		t.Errorf("Expected first route service to be 'user-service', but got '%s'", routes[0].Service)
	}

	fmt.Println("✅ 真实转换成功！")
}

func TestRealLuaPluginParsing(t *testing.T) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🧪 测试 3: 真实 Lua 插件解析")
	fmt.Println(strings.Repeat("-", 80))

	fmt.Println("📥 输入的 Lua 插件配置:")
	fmt.Printf("```lua\n%s\n```\n", testLuaPlugin)

	// REAL FUNCTION CALL
	plugin, err := parseNginxLuaPlugin(testLuaPlugin)
	if err != nil {
		t.Fatalf("❌ parseNginxLuaPlugin FAILED: %v", err)
	}

	output, _ := json.MarshalIndent(plugin, "", "  ")

	fmt.Println("📤 真实的解析输出:")
	fmt.Printf("```json\n%s\n```\n", string(output))

	if plugin.Phase != "access" {
		t.Errorf("Expected phase 'access', but got '%s'", plugin.Phase)
	}
	if _, exists := plugin.Config["auth_token"]; !exists {
		t.Error("Expected config 'auth_token' to be parsed")
	}

	fmt.Println("✅ 真实解析成功！")
}

func TestRealLuaToWasmConversionAndYaml(t *testing.T) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🧪 测试 4: 真实 Lua -> WASM 转换及 YAML 生成")
	fmt.Println(strings.Repeat("-", 80))

	// REAL FUNCTION CALL (parse)
	plugin, err := parseNginxLuaPlugin(testLuaPlugin)
	if err != nil {
		t.Fatalf("❌ parseNginxLuaPlugin FAILED: %v", err)
	}

	// REAL FUNCTION CALL (convert)
	wasmPlugin, err := convertLuaToWasmPlugin(plugin, "rust")
	if err != nil {
		t.Fatalf("❌ convertLuaToWasmPlugin FAILED: %v", err)
	}

	wasmOutput, _ := json.MarshalIndent(wasmPlugin, "", "  ")

	fmt.Println("📤 真实的 WASM 插件输出:")
	fmt.Printf("```json\n%s\n```\n", string(wasmOutput))

	if wasmPlugin.Phase != "AUTHN" {
		t.Errorf("Expected WASM phase 'AUTHN', but got '%s'", wasmPlugin.Phase)
	}

	fmt.Println("\n" + strings.Repeat("-", 40))

	// REAL FUNCTION CALL (generate YAML)
	yamlOutput := generateWasmPluginConfig(wasmPlugin, "production")

	fmt.Println("📤 真实的 Kubernetes YAML 输出:")
	fmt.Printf("```yaml\n%s\n```\n", yamlOutput)

	if !strings.Contains(yamlOutput, "kind: WasmPlugin") {
		t.Error("YAML output is missing 'kind: WasmPlugin'")
	}
	if !strings.Contains(yamlOutput, "namespace: production") {
		t.Error("YAML output is missing 'namespace: production'")
	}

	fmt.Println("✅ 真实转换和 YAML 生成成功！")
}

func TestRealPluginCompatibilityAnalysis(t *testing.T) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🧪 测试 5: 真实插件兼容性分析")
	fmt.Println(strings.Repeat("-", 80))

	fmt.Println("📥 输入的复杂 Lua 插件:")
	fmt.Printf("```lua\n%s\n```\n", testComplexLuaPlugin)

	// REAL FUNCTION CALL (parse)
	plugin, err := parseNginxLuaPlugin(testComplexLuaPlugin)
	if err != nil {
		t.Fatalf("❌ parseNginxLuaPlugin FAILED: %v", err)
	}

	// REAL FUNCTION CALL (analyze)
	result := analyzePluginCompatibility(plugin)

	output, _ := json.MarshalIndent(result, "", "  ")

	fmt.Println("📤 真实的兼容性分析输出:")
	fmt.Printf("```json\n%s\n```\n", string(output))

	if result.CompatibilityLevel != "manual" {
		t.Errorf("Expected compatibility level 'manual', but got '%s'", result.CompatibilityLevel)
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings about ngx.shared and http calls, but got none")
	}

	fmt.Println("✅ 真实分析成功！")
}
