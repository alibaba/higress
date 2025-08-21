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

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"ai-security-guard",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBody(onHttpStreamingResponseBody),
		wrapper.ProcessResponseBody(onHttpResponseBody),
	)
}

const (
	MaxRisk    = "max"
	HighRisk   = "high"
	MediumRisk = "medium"
	LowRisk    = "low"
	NoRisk     = "none"

	S4Sensitive = "S4"
	S3Sensitive = "S3"
	S2Sensitive = "S2"
	S1Sensitive = "S1"
	NoSensitive = "S0"

	ContentModerationType = "contentModeration"
	PromptAttackType      = "promptAttack"
	SensitiveDataType     = "sensitiveData"

	OpenAIResponseFormat       = `{"id": "%s","object":"chat.completion","model":"from-security-guard","choices":[{"index":0,"message":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":"stop"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	OpenAIStreamResponseChunk  = `data:{"id":"%s","object":"chat.completion.chunk","model":"from-security-guard","choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":null}]}`
	OpenAIStreamResponseEnd    = `data:{"id":"%s","object":"chat.completion.chunk","model":"from-security-guard","choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"stop"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`
	OpenAIStreamResponseFormat = OpenAIStreamResponseChunk + "\n\n" + OpenAIStreamResponseEnd + "\n\n" + `data: [DONE]`

	DefaultRequestCheckService       = "llm_query_moderation"
	DefaultResponseCheckService      = "llm_response_moderation"
	DefaultRequestJsonPath           = "messages.@reverse.0.content"
	DefaultResponseJsonPath          = "choices.0.message.content"
	DefaultStreamingResponseJsonPath = "choices.0.delta.content"
	DefaultDenyCode                  = 200
	DefaultDenyMessage               = "很抱歉，我无法回答您的问题"
	DefaultTimeout                   = 2000

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
	Detail    []Detail `json:"Detail,omitempty"`
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

type Detail struct {
	Suggestion string `json:"Suggestion,omitempty"`
	Type       string `json:"Type,omitempty"`
	Level      string `json:"Level,omitempty"`
}

type AISecurityConfig struct {
	client                        wrapper.HttpClient
	ak                            string
	sk                            string
	token                         string
	action                        string
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
	contentModerationLevelBar     string
	promptAttackLevelBar          string
	sensitiveDataLevelBar         string
	timeout                       uint32
	bufferLimit                   int
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

func levelToInt(riskLevel string) int {
	switch riskLevel {
	case MaxRisk, S4Sensitive:
		return 4
	case HighRisk, S3Sensitive:
		return 3
	case MediumRisk, S2Sensitive:
		return 2
	case LowRisk, S1Sensitive:
		return 1
	case NoRisk, NoSensitive:
		return 0
	default:
		return -1
	}
}

func isRiskLevelAcceptable(action string, data Data, config AISecurityConfig) bool {
	if action == "MultiModalGuard" {
		for _, detail := range data.Detail {
			switch detail.Type {
			case ContentModerationType:
				if levelToInt(detail.Level) >= levelToInt(config.contentModerationLevelBar) {
					return false
				}
			case PromptAttackType:
				if levelToInt(detail.Level) >= levelToInt(config.promptAttackLevelBar) {
					return false
				}
			case SensitiveDataType:
				if levelToInt(detail.Level) >= levelToInt(config.sensitiveDataLevelBar) {
					return false
				}
			}
		}
		return true
	} else {
		return levelToInt(data.RiskLevel) < levelToInt(config.contentModerationLevelBar)
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

func parseConfig(json gjson.Result, config *AISecurityConfig) error {
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
	config.action = json.Get("action").String()
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
	if obj := json.Get("contentModerationLevelBar"); obj.Exists() {
		config.contentModerationLevelBar = obj.String()
		if levelToInt(config.contentModerationLevelBar) <= 0 {
			return errors.New("invalid contentModerationLevelBar, value must be one of [max, high, medium, low]")
		}
	} else {
		config.contentModerationLevelBar = MaxRisk
	}
	if obj := json.Get("promptAttackLevelBar"); obj.Exists() {
		config.promptAttackLevelBar = obj.String()
		if levelToInt(config.promptAttackLevelBar) <= 0 {
			return errors.New("invalid promptAttackLevelBar, value must be one of [max, high, medium, low]")
		}
	} else {
		config.promptAttackLevelBar = MaxRisk
	}
	if obj := json.Get("sensitiveDataLevelBar"); obj.Exists() {
		config.sensitiveDataLevelBar = obj.String()
		if levelToInt(config.sensitiveDataLevelBar) <= 0 {
			return errors.New("invalid sensitiveDataLevelBar, value must be one of [S4, S3, S2, S1]")
		}
	} else {
		config.sensitiveDataLevelBar = S4Sensitive
	}
	if obj := json.Get("timeout"); obj.Exists() {
		config.timeout = uint32(obj.Int())
	} else {
		config.timeout = DefaultTimeout
	}
	if obj := json.Get("bufferLimit"); obj.Exists() {
		config.bufferLimit = int(obj.Int())
	} else {
		config.bufferLimit = 1000
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

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AISecurityConfig) types.Action {
	ctx.DisableReroute()
	if !config.checkRequest {
		log.Debugf("request checking is disabled")
		ctx.DontReadRequestBody()
	}
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AISecurityConfig, body []byte) types.Action {
	log.Debugf("checking request body...")
	startTime := time.Now().UnixMilli()
	content := gjson.GetBytes(body, config.requestContentJsonPath).String()
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
		if isRiskLevelAcceptable(config.action, response.Data, config) {
			if contentIndex >= len(content) {
				endTime := time.Now().UnixMilli()
				ctx.SetUserAttribute("safecheck_request_rt", endTime-startTime)
				ctx.SetUserAttribute("safecheck_status", "request pass")
				ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
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
		marshalledDenyMessage := marshalStr(denyMessage)
		if config.protocolOriginal {
			proxywasm.SendHttpResponse(uint32(config.denyCode), [][2]string{{"content-type", "application/json"}}, []byte(marshalledDenyMessage), -1)
		} else if gjson.GetBytes(body, "stream").Bool() {
			randomID := generateRandomID()
			jsonData := []byte(fmt.Sprintf(OpenAIStreamResponseFormat, randomID, marshalledDenyMessage, randomID))
			proxywasm.SendHttpResponse(uint32(config.denyCode), [][2]string{{"content-type", "text/event-stream;charset=UTF-8"}}, jsonData, -1)
		} else {
			randomID := generateRandomID()
			jsonData := []byte(fmt.Sprintf(OpenAIResponseFormat, randomID, marshalledDenyMessage))
			proxywasm.SendHttpResponse(uint32(config.denyCode), [][2]string{{"content-type", "application/json"}}, jsonData, -1)
		}
		ctx.DontReadResponseBody()
		config.incrementCounter("ai_sec_request_deny", 1)
		endTime := time.Now().UnixMilli()
		ctx.SetUserAttribute("safecheck_request_rt", endTime-startTime)
		ctx.SetUserAttribute("safecheck_status", "reqeust deny")
		if response.Data.Advice != nil {
			ctx.SetUserAttribute("safecheck_riskLabel", response.Data.Result[0].Label)
			ctx.SetUserAttribute("safecheck_riskWords", response.Data.Result[0].RiskWords)
		}
		ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
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
			"Action":            config.action,
			"AccessKeyId":       config.ak,
			"Timestamp":         timestamp,
			"Service":           config.requestCheckService,
			"ServiceParameters": fmt.Sprintf(`{"sessionId": "%s","content": "%s","requestFrom": "%s"}`, sessionID, marshalStr(contentPiece), AliyunUserAgent),
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
		err := config.client.Post(fmt.Sprintf("/?%s", reqParams.Encode()), [][2]string{{"User-Agent", AliyunUserAgent}}, nil, callback, config.timeout)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			proxywasm.ResumeHttpRequest()
		}
	}
	singleCall()
	return types.ActionPause
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config AISecurityConfig) types.Action {
	if !config.checkResponse {
		log.Debugf("response checking is disabled")
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}
	statusCode, _ := proxywasm.GetHttpResponseHeader(":status")
	if statusCode != "200" {
		log.Debugf("response is not 200, skip response body check")
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	ctx.SetContext("end_of_stream_received", false)
	ctx.SetContext("during_call", false)
	ctx.SetContext("risk_detected", false)
	sessionID, _ := generateHexID(20)
	ctx.SetContext("sessionID", sessionID)
	if strings.Contains(contentType, "text/event-stream") {
		ctx.NeedPauseStreamingResponse()
		return types.ActionContinue
	} else {
		ctx.BufferResponseBody()
		return types.HeaderStopIteration
	}
}

func onHttpStreamingResponseBody(ctx wrapper.HttpContext, config AISecurityConfig, data []byte, endOfStream bool) []byte {
	var bufferQueue [][]byte
	var singleCall func()
	callback := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		log.Info(string(responseBody))
		if statusCode != 200 || gjson.GetBytes(responseBody, "Code").Int() != 200 {
			if ctx.GetContext("end_of_stream_received").(bool) {
				proxywasm.ResumeHttpResponse()
			}
			ctx.SetContext("during_call", false)
			return
		}
		var response Response
		err := json.Unmarshal(responseBody, &response)
		if err != nil {
			log.Error("failed to unmarshal aliyun content security response at response phase")
			if ctx.GetContext("end_of_stream_received").(bool) {
				proxywasm.ResumeHttpResponse()
			}
			ctx.SetContext("during_call", false)
			return
		}
		if !isRiskLevelAcceptable(config.action, response.Data, config) {
			denyMessage := DefaultDenyMessage
			if response.Data.Advice != nil && response.Data.Advice[0].Answer != "" {
				denyMessage = "\n" + response.Data.Advice[0].Answer
			} else if config.denyMessage != "" {
				denyMessage = config.denyMessage
			}
			marshalledDenyMessage := marshalStr(denyMessage)
			randomID := generateRandomID()
			jsonData := []byte(fmt.Sprintf(OpenAIStreamResponseFormat, randomID, marshalledDenyMessage, randomID))
			proxywasm.InjectEncodedDataToFilterChain(jsonData, true)
			return
		}
		endStream := ctx.GetContext("end_of_stream_received").(bool) && ctx.BufferQueueSize() == 0
		proxywasm.InjectEncodedDataToFilterChain(bytes.Join(bufferQueue, []byte("")), endStream)
		bufferQueue = [][]byte{}
		if !endStream {
			ctx.SetContext("during_call", false)
			singleCall()
		}
	}
	singleCall = func() {
		if ctx.GetContext("during_call").(bool) {
			return
		}
		if ctx.BufferQueueSize() >= config.bufferLimit || ctx.GetContext("end_of_stream_received").(bool) {
			ctx.SetContext("during_call", true)
			var buffer string
			for ctx.BufferQueueSize() > 0 {
				front := ctx.PopBuffer()
				bufferQueue = append(bufferQueue, front)
				msg := gjson.GetBytes(front, config.responseStreamContentJsonPath).String()
				buffer += msg
				if len([]rune(buffer)) >= config.bufferLimit {
					break
				}
			}
			timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
			randomID, _ := generateHexID(16)
			log.Debugf("current content piece: %s", buffer)
			params := map[string]string{
				"Format":            "JSON",
				"Version":           "2022-03-02",
				"SignatureMethod":   "Hmac-SHA1",
				"SignatureNonce":    randomID,
				"SignatureVersion":  "1.0",
				"Action":            config.action,
				"AccessKeyId":       config.ak,
				"Timestamp":         timestamp,
				"Service":           config.responseCheckService,
				"ServiceParameters": fmt.Sprintf(`{"sessionId": "%s","content": "%s","requestFrom": "%s"}`, ctx.GetContext("sessionID").(string), marshalStr(buffer), AliyunUserAgent),
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
			err := config.client.Post(fmt.Sprintf("/?%s", reqParams.Encode()), [][2]string{{"User-Agent", AliyunUserAgent}}, nil, callback, config.timeout)
			if err != nil {
				log.Errorf("failed call the safe check service: %v", err)
				if ctx.GetContext("end_of_stream_received").(bool) {
					proxywasm.ResumeHttpResponse()
				}
			}
		}
	}
	if !ctx.GetContext("risk_detected").(bool) {
		ctx.PushBuffer(data)
		ctx.SetContext("end_of_stream_received", endOfStream)
		if !ctx.GetContext("during_call").(bool) {
			singleCall()
		}
	} else if endOfStream {
		proxywasm.ResumeHttpResponse()
	}
	return []byte{}
}

func onHttpResponseBody(ctx wrapper.HttpContext, config AISecurityConfig, body []byte) types.Action {
	log.Debugf("checking response body...")
	startTime := time.Now().UnixMilli()
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	isStreamingResponse := strings.Contains(contentType, "event-stream")
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
		if isRiskLevelAcceptable(config.action, response.Data, config) {
			if contentIndex >= len(content) {
				endTime := time.Now().UnixMilli()
				ctx.SetUserAttribute("safecheck_response_rt", endTime-startTime)
				ctx.SetUserAttribute("safecheck_status", "response pass")
				ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
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
		marshalledDenyMessage := marshalStr(denyMessage)
		if config.protocolOriginal {
			proxywasm.SendHttpResponse(uint32(config.denyCode), [][2]string{{"content-type", "application/json"}}, []byte(marshalledDenyMessage), -1)
		} else if isStreamingResponse {
			randomID := generateRandomID()
			jsonData := []byte(fmt.Sprintf(OpenAIStreamResponseFormat, randomID, marshalledDenyMessage, randomID))
			proxywasm.SendHttpResponse(uint32(config.denyCode), [][2]string{{"content-type", "text/event-stream;charset=UTF-8"}}, jsonData, -1)
		} else {
			randomID := generateRandomID()
			jsonData := []byte(fmt.Sprintf(OpenAIResponseFormat, randomID, marshalledDenyMessage))
			proxywasm.SendHttpResponse(uint32(config.denyCode), [][2]string{{"content-type", "application/json"}}, jsonData, -1)
		}
		config.incrementCounter("ai_sec_response_deny", 1)
		endTime := time.Now().UnixMilli()
		ctx.SetUserAttribute("safecheck_response_rt", endTime-startTime)
		ctx.SetUserAttribute("safecheck_status", "response deny")
		if response.Data.Advice != nil {
			ctx.SetUserAttribute("safecheck_riskLabel", response.Data.Result[0].Label)
			ctx.SetUserAttribute("safecheck_riskWords", response.Data.Result[0].RiskWords)
		}
		ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
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
			"Action":            config.action,
			"AccessKeyId":       config.ak,
			"Timestamp":         timestamp,
			"Service":           config.responseCheckService,
			"ServiceParameters": fmt.Sprintf(`{"sessionId": "%s","content": "%s","requestFrom": "%s"}`, sessionID, marshalStr(contentPiece), AliyunUserAgent),
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
		err := config.client.Post(fmt.Sprintf("/?%s", reqParams.Encode()), [][2]string{{"User-Agent", AliyunUserAgent}}, nil, callback, config.timeout)
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

func marshalStr(raw string) string {
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
