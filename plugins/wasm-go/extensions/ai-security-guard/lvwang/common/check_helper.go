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

// SendDenyResponse constructs and sends a deny HTTP response,
// sets deny metrics and log attributes.
func SendDenyResponse(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte, response cfg.Response, startTime int64) {
	consumer, _ := ctx.GetContext("consumer").(string)
	denyBody, err := cfg.BuildDenyResponseBody(response, config, consumer)
	if err != nil {
		log.Errorf("failed to build deny response body: %v", err)
		proxywasm.ResumeHttpRequest()
		return
	}
	if config.ProtocolOriginal {
		proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "application/json"}}, denyBody, -1)
	} else if gjson.GetBytes(body, "stream").Bool() {
		randomID := utils.GenerateRandomChatID()
		marshalledDenyMessage := wrapper.MarshalStr(string(denyBody))
		jsonData := []byte(fmt.Sprintf(cfg.OpenAIStreamResponseFormat, randomID, marshalledDenyMessage, randomID))
		proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "text/event-stream;charset=UTF-8"}}, jsonData, -1)
	} else {
		randomID := utils.GenerateRandomChatID()
		marshalledDenyMessage := wrapper.MarshalStr(string(denyBody))
		jsonData := []byte(fmt.Sprintf(cfg.OpenAIResponseFormat, randomID, marshalledDenyMessage))
		proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "application/json"}}, jsonData, -1)
	}
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
