// MatchRule defines a rule for matching requests
type MatchRule struct {
	// Domain pattern, supports wildcards
	MatchRuleDomain string `json:"match_rule_domain,omitempty"`
	// Path pattern to match
	MatchRulePath string `json:"match_rule_path,omitempty"`
	// Type of match rule: exact, prefix, suffix, contains, regex
	MatchRuleType string `json:"match_rule_type,omitempty"`
}

// McpServer defines the configuration for MCP (Model Context Protocol) server
type McpServer struct {
	// Flag to control whether MCP server is enabled
	Enable bool `json:"enable,omitempty"`
	// Redis Config for MCP server
	Redis *RedisConfig `json:"redis,omitempty"`
	// The suffix to be appended to SSE paths, default is "/sse"
	SsePathSuffix string `json:"sse_path_suffix,omitempty"`
	// List of SSE servers Configs
	Servers []*SSEServer `json:"servers,omitempty"`
	// List of match rules for filtering requests
	MatchList []*MatchRule `json:"match_list,omitempty"`
}

func deepCopyMcpServer(mcp *McpServer) (*McpServer, error) {
	newMcp := NewDefaultMcpServer()
	newMcp.Enable = mcp.Enable

	if mcp.Redis != nil {
		newMcp.Redis = &RedisConfig{
			Address:  mcp.Redis.Address,
			Username: mcp.Redis.Username,
			Password: mcp.Redis.Password,
			DB:       mcp.Redis.DB,
		}
	}

	newMcp.SsePathSuffix = mcp.SsePathSuffix

	if len(mcp.Servers) > 0 {
		newMcp.Servers = make([]*SSEServer, len(mcp.Servers))
		for i, server := range mcp.Servers {
			newServer := &SSEServer{
				Name: server.Name,
				Path: server.Path,
				Type: server.Type,
			}
			if server.Config != nil {
				newServer.Config = make(map[string]interface{})
				for k, v := range server.Config {
					newServer.Config[k] = v
				}
			}
			newMcp.Servers[i] = newServer
		}
	}

	if len(mcp.MatchList) > 0 {
		newMcp.MatchList = make([]*MatchRule, len(mcp.MatchList))
		for i, rule := range mcp.MatchList {
			newMcp.MatchList[i] = &MatchRule{
				MatchRuleDomain: rule.MatchRuleDomain,
				MatchRulePath:   rule.MatchRulePath,
				MatchRuleType:   rule.MatchRuleType,
			}
		}
	}

	return newMcp, nil
}

func (m *McpServerController) constructMcpServerStruct(mcp *McpServer) string {
	// 构建 servers 配置
	servers := "[]"
	if len(mcp.Servers) > 0 {
		serverConfigs := make([]string, len(mcp.Servers))
		for i, server := range mcp.Servers {
			serverConfig := fmt.Sprintf(`{
				"name": "%s",
				"path": "%s",
				"type": "%s"`,
				server.Name, server.Path, server.Type)

			if len(server.Config) > 0 {
				config, _ := json.Marshal(server.Config)
				serverConfig += fmt.Sprintf(`,
				"config": %s`, string(config))
			}

			serverConfig += "}"
			serverConfigs[i] = serverConfig
		}
		servers = fmt.Sprintf("[%s]", strings.Join(serverConfigs, ","))
	}

	// 构建 match_list 配置
	matchList := "[]"
	if len(mcp.MatchList) > 0 {
		matchConfigs := make([]string, len(mcp.MatchList))
		for i, rule := range mcp.MatchList {
			matchConfigs[i] = fmt.Sprintf(`{
				"match_rule_domain": "%s",
				"match_rule_path": "%s",
				"match_rule_type": "%s"
			}`, rule.MatchRuleDomain, rule.MatchRulePath, rule.MatchRuleType)
		}
		matchList = fmt.Sprintf("[%s]", strings.Join(matchConfigs, ","))
	}

	// 构建完整的配置结构
	structFmt := `{
		"name": "envoy.filters.http.golang",
		"typed_config": {
			"@type": "type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config",
			"value": {
				"library_id": "mcp-server",
				"library_path": "/var/lib/istio/envoy/mcp-server.so",
				"plugin_name": "mcp-server",
				"plugin_config": {
					"@type": "type.googleapis.com/xds.type.v3.TypedStruct",
					"value": {
						"redis": {
							"address": "%s",
							"username": "%s",
							"password": "%s",
							"db": %d
						},
						"sse_path_suffix": "%s",
						"servers": %s,
						"match_list": %s
					}
				}
			}
		}
	}`

	return fmt.Sprintf(structFmt,
		mcp.Redis.Address,
		mcp.Redis.Username,
		mcp.Redis.Password,
		mcp.Redis.DB,
		mcp.SsePathSuffix,
		servers,
		matchList)
} 