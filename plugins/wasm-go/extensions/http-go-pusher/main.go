package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"http-log-pusher",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

// PluginConfig 定义插件配置 (对应 WasmPlugin 资源中的 pluginConfig)
type PluginConfig struct {
	CollectorClientName string `json:"collector_service"` // Envoy 集群名，例如 "outbound|8080||collector-service.default.svc.cluster.local"
	CollectorPath       string `json:"collector_path"`    // 接收日志的 API 路径，例如 "/api/log"
	SampleRate          int64  `json:"sample_rate"`       // 采样率 0-100
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
	config.CollectorClientName = jsonConf.Get("collector_service").String()
	if config.CollectorClientName == "" {
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
	return nil
}

// ---------------- 核心逻辑 ----------------

// 1. 处理请求头
func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	// 获取所有请求头并暂存
	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		log.Errorf("failed to get request headers: %v", err)
	}
	ctx.SetContext("req_headers", headers)
	ctx.SetContext("start_time", time.Now().UnixMilli())

	// 必须允许继续，否则请求会卡住
	// 如果需要读取 Body，必须在 return 时不打断流
	return types.ActionContinue
}

// 2. 处理请求体
func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {
	if len(body) > 0 {
		// 注意：大包体可能会分多次回调，生产环境建议限制长度或做截断
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

	// 2. 发送异步请求给 Collector
	// 注意：DispatchHttpCall 是异步的，不会阻塞当前客户端响应
	headers := [][2]string{
		{":method", "POST"},
		{":path", config.CollectorPath},
		{":authority", "collector"},
		{"Content-Type", "application/json"},
	}

	// 这里的 5000 是超时时间(ms)
	if _, err := proxywasm.DispatchHttpCall(
		config.CollectorClientName, 
		headers, 
		payload, 
		nil, // 不接收回调，Fire-and-forget
		5000,
	); err != nil {
		log.Errorf("dispatch http call failed: %v", err)
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