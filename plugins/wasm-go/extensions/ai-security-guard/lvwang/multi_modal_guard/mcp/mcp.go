package mcp

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/lvwang/common"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/utils"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const DenyResponse = `{"jsonrpc":"2.0","id":0,"error":{"code":403,"message":"Blocked by security guard"}}`
const DenySSEResponse = `event: message
data: {"jsonrpc":"2.0","id":0,"error":{"code":403,"message":"Blocked by security guard"}}

`

func HandleMcpStreamingResponseBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, data []byte, endOfStream bool) []byte {
	consumer, _ := ctx.GetContext("consumer").(string)
	var frontBuffer []byte
	var singleCall func()
	callback := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		defer func() {
			ctx.SetContext("during_call", false)
			singleCall()
		}()
		log.Info(string(responseBody))
		if statusCode != 200 || gjson.GetBytes(responseBody, "Code").Int() != 200 {
			proxywasm.InjectEncodedDataToFilterChain(frontBuffer, false)
			return
		}
		var response cfg.Response
		err := json.Unmarshal(responseBody, &response)
		if err != nil {
			log.Error("failed to unmarshal aliyun content security response at response phase")
			proxywasm.InjectEncodedDataToFilterChain(frontBuffer, false)
			return
		}
		if !cfg.IsRiskLevelAcceptable(config.Action, response.Data, config, consumer) {
			proxywasm.InjectEncodedDataToFilterChain([]byte(DenySSEResponse), true)
		} else {
			proxywasm.InjectEncodedDataToFilterChain(frontBuffer, false)
		}
	}
	singleCall = func() {
		if ctx.GetContext("during_call").(bool) {
			return
		}
		if ctx.BufferQueueSize() > 0 {
			frontBuffer = ctx.PopBuffer()
			index := strings.Index(string(frontBuffer), "data:")
			msg := gjson.GetBytes(frontBuffer[index:], config.ResponseStreamContentJsonPath).String()
			log.Debugf("current content piece: %s", msg)
			ctx.SetContext("during_call", true)
			checkService := config.GetResponseCheckService(consumer)
			sessionID, _ := utils.GenerateHexID(20)
			path, headers, body := common.GenerateRequestForText(config, config.Action, checkService, msg, sessionID)
			err := config.Client.Post(path, headers, body, callback, config.Timeout)
			if err != nil {
				log.Errorf("failed call the safe check service: %v", err)
				proxywasm.InjectEncodedDataToFilterChain(frontBuffer, false)
				ctx.SetContext("during_call", false)
			}
		}
	}
	index := strings.Index(string(data), "data:")
	if index != -1 {
		event := data[index:]
		if gjson.GetBytes(event, config.ResponseStreamContentJsonPath).Exists() {
			ctx.PushBuffer(data)
			if !ctx.GetContext("during_call").(bool) {
				singleCall()
			}
			return []byte{}
		}
	}
	proxywasm.InjectEncodedDataToFilterChain(data, false)
	return []byte{}
}

func HandleMcpResponseBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	consumer, _ := ctx.GetContext("consumer").(string)
	log.Debugf("checking response body...")
	startTime := time.Now().UnixMilli()
	content := gjson.GetBytes(body, config.ResponseContentJsonPath).String()
	log.Debugf("Raw response content is: %s", content)
	if len(content) == 0 {
		log.Info("response content is empty. skip")
		return types.ActionContinue
	}
	contentIndex := 0
	sessionID, _ := utils.GenerateHexID(20)
	var singleCall func()
	callback := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		log.Info(string(responseBody))
		if statusCode != 200 || gjson.GetBytes(responseBody, "Code").Int() != 200 {
			proxywasm.ResumeHttpResponse()
			return
		}
		var response cfg.Response
		err := json.Unmarshal(responseBody, &response)
		if err != nil {
			log.Error("failed to unmarshal aliyun content security response at response phase")
			proxywasm.ResumeHttpResponse()
			return
		}
		if cfg.IsRiskLevelAcceptable(config.Action, response.Data, config, consumer) {
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
		proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "application/json"}}, []byte(DenyResponse), -1)
		config.IncrementCounter("ai_sec_response_deny", 1)
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
		var nextContentIndex int
		if contentIndex+cfg.LengthLimit >= len(content) {
			nextContentIndex = len(content)
		} else {
			nextContentIndex = contentIndex + cfg.LengthLimit
		}
		contentPiece := content[contentIndex:nextContentIndex]
		contentIndex = nextContentIndex
		log.Debugf("current content piece: %s", contentPiece)
		checkService := config.GetResponseCheckService(consumer)
		path, headers, body := common.GenerateRequestForText(config, config.Action, checkService, contentPiece, sessionID)
		err := config.Client.Post(path, headers, body, callback, config.Timeout)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			proxywasm.ResumeHttpResponse()
		}
	}
	singleCall()
	return types.ActionPause
}
