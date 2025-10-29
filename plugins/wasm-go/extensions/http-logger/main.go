package main

import (
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

// 常量定义
const (
	pluginName  = "http-logger"
	maxBodySize = 1024
)

// 支持的content-type类型
var allowedContentTypes = []string{
	"application/x-www-form-urlencoded",
	"application/json",
	"text/plain",
}

// 配置结构
type Config struct {
	LogRequestHeaders  bool
	LogRequestBody     bool
	LogResponseHeaders bool
	LogResponseBody    bool
}

// 初始化插件
func init() {
	wrapper.SetCtx(
		pluginName,
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
		wrapper.ProcessResponseBody(onHttpResponseBody),
	)
}

// 解析配置
func parseConfig(json gjson.Result, config *Config) error {
	// 默认全部开启
	config.LogRequestHeaders = json.Get("log_request_headers").Bool()
	config.LogRequestBody = json.Get("log_request_body").Bool()
	config.LogResponseHeaders = json.Get("log_response_headers").Bool()
	config.LogResponseBody = json.Get("log_response_body").Bool()
	return nil
}

// 检查content-type是否可读
func checkContentReadable(contentType string) bool {
	if contentType == "" {
		return false
	}
	for _, allowedType := range allowedContentTypes {
		if strings.Contains(contentType, allowedType) {
			return true
		}
	}
	return false
}

// 处理请求头
func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config) types.Action {
	// 获取所有请求头
	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		log.Errorf("Failed to get request headers: %v", err)
		return types.ActionContinue
	}
	// 构建请求头字符串
	var headersStr string
	var contentType string
	for _, header := range headers {
		key := header[0]
		value := header[1]
		if strings.ToLower(key) == "content-type" {
			contentType = value
		}
		headersStr += key + "=" + value + ", "
	}
	// 如果需要记录请求头，立即打印
	if config.LogRequestHeaders {
		log.Info("request Headers: [" + headersStr + "]")
	}
	// 保存content-type到context供请求体处理使用
	ctx.SetContext("request_content_type", contentType)
	return types.ActionContinue
}

// 处理请求体
func onHttpRequestBody(ctx wrapper.HttpContext, config Config, body []byte) types.Action {
	// 如果不需要记录请求体，直接返回
	if !config.LogRequestBody {
		return types.ActionContinue
	}
	// 获取content-type
	var contentType string
	if val, ok := ctx.GetContext("request_content_type").(string); ok {
		contentType = val
	}
	// 检查content-type是否可读
	if !checkContentReadable(contentType) {
		return types.ActionContinue
	}
	requestBody := string(body)
	// 限制大小
	if len(requestBody) > maxBodySize {
		requestBody = requestBody[:maxBodySize] + "<truncated>"
	}
	// 转义换行符
	requestBody = strings.ReplaceAll(requestBody, "\n", "\\n")
	// 打印日志
	log.Info("request Body: [" + requestBody + "]")
	return types.ActionContinue
}

// 处理响应头
func onHttpResponseHeaders(ctx wrapper.HttpContext, config Config) types.Action {
	// 获取所有响应头
	headers, err := proxywasm.GetHttpResponseHeaders()
	if err != nil {
		log.Errorf("Failed to get response headers: %v", err)
		return types.ActionContinue
	}
	// 构建响应头字符串
	var headersStr string
	var contentType string
	var hasContentEncoding bool
	for _, header := range headers {
		key := header[0]
		value := header[1]

		if strings.ToLower(key) == "content-type" {
			contentType = value
		} else if strings.ToLower(key) == "content-encoding" {
			hasContentEncoding = true
		}

		headersStr += key + "=" + value + ", "
	}
	// 如果需要记录响应头，立即打印
	if config.LogResponseHeaders {
		log.Info("response Headers: [" + headersStr + "]")
	}
	// 保存content-type和content-encoding到context供响应体处理使用
	ctx.SetContext("response_content_type", contentType)
	ctx.SetContext("response_content_encoding", hasContentEncoding)

	return types.ActionContinue
}

// 处理响应体
func onHttpResponseBody(ctx wrapper.HttpContext, config Config, body []byte) types.Action {
	// 如果不需要记录响应体，直接返回
	if !config.LogResponseBody {
		return types.ActionContinue
	}
	// 获取content-type和content-encoding
	var contentType string
	if val, ok := ctx.GetContext("response_content_type").(string); ok {
		contentType = val
	}
	var hasContentEncoding bool
	if val, ok := ctx.GetContext("response_content_encoding").(bool); ok {
		hasContentEncoding = val
	}
	// 检查content-type是否可读，且没有content-encoding
	if !checkContentReadable(contentType) || hasContentEncoding {
		return types.ActionContinue
	}
	responseBody := string(body)
	// 限制大小
	if len(responseBody) > maxBodySize {
		responseBody = responseBody[:maxBodySize] + "<truncated>"
	}
	// 转义换行符
	responseBody = strings.ReplaceAll(responseBody, "\n", "\\n")
	// 打印日志
	log.Info("response Body: [" + responseBody + "]")
	return types.ActionContinue
}
