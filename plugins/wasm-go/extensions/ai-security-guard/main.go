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
	MaxRisk    = "max"
	HighRisk   = "high"
	MediumRisk = "medium"
	LowRisk    = "low"
	NoRisk     = "none"

	OpenAIResponseFormat       = `{"id": "%s","object":"chat.completion","model":"%s","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":"stop"}]}`
	OpenAIStreamResponseChunk  = `data:{"id":"%s","object":"chat.completion.chunk","model":"%s","choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":null}]}`
	OpenAIStreamResponseEnd    = `data:{"id":"%s","object":"chat.completion.chunk","model":"%s","choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"stop"}]}`
	OpenAIStreamResponseFormat = OpenAIStreamResponseChunk + "\n\n" + OpenAIStreamResponseEnd + "\n\n" + `data: [DONE]`

	DefaultRequestCheckService       = "llm_query_moderation"
	DefaultResponseCheckService      = "llm_response_moderation"
	DefaultRequestJsonPath           = "messages.@reverse.0.content"
	DefaultResponseJsonPath          = "choices.0.message.content"
	DefaultStreamingResponseJsonPath = "choices.0.delta.content"
	DefaultDenyCode                  = 200
	DefaultDenyMessage               = "很抱歉，我无法回答您的问题"

	AliyunUserAgent = "CIPFrom/AIGateway"
	LengthLimit     = 1800
)

type Response struct {
	Code      int    `json:"Code"`
	Message   string `json:"Message"`
	RequestId string `json:"RequestId"`
	Data      Data   `json:"Data"`
}

type Data struct {
	RiskLevel string   `json:"RiskLevel"`
	Result    []Result `json:"Result,omitempty"`
	Advice    []Advice `json:"Advice,omitempty"`
}

type Result struct {
	RiskWords   string  `json:"RiskWords,omitempty"`
	Description string  `json:"Description,omitempty"`
	Confidence  float64 `json:"Confidence,omitempty"`
	Label       string  `json:"Label,omitempty"`
}

type Advice struct {
	Answer     string `json:"Answer,omitempty"`
	HitLabel   string `json:"HitLabel,omitempty"`
	HitLibName string `json:"HitLibName,omitempty"`
}

type AISecurityConfig struct {
	client                        wrapper.HttpClient
	ak                            string
	sk                            string
	token                         string
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
	riskLevelBar                  string
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

func riskLevelToInt(riskLevel string) int {
	switch riskLevel {
	case MaxRisk:
		return 4
	case HighRisk:
		return 3
	case MediumRisk:
		return 2
	case LowRisk:
		return 1
	case NoRisk:
		return 0
	default:
		return -1
	}
}

func urlEncoding(rawStr string) string {
	encodedStr := url.PathEscape(rawStr)
	encodedStr = strings.ReplaceAll(encodedStr, "+", "%2B")
	encodedStr = strings.ReplaceAll(encodedStr, ":", "%3A")
	encodedStr = strings.ReplaceAll(encodedStr, "=", "%3D")
	encodedStr = strings.ReplaceAll(encodedStr, "&", "%26")
	encodedStr = strings.ReplaceAll(encodedStr, "$", "%24")
	encodedStr = strings.ReplaceAll(encodedStr, "@", "%40")
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
	config.token = json.Get("securityToken").String()
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
	if obj := json.Get("riskLevelBar"); obj.Exists() {
		config.riskLevelBar = obj.String()
		if riskLevelToInt(config.riskLevelBar) <= 0 {
			return errors.New("invalid risk level, value must be one of [max, high, medium, low]")
		}
	} else {
		config.riskLevelBar = HighRisk
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
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AISecurityConfig, body []byte, log wrapper.Log) types.Action {
	log.Debugf("checking request body...")
	content := gjson.GetBytes(body, config.requestContentJsonPath).String()
	model := gjson.GetBytes(body, "model").String()
	ctx.SetContext("requestModel", model)
	log.Debugf("Raw request content is: %s", content)
	if len(content) == 0 {
		log.Info("request content is empty. skip")
		return types.ActionContinue
	}
	contentIndex := 0
	sessionID, _ := generateHexID(20)
	var singleCall func()
	callback := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		log.Info(string(responseBody))
		if statusCode != 200 || gjson.GetBytes(responseBody, "Code").Int() != 200 {
			proxywasm.ResumeHttpRequest()
			return
		}
		var response Response
		err := json.Unmarshal(responseBody, &response)
		if err != nil {
			log.Error("failed to unmarshal aliyun content security response at request phase")
			proxywasm.ResumeHttpRequest()
			return
		}
		if riskLevelToInt(response.Data.RiskLevel) < riskLevelToInt(config.riskLevelBar) {
			if contentIndex >= len(content) {
				proxywasm.ResumeHttpRequest()
			} else {
				singleCall()
			}
			return
		}
		denyMessage := DefaultDenyMessage
		if config.denyMessage != "" {
			denyMessage = config.denyMessage
		} else if response.Data.Advice != nil && response.Data.Advice[0].Answer != "" {
			denyMessage = response.Data.Advice[0].Answer
		}
		marshalledDenyMessage := marshalStr(denyMessage, log)
		if config.protocolOriginal {
			proxywasm.SendHttpResponse(uint32(config.denyCode), [][2]string{{"content-type", "application/json"}}, []byte(marshalledDenyMessage), -1)
		} else if gjson.GetBytes(body, "stream").Bool() {
			randomID := generateRandomID()
			jsonData := []byte(fmt.Sprintf(OpenAIStreamResponseFormat, randomID, model, marshalledDenyMessage, randomID, model))
			proxywasm.SendHttpResponse(uint32(config.denyCode), [][2]string{{"content-type", "text/event-stream;charset=UTF-8"}}, jsonData, -1)
		} else {
			randomID := generateRandomID()
			jsonData := []byte(fmt.Sprintf(OpenAIResponseFormat, randomID, model, marshalledDenyMessage))
			proxywasm.SendHttpResponse(uint32(config.denyCode), [][2]string{{"content-type", "application/json"}}, jsonData, -1)
		}
		ctx.DontReadResponseBody()
		config.incrementCounter("ai_sec_request_deny", 1)
		proxywasm.ResumeHttpRequest()
	}
	singleCall = func() {
		timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		randomID, _ := generateHexID(16)
		var nextContentIndex int
		if contentIndex+LengthLimit >= len(content) {
			nextContentIndex = len(content)
		} else {
			nextContentIndex = contentIndex + LengthLimit
		}
		contentPiece := content[contentIndex:nextContentIndex]
		contentIndex = nextContentIndex
		log.Debugf("current content piece: %s", contentPiece)
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
			"ServiceParameters": fmt.Sprintf(`{"sessionId": "%s","content": "%s"}`, sessionID, marshalStr(contentPiece, log)),
		}
		if config.token != "" {
			params["SecurityToken"] = config.token
		}
		signature := getSign(params, config.sk+"&")
		reqParams := url.Values{}
		for k, v := range params {
			reqParams.Add(k, v)
		}
		reqParams.Add("Signature", signature)
		err := config.client.Post(fmt.Sprintf("/?%s", reqParams.Encode()), [][2]string{{"User-Agent", AliyunUserAgent}}, nil, callback)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			proxywasm.ResumeHttpRequest()
		}
	}
	singleCall()
	return types.ActionPause
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
	if !config.checkResponse {
		log.Debugf("response checking is disabled")
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}
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
		content = gjson.GetBytes(body, config.responseContentJsonPath).String()
	}
	log.Debugf("Raw response content is: %s", content)
	if len(content) == 0 {
		log.Info("response content is empty. skip")
		return types.ActionContinue
	}
	contentIndex := 0
	sessionID, _ := generateHexID(20)
	var singleCall func()
	callback := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		log.Info(string(responseBody))
		if statusCode != 200 || gjson.GetBytes(responseBody, "Code").Int() != 200 {
			proxywasm.ResumeHttpResponse()
			return
		}
		var response Response
		err := json.Unmarshal(responseBody, &response)
		if err != nil {
			log.Error("failed to unmarshal aliyun content security response at response phase")
			proxywasm.ResumeHttpResponse()
			return
		}
		if riskLevelToInt(response.Data.RiskLevel) < riskLevelToInt(config.riskLevelBar) {
			if contentIndex >= len(content) {
				proxywasm.ResumeHttpResponse()
			} else {
				singleCall()
			}
			return
		}
		denyMessage := DefaultDenyMessage
		if config.denyMessage != "" {
			denyMessage = config.denyMessage
		} else if response.Data.Advice != nil && response.Data.Advice[0].Answer != "" {
			denyMessage = response.Data.Advice[0].Answer
		}
		marshalledDenyMessage := marshalStr(denyMessage, log)
		var jsonData []byte
		if config.protocolOriginal {
			jsonData = []byte(marshalledDenyMessage)
		} else if isStreamingResponse {
			randomID := generateRandomID()
			jsonData = []byte(fmt.Sprintf(OpenAIStreamResponseFormat, randomID, model, marshalledDenyMessage, randomID, model))
		} else {
			randomID := generateRandomID()
			jsonData = []byte(fmt.Sprintf(OpenAIResponseFormat, randomID, model, marshalledDenyMessage))
		}
		delete(hdsMap, "content-length")
		hdsMap[":status"] = []string{fmt.Sprint(config.denyCode)}
		proxywasm.ReplaceHttpResponseHeaders(reconvertHeaders(hdsMap))
		proxywasm.ReplaceHttpResponseBody(jsonData)
		config.incrementCounter("ai_sec_response_deny", 1)
		proxywasm.ResumeHttpResponse()
	}
	singleCall = func() {
		timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		randomID, _ := generateHexID(16)
		var nextContentIndex int
		if contentIndex+LengthLimit >= len(content) {
			nextContentIndex = len(content)
		} else {
			nextContentIndex = contentIndex + LengthLimit
		}
		contentPiece := content[contentIndex:nextContentIndex]
		contentIndex = nextContentIndex
		log.Debugf("current content piece: %s", contentPiece)
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
			"ServiceParameters": fmt.Sprintf(`{"sessionId": "%s","content": "%s"}`, sessionID, marshalStr(contentPiece, log)),
		}
		if config.token != "" {
			params["SecurityToken"] = config.token
		}
		signature := getSign(params, config.sk+"&")
		reqParams := url.Values{}
		for k, v := range params {
			reqParams.Add(k, v)
		}
		reqParams.Add("Signature", signature)
		err := config.client.Post(fmt.Sprintf("/?%s", reqParams.Encode()), [][2]string{{"User-Agent", AliyunUserAgent}}, nil, callback)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			proxywasm.ResumeHttpResponse()
		}
	}
	singleCall()
	return types.ActionPause
}

func extractMessageFromStreamingBody(data []byte, jsonPath string) string {
	chunks := bytes.Split(bytes.TrimSpace(data), []byte("\n\n"))
	strChunks := []string{}
	for _, chunk := range chunks {
		// Example: "choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":null}]
		strChunks = append(strChunks, gjson.GetBytes(chunk, jsonPath).String())
	}
	return strings.Join(strChunks, "")
}

func marshalStr(raw string, log wrapper.Log) string {
	helper := map[string]string{
		"placeholder": raw,
	}
	marshalledHelper, _ := json.Marshal(helper)
	marshalledRaw := gjson.GetBytes(marshalledHelper, "placeholder").Raw
	if len(marshalledRaw) >= 2 {
		return marshalledRaw[1 : len(marshalledRaw)-1]
	} else {
		log.Errorf("failed to marshal json string, raw string is: %s", raw)
		return ""
	}
}
