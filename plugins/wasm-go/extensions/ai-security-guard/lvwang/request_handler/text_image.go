package request_handler

// https://help.aliyun.com/document_detail/2937221.html?spm=a2c4g.11186623.help-menu-28415.d_4_3.5e66340d38SMdd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	cfg "github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/lvwang"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-security-guard/utils"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func parseContent(json gjson.Result) (text, imgUrl, imgBase64 string) {
	if json.IsArray() {
		for _, item := range json.Array() {
			switch item.Get("type").String() {
			case "text":
				text += item.Get("text").String()
			case "image_url":
				if strings.HasPrefix(item.Get("image_url.url").String(), "http") {
					imgUrl = "url"
				} else {
					imgBase64 = "base64"
				}
			}
		}
	} else {
		text = json.String()
	}
	return text, imgUrl, imgBase64
}

func HandleTextAndImageRequestBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	consumer, _ := ctx.GetContext("consumer").(string)
	checkService := config.GetRequestCheckService(consumer)
	startTime := time.Now().UnixMilli()
	// content := gjson.GetBytes(body, config.RequestContentJsonPath).String()
	content, imgUrl, imgBase64 := parseContent(gjson.GetBytes(body, config.RequestContentJsonPath))
	// content, _, _ := parseContent(gjson.GetBytes(body, config.RequestContentJsonPath))
	log.Debugf("Raw request content is: %s", content)
	if len(content) == 0 {
		log.Info("request content is empty. skip")
		return types.ActionContinue
	}
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
			log.Error("failed to unmarshal aliyun content security response at request phase")
			proxywasm.ResumeHttpRequest()
			return
		}
		if cfg.IsRiskLevelAcceptable(config.Action, response.Data, config, consumer) {
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
		denyMessage := utils.DefaultDenyMessage
		if config.DenyMessage != "" {
			denyMessage = config.DenyMessage
		} else if response.Data.Advice != nil && response.Data.Advice[0].Answer != "" {
			denyMessage = response.Data.Advice[0].Answer
		}
		marshalledDenyMessage := wrapper.MarshalStr(denyMessage)
		if config.ProtocolOriginal {
			proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "application/json"}}, []byte(marshalledDenyMessage), -1)
		} else if gjson.GetBytes(body, "stream").Bool() {
			randomID := utils.GenerateRandomChatID()
			jsonData := []byte(fmt.Sprintf(utils.OpenAIStreamResponseFormat, randomID, marshalledDenyMessage, randomID))
			proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "text/event-stream;charset=UTF-8"}}, jsonData, -1)
		} else {
			randomID := utils.GenerateRandomChatID()
			jsonData := []byte(fmt.Sprintf(utils.OpenAIResponseFormat, randomID, marshalledDenyMessage))
			proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "application/json"}}, jsonData, -1)
		}
		ctx.DontReadResponseBody()
		config.IncrementCounter("ai_sec_request_deny", 1)
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
		var nextContentIndex int
		if contentIndex+utils.LengthLimit >= len(content) {
			nextContentIndex = len(content)
		} else {
			nextContentIndex = contentIndex + utils.LengthLimit
		}
		contentPiece := content[contentIndex:nextContentIndex]
		contentIndex = nextContentIndex
		log.Debugf("current content piece: %s", contentPiece)
		path, headers, body := lvwang.GenerateRequestForText(config, config.Action, checkService, contentPiece, sessionID)
		err := config.Client.Post(path, headers, body, callback, config.Timeout)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			proxywasm.ResumeHttpRequest()
		}
	}
	// check image
	if imgUrl != "" || imgBase64 != "" {
		checkService := config.GetRequestCheckService(consumer)
		path, headers, body := lvwang.GenerateRequestForImage(config, config.Action, checkService, imgUrl, imgBase64)
		err := config.Client.Post(path, headers, body, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			log.Info(string(responseBody))
			if statusCode != 200 || gjson.GetBytes(responseBody, "Code").Int() != 200 {
				proxywasm.ResumeHttpRequest()
				return
			}
			var response cfg.Response
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				log.Error("failed to unmarshal aliyun content security response at request phase")
				proxywasm.ResumeHttpRequest()
				return
			}
			if cfg.IsRiskLevelAcceptable(config.Action, response.Data, config, consumer) {
				endTime := time.Now().UnixMilli()
				ctx.SetUserAttribute("safecheck_request_rt", endTime-startTime)
				ctx.SetUserAttribute("safecheck_status", "request pass")
				ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
				// start checking text
				singleCall()
			}
			denyMessage := utils.DefaultDenyMessage
			if config.DenyMessage != "" {
				denyMessage = config.DenyMessage
			} else if response.Data.Advice != nil && response.Data.Advice[0].Answer != "" {
				denyMessage = response.Data.Advice[0].Answer
			}
			marshalledDenyMessage := wrapper.MarshalStr(denyMessage)
			if config.ProtocolOriginal {
				proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "application/json"}}, []byte(marshalledDenyMessage), -1)
			} else if gjson.GetBytes(body, "stream").Bool() {
				randomID := utils.GenerateRandomChatID()
				jsonData := []byte(fmt.Sprintf(utils.OpenAIStreamResponseFormat, randomID, marshalledDenyMessage, randomID))
				proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "text/event-stream;charset=UTF-8"}}, jsonData, -1)
			} else {
				randomID := utils.GenerateRandomChatID()
				jsonData := []byte(fmt.Sprintf(utils.OpenAIResponseFormat, randomID, marshalledDenyMessage))
				proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "application/json"}}, jsonData, -1)
			}
			ctx.DontReadResponseBody()
			config.IncrementCounter("ai_sec_request_deny", 1)
			endTime := time.Now().UnixMilli()
			ctx.SetUserAttribute("safecheck_request_rt", endTime-startTime)
			ctx.SetUserAttribute("safecheck_status", "reqeust deny")
			if response.Data.Advice != nil {
				ctx.SetUserAttribute("safecheck_riskLabel", response.Data.Result[0].Label)
				ctx.SetUserAttribute("safecheck_riskWords", response.Data.Result[0].RiskWords)
			}
			ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
		}, config.Timeout)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			proxywasm.ResumeHttpRequest()
		}
	} else {
		singleCall()
	}
	return types.ActionPause
}
