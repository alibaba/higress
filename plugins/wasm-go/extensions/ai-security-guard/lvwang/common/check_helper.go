package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/utils"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

// SelectDenyMessage chooses the deny message based on config and response.
// Priority: config.DenyMessage > response advice answer > default.
func SelectDenyMessage(configDenyMsg string, response cfg.Response) string {
	if configDenyMsg != "" {
		return configDenyMsg
	}
	if len(response.Data.Advice) > 0 && response.Data.Advice[0].Answer != "" {
		return response.Data.Advice[0].Answer
	}
	return cfg.DefaultDenyMessage
}

// DenyResponseResult holds the content type and body for a deny response.
type DenyResponseResult struct {
	ContentType string
	Body        []byte
}

// BuildDenyResponseBody constructs the deny response body based on protocol settings.
func BuildDenyResponseBody(protocolOriginal bool, isStream bool, denyCode int64, denyMessage string) DenyResponseResult {
	marshalledDenyMessage := wrapper.MarshalStr(denyMessage)
	if protocolOriginal {
		return DenyResponseResult{
			ContentType: "application/json",
			Body:        []byte(marshalledDenyMessage),
		}
	}
	if isStream {
		randomID := utils.GenerateRandomChatID()
		jsonData := []byte(fmt.Sprintf(cfg.OpenAIStreamResponseFormat, randomID, marshalledDenyMessage, randomID))
		return DenyResponseResult{
			ContentType: "text/event-stream;charset=UTF-8",
			Body:        jsonData,
		}
	}
	randomID := utils.GenerateRandomChatID()
	jsonData := []byte(fmt.Sprintf(cfg.OpenAIResponseFormat, randomID, marshalledDenyMessage))
	return DenyResponseResult{
		ContentType: "application/json",
		Body:        jsonData,
	}
}

// SendDenyResponse constructs and sends a deny HTTP response,
// sets deny metrics and log attributes.
func SendDenyResponse(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte, response cfg.Response, startTime int64) {
	denyMessage := SelectDenyMessage(config.DenyMessage, response)
	isStream := gjson.GetBytes(body, "stream").Bool()
	result := BuildDenyResponseBody(config.ProtocolOriginal, isStream, config.DenyCode, denyMessage)
	proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", result.ContentType}}, result.Body, -1)
	ctx.DontReadResponseBody()
	config.IncrementCounter("ai_sec_request_deny", 1)
	endTime := time.Now().UnixMilli()
	ctx.SetUserAttribute("safecheck_request_rt", endTime-startTime)
	ctx.SetUserAttribute("safecheck_status", "reqeust deny")
	if len(response.Data.Result) > 0 {
		ctx.SetUserAttribute("safecheck_riskLabel", response.Data.Result[0].Label)
		ctx.SetUserAttribute("safecheck_riskWords", response.Data.Result[0].RiskWords)
	}
	ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
}

// SetRequestPassAttributes sets log attributes for a passed request security check.
func SetRequestPassAttributes(ctx wrapper.HttpContext, startTime int64) {
	endTime := time.Now().UnixMilli()
	ctx.SetUserAttribute("safecheck_request_rt", endTime-startTime)
	ctx.SetUserAttribute("safecheck_status", "request pass")
	ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
}

// TextCheckFunc generates an HTTP request for checking a content piece.
type TextCheckFunc func(contentPiece string, sessionID string) (path string, headers [][2]string, body []byte)

// RunChunkedTextCheck splits content into pieces of cfg.LengthLimit and checks each
// via textCheckFn. Calls onAllPassed when all pieces pass; sends deny response on risk.
func RunChunkedTextCheck(
	ctx wrapper.HttpContext,
	config cfg.AISecurityConfig,
	body []byte,
	content string,
	startTime int64,
	consumer string,
	textCheckFn TextCheckFunc,
	onAllPassed func(),
) {
	contentIndex := 0
	sessionID, _ := utils.GenerateHexID(20)
	var singleCall func()
	callback := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		log.Info(string(responseBody))
		if statusCode != 200 || gjson.GetBytes(responseBody, "Code").Int() != 200 {
			proxywasm.ResumeHttpRequest()
			return
		}
		var response cfg.Response
		err := json.Unmarshal(responseBody, &response)
		if err != nil {
			log.Errorf("failed to unmarshal content security response: %v", err)
			proxywasm.ResumeHttpRequest()
			return
		}
		if cfg.IsRiskLevelAcceptable(config.Action, response.Data, config, consumer) {
			if contentIndex >= len(content) {
				SetRequestPassAttributes(ctx, startTime)
				onAllPassed()
			} else {
				singleCall()
			}
			return
		}
		SendDenyResponse(ctx, config, body, response, startTime)
	}
	singleCall = func() {
		nextContentIndex := contentIndex + cfg.LengthLimit
		if nextContentIndex > len(content) {
			nextContentIndex = len(content)
		}
		contentPiece := content[contentIndex:nextContentIndex]
		contentIndex = nextContentIndex
		log.Debugf("current content piece: %s", contentPiece)
		path, headers, reqBody := textCheckFn(contentPiece, sessionID)
		err := config.Client.Post(path, headers, reqBody, callback, config.Timeout)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			proxywasm.ResumeHttpRequest()
		}
	}
	singleCall()
}
