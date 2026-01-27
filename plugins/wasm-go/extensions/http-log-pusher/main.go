package main

import (
	"encoding/json"
	"errors"
	"net/http"
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
	CollectorClientName string             `json:"collector_service"` // Envoy 集群名，例如 "outbound|8080||collector-service.default.svc.cluster.local"
	CollectorPath       string             `json:"collector_path"`    // 接收日志的 API 路径，例如 "/api/log"
	SampleRate          int64              `json:"sample_rate"`       // 采样率 0-100
	CollectorClient     wrapper.HttpClient `json:"-"`                 // HTTP 客户端，用于发送日志
}

// LogEntry 定义发给 Collector 的 JSON 数据结构
type LogEntry struct {
	Timestamp   int64             `json:"timestamp"`
	ReqHeaders  map[string]string `json:"req_headers"`
	ReqBody     string            `json:"req_body"`
	RespHeaders map[string]string `json:"resp_headers"`
	RespBody    string            `json:"resp_body"`
	Status      int               `json:"status"`
}

// 解析配置
func parseConfig(jsonConf gjson.Result, config *PluginConfig, log wrapper.Log) error {
	log.Infof("[http-log-pusher] parsing config: %s", jsonConf.String())
	
	config.CollectorClientName = jsonConf.Get("collector_service").String()
	if config.CollectorClientName == "" {
		log.Errorf("[http-log-pusher] collector_service is required in config")
		return errors.New("collector_service is required in config")
	}
	config.CollectorPath = jsonConf.Get("collector_path").String()
	if config.CollectorPath == "" {
		config.CollectorPath = "/"
	}
	config.SampleRate = jsonConf.Get("sample_rate").Int()
	if config.SampleRate == 0 {
		config.SampleRate = 100 // 默认全采
	}
	
	// 创建 HTTP 客户端用于发送日志
	config.CollectorClient = wrapper.NewClusterClient(wrapper.StaticIpCluster{
		ServiceName: config.CollectorClientName,
	})
	
	log.Infof("[http-log-pusher] config parsed successfully: collector_service=%s, collector_path=%s, sample_rate=%d",
		config.CollectorClientName, config.CollectorPath, config.SampleRate)
	return nil
}

// ---------------- 核心逻辑 ----------------

// 1. 处理请求头
func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	log.Debugf("[http-log-pusher] onHttpRequestHeaders called, path=%s, method=%s", ctx.Path(), ctx.Method())
	
	// 获取所有请求头并暂存
	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		log.Errorf("[http-log-pusher] failed to get request headers: %v", err)
	}
	ctx.SetContext("req_headers", headers)
	ctx.SetContext("start_time", time.Now().UnixMilli())

	// 必须允许继续，否则请求会卡住
	// 如果需要读取 Body，必须在 return 时不打断流
	return types.ActionContinue
}

// 2. 处理请求体
func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {
	log.Debugf("[http-log-pusher] onHttpRequestBody called, body_size=%d", len(body))
	
	if len(body) > 0 {
		// 注意：大包体可能会分多次回调，生产环境建议限制长度或做截断
		ctx.SetContext("req_body", string(body))
	}
	return types.ActionContinue
}

// 3. 处理响应头
func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	log.Debugf("[http-log-pusher] onHttpResponseHeaders called")
	
	headers, _ := proxywasm.GetHttpResponseHeaders()
	ctx.SetContext("resp_headers", headers)
	return types.ActionContinue
}

// 4. 处理响应体 (也是发送日志的最佳时机)
func onHttpResponseBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {
	log.Debugf("[http-log-pusher] onHttpResponseBody called, body_size=%d", len(body))
	
	// 1. 组装数据
	reqHeaders, _ := ctx.GetContext("req_headers").([][2]string)
	reqBody, _ := ctx.GetContext("req_body").(string)
	respHeaders, _ := ctx.GetContext("resp_headers").([][2]string)
	startTime, _ := ctx.GetContext("start_time").(int64)

	entry := LogEntry{
		Timestamp:   startTime,
		ReqHeaders:  toMap(reqHeaders),
		ReqBody:     reqBody,
		RespHeaders: toMap(respHeaders),
		RespBody:    string(body), // 同样注意截断
		Status:      200,          // 简化处理，实际可从 headers ":status" 获取
	}

	payload, _ := json.Marshal(entry)
	log.Infof("[http-log-pusher] sending log to collector: %s%s, payload_size=%d",
		config.CollectorClientName, config.CollectorPath, len(payload))

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