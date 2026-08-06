// Package tools provides Nginx configuration parsing and analysis capabilities.
// This intelligent parser extracts semantic information from Nginx configs for AI reasoning.
package tools

import (
	"fmt"
	"regexp"
	"strings"
)

// NginxConfig 表示解析后的 Nginx 配置结构
type NginxConfig struct {
	Servers   []NginxServer   `json:"servers"`
	Upstreams []NginxUpstream `json:"upstreams"`
	Raw       string          `json:"raw"`
}

// NginxServer 表示一个 server 块
type NginxServer struct {
	Listen      []string            `json:"listen"`        // 监听端口和地址
	ServerNames []string            `json:"server_names"`  // 域名列表
	Locations   []NginxLocation     `json:"locations"`     // location 块列表
	SSL         *NginxSSL           `json:"ssl,omitempty"` // SSL 配置
	Directives  map[string][]string `json:"directives"`    // 其他指令
}

// NginxLocation 表示一个 location 块
type NginxLocation struct {
	Path       string              `json:"path"`                 // 路径
	Modifier   string              `json:"modifier"`             // 修饰符（=, ~, ~*, ^~）
	ProxyPass  string              `json:"proxy_pass,omitempty"` // 代理目标
	Rewrite    []string            `json:"rewrite,omitempty"`    // rewrite 规则
	Return     *NginxReturn        `json:"return,omitempty"`     // return 指令
	Directives map[string][]string `json:"directives"`           // 其他指令
}

// NginxSSL 表示 SSL 配置
type NginxSSL struct {
	Certificate    string   `json:"certificate,omitempty"`
	CertificateKey string   `json:"certificate_key,omitempty"`
	Protocols      []string `json:"protocols,omitempty"`
	Ciphers        string   `json:"ciphers,omitempty"`
}

// NginxReturn 表示 return 指令
type NginxReturn struct {
	Code int    `json:"code"`
	URL  string `json:"url,omitempty"`
	Text string `json:"text,omitempty"`
}

// NginxUpstream 表示 upstream 块
type NginxUpstream struct {
	Name    string   `json:"name"`
	Servers []string `json:"servers"`
	Method  string   `json:"method,omitempty"` // 负载均衡方法
}

// ParseNginxConfig 解析 Nginx 配置内容
func ParseNginxConfig(content string) (*NginxConfig, error) {
	config := &NginxConfig{
		Raw:       content,
		Servers:   []NginxServer{},
		Upstreams: []NginxUpstream{},
	}

	// 解析 upstream 块
	upstreams := extractUpstreams(content)
	config.Upstreams = upstreams

	// 解析 server 块
	servers := extractServers(content)
	for _, serverContent := range servers {
		server := parseServer(serverContent)
		config.Servers = append(config.Servers, server)
	}

	return config, nil
}

// extractUpstreams 提取所有 upstream 块
func extractUpstreams(content string) []NginxUpstream {
	upstreams := []NginxUpstream{}
	upstreamRegex := regexp.MustCompile(`upstream\s+(\S+)\s*\{([^}]*)\}`)
	matches := upstreamRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			name := match[1]
			body := match[2]
			upstream := NginxUpstream{
				Name:    name,
				Servers: []string{},
			}

			// 提取 server 指令
			serverRegex := regexp.MustCompile(`server\s+([^;]+);`)
			serverMatches := serverRegex.FindAllStringSubmatch(body, -1)
			for _, sm := range serverMatches {
				if len(sm) >= 2 {
					upstream.Servers = append(upstream.Servers, strings.TrimSpace(sm[1]))
				}
			}

			// 检测负载均衡方法
			if strings.Contains(body, "ip_hash") {
				upstream.Method = "ip_hash"
			} else if strings.Contains(body, "least_conn") {
				upstream.Method = "least_conn"
			}

			upstreams = append(upstreams, upstream)
		}
	}

	return upstreams
}

// extractServers 提取所有 server 块的内容
func extractServers(content string) []string {
	servers := []string{}

	// 简单的大括号匹配提取
	lines := strings.Split(content, "\n")
	inServer := false
	braceCount := 0
	var currentServer strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 检测 server 块开始
		if strings.HasPrefix(trimmed, "server") && strings.Contains(trimmed, "{") {
			inServer = true
			currentServer.Reset()
			currentServer.WriteString(line + "\n")
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
			continue
		}

		if inServer {
			currentServer.WriteString(line + "\n")
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			if braceCount == 0 {
				servers = append(servers, currentServer.String())
				inServer = false
			}
		}
	}

	return servers
}

// parseServer 解析单个 server 块
func parseServer(content string) NginxServer {
	server := NginxServer{
		Listen:      []string{},
		ServerNames: []string{},
		Locations:   []NginxLocation{},
		Directives:  make(map[string][]string),
	}

	lines := strings.Split(content, "\n")

	// 解析 listen 指令
	listenRegex := regexp.MustCompile(`^\s*listen\s+([^;]+);`)
	for _, line := range lines {
		if match := listenRegex.FindStringSubmatch(line); match != nil {
			server.Listen = append(server.Listen, strings.TrimSpace(match[1]))
		}
	}

	// 解析 server_name 指令
	serverNameRegex := regexp.MustCompile(`^\s*server_name\s+([^;]+);`)
	for _, line := range lines {
		if match := serverNameRegex.FindStringSubmatch(line); match != nil {
			names := strings.Fields(match[1])
			server.ServerNames = append(server.ServerNames, names...)
		}
	}

	// 解析 SSL 配置
	server.SSL = parseSSL(content)

	// 解析 location 块
	server.Locations = extractLocations(content)

	// 解析其他常见指令
	commonDirectives := []string{
		"root", "index", "access_log", "error_log",
		"client_max_body_size", "proxy_set_header",
	}

	for _, directive := range commonDirectives {
		pattern := fmt.Sprintf(`(?m)^\s*%s\s+([^;]+);`, directive)
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				server.Directives[directive] = append(server.Directives[directive], strings.TrimSpace(match[1]))
			}
		}
	}

	return server
}

// parseSSL 解析 SSL 配置
func parseSSL(content string) *NginxSSL {
	hasSSL := strings.Contains(content, "ssl") || strings.Contains(content, "443")
	if !hasSSL {
		return nil
	}

	ssl := &NginxSSL{
		Protocols: []string{},
	}

	// 提取证书路径
	certRegex := regexp.MustCompile(`ssl_certificate\s+([^;]+);`)
	if match := certRegex.FindStringSubmatch(content); match != nil {
		ssl.Certificate = strings.TrimSpace(match[1])
	}

	// 提取私钥路径
	keyRegex := regexp.MustCompile(`ssl_certificate_key\s+([^;]+);`)
	if match := keyRegex.FindStringSubmatch(content); match != nil {
		ssl.CertificateKey = strings.TrimSpace(match[1])
	}

	// 提取协议
	protocolRegex := regexp.MustCompile(`ssl_protocols\s+([^;]+);`)
	if match := protocolRegex.FindStringSubmatch(content); match != nil {
		ssl.Protocols = strings.Fields(match[1])
	}

	// 提取加密套件
	cipherRegex := regexp.MustCompile(`ssl_ciphers\s+([^;]+);`)
	if match := cipherRegex.FindStringSubmatch(content); match != nil {
		ssl.Ciphers = strings.TrimSpace(match[1])
	}

	return ssl
}

// extractLocations 提取所有 location 块
func extractLocations(content string) []NginxLocation {
	locations := []NginxLocation{}

	// 匹配 location 块
	locationRegex := regexp.MustCompile(`location\s+(=|~|~\*|\^~)?\s*([^\s{]+)\s*\{([^}]*)\}`)
	matches := locationRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			modifier := strings.TrimSpace(match[1])
			path := strings.TrimSpace(match[2])
			body := match[3]

			location := NginxLocation{
				Path:       path,
				Modifier:   modifier,
				Rewrite:    []string{},
				Directives: make(map[string][]string),
			}

			// 提取 proxy_pass
			proxyPassRegex := regexp.MustCompile(`proxy_pass\s+([^;]+);`)
			if ppMatch := proxyPassRegex.FindStringSubmatch(body); ppMatch != nil {
				location.ProxyPass = strings.TrimSpace(ppMatch[1])
			}

			// 提取 rewrite 规则
			rewriteRegex := regexp.MustCompile(`rewrite\s+([^;]+);`)
			rewriteMatches := rewriteRegex.FindAllStringSubmatch(body, -1)
			for _, rm := range rewriteMatches {
				if len(rm) >= 2 {
					location.Rewrite = append(location.Rewrite, strings.TrimSpace(rm[1]))
				}
			}

			// 提取 return 指令
			returnRegex := regexp.MustCompile(`return\s+(\d+)(?:\s+([^;]+))?;`)
			if retMatch := returnRegex.FindStringSubmatch(body); retMatch != nil {
				code := 0
				fmt.Sscanf(retMatch[1], "%d", &code)
				location.Return = &NginxReturn{
					Code: code,
				}
				if len(retMatch) >= 3 {
					urlOrText := strings.TrimSpace(retMatch[2])
					if strings.HasPrefix(urlOrText, "http") {
						location.Return.URL = urlOrText
					} else {
						location.Return.Text = urlOrText
					}
				}
			}

			// 提取其他指令
			commonDirectives := []string{
				"proxy_set_header", "proxy_redirect", "proxy_read_timeout",
				"add_header", "alias", "root", "try_files",
			}

			for _, directive := range commonDirectives {
				pattern := fmt.Sprintf(`(?m)^\s*%s\s+([^;]+);`, directive)
				regex := regexp.MustCompile(pattern)
				matches := regex.FindAllStringSubmatch(body, -1)
				for _, m := range matches {
					if len(m) >= 2 {
						location.Directives[directive] = append(location.Directives[directive], strings.TrimSpace(m[1]))
					}
				}
			}

			locations = append(locations, location)
		}
	}

	return locations
}

// AnalyzeNginxConfig 分析 Nginx 配置，生成用于 AI 的分析报告
func AnalyzeNginxConfig(config *NginxConfig) *NginxAnalysis {
	analysis := &NginxAnalysis{
		ServerCount: len(config.Servers),
		Features:    make(map[string]bool),
		Complexity:  "simple",
		Suggestions: []string{},
	}

	totalLocations := 0
	hasSSL := false
	hasRewrite := false
	hasUpstream := len(config.Upstreams) > 0
	hasComplexRouting := false
	uniqueDomains := make(map[string]bool)

	for _, server := range config.Servers {
		// 统计域名
		for _, name := range server.ServerNames {
			uniqueDomains[name] = true
		}

		// 统计 location
		totalLocations += len(server.Locations)

		// 检测 SSL
		if server.SSL != nil {
			hasSSL = true
			analysis.Features["ssl"] = true
		}

		// 检测 location 特性
		for _, loc := range server.Locations {
			if loc.ProxyPass != "" {
				analysis.Features["proxy"] = true
			}
			if len(loc.Rewrite) > 0 {
				hasRewrite = true
				analysis.Features["rewrite"] = true
			}
			if loc.Return != nil {
				analysis.Features["return"] = true
				if loc.Return.Code >= 300 && loc.Return.Code < 400 {
					analysis.Features["redirect"] = true
				}
			}
			if loc.Modifier != "" {
				hasComplexRouting = true
				analysis.Features["complex_routing"] = true
			}
			// 检测其他指令
			if _, ok := loc.Directives["proxy_set_header"]; ok {
				analysis.Features["header_manipulation"] = true
			}
			if _, ok := loc.Directives["add_header"]; ok {
				analysis.Features["response_headers"] = true
			}
		}
	}

	analysis.LocationCount = totalLocations
	analysis.DomainCount = len(uniqueDomains)

	// 判断复杂度
	if analysis.ServerCount > 3 || totalLocations > 10 || (hasRewrite && hasSSL && hasComplexRouting) {
		analysis.Complexity = "high"
	} else if analysis.ServerCount > 1 || totalLocations > 5 || hasRewrite || hasSSL || hasUpstream {
		analysis.Complexity = "medium"
	}

	// 生成建议
	if analysis.Features["proxy"] {
		analysis.Suggestions = append(analysis.Suggestions, "proxy_pass 将转换为 Ingress/HTTPRoute 的 backend 配置")
	}
	if analysis.Features["rewrite"] {
		analysis.Suggestions = append(analysis.Suggestions, "rewrite 规则需要使用 Higress 注解实现，如 higress.io/rewrite-target")
	}
	if analysis.Features["ssl"] {
		analysis.Suggestions = append(analysis.Suggestions, "SSL 证书需要创建 Kubernetes Secret，并在 Ingress 中引用")
	}
	if analysis.Features["redirect"] {
		analysis.Suggestions = append(analysis.Suggestions, "redirect 可以使用 Higress 的重定向注解或插件实现")
	}
	if hasUpstream {
		analysis.Suggestions = append(analysis.Suggestions, "upstream 负载均衡将由 Kubernetes Service 和 Endpoints 实现")
	}
	if analysis.Features["header_manipulation"] {
		analysis.Suggestions = append(analysis.Suggestions, "请求头操作可以使用 Higress 注解或 custom-response 插件实现")
	}

	return analysis
}

// NginxAnalysis 表示 Nginx 配置分析结果
type NginxAnalysis struct {
	ServerCount   int             `json:"server_count"`
	LocationCount int             `json:"location_count"`
	DomainCount   int             `json:"domain_count"`
	Features      map[string]bool `json:"features"`
	Complexity    string          `json:"complexity"` // simple, medium, high
	Suggestions   []string        `json:"suggestions"`
}
