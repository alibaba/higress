//go:build higress_integration
// +build higress_integration

package mcptools

import (
	"fmt"
	"strings"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
)

// RegisterNginxConfigTools registers Nginx configuration analysis and conversion tools
func RegisterNginxConfigTools(server *common.MCPServer, ctx *MigrationContext) {
	// Tool 1: Parse and analyze Nginx configuration
	server.RegisterTool(common.NewTool(
		"parse_nginx_config",
		"解析和分析 Nginx 配置文件，识别配置结构和复杂度",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"config_content": map[string]interface{}{
					"type":        "string",
					"description": "Nginx 配置文件内容",
				},
			},
			"required": []string{"config_content"},
		},
		func(args map[string]interface{}) (string, error) {
			return parseNginxConfig(args, ctx)
		},
	))

	// Tool 2: Convert Nginx config to Higress HTTPRoute
	server.RegisterTool(common.NewTool(
		"convert_to_higress",
		"将 Nginx 配置转换为 Higress HTTPRoute 和 Service 资源",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"config_content": map[string]interface{}{
					"type":        "string",
					"description": "Nginx 配置文件内容",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "目标 Kubernetes 命名空间",
					"default":     "default",
				},
			},
			"required": []string{"config_content"},
		},
		func(args map[string]interface{}) (string, error) {
			return convertToHigress(args, ctx)
		},
	))
}

func parseNginxConfig(args map[string]interface{}, ctx *MigrationContext) (string, error) {
	configContent, ok := args["config_content"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid config_content parameter")
	}

	// Simple analysis
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

	analysis := fmt.Sprintf(`Nginx配置分析结果

基础信息:
- Server块: %d个
- Location块: %d个  
- SSL配置: %t
- 反向代理: %t
- URL重写: %t

复杂度: %s

迁移建议:`, serverCount, locationCount, hasSSL, hasProxy, hasRewrite, complexity)

	if hasProxy {
		analysis += "\n- 反向代理将转换为HTTPRoute backendRefs"
	}
	if hasRewrite {
		analysis += "\n- URL重写将使用URLRewrite过滤器"
	}
	if hasSSL {
		analysis += "\n- SSL配置需要迁移到Gateway资源"
	}

	return analysis, nil
}

func convertToHigress(args map[string]interface{}, ctx *MigrationContext) (string, error) {
	configContent, ok := args["config_content"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid config_content parameter")
	}

	namespace := ctx.DefaultNamespace
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	// Extract hostname
	hostname := ctx.DefaultHostname
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

	// Generate route name
	routeName := generateRouteName(hostname, ctx)
	serviceName := generateServiceName(hostname, ctx)

	yamlConfig := fmt.Sprintf(`转换后的Higress配置

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
        value: /
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

转换完成

应用步骤:
1. 保存为 higress-config.yaml
2. 执行: kubectl apply -f higress-config.yaml
3. 验证: kubectl get httproute -n %s`,
		routeName, namespace,
		ctx.GatewayName, ctx.GatewayNamespace, hostname,
		serviceName, ctx.ServicePort,
		serviceName, namespace,
		ctx.ServicePort, ctx.TargetPort, namespace)

	return yamlConfig, nil
}

func generateRouteName(hostname string, ctx *MigrationContext) string {
	prefix := "nginx-migrated"
	if ctx.RoutePrefix != "" {
		prefix = ctx.RoutePrefix
	}

	if hostname == "" || hostname == ctx.DefaultHostname {
		return fmt.Sprintf("%s-route", prefix)
	}
	// Replace dots and special characters for valid k8s name
	safeName := hostname
	for _, char := range []string{".", "_", ":"} {
		safeName = strings.ReplaceAll(safeName, char, "-")
	}
	return fmt.Sprintf("%s-%s", prefix, safeName)
}

func generateServiceName(hostname string, ctx *MigrationContext) string {
	if hostname == "" || hostname == ctx.DefaultHostname {
		return "backend-service"
	}
	safeName := hostname
	for _, char := range []string{".", "_", ":"} {
		safeName = strings.ReplaceAll(safeName, char, "-")
	}
	return fmt.Sprintf("%s-service", safeName)
}
