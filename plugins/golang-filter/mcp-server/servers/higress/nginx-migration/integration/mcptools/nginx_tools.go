//go:build higress_integration
// +build higress_integration

package mcptools

import (
	"encoding/json"
	"fmt"
	"log"
	"nginx-migration-mcp/internal/rag"
	"nginx-migration-mcp/tools"
	"strings"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
)

// RegisterNginxConfigTools 注册 Nginx 配置分析和转换工具
func RegisterNginxConfigTools(server *common.MCPServer, ctx *MigrationContext) {
	RegisterSimpleTool(
		server,
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
	)

	RegisterSimpleTool(
		server,
		"convert_to_higress",
		"将 Nginx 配置转换为 Higress Ingress 和 Service 资源（主要方式）或 HTTPRoute（可选）",
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
				"use_gateway_api": map[string]interface{}{
					"type":        "boolean",
					"description": "是否使用 Gateway API (HTTPRoute)。默认 false，使用 Ingress",
					"default":     false,
				},
			},
			"required": []string{"config_content"},
		},
		func(args map[string]interface{}) (string, error) {
			return convertToHigress(args, ctx)
		},
	)
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

	// 收集配置特性用于 RAG 查询
	features := []string{}
	if hasProxy {
		features = append(features, "反向代理")
	}
	if hasRewrite {
		features = append(features, "URL重写")
	}
	if hasSSL {
		features = append(features, "SSL配置")
	}

	// === RAG 增强：查询 Nginx 配置迁移最佳实践 ===
	var ragContext *rag.RAGContext
	if ctx.RAGManager != nil && ctx.RAGManager.IsEnabled() && len(features) > 0 {
		query := fmt.Sprintf("Nginx %s 迁移到 Higress 的配置方法和最佳实践", strings.Join(features, "、"))
		var err error
		ragContext, err = ctx.RAGManager.QueryForTool("parse_nginx_config", query, "nginx_migration")
		if err != nil {
			log.Printf("  RAG query failed for parse_nginx_config: %v", err)
		}
	}

	// 构建分析结果
	var result strings.Builder

	// RAG 上下文（如果有）
	if ragContext != nil && ragContext.Enabled && len(ragContext.Documents) > 0 {
		result.WriteString("📚 知识库迁移指南:\n\n")
		result.WriteString(ragContext.FormatContextForAI())
		result.WriteString("\n---\n\n")
	}

	result.WriteString(fmt.Sprintf(`Nginx配置分析结果

基础信息:
- Server块: %d个
- Location块: %d个  
- SSL配置: %t
- 反向代理: %t
- URL重写: %t

复杂度: %s

迁移建议:`, serverCount, locationCount, hasSSL, hasProxy, hasRewrite, complexity))

	if hasProxy {
		result.WriteString("\n- 反向代理将转换为Ingress backend配置")
	}
	if hasRewrite {
		result.WriteString("\n- URL重写将使用Higress注解 (higress.io/rewrite-target)")
	}
	if hasSSL {
		result.WriteString("\n- SSL配置将转换为Ingress TLS配置")
	}

	return result.String(), nil
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

	// 检查是否使用 Gateway API
	useGatewayAPI := false
	if val, ok := args["use_gateway_api"].(bool); ok {
		useGatewayAPI = val
	}

	// ===  使用增强的解析器解析 Nginx 配置 ===
	nginxConfig, err := tools.ParseNginxConfig(configContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse Nginx config: %v", err)
	}

	// 分析配置
	analysis := tools.AnalyzeNginxConfig(nginxConfig)

	// === RAG 增强：查询转换示例和最佳实践 ===
	var ragContext string
	if ctx.RAGManager != nil && ctx.RAGManager.IsEnabled() {
		// 构建查询关键词
		queryBuilder := []string{"Nginx 配置转换到 Higress"}

		if useGatewayAPI {
			queryBuilder = append(queryBuilder, "Gateway API HTTPRoute")
		} else {
			queryBuilder = append(queryBuilder, "Kubernetes Ingress")
		}

		// 根据特性添加查询关键词
		if analysis.Features["ssl"] {
			queryBuilder = append(queryBuilder, "SSL TLS 证书配置")
		}
		if analysis.Features["rewrite"] {
			queryBuilder = append(queryBuilder, "URL 重写 rewrite 规则")
		}
		if analysis.Features["redirect"] {
			queryBuilder = append(queryBuilder, "重定向 redirect")
		}
		if analysis.Features["header_manipulation"] {
			queryBuilder = append(queryBuilder, "请求头 响应头处理")
		}
		if len(nginxConfig.Upstreams) > 0 {
			queryBuilder = append(queryBuilder, "负载均衡 upstream")
		}

		queryString := strings.Join(queryBuilder, " ")
		log.Printf("🔍 RAG Query: %s", queryString)

		ragResult, err := ctx.RAGManager.QueryForTool(
			"convert_to_higress",
			queryString,
			"nginx_to_higress",
		)

		if err == nil && ragResult.Enabled && len(ragResult.Documents) > 0 {
			log.Printf("✅ RAG: Found %d documents for conversion", len(ragResult.Documents))
			ragContext = "\n\n## 📚 参考文档（来自知识库）\n\n" + ragResult.FormatContextForAI()
		} else {
			if err != nil {
				log.Printf("⚠️  RAG query failed: %v", err)
			}
		}
	}

	// === 将配置数据转换为 JSON 供 AI 使用 ===
	configJSON, _ := json.MarshalIndent(nginxConfig, "", "  ")
	analysisJSON, _ := json.MarshalIndent(analysis, "", "  ")

	// === 构建返回消息 ===
	var result strings.Builder

	result.WriteString(fmt.Sprintf(`📋 Nginx 配置解析完成

## 配置概览
- Server 块: %d
- Location 块: %d
- 域名: %d 个
- 复杂度: %s
- 目标格式: %s
- 命名空间: %s

## 检测到的特性
%s

## 迁移建议
%s
%s

---

## Nginx 配置结构

`+"```json"+`
%s
`+"```"+`

## 分析结果

`+"```json"+`
%s
`+"```"+`
%s
`,
		analysis.ServerCount,
		analysis.LocationCount,
		analysis.DomainCount,
		analysis.Complexity,
		func() string {
			if useGatewayAPI {
				return "Gateway API (HTTPRoute)"
			}
			return "Kubernetes Ingress"
		}(),
		namespace,
		formatFeaturesForOutput(analysis.Features),
		formatSuggestionsForOutput(analysis.Suggestions),
		func() string {
			if ragContext != "" {
				return "\n\n✅ 已加载知识库参考文档"
			}
			return ""
		}(),
		string(configJSON),
		string(analysisJSON),
		ragContext,
	))

	return result.String(), nil
}

// generateIngressConfig 生成 Ingress 资源配置（主要方式）
func generateIngressConfig(ingressName, namespace, hostname, serviceName string, ctx *MigrationContext) string {
	return fmt.Sprintf(`转换后的Higress配置（使用 Ingress - 推荐方式）

apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s
  namespace: %s
  annotations:
    higress.io/migrated-from: "nginx"
    higress.io/ingress.class: "higress"
spec:
  ingressClassName: higress
  rules:
  - host: %s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: %d

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
    protocol: TCP

转换完成

应用步骤:
1. 保存为 higress-config.yaml
2. 执行: kubectl apply -f higress-config.yaml
3. 验证: kubectl get ingress -n %s

说明:
- 使用 Ingress 是 Higress 的主要使用方式，兼容性最好
- 如需使用 Gateway API (HTTPRoute)，请设置参数 use_gateway_api=true`,
		ingressName, namespace,
		hostname,
		serviceName, ctx.ServicePort,
		serviceName, namespace,
		ctx.ServicePort, ctx.TargetPort,
		namespace)
}

// generateHTTPRouteConfig 生成 HTTPRoute 资源配置（备用选项）
func generateHTTPRouteConfig(routeName, namespace, hostname, serviceName string, ctx *MigrationContext) string {
	return fmt.Sprintf(`转换后的Higress配置（使用 Gateway API - 可选方式）

注意: Gateway API 在 Higress 中默认关闭，使用前需要确认已启用。

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
    protocol: TCP

转换完成

应用步骤:
1. 确认 Gateway API 已启用: PILOT_ENABLE_GATEWAY_API=true
2. 保存为 higress-config.yaml
3. 执行: kubectl apply -f higress-config.yaml
4. 验证: kubectl get httproute -n %s

说明:
- Gateway API 是可选功能，默认关闭
- 推荐使用 Ingress (设置 use_gateway_api=false)`,
		routeName, namespace,
		ctx.GatewayName, ctx.GatewayNamespace, hostname,
		serviceName, ctx.ServicePort,
		serviceName, namespace,
		ctx.ServicePort, ctx.TargetPort,
		namespace)
}

func generateIngressName(hostname string, ctx *MigrationContext) string {
	prefix := "nginx-migrated"
	if ctx.RoutePrefix != "" {
		prefix = ctx.RoutePrefix
	}

	if hostname == "" || hostname == ctx.DefaultHostname {
		return fmt.Sprintf("%s-ingress", prefix)
	}
	// Replace dots and special characters for valid k8s name
	safeName := hostname
	for _, char := range []string{".", "_", ":"} {
		safeName = strings.ReplaceAll(safeName, char, "-")
	}
	return fmt.Sprintf("%s-%s", prefix, safeName)
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

// formatFeaturesForOutput 格式化特性列表用于输出
func formatFeaturesForOutput(features map[string]bool) string {
	featureNames := map[string]string{
		"ssl":                 "SSL/TLS 加密",
		"proxy":               "反向代理",
		"rewrite":             "URL 重写",
		"redirect":            "重定向",
		"return":              "返回指令",
		"complex_routing":     "复杂路由匹配",
		"header_manipulation": "请求头操作",
		"response_headers":    "响应头操作",
	}

	var result []string
	for key, enabled := range features {
		if enabled {
			if name, ok := featureNames[key]; ok {
				result = append(result, fmt.Sprintf("- ✅ %s", name))
			} else {
				result = append(result, fmt.Sprintf("- ✅ %s", key))
			}
		}
	}

	if len(result) == 0 {
		return "- 基础配置（无特殊特性）"
	}
	return strings.Join(result, "\n")
}

// formatSuggestionsForOutput 格式化建议列表用于输出
func formatSuggestionsForOutput(suggestions []string) string {
	if len(suggestions) == 0 {
		return "- 无特殊建议"
	}
	var result []string
	for _, s := range suggestions {
		result = append(result, fmt.Sprintf("- 💡 %s", s))
	}
	return strings.Join(result, "\n")
}
