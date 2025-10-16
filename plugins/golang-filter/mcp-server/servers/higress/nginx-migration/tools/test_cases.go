// Package tools provides test cases for nginx migration functionality
// Added on 2025.9.29 - Test cases for validation
package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Test data for Nginx configuration
const sampleNginxConfig = `
server {
    listen 80;
    listen 443 ssl;
    server_name example.com www.example.com;
    
    root /var/www/html;
    index index.html index.php;
    
    location / {
        try_files $uri $uri/ =404;
    }
    
    location /api {
        proxy_pass http://backend-service:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        rewrite ^/api/(.*) /$1 break;
    }
    
    location /static {
        root /var/www/static;
        expires 1y;
    }
    
    location ~ \.php$ {
        fastcgi_pass php-fpm:9000;
        fastcgi_index index.php;
        include fastcgi_params;
    }
}

server {
    listen 8080;
    server_name admin.example.com;
    
    location / {
        proxy_pass http://admin-backend:3000;
        proxy_set_header Authorization "Bearer secret";
    }
}
`

// Test data for Nginx Lua plugin
const sampleLuaPlugin = `
# Rate limiting plugin
lua_package_path "/usr/local/openresty/lualib/resty/?.lua;;";

access_by_lua_block {
    local limit_req = require "resty.limit.req"
    local redis = require "resty.redis"
    
    -- Rate limiting configuration
    local lim, err = limit_req.new("my_limit_req_store", 200, 100)
    if not lim then
        ngx.log(ngx.ERR, "failed to instantiate a resty.limit.req object: ", err)
        return ngx.exit(500)
    end
    
    -- Get client IP
    local key = ngx.var.binary_remote_addr
    local delay, err = lim:incoming(key, true)
    
    if not delay then
        if err == "rejected" then
            ngx.header["X-RateLimit-Limit"] = "200"
            ngx.header["X-RateLimit-Remaining"] = "0"
            return ngx.exit(429)
        end
        ngx.log(ngx.ERR, "failed to limit req: ", err)
        return ngx.exit(500)
    end
    
    -- Set rate limit headers
    ngx.header["X-RateLimit-Limit"] = "200"
    ngx.header["X-RateLimit-Remaining"] = "100"
    
    if delay >= 0.001 then
        ngx.sleep(delay)
    end
}

set $rate_limit_key "default";
set $rate_limit_window 60;
`

// Test data for complex Lua plugin with multiple phases
const complexLuaPlugin = `
# Authentication and logging plugin
lua_shared_dict user_cache 10m;
lua_shared_dict stats_cache 5m;

init_by_lua_block {
    require "resty.core"
    local http = require "resty.http"
    local cjson = require "cjson"
}

access_by_lua_block {
    local http = require "resty.http"
    local cjson = require "cjson"
    local user_cache = ngx.shared.user_cache
    
    -- Extract JWT token
    local auth_header = ngx.var.http_authorization
    if not auth_header then
        ngx.status = 401
        ngx.say('{"error": "Missing authorization header"}')
        return ngx.exit(401)
    end
    
    local token = string.match(auth_header, "Bearer%s+(.+)")
    if not token then
        ngx.status = 401
        ngx.say('{"error": "Invalid authorization format"}')
        return ngx.exit(401)
    end
    
    -- Check cache first
    local user_info = user_cache:get(token)
    if user_info then
        ngx.var.user_id = user_info
        return
    end
    
    -- Validate with auth service
    local httpc = http.new()
    httpc:set_timeout(5000)
    
    local res, err = httpc:request_uri("http://auth-service:8080/validate", {
        method = "POST",
        body = cjson.encode({token = token}),
        headers = {
            ["Content-Type"] = "application/json",
        }
    })
    
    if not res then
        ngx.log(ngx.ERR, "Auth service error: ", err)
        return ngx.exit(500)
    end
    
    if res.status ~= 200 then
        ngx.status = 401
        ngx.say('{"error": "Invalid token"}')
        return ngx.exit(401)
    end
    
    local user_data = cjson.decode(res.body)
    user_cache:set(token, user_data.user_id, 300)
    ngx.var.user_id = user_data.user_id
}

header_filter_by_lua_block {
    -- Add response headers
    ngx.header["X-Request-ID"] = ngx.var.request_id
    ngx.header["X-User-ID"] = ngx.var.user_id
}

log_by_lua_block {
    local stats_cache = ngx.shared.stats_cache
    local user_id = ngx.var.user_id
    
    if user_id then
        local key = "user_requests:" .. user_id
        local count = stats_cache:get(key) or 0
        stats_cache:set(key, count + 1, 3600)
    end
}

set $user_id "";
`

// TestNginxConfigParsing tests the basic nginx configuration parsing
func TestNginxConfigParsing() error {
	fmt.Println("=== Testing Nginx Configuration Parsing ===")

	config, err := parseNginxConfig(sampleNginxConfig)
	if err != nil {
		return fmt.Errorf("failed to parse nginx config: %w", err)
	}

	fmt.Printf("Parsed %d server blocks\n", len(config.ServerBlocks))

	for i, server := range config.ServerBlocks {
		fmt.Printf("Server %d:\n", i+1)
		fmt.Printf("  Listen: %v\n", server.Listen)
		fmt.Printf("  Server Names: %v\n", server.ServerName)
		fmt.Printf("  Locations: %d\n", len(server.Location))

		for j, location := range server.Location {
			fmt.Printf("    Location %d: %s -> %s\n", j+1, location.Path, location.ProxyPass)
		}
		fmt.Println()
	}

	return nil
}

// TestNginxToHigressConversion tests the conversion from nginx to higress format
func TestNginxToHigressConversion() error {
	fmt.Println("=== Testing Nginx to Higress Conversion ===")

	config, err := parseNginxConfig(sampleNginxConfig)
	if err != nil {
		return fmt.Errorf("failed to parse nginx config: %w", err)
	}

	routes, err := convertToHigressRoutes(*config, "test-namespace")
	if err != nil {
		return fmt.Errorf("failed to convert to higress routes: %w", err)
	}

	fmt.Printf("Generated %d Higress routes:\n", len(routes))

	for i, route := range routes {
		fmt.Printf("Route %d:\n", i+1)
		fmt.Printf("  Name: %s\n", route.Name)
		fmt.Printf("  Host: %s\n", route.Host)
		fmt.Printf("  Path: %s\n", route.Path)
		fmt.Printf("  Service: %s:%d\n", route.Service, route.Port)
		if route.Rewrite != nil {
			fmt.Printf("  Rewrite: %s\n", route.Rewrite.Path)
		}
		fmt.Println()
	}

	return nil
}

// TestLuaPluginParsing tests lua plugin parsing functionality
func TestLuaPluginParsing() error {
	fmt.Println("=== Testing Lua Plugin Parsing ===")

	// Test simple plugin
	fmt.Println("--- Simple Rate Limiting Plugin ---")
	plugin, err := parseNginxLuaPlugin(sampleLuaPlugin)
	if err != nil {
		return fmt.Errorf("failed to parse simple lua plugin: %w", err)
	}

	pluginJSON, _ := json.MarshalIndent(plugin, "", "  ")
	fmt.Printf("Parsed Plugin:\n%s\n\n", string(pluginJSON))

	// Test complex plugin
	fmt.Println("--- Complex Authentication Plugin ---")
	complexPlugin, err := parseNginxLuaPlugin(complexLuaPlugin)
	if err != nil {
		return fmt.Errorf("failed to parse complex lua plugin: %w", err)
	}

	complexJSON, _ := json.MarshalIndent(complexPlugin, "", "  ")
	fmt.Printf("Parsed Complex Plugin:\n%s\n\n", string(complexJSON))

	return nil
}

// TestLuaToWasmConversion tests lua to wasm plugin conversion
func TestLuaToWasmConversion() error {
	fmt.Println("=== Testing Lua to WASM Conversion ===")

	plugin, err := parseNginxLuaPlugin(sampleLuaPlugin)
	if err != nil {
		return fmt.Errorf("failed to parse lua plugin: %w", err)
	}

	// Test different target languages
	languages := []string{"rust", "go", "cpp", "assemblyscript"}

	for _, lang := range languages {
		fmt.Printf("--- Converting to %s ---\n", lang)

		wasmPlugin, err := convertLuaToWasmPlugin(plugin, lang)
		if err != nil {
			return fmt.Errorf("failed to convert to %s wasm plugin: %w", lang, err)
		}

		wasmJSON, _ := json.MarshalIndent(wasmPlugin, "", "  ")
		fmt.Printf("WASM Plugin (%s):\n%s\n\n", lang, string(wasmJSON))

		// Test YAML generation
		yamlContent := generateWasmPluginConfig(wasmPlugin, "test-namespace")
		fmt.Printf("Generated YAML for %s:\n%s\n", lang, yamlContent)
	}

	return nil
}

// TestPluginCompatibilityAnalysis tests plugin compatibility analysis
func TestPluginCompatibilityAnalysis() error {
	fmt.Println("=== Testing Plugin Compatibility Analysis ===")

	testPlugins := []string{sampleLuaPlugin, complexLuaPlugin}
	pluginNames := []string{"Simple Rate Limiting", "Complex Authentication"}

	for i, pluginConfig := range testPlugins {
		fmt.Printf("--- Analyzing %s Plugin ---\n", pluginNames[i])

		plugin, err := parseNginxLuaPlugin(pluginConfig)
		if err != nil {
			return fmt.Errorf("failed to parse plugin %d: %w", i+1, err)
		}

		result := analyzePluginCompatibility(plugin)

		fmt.Printf("Compatibility Level: %s\n", result.CompatibilityLevel)

		if len(result.MigrationNotes) > 0 {
			fmt.Println("Migration Notes:")
			for _, note := range result.MigrationNotes {
				fmt.Printf("  - %s\n", note)
			}
		}

		if len(result.RequiredChanges) > 0 {
			fmt.Println("Required Changes:")
			for _, change := range result.RequiredChanges {
				fmt.Printf("  - %s\n", change)
			}
		}

		if len(result.Warnings) > 0 {
			fmt.Println("Warnings:")
			for _, warning := range result.Warnings {
				fmt.Printf("  - ⚠️  %s\n", warning)
			}
		}

		fmt.Println()
	}

	return nil
}

// TestMigrationReport tests migration report generation
func TestMigrationReport() error {
	fmt.Println("=== Testing Migration Report Generation ===")

	var plugins []NginxLuaPlugin

	// Parse test plugins
	plugin1, err := parseNginxLuaPlugin(sampleLuaPlugin)
	if err != nil {
		return fmt.Errorf("failed to parse plugin 1: %w", err)
	}
	plugins = append(plugins, *plugin1)

	plugin2, err := parseNginxLuaPlugin(complexLuaPlugin)
	if err != nil {
		return fmt.Errorf("failed to parse plugin 2: %w", err)
	}
	plugins = append(plugins, *plugin2)

	report := generatePluginMigrationReport(plugins)
	fmt.Printf("Migration Report:\n%s\n", report)

	return nil
}

// RunAllTests executes all test cases
func RunAllTests() {
	fmt.Println("🚀 Starting Nginx Migration Tools Test Suite")
	fmt.Println(strings.Repeat("=", 60))

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Nginx Configuration Parsing", TestNginxConfigParsing},
		{"Nginx to Higress Conversion", TestNginxToHigressConversion},
		{"Lua Plugin Parsing", TestLuaPluginParsing},
		{"Lua to WASM Conversion", TestLuaToWasmConversion},
		{"Plugin Compatibility Analysis", TestPluginCompatibilityAnalysis},
		{"Migration Report Generation", TestMigrationReport},
	}

	passed := 0
	failed := 0

	for _, test := range tests {
		fmt.Printf("\n🧪 Running: %s\n", test.name)
		if err := test.fn(); err != nil {
			fmt.Printf("❌ FAILED: %s - %v\n", test.name, err)
			failed++
		} else {
			fmt.Printf("✅ PASSED: %s\n", test.name)
			passed++
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("📊 Test Results: %d passed, %d failed\n", passed, failed)

	if failed > 0 {
		os.Exit(1)
	}
}

// TestBasicMigrationTools tests basic migration functionality without LLM dependency
func TestBasicMigrationTools() error {
	fmt.Println("=== Testing Basic Migration Tools ===")

	// Test nginx config parsing
	testConfig := `server {
		listen 80;
		server_name example.com;
		location / {
			proxy_pass http://backend:8080;
		}
	}`

	config, err := ParseNginxConfig(testConfig)
	if err != nil {
		return fmt.Errorf("failed to parse nginx config: %v", err)
	}

	if len(config.ServerBlocks) == 0 {
		return fmt.Errorf("no server blocks found in parsed config")
	}

	fmt.Printf("✅ Basic migration tools working correctly\n")
	fmt.Printf("   Server blocks found: %d\n", len(config.ServerBlocks))

	// Note: We're not making actual API calls to avoid costs and dependencies
	fmt.Println("ℹ️  LLM integration ready for use with valid API key")

	return nil
}
