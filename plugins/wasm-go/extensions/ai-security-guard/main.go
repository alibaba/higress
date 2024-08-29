package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
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
	NormalResponseFormat = `{"id": "chatcmpl-123","object": "chat.completion","created": 1677652288,"model": "gpt-4o-mini","system_fingerprint": "fp_44709d6fcb","choices": [{"index": 0,"message": {"role": "assistant","content": "%s",},"logprobs": null,"finish_reason": "stop"}]}`
	StreamResponseChunk  = `data:{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4o-mini", "system_fingerprint": "fp_44709d6fcb", "choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":null}]}`
	StreamResponseEnd    = `data:{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4o-mini", "system_fingerprint": "fp_44709d6fcb", "choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"stop"}]}`
	StreamResponseFormat = StreamResponseChunk + "\n\n" + StreamResponseEnd
	TracingPrefix        = "trace_span_tag."
)

type AISecurityConfig struct {
	client        wrapper.HttpClient
	ak            string
	sk            string
	checkRequest  bool
	checkResponse bool
}

type StandardResponse struct {
	ID                string              `json:"id"`
	Choices           []Choice            `json:"choices"`
	Created           int64               `json:"created,omitempty"`
	Model             string              `json:"model,omitempty"`
	SystemFingerprint string              `json:"system_fingerprint,omitempty"`
	Object            string              `json:"object,omitempty"`
	Usage             chatCompletionUsage `json:"usage,omitempty"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

func urlEncoding(rawStr string) string {
	encodedStr := url.PathEscape(rawStr)
	encodedStr = strings.ReplaceAll(encodedStr, "+", "%20")
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
	// proxywasm.LogInfo(signStr)
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
	domain := json.Get("domain").String()
	if serviceName == "" || servicePort == 0 || domain == "" {
		return errors.New("invalid service config")
	}
	config.ak = json.Get("ak").String()
	config.sk = json.Get("sk").String()
	if config.ak == "" || config.sk == "" {
		return errors.New("invalid AK/SK config")
	}
	config.checkRequest = json.Get("checkRequest").Bool()
	config.checkResponse = json.Get("checkResponse").Bool()
	config.client = wrapper.NewClusterClient(wrapper.DnsCluster{
		ServiceName: serviceName,
		Port:        servicePort,
		Domain:      domain,
	})
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AISecurityConfig, log wrapper.Log) types.Action {
	if !config.checkRequest {
		ctx.DontReadRequestBody()
	}
	if !config.checkResponse {
		ctx.DontReadResponseBody()
	}
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AISecurityConfig, body []byte, log wrapper.Log) types.Action {
	messages := gjson.GetBytes(body, "messages").Array()
	stream := gjson.GetBytes(body, "stream").Bool()
	if len(messages) > 0 {
		role := messages[len(messages)-1].Get("role").String()
		content := messages[len(messages)-1].Get("content").String()
		if role != "user" {
			return types.ActionContinue
		}
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
			"Service":           "llm_query_moderation",
			"ServiceParameters": `{"content": "` + content + `"}`,
		}
		signature := getSign(params, config.sk+"&")
		reqParams := url.Values{}
		for k, v := range params {
			reqParams.Add(k, v)
		}
		reqParams.Add("Signature", signature)
		config.client.Post(fmt.Sprintf("/?%s", reqParams.Encode()), nil, nil,
			func(statusCode int, responseHeaders http.Header, responseBody []byte) {
				respData := gjson.GetBytes(responseBody, "Data")
				if respData.Exists() {
					respAdvice := respData.Get("Advice")
					respResult := respData.Get("Result")
					if respAdvice.Exists() {
						proxywasm.SetProperty([]string{TracingPrefix, "ai_sec_risklabel"}, []byte(respResult.Array()[0].Get("Label").String()))
						proxywasm.SetProperty([]string{TracingPrefix, "ai_sec_deny_phase"}, []byte("request"))
						if stream {
							jsonData := []byte(fmt.Sprintf(StreamResponseFormat, respAdvice.Array()[0].Get("Answer").String()))
							proxywasm.SendHttpResponse(200, [][2]string{{"content-type", "text/event-stream;charset=UTF-8"}}, jsonData, -1)
						} else {
							jsonData := []byte(fmt.Sprintf(StreamResponseFormat, respAdvice.Array()[0].Get("Answer").String()))
							proxywasm.SendHttpResponse(200, [][2]string{{"content-type", "application/json"}}, jsonData, -1)
						}
					} else if respResult.Array()[0].Get("Label").String() != "nonLabel" {
						proxywasm.SetProperty([]string{TracingPrefix, "ai_sec_risklabel"}, []byte(respResult.Array()[0].Get("Label").String()))
						proxywasm.SetProperty([]string{TracingPrefix, "ai_sec_deny_phase"}, []byte("request"))
						if stream {
							jsonData := []byte(fmt.Sprintf(StreamResponseFormat, "很抱歉，我不能对您的问题做出回答。"))
							proxywasm.SendHttpResponse(200, [][2]string{{"content-type", "text/event-stream;charset=UTF-8"}}, jsonData, -1)
						} else {
							jsonData := []byte(fmt.Sprintf(NormalResponseFormat, "很抱歉，我不能对您的问题做出回答。"))
							proxywasm.SendHttpResponse(200, [][2]string{{"content-type", "application/json"}}, jsonData, -1)
						}
					} else {
						proxywasm.ResumeHttpRequest()
					}
				} else {
					proxywasm.ResumeHttpRequest()
				}
			},
		)
		return types.ActionPause
	} else {
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
	content := extractResponseMessage(body)
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
			"Service":           "llm_response_moderation",
			"ServiceParameters": `{"content": "` + content + `"}`,
		}
		signature := getSign(params, config.sk+"&")
		reqParams := url.Values{}
		for k, v := range params {
			reqParams.Add(k, v)
		}
		reqParams.Add("Signature", signature)
		config.client.Post(fmt.Sprintf("/?%s", reqParams.Encode()), nil, nil,
			func(statusCode int, responseHeaders http.Header, responseBody []byte) {
				defer proxywasm.ResumeHttpResponse()
				respData := gjson.GetBytes(responseBody, "Data")
				if respData.Exists() {
					respAdvice := respData.Get("Advice")
					respResult := respData.Get("Result")
					if respAdvice.Exists() {
						hdsMap := ctx.GetContext("headers").(map[string][]string)
						var jsonData []byte
						if strings.Contains(strings.Join(hdsMap["content-type"], ";"), "event-stream") {
							jsonData = []byte(fmt.Sprintf(StreamResponseFormat, respAdvice.Array()[0].Get("Answer").String()))
						} else {
							jsonData = []byte(fmt.Sprintf(NormalResponseFormat, respAdvice.Array()[0].Get("Answer").String()))
						}
						delete(hdsMap, "content-length")
						hdsMap[":status"] = []string{"200"}
						proxywasm.ReplaceHttpResponseHeaders(reconvertHeaders(hdsMap))
						proxywasm.ReplaceHttpResponseBody(jsonData)
						proxywasm.SetProperty([]string{TracingPrefix, "ai_sec_risklabel"}, []byte(respResult.Array()[0].Get("Label").String()))
						proxywasm.SetProperty([]string{TracingPrefix, "ai_sec_deny_phase"}, []byte("response"))
					} else if respResult.Array()[0].Get("Label").String() != "nonLabel" {
						hdsMap := ctx.GetContext("headers").(map[string][]string)
						var jsonData []byte
						if strings.Contains(strings.Join(hdsMap["content-type"], ";"), "event-stream") {
							jsonData = []byte(fmt.Sprintf(StreamResponseFormat, "很抱歉，我不能对您的问题做出回答。"))
						} else {
							jsonData = []byte(fmt.Sprintf(NormalResponseFormat, "很抱歉，我不能对您的问题做出回答。"))
						}
						delete(hdsMap, "content-length")
						hdsMap[":status"] = []string{"200"}
						proxywasm.ReplaceHttpResponseHeaders(reconvertHeaders(hdsMap))
						proxywasm.ReplaceHttpResponseBody(jsonData)
						proxywasm.SetProperty([]string{TracingPrefix, "ai_sec_risklabel"}, []byte(respResult.Array()[0].Get("Label").String()))
						proxywasm.SetProperty([]string{TracingPrefix, "ai_sec_deny_phase"}, []byte("response"))
					}
				}
			},
		)
		return types.ActionPause
	} else {
		return types.ActionContinue
	}
}

func extractResponseMessage(data []byte) string {
	chunks := bytes.Split(bytes.TrimSpace(data), []byte("\n\n"))
	strChunks := []string{}
	for _, chunk := range chunks {
		// Example: "choices":[{"index":0,"delta":{"role":"assistant","content":"%s"},"logprobs":null,"finish_reason":null}]
		jsonObj := gjson.GetBytes(chunk, "choices.0.delta.content")
		if jsonObj.Exists() {
			strChunks = append(strChunks, jsonObj.String())
		}
	}
	return strings.Join(strChunks, "")
}
