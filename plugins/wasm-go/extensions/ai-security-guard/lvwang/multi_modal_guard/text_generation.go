package multi_modal_guard

import (
	"bytes"
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
				imgContent := item.Get("image_url.url").String()
				if strings.HasPrefix(imgContent, "data:image") {
					imgBase64 = imgContent
				} else {
					imgUrl = imgContent
				}
			}
		}
	} else {
		text = json.String()
	}
	return text, imgUrl, imgBase64
}

func HandleTextGenerationRequestBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	consumer, _ := ctx.GetContext("consumer").(string)
	checkService := config.GetRequestCheckService(consumer)
	checkImageService := config.GetRequestImageCheckService(consumer)
	startTime := time.Now().UnixMilli()
	// content := gjson.GetBytes(body, config.RequestContentJsonPath).String()
	content, imgUrl, imgBase64 := parseContent(gjson.GetBytes(body, config.RequestContentJsonPath))
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
			log.Errorf("%+v", err)
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
		denyMessage := cfg.DefaultDenyMessage
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
			jsonData := []byte(fmt.Sprintf(cfg.OpenAIStreamResponseFormat, randomID, marshalledDenyMessage, randomID))
			proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "text/event-stream;charset=UTF-8"}}, jsonData, -1)
		} else {
			randomID := utils.GenerateRandomChatID()
			jsonData := []byte(fmt.Sprintf(cfg.OpenAIResponseFormat, randomID, marshalledDenyMessage))
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
		if contentIndex+cfg.LengthLimit >= len(content) {
			nextContentIndex = len(content)
		} else {
			nextContentIndex = contentIndex + cfg.LengthLimit
		}
		contentPiece := content[contentIndex:nextContentIndex]
		contentIndex = nextContentIndex
		log.Debugf("current content piece: %s", contentPiece)
		path, headers, body := lvwang.GenerateRequestForText(config, cfg.MultiModalGuard, checkService, contentPiece, sessionID)
		err := config.Client.Post(path, headers, body, callback, config.Timeout)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			proxywasm.ResumeHttpRequest()
		}
	}
	// check image
	if imgUrl != "" || imgBase64 != "" {
		path, headers, body := lvwang.GenerateRequestForImage(config, cfg.MultiModalGuardForBase64, checkImageService, imgUrl, imgBase64)
		err := config.Client.Post(path, headers, body, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			log.Info(string(responseBody))
			if statusCode != 200 || gjson.GetBytes(responseBody, "Code").Int() != 200 {
				// start checking text
				singleCall()
				return
			}
			var response cfg.Response
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				log.Errorf("%+v", err)
				// start checking text
				singleCall()
				return
			}
			if cfg.IsRiskLevelAcceptable(config.Action, response.Data, config, consumer) {
				endTime := time.Now().UnixMilli()
				ctx.SetUserAttribute("safecheck_request_rt", endTime-startTime)
				ctx.SetUserAttribute("safecheck_status", "request pass")
				ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
				// start checking text
				singleCall()
				return
			}
			denyMessage := cfg.DefaultDenyMessage
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
				jsonData := []byte(fmt.Sprintf(cfg.OpenAIStreamResponseFormat, randomID, marshalledDenyMessage, randomID))
				proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "text/event-stream;charset=UTF-8"}}, jsonData, -1)
			} else {
				randomID := utils.GenerateRandomChatID()
				jsonData := []byte(fmt.Sprintf(cfg.OpenAIResponseFormat, randomID, marshalledDenyMessage))
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

func HandleTextGenerationResponseHeader(ctx wrapper.HttpContext, config cfg.AISecurityConfig) types.Action {
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	ctx.SetContext("end_of_stream_received", false)
	ctx.SetContext("during_call", false)
	ctx.SetContext("risk_detected", false)
	sessionID, _ := utils.GenerateHexID(20)
	ctx.SetContext("sessionID", sessionID)
	if strings.Contains(contentType, "text/event-stream") {
		ctx.NeedPauseStreamingResponse()
		return types.ActionContinue
	} else {
		ctx.BufferResponseBody()
		return types.HeaderStopIteration
	}
}

func HandleTextGenerationStreamingResponseBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, data []byte, endOfStream bool) []byte {
	consumer, _ := ctx.GetContext("consumer").(string)
	var sessionID string
	if ctx.GetContext("sessionID") == nil {
		sessionID, _ = utils.GenerateHexID(20)
		ctx.SetContext("sessionID", sessionID)
	} else {
		sessionID, _ = ctx.GetContext("sessionID").(string)
	}
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
		var response cfg.Response
		err := json.Unmarshal(responseBody, &response)
		if err != nil {
			log.Error("failed to unmarshal aliyun content security response at response phase")
			if ctx.GetContext("end_of_stream_received").(bool) {
				proxywasm.ResumeHttpResponse()
			}
			ctx.SetContext("during_call", false)
			return
		}
		if !cfg.IsRiskLevelAcceptable(config.Action, response.Data, config, consumer) {
			denyMessage := cfg.DefaultDenyMessage
			if response.Data.Advice != nil && response.Data.Advice[0].Answer != "" {
				denyMessage = "\n" + response.Data.Advice[0].Answer
			} else if config.DenyMessage != "" {
				denyMessage = config.DenyMessage
			}
			marshalledDenyMessage := wrapper.MarshalStr(denyMessage)
			randomID := utils.GenerateRandomChatID()
			jsonData := []byte(fmt.Sprintf(cfg.OpenAIStreamResponseFormat, randomID, marshalledDenyMessage, randomID))
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
		if ctx.BufferQueueSize() >= config.BufferLimit || ctx.GetContext("end_of_stream_received").(bool) {
			var buffer string
			for ctx.BufferQueueSize() > 0 {
				front := ctx.PopBuffer()
				bufferQueue = append(bufferQueue, front)
				msg := gjson.GetBytes(front, config.ResponseStreamContentJsonPath).String()
				buffer += msg
				if len([]rune(buffer)) >= config.BufferLimit {
					break
				}
			}
			// if streaming body has reasoning_content, buffer maybe empty
			log.Debugf("current content piece: %s", buffer)
			if len(buffer) == 0 {
				return
			}
			ctx.SetContext("during_call", true)
			log.Debugf("current content piece: %s", buffer)
			checkService := config.GetResponseCheckService(consumer)
			path, headers, body := lvwang.GenerateRequestForText(config, config.Action, checkService, buffer, sessionID)
			err := config.Client.Post(path, headers, body, callback, config.Timeout)
			if err != nil {
				log.Errorf("failed call the safe check service: %v", err)
				if ctx.GetContext("end_of_stream_received").(bool) {
					proxywasm.ResumeHttpResponse()
				}
			}
		}
	}
	if !ctx.GetContext("risk_detected").(bool) {
		for _, chunk := range bytes.Split(bytes.TrimSpace(wrapper.UnifySSEChunk(data)), []byte("\n\n")) {
			ctx.PushBuffer([]byte(string(chunk) + "\n\n"))
		}
		ctx.SetContext("end_of_stream_received", endOfStream)
		if !ctx.GetContext("during_call").(bool) {
			singleCall()
		}
	} else if endOfStream {
		proxywasm.ResumeHttpResponse()
	}
	return []byte{}
}

func HandleTextGenerationResponseBody(ctx wrapper.HttpContext, config cfg.AISecurityConfig, body []byte) types.Action {
	consumer, _ := ctx.GetContext("consumer").(string)
	log.Debugf("checking response body...")
	startTime := time.Now().UnixMilli()
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	isStreamingResponse := strings.Contains(contentType, "event-stream")
	var content string
	if isStreamingResponse {
		content = utils.ExtractMessageFromStreamingBody(body, config.ResponseStreamContentJsonPath)
	} else {
		content = gjson.GetBytes(body, config.ResponseContentJsonPath).String()
	}
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
		denyMessage := cfg.DefaultDenyMessage
		if config.DenyMessage != "" {
			denyMessage = config.DenyMessage
		} else if response.Data.Advice != nil && response.Data.Advice[0].Answer != "" {
			denyMessage = response.Data.Advice[0].Answer
		}
		marshalledDenyMessage := wrapper.MarshalStr(denyMessage)
		if config.ProtocolOriginal {
			proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "application/json"}}, []byte(marshalledDenyMessage), -1)
		} else if isStreamingResponse {
			randomID := utils.GenerateRandomChatID()
			jsonData := []byte(fmt.Sprintf(cfg.OpenAIStreamResponseFormat, randomID, marshalledDenyMessage, randomID))
			proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "text/event-stream;charset=UTF-8"}}, jsonData, -1)
		} else {
			randomID := utils.GenerateRandomChatID()
			jsonData := []byte(fmt.Sprintf(cfg.OpenAIResponseFormat, randomID, marshalledDenyMessage))
			proxywasm.SendHttpResponse(uint32(config.DenyCode), [][2]string{{"content-type", "application/json"}}, jsonData, -1)
		}
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
		path, headers, body := lvwang.GenerateRequestForText(config, config.Action, checkService, contentPiece, sessionID)
		err := config.Client.Post(path, headers, body, callback, config.Timeout)
		if err != nil {
			log.Errorf("failed call the safe check service: %v", err)
			proxywasm.ResumeHttpResponse()
		}
	}
	singleCall()
	return types.ActionPause
}
