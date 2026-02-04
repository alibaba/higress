package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	proxywasm.LogInfo("[http-log-pusher] plugin initializing...")
	wrapper.SetCtx(
		"http-log-pusher",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
	proxywasm.LogInfo("[http-log-pusher] plugin loaded successfully")
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
	
	// 详细数据 (可选)
	ReqHeaders  map[string]string `json:"req_headers,omitempty"`  // 完整请求头
	ReqBody     string            `json:"req_body,omitempty"`     // 请求体
	RespHeaders map[string]string `json:"resp_headers,omitempty"` // 完整响应头
	RespBody    string            `json:"resp_body,omitempty"`    // 响应体
}

// 解析配置
func parseConfig(jsonConf gjson.Result, config *PluginConfig, log wrapper.Log) error {
	log.Infof("[http-log-pusher] parsing config: %s", jsonConf.String())
	
	config.CollectorServiceName = jsonConf.Get("collector_service_name").String()
	config.CollectorHost = jsonConf.Get("collector_host").String()
	config.CollectorPort = jsonConf.Get("collector_port").Int()
	
	// 校验必填参数
	if config.CollectorServiceName == "" || config.CollectorHost == "" || config.CollectorPort == 0 {
		log.Errorf("[http-log-pusher] either collector_service_name or (collector_host + collector_port) is required")
		return errors.New("either collector_service_name or (collector_host + collector_port) is required")
	}
	
	config.CollectorPath = jsonConf.Get("collector_path").String()
	if config.CollectorPath == "" {
		config.CollectorPath = "/"
	}
	
	// 创建 HTTP 客户端用于发送日志
	// 优先使用 host + port 方式,更稳定可靠
	log.Infof("[http-log-pusher] using host+port cluster: host=%s, port=%d", config.CollectorHost, config.CollectorPort)
	config.CollectorClient = wrapper.NewClusterClient(wrapper.DnsCluster{
		ServiceName: config.CollectorServiceName,
		Port:        config.CollectorPort,
		Domain:        config.CollectorHost,
	})
	
	return nil
}


// ---------------- 核心逻辑 ----------------

// 1. 处理请求头
func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	// 获取所有请求头并暂存
	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		log.Errorf("[http-log-pusher] failed to get request headers: %v", err)
	}
	ctx.SetContext("req_headers", headers)
	ctx.SetContext("start_time", time.Now().UnixMilli())

	// 必须允许继续,否则请求会卡住
	// 如果需要读取 Body,必须在 return 时不打断流
	return types.ActionContinue
}

// 2. 处理请求体
func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {
	if len(body) > 0 {
		// 注意:大包体可能会分多次回调,生产环境建议限制长度或做截断
		ctx.SetContext("req_body", string(body))
	}
	return types.ActionContinue
}

// 3. 处理响应头
func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	headers, _ := proxywasm.GetHttpResponseHeaders()
	ctx.SetContext("resp_headers", headers)
	return types.ActionContinue
}

// 4. 处理响应体 (也是发送日志的最佳时机)
func onHttpResponseBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {
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
	routeName := getEnvoyProperty("route_name", "")
	sni := getEnvoyProperty("requested_server_name", "")
	aiLog := getEnvoyProperty("wasm.ai_log", "")
	
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
		RouteName:           routeName,
		RequestedServerName: sni,
		
		// AI 日志
		AILog: aiLog,
		
		// 详细数据 (可选，根据需要采集)
		ReqHeaders:  toMap(reqHeaders),
		ReqBody:     reqBody,
		RespHeaders: toMap(respHeaders),
		RespBody:    string(body),
	}

	payload, _ := json.Marshal(entry)
	
	// 获取最终使用的集群名
	clusterName := config.CollectorClient.ClusterName()
	
	log.Infof("[http-log-pusher] preparing http call: cluster=%s, path=%s, payload_size=%d",
		clusterName, config.CollectorPath, len(payload))

	// 2. 发送异步请求给 Collector
	// 使用 wrapper.HttpClient.Post 方法，它会自动处理 headers
	headers := [][2]string{
		{"Content-Type", "application/json"},
	}

	// 这里的 5000 是超时时间(ms)
	// Fire-and-forget: 回调函数简单记录结果
	err := config.CollectorClient.Post(
		config.CollectorPath,
		headers,
		payload,
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode == 200 || statusCode == 204 {
				log.Infof("[http-log-pusher] log sent successfully, status=%d", statusCode)
			} else {
				log.Warnf("[http-log-pusher] collector returned status=%d, body=%s", statusCode, string(responseBody))
			}
		},
		5000, // 超时 5 秒
	)
	if err != nil {
		log.Errorf("[http-log-pusher] failed to dispatch http call: %v", err)
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
	case "wasm.ai_log":
		propertyPath = []string{"wasm", "ai_log"}
	default:
		return defaultValue
	}
	
	value, err := proxywasm.GetProperty(propertyPath)
	if err != nil || len(value) == 0 {
		return defaultValue
	}
	return string(value)
}

// 获取 Envoy 属性 (int64 类型)
func getEnvoyPropertyInt64(path string, defaultValue int64) int64 {
	var propertyPath []string
	
	switch path {
	case "request.total_size":
		propertyPath = []string{"request", "total_size"}
	case "response.total_size":
		propertyPath = []string{"response", "total_size"}
	default:
		return defaultValue
	}
	
	value, err := proxywasm.GetProperty(propertyPath)
	if err != nil || len(value) == 0 {
		return defaultValue
	}
	
	// 尝试解析为 int64
	if len(value) >= 8 {
		// Envoy 返回的是 little-endian 字节序
		var result int64
		for i := 0; i < 8 && i < len(value); i++ {
			result |= int64(value[i]) << (i * 8)
		}
		return result
	}
	
	return defaultValue
}