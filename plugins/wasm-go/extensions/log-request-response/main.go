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
		"log-request-response",
		// Set custom function for parsing plugin configuration
		wrapper.ParseConfigBy(parseConfig),
		// Set custom function for processing request headers
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		// Set custom function for processing request body
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		// Set custom function for processing response headers
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		// Set custom function for processing response body
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

// PluginConfig Custom plugin configuration
type PluginConfig struct {
	// Whether to log request headers
	logRequestHeaders bool
	// Whether to log request body
	logRequestBody bool
	// Whether to log response headers
	logResponseHeaders bool
	// Whether to log response body
	logResponseBody bool
	// Content types for request body logging (Content-Type)
	requestBodyContentTypes []string
	// Maximum size limit for logging (bytes)
	maxBodySize int
}

// The YAML configuration filled in the console will be automatically converted to JSON,
// so we can directly parse the configuration from this JSON parameter
func parseConfig(json gjson.Result, config *PluginConfig, log wrapper.Log) error {
	config.logRequestHeaders = json.Get("logRequestHeaders").Bool()
	config.logRequestBody = json.Get("logRequestBody").Bool()
	config.logResponseHeaders = json.Get("logResponseHeaders").Bool()
	config.logResponseBody = json.Get("logResponseBody").Bool()

	if contentTypes := json.Get("requestBodyContentTypes").Array(); len(contentTypes) > 0 {
		for _, ct := range contentTypes {
			config.requestBodyContentTypes = append(config.requestBodyContentTypes, ct.String())
		}
	} else {
		// Default common content types to record
		config.requestBodyContentTypes = []string{
			"application/json",
			"application/xml",
			"application/x-www-form-urlencoded",
			"text/plain",
		}
	}

	maxSize := json.Get("maxBodySize").Int()
	if maxSize <= 0 {
		// Default maximum size is 10KB
		config.maxBodySize = 10 * 1024
	} else {
		config.maxBodySize = int(maxSize)
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
	if config.logRequestHeaders {
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
			log.Errorf("failed to set %s in filter state, raw is %s, err is %v", LogKeyRequestHeaders, jsonStr, err)
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
	if !config.logRequestBody || (method != "POST" && method != "PUT" && method != "PATCH") {
		ctx.SetContext("skip_request_body", true)
		return types.HeaderContinue
	}

	// Check if the content type is in the configured list for logging
	shouldLogBody := false
	for _, allowedType := range config.requestBodyContentTypes {
		if strings.Contains(contentType, allowedType) {
			shouldLogBody = true
			break
		}
	}
	ctx.SetContext("skip_request_body", !shouldLogBody)

	return types.HeaderContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {
	skipRequestBody, ok := ctx.GetContext("skip_request_body").(bool)
	if ok && skipRequestBody {
		return types.ActionContinue
	}

	// Get request body with size limitation
	bodySize := len(body)
	if bodySize > config.maxBodySize {
		bodySize = config.maxBodySize
	}

	if bodySize > 0 {
		bodyStr := string(body[:bodySize])

		// For request body, we use marshalStr to ensure proper escaping
		marshalledBodyStr := marshalStr(bodyStr)
		if err := proxywasm.SetProperty([]string{LogKeyRequestBody}, []byte(marshalledBodyStr)); err != nil {
			log.Errorf("failed to set %s in filter state, raw is %s, err is %v", LogKeyRequestBody, bodyStr, err)
		}
	}

	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig, log wrapper.Log) types.Action {
	// Get all response headers
	headers, err := proxywasm.GetHttpResponseHeaders()
	if err != nil {
		log.Errorf("Failed to get response headers: %v", err)
		return types.ActionContinue
	}

	// Check if response headers need to be logged
	if config.logResponseHeaders {
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
			log.Errorf("failed to set %s in filter state, raw is %s, err is %v", LogKeyResponseHeaders, jsonStr, err)
		}
	}

	// Check if response body needs to be logged
	if !config.logResponseBody {
		ctx.SetContext("skip_response_body", true)
		return types.ActionContinue
	}

	// Using HeaderContinue from ABI version 0.2.100 indicates that the current filter has completed processing
	// and can be passed to the next filter
	return types.HeaderContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config PluginConfig, body []byte, log wrapper.Log) types.Action {
	skipResponseBody, ok := ctx.GetContext("skip_response_body").(bool)
	if ok && skipResponseBody {
		return types.ActionContinue
	}

	// Get response body with size limitation
	bodySize := len(body)
	if bodySize > config.maxBodySize {
		bodySize = config.maxBodySize
	}

	if bodySize > 0 {
		bodyStr := string(body[:bodySize])

		// For response body, we use marshalStr to ensure proper escaping
		marshalledBodyStr := marshalStr(bodyStr)
		if err := proxywasm.SetProperty([]string{LogKeyResponseBody}, []byte(marshalledBodyStr)); err != nil {
			log.Errorf("failed to set %s in filter state, raw is %s, err is %v", LogKeyResponseBody, bodyStr, err)
		}
	}

	return types.ActionContinue
}
