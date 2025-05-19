package main

import (
	"encoding/json"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Constants for log keys in Filter State
const (
	pluginName            = "log-request-response"
	LogKeyRequestHeaders  = "log-request-headers"
	LogKeyRequestBody     = "log-request-body"
	LogKeyResponseHeaders = "log-response-headers"
	LogKeyResponseBody    = "log-response-body"
)

// HTTP/2 header name mapping
var http2HeaderMap = map[string]string{
	":authority": "authority",
	":method":    "method",
	":path":      "path",
	":scheme":    "scheme",
	":status":    "status",
}

func main() {
	wrapper.SetCtx(
		// Plugin name
		pluginName,
		// Set custom function for parsing plugin configuration
		wrapper.ParseConfigBy(parseConfig),
		// Set custom function for processing request headers
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		// Set custom function for processing streaming request body
		wrapper.ProcessStreamingRequestBodyBy(onStreamingRequestBody),
		// Set custom function for processing response headers
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		// Set custom function for processing streaming response body
		wrapper.ProcessStreamingResponseBodyBy(onStreamingResponseBody),
	)
}

// PluginConfig Custom plugin configuration
type PluginConfig struct {
	// Request configuration
	Request struct {
		// Headers configuration
		Headers struct {
			// Whether to enable request headers logging
			Enabled bool
		}
		// Body configuration
		Body struct {
			// Whether to enable request body logging
			Enabled bool
			// Maximum size limit for logging (bytes)
			MaxSize int
			// Content types to be logged
			ContentTypes []string
		}
	}
	// Response configuration
	Response struct {
		// Headers configuration
		Headers struct {
			// Whether to enable response headers logging
			Enabled bool
		}
		// Body configuration
		Body struct {
			// Whether to enable response body logging
			Enabled bool
			// Maximum size limit for logging (bytes)
			MaxSize int
			// Content types to be logged
			ContentTypes []string
		}
	}
}

// The YAML configuration filled in the console will be automatically converted to JSON,
// so we can directly parse the configuration from this JSON parameter
func parseConfig(json gjson.Result, config *PluginConfig, log wrapper.Log) error {
	// Parse request headers configuration
	config.Request.Headers.Enabled = json.Get("request.headers.enabled").Bool()

	// Parse request body configuration
	config.Request.Body.Enabled = json.Get("request.body.enabled").Bool()
	config.Request.Body.MaxSize = int(json.Get("request.body.maxSize").Int())

	// Set default maximum size for request body
	if config.Request.Body.MaxSize <= 0 {
		config.Request.Body.MaxSize = 10 * 1024 // Default 10KB
	}

	// Parse request body content types
	if contentTypes := json.Get("request.body.contentTypes").Array(); len(contentTypes) > 0 {
		for _, ct := range contentTypes {
			config.Request.Body.ContentTypes = append(config.Request.Body.ContentTypes, ct.String())
		}
	} else {
		// Default content types
		config.Request.Body.ContentTypes = []string{
			"application/json",
			"application/xml",
			"application/x-www-form-urlencoded",
			"text/plain",
		}
	}

	// Parse response headers configuration
	config.Response.Headers.Enabled = json.Get("response.headers.enabled").Bool()

	// Parse response body configuration
	config.Response.Body.Enabled = json.Get("response.body.enabled").Bool()
	config.Response.Body.MaxSize = int(json.Get("response.body.maxSize").Int())

	// Set default maximum size for response body
	if config.Response.Body.MaxSize <= 0 {
		config.Response.Body.MaxSize = 10 * 1024 // Default 10KB
	}

	// Parse response body content types
	if contentTypes := json.Get("response.body.contentTypes").Array(); len(contentTypes) > 0 {
		for _, ct := range contentTypes {
			config.Response.Body.ContentTypes = append(config.Response.Body.ContentTypes, ct.String())
		}
	} else {
		// Default content types
		config.Response.Body.ContentTypes = []string{
			"application/json",
			"application/xml",
			"text/plain",
			"text/html",
		}
	}

	return nil
}

// normalizeHeaderName standardizes HTTP/2 header names by removing the colon prefix
// or mapping them to more standard names
func normalizeHeaderName(name string) string {
	// If it's a known HTTP/2 header, map it to a standard name
	if standardName, exists := http2HeaderMap[name]; exists {
		return standardName
	}

	// For other headers that might start with colon, just remove the colon
	if strings.HasPrefix(name, ":") {
		return name[1:]
	}

	// Return the original name for regular headers
	return name
}

// marshalStr properly escapes a JSON string to ensure it can be safely included in another JSON
func marshalStr(raw string) string {
	// e.g. {"field1":"value1","field2":"value2"}
	helper := map[string]string{
		"placeholder": raw,
	}
	marshalledHelper, _ := json.Marshal(helper)
	marshalledRaw := gjson.GetBytes(marshalledHelper, "placeholder").Raw
	if len(marshalledRaw) >= 2 {
		// e.g. {\"field1\":\"value1\",\"field2\":\"value2\"}
		return marshalledRaw[1 : len(marshalledRaw)-1]
	} else {
		proxywasm.LogErrorf("failed to marshal json string, raw string is: %s", raw)
		return ""
	}
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	// Get all request headers
	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		log.Errorf("Failed to get request headers: %v", err)
		return types.ActionContinue
	}

	method := ""
	contentType := ""

	// Check if request headers need to be logged
	if config.Request.Headers.Enabled {
		jsonStr := "{}"
		for _, header := range headers {
			var err error
			normalizedName := normalizeHeaderName(header[0])
			jsonStr, err = sjson.Set(jsonStr, normalizedName, header[1])
			if err != nil {
				log.Errorf("Failed to convert request headers to JSON: %v", err)
			}
		}

		marshalledJsonStr := marshalStr(jsonStr)
		if err := proxywasm.SetProperty([]string{LogKeyRequestHeaders}, []byte(marshalledJsonStr)); err != nil {
			log.Errorf("failed to set %s in filter state, err: %v, raw:\n%s", LogKeyRequestHeaders, err, jsonStr)
		}
	}

	// Get request method and Content-Type for subsequent processing
	for _, header := range headers {
		if strings.ToLower(header[0]) == ":method" {
			method = header[1]
		} else if strings.ToLower(header[0]) == "content-type" {
			contentType = header[1]
		}
	}

	// For non-POST/PUT/PATCH requests, or if request body logging is not enabled, no need to log the request body
	if !config.Request.Body.Enabled || (method != "POST" && method != "PUT" && method != "PATCH") {
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}

	// Check if the content type is in the configured list for logging
	shouldLogBody := false
	for _, allowedType := range config.Request.Body.ContentTypes {
		if strings.Contains(contentType, allowedType) {
			shouldLogBody = true
			break
		}
	}

	if !shouldLogBody {
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}

	// Initialize a buffer to accumulate request body chunks
	ctx.SetContext("request_body_buffer", []byte{})

	return types.ActionContinue
}

// onStreamingRequestBody processes each chunk of the request body in streaming mode
// This allows us to log the request body without affecting the original request
func onStreamingRequestBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isEndStream bool, log wrapper.Log) []byte {
	// If request body logging is not enabled or max size is <= 0, just return the chunk as is
	if !config.Request.Body.Enabled || config.Request.Body.MaxSize <= 0 {
		return chunk
	}

	// Get the buffer from context
	buffer, _ := ctx.GetContext("request_body_buffer").([]byte)

	// If we haven't reached max size yet, append chunk to buffer
	if len(buffer) < config.Request.Body.MaxSize {
		// Calculate how much of this chunk we can add
		remainingCapacity := config.Request.Body.MaxSize - len(buffer)
		if remainingCapacity > 0 {
			if len(chunk) <= remainingCapacity {
				buffer = append(buffer, chunk...)
			} else {
				buffer = append(buffer, chunk[:remainingCapacity]...)
			}
			ctx.SetContext("request_body_buffer", buffer)
		}
	}

	// When we reach the end of stream, create log entry
	if isEndStream && len(buffer) > 0 {
		bodyStr := string(buffer)
		marshalledBodyStr := marshalStr(bodyStr)
		if err := proxywasm.SetProperty([]string{LogKeyRequestBody}, []byte(marshalledBodyStr)); err != nil {
			log.Errorf("failed to set %s in filter state, err: %v, raw:\n%s", LogKeyRequestBody, err, bodyStr)
		}
	}

	// Always return the original chunk unmodified
	return chunk
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	// Get all response headers
	headers, err := proxywasm.GetHttpResponseHeaders()
	if err != nil {
		log.Errorf("Failed to get response headers: %v", err)
		return types.ActionContinue
	}

	// Check if response headers need to be logged
	if config.Response.Headers.Enabled {
		jsonStr := "{}"
		for _, header := range headers {
			var err error
			normalizedName := normalizeHeaderName(header[0])
			jsonStr, err = sjson.Set(jsonStr, normalizedName, header[1])
			if err != nil {
				log.Errorf("Failed to convert response headers to JSON: %v", err)
			}
		}

		marshalledJsonStr := marshalStr(jsonStr)
		if err := proxywasm.SetProperty([]string{LogKeyResponseHeaders}, []byte(marshalledJsonStr)); err != nil {
			log.Errorf("failed to set %s in filter state, err: %v, raw:\n%s", LogKeyResponseHeaders, err, jsonStr)
		}
	}

	// Check if response body needs to be logged
	if !config.Response.Body.Enabled {
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}

	// Check Content-Type for response body logging
	contentType := ""
	for _, header := range headers {
		if strings.ToLower(header[0]) == "content-type" {
			contentType = header[1]
			break
		}
	}

	// Skip response body logging if content type is not in the configured list
	if contentType != "" {
		shouldLogBody := false
		for _, allowedType := range config.Response.Body.ContentTypes {
			if strings.Contains(contentType, allowedType) {
				shouldLogBody = true
				break
			}
		}

		if !shouldLogBody {
			ctx.DontReadResponseBody()
			return types.ActionContinue
		}
	}

	// Initialize a buffer to accumulate response body chunks
	ctx.SetContext("response_body_buffer", []byte{})

	return types.ActionContinue
}

// onStreamingResponseBody processes each chunk of the response body in streaming mode
// This allows us to log the response body without affecting the original response
func onStreamingResponseBody(ctx wrapper.HttpContext, config PluginConfig, chunk []byte, isEndStream bool, log wrapper.Log) []byte {
	// If response body logging is not enabled or max size is <= 0, just return the chunk as is
	if !config.Response.Body.Enabled || config.Response.Body.MaxSize <= 0 {
		return chunk
	}

	// Get the buffer from context
	buffer, _ := ctx.GetContext("response_body_buffer").([]byte)

	// If we haven't reached max size yet, append chunk to buffer
	if len(buffer) < config.Response.Body.MaxSize {
		// Calculate how much of this chunk we can add
		remainingCapacity := config.Response.Body.MaxSize - len(buffer)
		if remainingCapacity > 0 {
			if len(chunk) <= remainingCapacity {
				buffer = append(buffer, chunk...)
			} else {
				buffer = append(buffer, chunk[:remainingCapacity]...)
			}
			ctx.SetContext("response_body_buffer", buffer)
		}
	}

	// When we reach the end of stream, create log entry
	if isEndStream && len(buffer) > 0 {
		bodyStr := string(buffer)
		marshalledBodyStr := marshalStr(bodyStr)
		if err := proxywasm.SetProperty([]string{LogKeyResponseBody}, []byte(marshalledBodyStr)); err != nil {
			log.Errorf("failed to set %s in filter state, err: %v, raw:\n%s", LogKeyResponseBody, err, bodyStr)
		}
	}

	// Always return the original chunk unmodified
	return chunk
}
