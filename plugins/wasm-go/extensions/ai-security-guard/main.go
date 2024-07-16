package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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

type AISecurityConfig struct {
	client wrapper.HttpClient
	ak     string
	sk     string
}

type StandardResponse struct {
	Code    int    `json:"Code"`
	Phase   string `json:"BlockPhase"`
	Message string `json:"Message"`
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
	fmt.Println(signStr)
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
	config.ak = json.Get("ak").String()
	config.sk = json.Get("sk").String()
	if serviceName == "" || servicePort == 0 || domain == "" {
		return errors.New("invalid service config")
	}
	config.client = wrapper.NewClusterClient(wrapper.DnsCluster{
		ServiceName: serviceName,
		Port:        servicePort,
		Domain:      domain,
	})
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AISecurityConfig, log wrapper.Log) types.Action {
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config AISecurityConfig, body []byte, log wrapper.Log) types.Action {
	messages := gjson.GetBytes(body, "messages").Array()
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
						sr := StandardResponse{
							Code:    403,
							Phase:   "Request",
							Message: respAdvice.Array()[0].Get("Answer").String(),
						}
						jsonData, _ := json.MarshalIndent(sr, "", "    ")
						label := respResult.Array()[0].Get("Label").String()
						proxywasm.SetProperty([]string{"risklabel"}, []byte(label))
						proxywasm.SendHttpResponseWithDetail(403, "ai-security-guard.label."+label, [][2]string{{"content-type", "application/json"}}, jsonData, -1)
					} else if respResult.Array()[0].Get("Label").String() != "nonLabel" {
						sr := StandardResponse{
							Code:    403,
							Phase:   "Request",
							Message: "risk detected",
						}
						jsonData, _ := json.MarshalIndent(sr, "", "    ")
						proxywasm.SetProperty([]string{"risklabel"}, []byte(respResult.Array()[0].Get("Label").String()))
						proxywasm.SendHttpResponseWithDetail(403, "ai-security-guard.risk_detected", [][2]string{{"content-type", "application/json"}}, jsonData, -1)
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
	messages := gjson.GetBytes(body, "choices").Array()
	if len(messages) > 0 {
		content := messages[0].Get("message").Get("content").String()
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
						sr := StandardResponse{
							Code:    403,
							Phase:   "Response",
							Message: respAdvice.Array()[0].Get("Answer").String(),
						}
						jsonData, _ := json.MarshalIndent(sr, "", "    ")
						hdsMap := ctx.GetContext("headers").(map[string][]string)
						delete(hdsMap, "content-length")
						hdsMap[":status"] = []string{"403"}
						proxywasm.ReplaceHttpResponseHeaders(reconvertHeaders(hdsMap))
						proxywasm.ReplaceHttpResponseBody(jsonData)
						proxywasm.SetProperty([]string{"risklabel"}, []byte(respResult.Array()[0].Get("Label").String()))
					} else if respResult.Array()[0].Get("Label").String() != "nonLabel" {
						sr := StandardResponse{
							Code:    403,
							Phase:   "Response",
							Message: "risk detected",
						}
						jsonData, _ := json.MarshalIndent(sr, "", "    ")
						hdsMap := ctx.GetContext("headers").(map[string][]string)
						delete(hdsMap, "content-length")
						hdsMap[":status"] = []string{"403"}
						proxywasm.ReplaceHttpResponseHeaders(reconvertHeaders(hdsMap))
						proxywasm.ReplaceHttpResponseBody(jsonData)
						proxywasm.SetProperty([]string{"risklabel"}, []byte(respResult.Array()[0].Get("Label").String()))
					}
				}
			},
		)
		return types.ActionPause
	} else {
		return types.ActionContinue
	}
}
