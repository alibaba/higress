package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	mrand "math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"ai-security-guard",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

const (
	OpenAIResponseFormat       = `{"id": "%s","object":"chat.completion","model":%s,"choices":[{"index":0,"message":{"role":"assistant","content":%s},"logprobs":null,"finish_reason":"stop"}]}`
	OpenAIStreamResponseChunk  = `data:{"id":"%s","object":"chat.completion.chunk","model":%s,"choices":[{"index":0,"delta":{"role":"assistant","content":%s},"logprobs":null,"finish_reason":null}]}`
	OpenAIStreamResponseEnd    = `data:{"id":"%s","object":"chat.completion.chunk","model": %s,"choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"stop"}]}`
	OpenAIStreamResponseFormat = OpenAIStreamResponseChunk + "\n\n" + OpenAIStreamResponseEnd + "\n\n" + `data: [DONE]`

	TracingPrefix = "trace_span_tag."

	DefaultRequestCheckService       = "llm_query_moderation"
	DefaultResponseCheckService      = "llm_response_moderation"
	DefaultRequestJsonPath           = "messages.@reverse.0.content"
	DefaultResponseJsonPath          = "choices.0.message.content"
	DefaultStreamingResponseJsonPath = "choices.0.delta.content"
	DefaultDenyCode                  = 200
	DefaultDenyMessage               = "很抱歉，我无法回答您的问题"

	AliyunUserAgent = "CIPFrom/AIGateway"
)

type AISecurityConfig struct {
	client                        wrapper.HttpClient
	ak                            string
	sk                            string
	checkRequest                  bool
	requestCheckService           string
	requestContentJsonPath        string
	checkResponse                 bool
	responseCheckService          string
	responseContentJsonPath       string
	responseStreamContentJsonPath string
	denyCode                      int64
	denyMessage                   string
	protocolOriginal              bool
	metrics                       map[string]proxywasm.MetricCounter
}

func (config *AISecurityConfig) incrementCounter(metricName string, inc uint64) {
	counter, ok := config.metrics[metricName]
	if !ok {
		counter = proxywasm.DefineCounterMetric(metricName)
		config.metrics[metricName] = counter
	}
	counter.Increment(inc)
}

func urlEncoding(rawStr string) string {
	encodedStr := url.PathEscape(rawStr)
	encodedStr = strings.ReplaceAll(encodedStr, "+", "%2B")
	encodedStr = strings.ReplaceAll(encodedStr, ":", "%3A")
	encodedStr = strings.ReplaceAll(encodedStr, "=", "%3D")
	encodedStr = strings.ReplaceAll(encodedStr, "&", "%26")
	return encodedStr
}

func hmacSha1(message, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha1.New, key)
	h.Write([]byte(message))
	hash := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(hash)
}

func getSign(params map[string]string, secret string) string {
	paramArray := []string{}
	for k, v := range params {
		paramArray = append(paramArray, urlEncoding(k)+"="+urlEncoding(v))
	}
	sort.Slice(paramArray, func(i, j int) bool {
		return paramArray[i] <= paramArray[j]
	})
	canonicalStr := strings.Join(paramArray, "&")
	signStr := "POST&%2F&" + urlEncoding(canonicalStr)
	proxywasm.LogDebugf("String to sign is: %s", signStr)
	return hmacSha1(signStr, secret)
}

func generateHexID(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func parseConfig(json gjson.Result, config *AISecurityConfig, log wrapper.Log) error {
	serviceName := json.Get("serviceName").String()
	servicePort := json.Get("servicePort").Int()
	serviceHost := json.Get("serviceHost").String()
	if serviceName == "" || servicePort == 0 || serviceHost == "" {
		return errors.New("invalid service config")
	}
	config.ak = json.Get("accessKey").String()
	config.sk = json.Get("secretKey").String()
	if config.ak == "" || config.sk == "" {
		return errors.New("invalid AK/SK config")
	}
	config.checkRequest = json.Get("checkRequest").Bool()
	config.checkResponse = json.Get("checkResponse").Bool()
	config.protocolOriginal = json.Get("protocol").String() == "original"
	config.denyMessage = json.Get("denyMessage").String()
	if obj := json.Get("denyCode"); obj.Exists() {
		config.denyCode = obj.Int()
	} else {
		config.denyCode = DefaultDenyCode
	}
	if obj := json.Get("requestCheckService"); obj.Exists() {
		config.requestCheckService = obj.String()
	} else {
		config.requestCheckService = DefaultRequestCheckService
	}
	if obj := json.Get("responseCheckService"); obj.Exists() {
		config.responseCheckService = obj.String()
	} else {
		config.responseCheckService = DefaultResponseCheckService
	}
	if obj := json.Get("requestContentJsonPath"); obj.Exists() {
		config.requestContentJsonPath = obj.String()
	} else {
		config.requestContentJsonPath = DefaultRequestJsonPath
	}
	if obj := json.Get("responseContentJsonPath"); obj.Exists() {
		config.responseContentJsonPath = obj.String()
	} else {
		config.responseContentJsonPath = DefaultResponseJsonPath
	}
	if obj := json.Get("responseStreamContentJsonPath"); obj.Exists() {
		config.responseStreamContentJsonPath = obj.String()
	} else {
		config.responseStreamContentJsonPath = DefaultStreamingResponseJsonPath
	}
	config.client = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: servicePort,
		Host: serviceHost,
	})
	config.metrics = make(map[string]proxywasm.MetricCounter)
	return nil
}

func generateRandomID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 29)
	for i := range b {
		b[i] = charset[mrand.Intn(len(charset))]
	}
	return "chatcmpl-" + string(b)
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AISecurityConfig, log wrapper.Log) types.Action {
	if !config.checkRequest {
		log.Debugf("request checking is disabled")
		ctx.DontReadRequestBody()
	}
	if !config.checkResponse {
		log.Debugf("response checking is disabled")
		ctx.DontReadResponseBody()
	}
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AISecurityConfig, body []byte, log wrapper.Log) types.Action {
	log.Debugf("checking request body...")
	content := gjson.GetBytes(body, config.requestContentJsonPath).Raw
	model := gjson.GetBytes(body, "model").Raw
	ctx.SetContext("requestModel", model)
	log.Debugf("Raw response content is: %s", content)
	if len(content) > 0 {
		timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		randomID, _ := generateHexID(16)
		params := map[string]string{
			"Format":            "JSON",
			"Version":           "2022-03-02",
			"SignatureMethod":   "Hmac-SHA1",
			"SignatureNonce":    randomID,
			"SignatureVersion":  "1.0",
			"Action":            "TextModerationPlus",
			"AccessKeyId":       config.ak,
			"Timestamp":         timestamp,
			"Service":           config.requestCheckService,
			"ServiceParameters": fmt.Sprintf(`{"content": %s}`, content),
		}
		signature := getSign(params, config.sk+"&")
		reqParams := url.Values{}
		for k, v := range params {
			reqParams.Add(k, v)
		}
		reqParams.Add("Signature", signature)
		err := config.client.Post(fmt.Sprintf("/?%s", reqParams.Encode()), [][2]string{{"User-Agent", AliyunUserAgent}}, nil,
			func(statusCode int, responseHeaders http.Header, responseBody []byte) {
				if statusCode != 200 {
					log.Error(string(responseBody))
					proxywasm.ResumeHttpRequest()
					return
				}
				respData := gjson.GetBytes(responseBody, "Data")
				if respData.Exists() {
					respAdvice := respData.Get("Advice")
					respResult := respData.Get("Result")
					var denyMessage string
					messageNeedSerialization := true
					if config.protocolOriginal {
						// not openai
						if config.denyMessage != "" {
							denyMessage = config.denyMessage
						} else if respAdvice.Exists() {
							denyMessage = respAdvice.Array()[0].Get("Answer").Raw
							messageNeedSerialization = false
						} else {
							denyMessage = DefaultDenyMessage
						}
					} else {
						// openai
						if respAdvice.Exists() {
							denyMessage = respAdvice.Array()[0].Get("Answer").Raw
							messageNeedSerialization = false
						} else if config.denyMessage != "" {
							denyMessage = config.denyMessage
						} else {
							denyMessage = DefaultDenyMessage
						}
					}
					if messageNeedSerialization {
						if data, err := json.Marshal(denyMessage); err == nil {
							denyMessage = string(data)
						} else {
							denyMessage = fmt.Sprintf("\"%s\"", DefaultDenyMessage)
						}
					}
					if respResult.Array()[0].Get("Label").String() != "nonLabel" {
						proxywasm.SetProperty([]string{TracingPrefix, "ai_sec_risklabel"}, []byte(respResult.Array()[0].Get("Label").String()))
						proxywasm.SetProperty([]string{TracingPrefix, "ai_sec_deny_phase"}, []byte("request"))
						config.incrementCounter("ai_sec_request_deny", 1)
						if config.protocolOriginal {
							proxywasm.SendHttpResponse(uint32(config.denyCode), [][2]string{{"content-type", "application/json"}}, []byte(denyMessage), -1)
						} else if gjson.GetBytes(body, "stream").Bool() {
							randomID := generateRandomID()
							jsonData := []byte(fmt.Sprintf(OpenAIStreamResponseFormat, randomID, model, denyMessage, randomID, model))
							proxywasm.SendHttpResponse(uint32(config.denyCode), [][2]string{{"content-type", "text/event-stream;charset=UTF-8"}}, jsonData, -1)
						} else {
							randomID := generateRandomID()
							jsonData := []byte(fmt.Sprintf(OpenAIResponseFormat, randomID, model, denyMessage))
							proxywasm.SendHttpResponse(uint32(config.denyCode), [][2]string{{"content-type", "application/json"}}, jsonData, -1)
						}
						ctx.DontReadResponseBody()
					} else {
						proxywasm.ResumeHttpRequest()
					}
				} else {
					proxywasm.ResumeHttpRequest()
				}
			},
		)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			return types.ActionContinue
		}
		return types.ActionPause
	} else {
		log.Debugf("request content is empty. skip")
		return types.ActionContinue
	}
}

func convertHeaders(hs [][2]string) map[string][]string {
	ret := make(map[string][]string)
	for _, h := range hs {
		k, v := strings.ToLower(h[0]), h[1]
		ret[k] = append(ret[k], v)
	}
	return ret
}

// headers: map[string][]string -> [][2]string
func reconvertHeaders(hs map[string][]string) [][2]string {
	var ret [][2]string
	for k, vs := range hs {
		for _, v := range vs {
			ret = append(ret, [2]string{k, v})
		}
	}
	sort.SliceStable(ret, func(i, j int) bool {
		return ret[i][0] < ret[j][0]
	})
	return ret
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config AISecurityConfig, log wrapper.Log) types.Action {
	headers, err := proxywasm.GetHttpResponseHeaders()
	if err != nil {
		log.Warnf("failed to get response headers: %v", err)
		return types.ActionContinue
	}
	hdsMap := convertHeaders(headers)
	ctx.SetContext("headers", hdsMap)
	return types.HeaderStopIteration
}

func onHttpResponseBody(ctx wrapper.HttpContext, config AISecurityConfig, body []byte, log wrapper.Log) types.Action {
	log.Debugf("checking response body...")
	hdsMap := ctx.GetContext("headers").(map[string][]string)
	isStreamingResponse := strings.Contains(strings.Join(hdsMap["content-type"], ";"), "event-stream")
	model := ctx.GetStringContext("requestModel", "unknown")
	var content string
	if isStreamingResponse {
		content = extractMessageFromStreamingBody(body, config.responseStreamContentJsonPath)
	} else {
		content = gjson.GetBytes(body, config.responseContentJsonPath).Raw
	}
	log.Debugf("Raw response content is: %s", content)
	if len(content) > 0 {
		timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		randomID, _ := generateHexID(16)
		params := map[string]string{
			"Format":            "JSON",
			"Version":           "2022-03-02",
			"SignatureMethod":   "Hmac-SHA1",
			"SignatureNonce":    randomID,
			"SignatureVersion":  "1.0",
			"Action":            "TextModerationPlus",
			"AccessKeyId":       config.ak,
			"Timestamp":         timestamp,
			"Service":           config.responseCheckService,
			"ServiceParameters": fmt.Sprintf(`{"content": %s}`, content),
		}
		signature := getSign(params, config.sk+"&")
		reqParams := url.Values{}
		for k, v := range params {
			reqParams.Add(k, v)
		}
		reqParams.Add("Signature", signature)
		err := config.client.Post(fmt.Sprintf("/?%s", reqParams.Encode()), [][2]string{{"User-Agent", AliyunUserAgent}}, nil,
			func(statusCode int, responseHeaders http.Header, responseBody []byte) {
				defer proxywasm.ResumeHttpResponse()
				if statusCode != 200 {
					log.Error(string(responseBody))
					return
				}
				respData := gjson.GetBytes(responseBody, "Data")
				if respData.Exists() {
					respAdvice := respData.Get("Advice")
					respResult := respData.Get("Result")
					var denyMessage string
					if config.protocolOriginal {
						// not openai
						if config.denyMessage != "" {
							denyMessage = config.denyMessage
						} else if respAdvice.Exists() {
							denyMessage = respAdvice.Array()[0].Get("Answer").Raw
						} else {
							denyMessage = DefaultDenyMessage
						}
					} else {
						// openai
						if respAdvice.Exists() {
							denyMessage = respAdvice.Array()[0].Get("Answer").Raw
						} else if config.denyMessage != "" {
							denyMessage = config.denyMessage
						} else {
							denyMessage = DefaultDenyMessage
						}
					}
					if respResult.Array()[0].Get("Label").String() != "nonLabel" {
						var jsonData []byte
						if config.protocolOriginal {
							jsonData = []byte(denyMessage)
						} else if strings.Contains(strings.Join(hdsMap["content-type"], ";"), "event-stream") {
							randomID := generateRandomID()
							jsonData = []byte(fmt.Sprintf(OpenAIStreamResponseFormat, randomID, model, denyMessage, randomID, model))
						} else {
							randomID := generateRandomID()
							jsonData = []byte(fmt.Sprintf(OpenAIResponseFormat, randomID, model, denyMessage))
						}
						delete(hdsMap, "content-length")
						hdsMap[":status"] = []string{fmt.Sprint(config.denyCode)}
						proxywasm.ReplaceHttpResponseHeaders(reconvertHeaders(hdsMap))
						proxywasm.ReplaceHttpResponseBody(jsonData)
						proxywasm.SetProperty([]string{TracingPrefix, "ai_sec_risklabel"}, []byte(respResult.Array()[0].Get("Label").String()))
						proxywasm.SetProperty([]string{TracingPrefix, "ai_sec_deny_phase"}, []byte("response"))
						config.incrementCounter("ai_sec_response_deny", 1)
					}
				}
			},
		)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			return types.ActionContinue
		}
		return types.ActionPause
	} else {
		log.Debugf("request content is empty. skip")
		return types.ActionContinue
	}
}

func extractMessageFromStreamingBody(data []byte, jsonPath string) string {
	chunks := bytes.Split(bytes.TrimSpace(data), []byte("\n\n"))
	strChunks := []string{}
	for _, chunk := range chunks {
		// Example: "choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":null}]
		jsonRaw := gjson.GetBytes(chunk, jsonPath).Raw
		if len(jsonRaw) > 2 {
			strChunks = append(strChunks, jsonRaw[1:len(jsonRaw)-1])
		}
	}
	return fmt.Sprintf(`"%s"`, strings.Join(strChunks, ""))
}
