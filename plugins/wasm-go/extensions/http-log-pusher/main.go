package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	pluginlog "github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/tokenusage"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	pluginlog.Info("[http-log-pusher] plugin initializing...")
	wrapper.SetCtx(
		"http-log-pusher",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
		wrapper.ProcessResponseBody(onHttpResponseBody),
	)
	pluginlog.Info("[http-log-pusher] plugin loaded successfully")
}

// PluginConfig 定义插件配置 (对应 WasmPlugin 资源中的 pluginConfig)
type PluginConfig struct {
	CollectorServiceName string             `json:"collector_service_name"` // fqdn,例如 "log-collector.higress-system.svc.cluster.local"
	CollectorHost       string             `json:"collector_host"`    // Collector 主机名或 IP,例如 "collector-service.default.svc.cluster.local" 或 "192.168.1.100"
	CollectorPort       int64              `json:"collector_port"`    // Collector 端口,例如 8080
	CollectorPath       string             `json:"collector_path"`    // 接收日志的 API 路径,例如 "/api/log"
	CollectorClient     wrapper.HttpClient `json:"-"`                 // HTTP 客户端,用于发送日志
}

// LogEntry 定义发给 Collector 的 JSON 数据结构 (参考 Envoy accessLogFormat)
type LogEntry struct {
	// 基础请求信息
	StartTime     string `json:"start_time"`               // 请求开始时间
	Authority     string `json:"authority"`                // Host/Authority
	Method        string `json:"method"`                   // HTTP 方法
	Path          string `json:"path"`                     // 请求路径
	Protocol      string `json:"protocol"`                 // HTTP 协议版本
	RequestID     string `json:"request_id"`               // X-Request-ID
	TraceID       string `json:"trace_id,omitempty"`       // X-B3-TraceID
	UserAgent     string `json:"user_agent,omitempty"`     // User-Agent
	XForwardedFor string `json:"x_forwarded_for,omitempty"` // X-Forwarded-For
	
	// 响应信息
	ResponseCode        int    `json:"response_code"`                  // 响应状态码
	ResponseFlags       string `json:"response_flags,omitempty"`       // Envoy 响应标志
	ResponseCodeDetails string `json:"response_code_details,omitempty"` // 响应码详情
	
	// 流量信息
	BytesReceived int64 `json:"bytes_received"` // 接收字节数
	BytesSent     int64 `json:"bytes_sent"`     // 发送字节数
	Duration      int64 `json:"duration"`       // 请求总耗时(ms)
	
	// 上游信息
	UpstreamCluster              string `json:"upstream_cluster,omitempty"`                // 上游集群名
	UpstreamHost                 string `json:"upstream_host,omitempty"`                   // 上游主机
	UpstreamServiceTime          string `json:"upstream_service_time,omitempty"`           // 上游服务耗时
	UpstreamTransportFailure     string `json:"upstream_transport_failure_reason,omitempty"` // 上游传输失败原因
	
	// 连接信息
	DownstreamLocalAddress  string `json:"downstream_local_address,omitempty"`  // 下游本地地址
	DownstreamRemoteAddress string `json:"downstream_remote_address,omitempty"` // 下游远程地址
	UpstreamLocalAddress    string `json:"upstream_local_address,omitempty"`    // 上游本地地址
	
	// 路由信息
	RouteName            string `json:"route_name,omitempty"`             // 路由名称
	RequestedServerName  string `json:"requested_server_name,omitempty"`  // SNI
	
	// AI 日志 (如果有)
	AILog string `json:"ai_log,omitempty"` // WASM AI 日志
	
	// 监控元数据字段
	InstanceID   string `json:"instance_id"`      // 实例ID
	API          string `json:"api"`              // API名称
	Model        string `json:"model"`            // 模型名称
	Consumer     string `json:"consumer"`         // 消费者
	Route        string `json:"route"`            // 路由
	Service      string `json:"service"`          // 服务
	MCPServer    string `json:"mcp_server"`       // MCP Server
	MCPTool      string `json:"mcp_tool"`         // MCP Tool
	InputTokens  int64  `json:"input_tokens"`     // 输入token数量
	OutputTokens int64  `json:"output_tokens"`    // 输出token数量
	TotalTokens  int64  `json:"total_tokens"`     // 总token数量
	
	// 详细数据 (可选)
	ReqHeaders  map[string]string `json:"req_headers,omitempty"`  // 完整请求头
	ReqBody     string            `json:"req_body,omitempty"`     // 请求体
	RespHeaders map[string]string `json:"resp_headers,omitempty"` // 完整响应头
	RespBody    string            `json:"resp_body,omitempty"`    // 响应体
}

// 解析配置
func parseConfig(jsonConf gjson.Result, config *PluginConfig) error {
	pluginlog.Infof("[http-log-pusher] parsing config: %s", jsonConf.String())
	
	config.CollectorServiceName = jsonConf.Get("collector_service_name").String()
	config.CollectorHost = jsonConf.Get("collector_host").String()
	config.CollectorPort = jsonConf.Get("collector_port").Int()
	
	// 校验必填参数
	if config.CollectorServiceName == "" || config.CollectorHost == "" || config.CollectorPort == 0 {
		pluginlog.Errorf("[http-log-pusher] either collector_service_name or (collector_host + collector_port) is required")
		return errors.New("either collector_service_name or (collector_host + collector_port) is required")
	}
	
	config.CollectorPath = jsonConf.Get("collector_path").String()
	if config.CollectorPath == "" {
		config.CollectorPath = "/"
	}
	
	// 创建 HTTP 客户端用于发送日志
	// 优先使用 host + port 方式,更稳定可靠
	pluginlog.Infof("[http-log-pusher] using host+port cluster: host=%s, port=%d", config.CollectorHost, config.CollectorPort)
	config.CollectorClient = wrapper.NewClusterClient(wrapper.DnsCluster{
		ServiceName: config.CollectorServiceName,
		Port:        config.CollectorPort,
		Domain:        config.CollectorHost,
	})
	
	return nil
}


// ---------------- 核心逻辑 ----------------

// 1. 处理请求头
func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
	// 获取所有请求头并暂存
	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		pluginlog.Errorf("[http-log-pusher] failed to get request headers: %v", err)
	}
	ctx.SetContext("req_headers", headers)
	ctx.SetContext("start_time", time.Now().UnixMilli())

	// 必须允许继续,否则请求会卡住
	// 如果需要读取 Body,必须在 return 时不打断流
	return types.ActionContinue
}

// 2. 处理请求体
func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte) types.Action {
	if len(body) > 0 {
		// 注意:大包体可能会分多次回调,生产环境建议限制长度或做截断
		ctx.SetContext("req_body", string(body))
	}
	return types.ActionContinue
}

// 3. 处理响应头
func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
	headers, _ := proxywasm.GetHttpResponseHeaders()
	ctx.SetContext("resp_headers", headers)
	return types.ActionContinue
}

// 4. 处理响应体 (也是发送日志的最佳时机)
func onHttpResponseBody(ctx wrapper.HttpContext, config PluginConfig, body []byte) types.Action {
	// 1. 组装数据 - 参考 Envoy accessLogFormat 字段
	reqHeaders, _ := ctx.GetContext("req_headers").([][2]string)
	reqBody, _ := ctx.GetContext("req_body").(string)
	respHeaders, _ := ctx.GetContext("resp_headers").([][2]string)
	startTime, _ := ctx.GetContext("start_time").(int64)
	
	// 提取响应状态码
	statusCode := 200
	for _, h := range respHeaders {
		if h[0] == ":status" {
			if code, err := parseStatusCode(h[1]); err == nil {
				statusCode = code
			}
			break
		}
	}
	
	// 提取关键请求头
	requestID := getHeaderValue(reqHeaders, "x-request-id")
	traceID := getHeaderValue(reqHeaders, "x-b3-traceid")
	userAgent := getHeaderValue(reqHeaders, "user-agent")
	xForwardedFor := getHeaderValue(reqHeaders, "x-forwarded-for")
	
	// 获取 Envoy 属性
	protocol := getEnvoyProperty("request.protocol", "HTTP/1.1")
	bytesReceived := getEnvoyPropertyInt64("request.total_size", 0)
	bytesSent := getEnvoyPropertyInt64("response.total_size", 0)
	responseFlags := getEnvoyProperty("response.flags", "")
	responseCodeDetails := getEnvoyProperty("response.code_details", "")
	upstreamCluster := getEnvoyProperty("cluster_name", "")
	upstreamHost := getEnvoyProperty("upstream_host", "")
	upstreamServiceTime := getEnvoyProperty("upstream_service_time", "")
	downstreamLocalAddr := getEnvoyProperty("downstream_local_address", "")
	downstreamRemoteAddr := getEnvoyProperty("downstream_remote_address", "")
	upstreamLocalAddr := getEnvoyProperty("upstream_local_address", "")
	sni := getEnvoyProperty("requested_server_name", "")
	// 从 Envoy Filter State 读取 AI 日志
	// ai-statistics 插件通过 WriteUserAttributeToLogWithKey() 将数据写入此处
	var aiLog string
	defer func() {
		if r := recover(); r != nil {
			pluginlog.Debugf("[http-log-pusher] recovered from panic when getting ai_log: %v", r)
			aiLog = "-" // panic 时使用默认值
		}
	}()
	
	aiLogBytes, err := proxywasm.GetProperty([]string{wrapper.AILogKey})
	if err == nil && len(aiLogBytes) > 0 {
		aiLog = string(aiLogBytes)
	} else {
		aiLog = "-" // 无 AI 日志时的默认值
	}
	
	// 提取监控所需的元数据字段
	instanceID := getInstanceID()
	apiName := getAPIName(ctx)
	modelName := getModelName(ctx)
	consumer := getConsumer()
	routeNameMeta := getRouteName()
	serviceName := getServiceName()
	mcpServer := getMCPServer()
	mcpTool := getMCPTool(ctx)
	
	// 提取token信息
	inputTokens := getInputTokens(ctx, body)
	outputTokens := getOutputTokens(ctx, body)
	totalTokens := getTotalTokens(ctx, body)
	
	// 计算耗时
	duration := time.Now().UnixMilli() - startTime
	
	entry := LogEntry{
		// 基础信息
		StartTime:     time.UnixMilli(startTime).Format(time.RFC3339),
		Authority:     ctx.Host(),
		Method:        ctx.Method(),
		Path:          ctx.Path(),
		Protocol:      protocol,
		RequestID:     requestID,
		TraceID:       traceID,
		UserAgent:     userAgent,
		XForwardedFor: xForwardedFor,
		
		// 响应信息
		ResponseCode:        statusCode,
		ResponseFlags:       responseFlags,
		ResponseCodeDetails: responseCodeDetails,
		
		// 流量信息
		BytesReceived: bytesReceived,
		BytesSent:     bytesSent,
		Duration:      duration,
		
		// 上游信息
		UpstreamCluster:          upstreamCluster,
		UpstreamHost:             upstreamHost,
		UpstreamServiceTime:      upstreamServiceTime,
		UpstreamTransportFailure: getEnvoyProperty("upstream_transport_failure_reason", ""),
		
		// 连接信息
		DownstreamLocalAddress:  downstreamLocalAddr,
		DownstreamRemoteAddress: downstreamRemoteAddr,
		UpstreamLocalAddress:    upstreamLocalAddr,
		
		// 路由信息
		RouteName:           routeNameMeta,
		RequestedServerName: sni,
		
		// AI 日志
		AILog: aiLog,
		
		// 监控元数据
		InstanceID:   instanceID,
		API:          apiName,
		Model:        modelName,
		Consumer:     consumer,
		Route:        routeNameMeta,
		Service:      serviceName,
		MCPServer:    mcpServer,
		MCPTool:      mcpTool,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
		
		// 详细数据 (可选，根据需要采集)
		ReqHeaders:  toMap(reqHeaders),
		ReqBody:     reqBody,
		RespHeaders: toMap(respHeaders),
		RespBody:    string(body),
	}

	// 🔍 调试日志：打印即将存储的所有字段内容
	pluginlog.Infof("[http-log-pusher] === 即将存储的日志内容 ===")
	// pluginlog.Infof("[http-log-pusher] 基础信息: StartTime=%s, Authority=%s, Method=%s, Path=%s, Protocol=%s", 
	// 	entry.StartTime, entry.Authority, entry.Method, entry.Path, entry.Protocol)
	// pluginlog.Infof("[http-log-pusher] 请求标识: RequestID=%s, TraceID=%s", entry.RequestID, entry.TraceID)
	// pluginlog.Infof("[http-log-pusher] 响应信息: ResponseCode=%d, ResponseFlags=%s", entry.ResponseCode, entry.ResponseFlags)
	// pluginlog.Infof("[http-log-pusher] 流量统计: BytesReceived=%d, BytesSent=%d, Duration=%d ms", 
	// 	entry.BytesReceived, entry.BytesSent, entry.Duration)
	// pluginlog.Infof("[http-log-pusher] 上游信息: UpstreamCluster=%s, UpstreamHost=%s", entry.UpstreamCluster, entry.UpstreamHost)
	pluginlog.Infof("[http-log-pusher] 监控元数据: InstanceID=%s, API=%s, Model=%s, Consumer=%s", 
		entry.InstanceID, entry.API, entry.Model, entry.Consumer)
	pluginlog.Infof("[http-log-pusher] 路由服务: Route=%s, Service=%s, MCPServer=%s, MCPTool=%s", 
		entry.Route, entry.Service, entry.MCPServer, entry.MCPTool)
	pluginlog.Infof("[http-log-pusher] Token信息: Input=%d, Output=%d, Total=%d", 
		entry.InputTokens, entry.OutputTokens, entry.TotalTokens)
	// pluginlog.Infof("[http-log-pusher] AI日志: AILog=%s", entry.AILog)
	pluginlog.Infof("[http-log-pusher] =========================")

	payload, _ := json.Marshal(entry)
	
	// 获取最终使用的集群名
	clusterName := config.CollectorClient.ClusterName()
	
	pluginlog.Infof("[http-log-pusher] preparing http call: cluster=%s, path=%s, payload_size=%d",
		clusterName, config.CollectorPath, len(payload))

	// 2. 发送异步请求给 Collector
	// 使用 wrapper.HttpClient.Post 方法，它会自动处理 headers
	headers := [][2]string{
		{"Content-Type", "application/json"},
	}

	// 这里的 5000 是超时时间(ms)
	// Fire-and-forget: 回调函数简单记录结果
	postErr := config.CollectorClient.Post(
		config.CollectorPath,
		headers,
		payload,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode == 200 || statusCode == 204 {
				pluginlog.Infof("[http-log-pusher] log sent successfully, status=%d", statusCode)
			} else {
				pluginlog.Warnf("[http-log-pusher] collector returned status=%d, body=%s", statusCode, string(responseBody))
			}
		},
		5000, // 超时 5 秒
	)
	if postErr != nil {
		pluginlog.Errorf("[http-log-pusher] failed to dispatch http call: %v", postErr)
	}

	return types.ActionContinue
}

// 辅助工具：Header 数组转 Map
func toMap(headers [][2]string) map[string]string {
	m := make(map[string]string)
	for _, h := range headers {
		m[h[0]] = h[1]
	}
	return m
}

// 从 Header 数组中获取指定 key 的值 (不区分大小写)
func getHeaderValue(headers [][2]string, key string) string {
	key = strings.ToLower(key)
	for _, h := range headers {
		if strings.ToLower(h[0]) == key {
			return h[1]
		}
	}
	return ""
}

// 解析状态码
func parseStatusCode(statusStr string) (int, error) {
	code, err := strconv.Atoi(statusStr)
	if err != nil {
		return 0, err
	}
	return code, nil
}

// 获取 Envoy 属性 (字符串类型)
func getEnvoyProperty(path string, defaultValue string) string {
	// Envoy 属性路径格式，参考: https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/advanced/attributes
	var propertyPath []string
	
	switch path {
	case "request.protocol":
		propertyPath = []string{"request", "protocol"}
	case "response.flags":
		propertyPath = []string{"response", "flags"}
	case "response.code_details":
		propertyPath = []string{"response", "code_details"}
	case "cluster_name":
		propertyPath = []string{"cluster_name"}
	case "upstream_host":
		propertyPath = []string{"upstream", "address"}
	case "upstream_service_time":
		propertyPath = []string{"upstream", "service_time"}
	case "upstream_transport_failure_reason":
		propertyPath = []string{"upstream", "transport_failure_reason"}
	case "downstream_local_address":
		propertyPath = []string{"connection", "local_address"}
	case "downstream_remote_address":
		propertyPath = []string{"connection", "remote_address"}
	case "upstream_local_address":
		propertyPath = []string{"upstream", "local_address"}
	case "route_name":
		propertyPath = []string{"route_name"}
	case "requested_server_name":
		propertyPath = []string{"connection", "requested_server_name"}
	case "instance_id":
		propertyPath = []string{"node", "id"}
	case "wasm.ai_log":
		propertyPath = []string{"wasm", "ai_log"}
	default:
		return defaultValue
	}
	
	// 添加 panic 恢复保护
	defer func() {
		if r := recover(); r != nil {
			pluginlog.Debugf("[http-log-pusher] recovered from panic when getting property %s: %v", path, r)
		}
	}()
	
	value, err := proxywasm.GetProperty(propertyPath)
	if err != nil || len(value) == 0 {
		return defaultValue
	}
	return string(value)
}

// 获取实例ID
func getInstanceID() string {
	// 1. 从 Envoy 属性获取实例ID（最安全的方式）
	instanceID := getEnvoyProperty("instance_id", "")
	if instanceID != "" {
		pluginlog.Debugf("[http-log-pusher] got instance_id from envoy property: %s", instanceID)
		return instanceID
	}
	
	// 2. 从请求头获取
	instanceID, _ = proxywasm.GetHttpRequestHeader("x-instance-id")
	if instanceID != "" {
		pluginlog.Debugf("[http-log-pusher] got instance_id from header: %s", instanceID)
		return instanceID
	}
	
	// 3. 尝试从节点名称获取（作为备选方案）
	defer func() {
		if r := recover(); r != nil {
			pluginlog.Debugf("[http-log-pusher] recovered from panic when getting node.id: %v", r)
		}
	}()
	
	nodeNameBytes, err := proxywasm.GetProperty([]string{"node", "id"})
	if err == nil && len(nodeNameBytes) > 0 {
		nodeName := string(nodeNameBytes)
		if nodeName != "" {
			pluginlog.Debugf("[http-log-pusher] got instance_id from node.id: %s", nodeName)
			return nodeName
		}
	}
	
	pluginlog.Debugf("[http-log-pusher] instance_id not found, using default")
	return "unknown"
}

// 获取API名称
func getAPIName(ctx wrapper.HttpContext) string {
	// 从路由名称解析
	routeName := getEnvoyProperty("route_name", "")
	if routeName != "" {
		// 格式: model-api-{api-name}-0
		parts := strings.Split(routeName, "-")
		if len(parts) >= 3 && parts[0] == "model" && parts[1] == "api" {
			// 提取从第3个部分开始的所有内容作为 API 名称
			// 例如: model-api-test-by-lisi-0 -> test-by-lisi
			apiName := strings.Join(parts[2:len(parts)-1], "-")
			return apiName
		}
	}
	
	pluginlog.Debugf("[http-log-pusher] api_name not determined from route/path")
	return "unknown"
}

// 获取模型名称
func getModelName(ctx wrapper.HttpContext) string {
	// 优先从 ai-statistics 获取
	model := ctx.GetUserAttribute("model")
	if model != nil {
		if modelStr, ok := model.(string); ok && modelStr != "" {
			return modelStr
		}
	}
	
	// 从请求体解析
	reqBody, _ := ctx.GetContext("req_body").(string)
	if reqBody != "" {
		modelFromReq := extractModelFromRequestBody(reqBody)
		if modelFromReq != "" {
			return modelFromReq
		}
	}
	
	pluginlog.Debugf("[http-log-pusher] model_name not found")
	return "unknown"
}

// 获取消费者信息
func getConsumer() string {
	// 优先从认证插件设置的头获取（jwt-auth/key-auth等插件认证通过后会设置此header）
	consumer, _ := proxywasm.GetHttpRequestHeader("x-mse-consumer")
	if consumer != "" {
		return consumer
	}
	
	// 从 Authorization 头解析完整凭证信息
	authHeader, _ := proxywasm.GetHttpRequestHeader("authorization")
	if authHeader != "" {
		// 解析 Bearer token - 存储完整token用于审计和查询
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			// 返回完整token以便后续审计查询
			// 注意：如果token过长可能影响日志存储，建议配合数据库字段长度设置
			return fmt.Sprintf("bearer:%s", token)
		}
		// 解析 Basic 认证
		if strings.HasPrefix(authHeader, "Basic ") {
			credential := strings.TrimPrefix(authHeader, "Basic ")
			return fmt.Sprintf("basic:%s", credential)
		}
		// 其他认证方式
		return fmt.Sprintf("auth:%s", authHeader)
	}
	
	// 检查其他常见的认证头
	apiKey, _ := proxywasm.GetHttpRequestHeader("x-api-key")
	if apiKey != "" {
		return fmt.Sprintf("apikey:%s", apiKey)
	}
	
	pluginlog.Debugf("[http-log-pusher] consumer not found")
	return "anonymous"
}

// 获取路由名称 - 区分MCP场景和Model API场景
func getRouteName() string {
	routeName := getEnvoyProperty("route_name", "")
	if routeName == "" {
		pluginlog.Debugf("[http-log-pusher] route_name not found")
		return "unknown"
	}
	
	// 判断是否为MCP场景
	if strings.Contains(routeName, "-mcp-") {
		// MCP场景：路由名称格式为 {mcp-server-name}-mcp-{mcp-tool-name}-0
		// 在Route字段中存储MCP Server名称（即mcp前面的部分）
		parts := strings.Split(routeName, "-")
		mcpIndex := -1
		for i, part := range parts {
			if part == "mcp" {
				mcpIndex = i
				break
			}
		}
		if mcpIndex > 0 {
			// 返回MCP Server名称
			return strings.Join(parts[:mcpIndex], "-")
		}
	}
	
	// Model API场景或其他场景：直接返回原始路由名称
	return routeName
}

// 获取服务名称
func getServiceName() string {
	// 从上游集群获取
	clusterName := getEnvoyProperty("cluster_name", "")
	if clusterName != "" {
		// 清理集群名称格式
		service := strings.TrimPrefix(clusterName, "outbound|")
		service = strings.TrimPrefix(service, "inbound|")
		parts := strings.Split(service, "|")
		if len(parts) > 0 {
			return parts[len(parts)-1] // 取最后一部分作为服务名
		}
		return service
	}
	
	pluginlog.Debugf("[http-log-pusher] service_name not found")
	return "unknown"
}

// 获取MCP Server
func getMCPServer() string {
	// 方法1: 从路由名称获取
	routeName := getEnvoyProperty("route_name", "")
	if routeName == "" {
		pluginlog.Debugf("[http-log-pusher] route_name not found")
		return "unknown"
	}
	
	return routeName
}

// 获取MCP Tool
func getMCPTool(ctx wrapper.HttpContext) string {
	// 方法1: 从标准MCP工具头获取（最准确）
	// Higress系统通过x-envoy-mcp-tool-name header传递工具名称
	toolName, err := proxywasm.GetHttpRequestHeader("x-envoy-mcp-tool-name")
	if err == nil && toolName != "" {
		pluginlog.Debugf("[http-log-pusher] got mcp_tool from header: %s", toolName)
		return toolName
	}
	
	// 方法2: 从请求体中提取工具名称（备选方案）
	// 适用于tools/call请求，从params.name字段提取
	requestBody := ctx.GetContext("req_body")
	if requestBody != nil {
		if bodyStr, ok := requestBody.(string); ok && bodyStr != "" {
			// 尝试从JSON请求体中提取tool name
			toolNameFromBody := extractToolNameFromJson(bodyStr)
			if toolNameFromBody != "" {
				pluginlog.Debugf("[http-log-pusher] got mcp_tool from request body: %s", toolNameFromBody)
				return toolNameFromBody
			}
		}
	}
	
	// 获取路径用于日志记录
	path := ctx.Path()
	pluginlog.Debugf("[http-log-pusher] mcp_tool not determined from header/body/path: %s", path)
	return "unknown"
}

// 获取输入token数量
func getInputTokens(ctx wrapper.HttpContext, respBody []byte) int64 {
	// 方法1: 从tokenusage包获取（优先）
	if usage := tokenusage.GetTokenUsage(ctx, respBody); usage.TotalToken > 0 {
		pluginlog.Debugf("[http-log-pusher] got tokens from tokenusage: input=%d, output=%d, total=%d", 
			usage.InputToken, usage.OutputToken, usage.TotalToken)
		return usage.InputToken
	}
	
	// 方法2: 从响应体直接解析usage字段
	if len(respBody) > 0 {
		// 解析OpenAI格式的usage字段
		inputTokens := gjson.GetBytes(respBody, "usage.prompt_tokens").Int()
		if inputTokens > 0 {
			pluginlog.Debugf("[http-log-pusher] got input_tokens from response body: %d", inputTokens)
			return inputTokens
		}
		
		// 解析Claude/Bedrock格式
		inputTokens = gjson.GetBytes(respBody, "usage.input_tokens").Int()
		if inputTokens > 0 {
			pluginlog.Debugf("[http-log-pusher] got input_tokens from response body (claude format): %d", inputTokens)
			return inputTokens
		}
	}
	
	pluginlog.Debugf("[http-log-pusher] input_tokens not found")
	return 0
}

// 获取输出token数量
func getOutputTokens(ctx wrapper.HttpContext, respBody []byte) int64 {
	// 方法1: 从tokenusage包获取（优先）
	if usage := tokenusage.GetTokenUsage(ctx, respBody); usage.TotalToken > 0 {
		return usage.OutputToken
	}
	
	// 方法2: 从响应体直接解析usage字段
	if len(respBody) > 0 {
		// 解析OpenAI格式的usage字段
		outputTokens := gjson.GetBytes(respBody, "usage.completion_tokens").Int()
		if outputTokens > 0 {
			pluginlog.Debugf("[http-log-pusher] got output_tokens from response body: %d", outputTokens)
			return outputTokens
		}
		
		// 解析Claude/Bedrock格式
		outputTokens = gjson.GetBytes(respBody, "usage.output_tokens").Int()
		if outputTokens > 0 {
			pluginlog.Debugf("[http-log-pusher] got output_tokens from response body (claude format): %d", outputTokens)
			return outputTokens
		}
	}
	
	pluginlog.Debugf("[http-log-pusher] output_tokens not found")
	return 0
}

// 获取总token数量
func getTotalTokens(ctx wrapper.HttpContext, respBody []byte) int64 {
	// 方法1: 从tokenusage包获取（优先）
	if usage := tokenusage.GetTokenUsage(ctx, respBody); usage.TotalToken > 0 {
		return usage.TotalToken
	}
	
	// 方法2: 从响应体直接解析usage字段
	if len(respBody) > 0 {
		totalTokens := gjson.GetBytes(respBody, "usage.total_tokens").Int()
		if totalTokens > 0 {
			pluginlog.Debugf("[http-log-pusher] got total_tokens from response body: %d", totalTokens)
			return totalTokens
		}
		
		// 解析Claude/Bedrock格式
		totalTokens = gjson.GetBytes(respBody, "usage.inputTokens").Int() + gjson.GetBytes(respBody, "usage.outputTokens").Int()
		if totalTokens > 0 {
			pluginlog.Debugf("[http-log-pusher] calculated total_tokens from claude format: %d", totalTokens)
			return totalTokens
		}
	}
	
	pluginlog.Debugf("[http-log-pusher] total_tokens not found")
	return 0
}

// 从请求体提取模型名称
func extractModelFromRequestBody(body string) string {
	result := gjson.Get(body, "model")
	if result.Exists() {
		return result.String()
	}
	return ""
}

// 从JSON请求体中提取MCP工具名称
func extractToolNameFromJson(body string) string {
	// 对于tools/call请求，工具名称在params.name字段中
	result := gjson.Get(body, "params.name")
	if result.Exists() {
		return result.String()
	}
	return ""
}

// 获取 Envoy 属性 (int64 类型)
func getEnvoyPropertyInt64(path string, defaultValue int64) int64 {
	// Envoy 属性路径格式，参考: https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/advanced/attributes
	var propertyPath []string
	
	switch path {
	case "request.total_size":
		propertyPath = []string{"request", "size"}
	case "response.total_size":
		propertyPath = []string{"response", "size"}
	default:
		return defaultValue
	}
	
	// 添加 panic 恢复保护
	defer func() {
		if r := recover(); r != nil {
			pluginlog.Debugf("[http-log-pusher] recovered from panic when getting int64 property %s: %v", path, r)
		}
	}()
	
	value, err := proxywasm.GetProperty(propertyPath)
	if err != nil || len(value) == 0 {
		return defaultValue
	}
	
	// 将字节转换为字符串再解析为int64
	strValue := string(value)
	intValue, err := strconv.ParseInt(strValue, 10, 64)
	if err != nil {
		return defaultValue
	}
	
	return intValue
}