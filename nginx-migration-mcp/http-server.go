// HTTP API Server for Nginx Migration Tools
// 允许通过HTTP API在远程机器上使用nginx迁移功能
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// HTTP API 请求结构
type APIRequest struct {
	ConfigContent  string `json:"config_content,omitempty"`
	LuaCode        string `json:"lua_code,omitempty"`
	Namespace      string `json:"namespace,omitempty"`
	TargetLanguage string `json:"target_language,omitempty"`
}

type APIResponse struct {
	Success bool   `json:"success"`
	Data    string `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type HTTPServer struct {
	port   string
	config *ServerConfig
}

func NewHTTPServer(port string, config *ServerConfig) *HTTPServer {
	return &HTTPServer{port: port, config: config}
}

func (s *HTTPServer) Start() {
	// 注册API路由
	http.HandleFunc("/api/parse-nginx", s.handleParseNginx)
	http.HandleFunc("/api/convert-to-higress", s.handleConvertToHigress)
	http.HandleFunc("/api/analyze-lua", s.handleAnalyzeLua)
	http.HandleFunc("/health", s.handleHealth)

	// 静态文件服务 (API文档)
	http.HandleFunc("/", s.handleDocs)

	log.Printf("🚀 Nginx迁移HTTP API服务器启动于端口 %s", s.port)
	log.Printf("📋 API文档: http://localhost:%s", s.port)
	log.Printf("🔍 健康检查: http://localhost:%s/health", s.port)

	if err := http.ListenAndServe(":"+s.port, nil); err != nil {
		log.Fatalf("❌ 服务器启动失败: %v", err)
	}
}

func (s *HTTPServer) handleParseNginx(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "POST" {
		s.sendError(w, http.StatusMethodNotAllowed, "只支持POST方法")
		return
	}

	var req APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "JSON解析失败: "+err.Error())
		return
	}

	if req.ConfigContent == "" {
		s.sendError(w, http.StatusBadRequest, "缺少config_content参数")
		return
	}

	// 使用原有的解析逻辑
	result := s.parseNginxConfig(req.ConfigContent)
	s.sendSuccess(w, result)
}

func (s *HTTPServer) handleConvertToHigress(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "POST" {
		s.sendError(w, http.StatusMethodNotAllowed, "只支持POST方法")
		return
	}

	var req APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "JSON解析失败: "+err.Error())
		return
	}

	if req.ConfigContent == "" {
		s.sendError(w, http.StatusBadRequest, "缺少config_content参数")
		return
	}

	namespace := s.config.Defaults.Namespace
	if req.Namespace != "" {
		namespace = req.Namespace
	}

	result := s.convertToHigress(req.ConfigContent, namespace)
	s.sendSuccess(w, result)
}

func (s *HTTPServer) handleAnalyzeLua(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "POST" {
		s.sendError(w, http.StatusMethodNotAllowed, "只支持POST方法")
		return
	}

	var req APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "JSON解析失败: "+err.Error())
		return
	}

	if req.LuaCode == "" {
		s.sendError(w, http.StatusBadRequest, "缺少lua_code参数")
		return
	}

	result := s.analyzeLuaPlugin(req.LuaCode)
	s.sendSuccess(w, result)
}

func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w)
	response := map[string]interface{}{
		"status":  "healthy",
		"service": s.config.Server.Name,
		"version": s.config.Server.Version,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *HTTPServer) handleDocs(w http.ResponseWriter, r *http.Request) {
	docs := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Nginx迁移API文档</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }
        .api { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .method { color: #fff; padding: 2px 8px; border-radius: 3px; margin-right: 10px; }
        .post { background: #28a745; }
        .get { background: #007bff; }
        pre { background: #f8f9fa; padding: 10px; border-radius: 3px; overflow-x: auto; }
        h1 { color: #333; } h2 { color: #666; }
    </style>
</head>
<body>
    <h1>🚀 Nginx迁移HTTP API</h1>
    <p>提供Nginx配置解析、转换和Lua插件分析功能</p>
    
    <div class="api">
        <h2><span class="method post">POST</span>/api/parse-nginx</h2>
        <p><strong>功能:</strong> 解析和分析Nginx配置文件</p>
        <pre>{
  "config_content": "server { listen 80; server_name example.com; }"
}</pre>
    </div>
    
    <div class="api">
        <h2><span class="method post">POST</span>/api/convert-to-higress</h2>
        <p><strong>功能:</strong> 转换Nginx配置为Higress HTTPRoute格式</p>
        <pre>{
  "config_content": "server { listen 80; server_name example.com; }",
  "namespace": "production"
}</pre>
    </div>
    
    <div class="api">
        <h2><span class="method post">POST</span>/api/analyze-lua</h2>
        <p><strong>功能:</strong> 分析Nginx Lua插件兼容性</p>
        <pre>{
  "lua_code": "access_by_lua_block { ngx.say('hello') }"
}</pre>
    </div>
    
    <div class="api">
        <h2><span class="method get">GET</span>/health</h2>
        <p><strong>功能:</strong> 健康检查接口</p>
    </div>
    
    <h2>📋 使用示例</h2>
    <pre>curl -X POST http://localhost:8080/api/parse-nginx \
  -H "Content-Type: application/json" \
  -d '{"config_content": "server { listen 80; server_name test.com; }"}'</pre>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, docs)
}

func (s *HTTPServer) enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func (s *HTTPServer) sendSuccess(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

func (s *HTTPServer) sendError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}

// 复用原有的nginx解析逻辑
func (s *HTTPServer) parseNginxConfig(configContent string) string {
	serverCount := strings.Count(configContent, "server {")
	locationCount := strings.Count(configContent, "location")
	hasSSL := strings.Contains(configContent, "ssl")
	hasProxy := strings.Contains(configContent, "proxy_pass")
	hasRewrite := strings.Contains(configContent, "rewrite")

	complexity := "Simple"
	if serverCount > 1 || (hasRewrite && hasSSL) {
		complexity = "Complex"
	} else if hasRewrite || hasSSL {
		complexity = "Medium"
	}

	return fmt.Sprintf(`🔍 Nginx配置分析结果

📊 基础信息:
- Server块: %d个
- Location块: %d个
- SSL配置: %t
- 反向代理: %t
- URL重写: %t

📈 复杂度: %s

🎯 迁移建议:
%s`, serverCount, locationCount, hasSSL, hasProxy, hasRewrite, complexity, s.getMigrationAdvice(hasProxy, hasRewrite, hasSSL))
}

func (s *HTTPServer) convertToHigress(configContent, namespace string) string {
	hostname := s.config.Defaults.Hostname
	lines := strings.Split(configContent, "\n")
	for _, line := range lines {
		if strings.Contains(line, "server_name") && !strings.Contains(line, "#") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				hostname = strings.TrimSuffix(parts[1], ";")
				break
			}
		}
	}

	return fmt.Sprintf(`🚀 转换后的Higress配置

apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: %s
  namespace: %s
  annotations:
    higress.io/migrated-from: "nginx"
spec:
  parentRefs:
  - name: %s
    namespace: %s
  hostnames:
  - %s
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: %s
    backendRefs:
    - name: %s
      port: %d

---
apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  selector:
    app: backend
  ports:
  - port: %d
    targetPort: %d

✅ 转换完成！可以使用kubectl apply应用配置。`,
		s.config.GenerateRouteName(hostname), namespace,
		s.config.Gateway.Name, s.config.Gateway.Namespace, hostname, s.config.Defaults.PathPrefix,
		s.config.GenerateServiceName(hostname), s.config.Service.DefaultPort,
		s.config.GenerateServiceName(hostname), namespace,
		s.config.Service.DefaultPort, s.config.Service.DefaultTarget)
}

func (s *HTTPServer) analyzeLuaPlugin(luaCode string) string {
	features := []string{}
	warnings := []string{}

	if strings.Contains(luaCode, "ngx.var") {
		features = append(features, "✓ ngx.var - Nginx变量")
	}
	if strings.Contains(luaCode, "ngx.req") {
		features = append(features, "✓ ngx.req - 请求API")
	}
	if strings.Contains(luaCode, "ngx.exit") {
		features = append(features, "✓ ngx.exit - 请求终止")
	}
	if strings.Contains(luaCode, "ngx.shared") {
		features = append(features, "⚠️ ngx.shared - 共享字典")
		warnings = append(warnings, "共享字典需要外部缓存替换")
	}
	if strings.Contains(luaCode, "ngx.location.capture") {
		features = append(features, "⚠️ ngx.location.capture - 内部请求")
		warnings = append(warnings, "需要改为HTTP客户端调用")
	}

	compatibility := "full"
	if len(warnings) > 0 {
		compatibility = "partial"
	}
	if len(warnings) > 2 {
		compatibility = "manual"
	}

	advice := s.getCompatibilityAdvice(compatibility)

	return fmt.Sprintf(`🔍 Lua插件兼容性分析

📊 检测特性:
%s

⚠️ 兼容性警告:
%s

📈 兼容性级别: %s

💡 迁移建议:
%s`, strings.Join(features, "\n"), strings.Join(warnings, "\n"), compatibility, advice)
}

func (s *HTTPServer) getMigrationAdvice(hasProxy, hasRewrite, hasSSL bool) string {
	advice := []string{}
	if hasProxy {
		advice = append(advice, "✓ 反向代理将转换为HTTPRoute backendRefs")
	}
	if hasRewrite {
		advice = append(advice, "✓ URL重写将使用URLRewrite过滤器")
	}
	if hasSSL {
		advice = append(advice, "✓ SSL配置需要迁移到Gateway资源")
	}
	if len(advice) == 0 {
		advice = append(advice, "✓ 基础配置可以直接转换")
	}
	return strings.Join(advice, "\n")
}

func (s *HTTPServer) getCompatibilityAdvice(level string) string {
	switch level {
	case "full":
		return "- 可直接迁移到WASM插件\n- 预计工作量: 1-2天"
	case "partial":
		return "- 需要部分重构\n- 预计工作量: 3-5天"
	case "manual":
		return "- 需要手动重写\n- 预计工作量: 1-2周"
	default:
		return "- 需要详细评估"
	}
}

func main() {
	config := LoadConfig()
	port := config.Server.Port
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	server := NewHTTPServer(port, config)
	server.Start()
}
