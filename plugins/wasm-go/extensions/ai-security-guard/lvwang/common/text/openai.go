package text

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
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

// printStackTrace 输出当前的调用栈信息
func printStackTrace() {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	log.Debugf("Stack trace:\n%s", string(buf[:n]))
}

// clearBufferState clears buffer-related state in context
// resetDuringCall: if true, resets during_call to false
func clearBufferState(ctx wrapper.HttpContext, resetDuringCall bool) [][]byte {
	// 输出调用栈
	printStackTrace()
	var emptyQueue [][]byte
	ctx.SetContext("bufferQueue", emptyQueue)
	ctx.SetContext("buffer_pending_content", "")
	ctx.SetContext("start_ts", time.Now().UnixMilli())
	if resetDuringCall {
		ctx.SetContext("during_call", false)
	}
	return emptyQueue
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
	if ctx.GetContext("bufferQueue") != nil {
		bufferQueue = ctx.GetContext("bufferQueue").([][]byte)
	} else {
		bufferQueue = [][]byte{}
	}
	buffer := ctx.GetStringContext("buffer_pending_content", "")
	startTs := ctx.GetContext("start_ts")
	if startTs == nil {
		startTs = time.Now().UnixMilli()
		ctx.SetContext("start_ts", startTs)
	}

	var singleCall func()
	callback := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		log.Info(string(responseBody))
		if statusCode != 200 || gjson.GetBytes(responseBody, "Code").Int() != 200 {
			if ctx.GetContext("end_of_stream_received").(bool) {
				proxywasm.ResumeHttpResponse()
			}
			// Clear state on error, reset start_ts for new time window
			bufferQueue = clearBufferState(ctx, true)
			return
		}
		var response cfg.Response
		err := json.Unmarshal(responseBody, &response)
		if err != nil {
			log.Error("failed to unmarshal aliyun content security response at response phase")
			if ctx.GetContext("end_of_stream_received").(bool) {
				proxywasm.ResumeHttpResponse()
			}
			// Clear state on error, reset start_ts for new time window
			bufferQueue = clearBufferState(ctx, true)
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
			// Clear state on deny
			bufferQueue = clearBufferState(ctx, false)
			return
		}
		endStream := ctx.GetContext("end_of_stream_received").(bool) && ctx.BufferQueueSize() == 0
		proxywasm.InjectEncodedDataToFilterChain(bytes.Join(bufferQueue, []byte("")), endStream)

		// Clear state after successful check
		bufferQueue = clearBufferState(ctx, true)

		if !endStream {
			singleCall()
		}
	}
	singleCall = func() {
		if ctx.GetContext("during_call").(bool) {
			return
		}
		endOfStreamReceived := ctx.GetContext("end_of_stream_received").(bool)
		if ctx.BufferQueueSize() == 0 && !endOfStreamReceived {
			return
		}
		log.Debugf("[singleCall] bufferQueue size: %d, buffer_rune_len: %d, endOfStreamReceived:%t", ctx.BufferQueueSize(), len([]rune(buffer)), endOfStreamReceived)
		var needFlush = false
		for ctx.BufferQueueSize() > 0 {
			front := ctx.PopBuffer()
			bufferQueue = append(bufferQueue, front)
			ctx.SetContext("bufferQueue", bufferQueue)
			msg := gjson.GetBytes(front, config.ResponseStreamContentJsonPath).String()
			buffer += msg
			bufferRuneLen := len([]rune(buffer))
			if bufferRuneLen >= config.BufferLimit {
				needFlush = true
			} else if config.BufferFlushTimeInterval > 0 {
				endTs := time.Now().UnixMilli()
				if endTs-startTs.(int64) > int64(config.BufferFlushTimeInterval) {
					needFlush = true
				}
			}

			needFlush = needFlush || endOfStreamReceived
			if !needFlush {
				ctx.SetContext("buffer_pending_content", buffer)
				log.Debugf("nothing to do! bufferQueue(buffer provider) size: %d, bufferqueue(buffer for consumer) size: %d, buffer_rune_len: %d",
					ctx.BufferQueueSize(), len(bufferQueue), bufferRuneLen)
				continue
			}

			// case 1: streaming body has reasoning_content, part of buffer maybe empty
			// case 2: streaming body has toolcall result, part of buffer maybe empty
			log.Debugf("current content piece: %s", buffer)
			if len(buffer) == 0 {
				proxywasm.InjectEncodedDataToFilterChain(bytes.Join(bufferQueue, []byte("")), endOfStreamReceived)
				bufferQueue = clearBufferState(ctx, false)
				continue
			}
			ctx.SetContext("during_call", true)
			checkService := config.GetResponseCheckService(consumer)
			path, headers, body := common.GenerateRequestForText(config, config.Action, checkService, buffer, sessionID)
			err := config.Client.Post(path, headers, body, callback, config.Timeout)
			if err != nil {
				log.Errorf("failed call the safe check service: %v", err)
				proxywasm.InjectEncodedDataToFilterChain(bytes.Join(bufferQueue, []byte("")), endOfStreamReceived)
				bufferQueue = clearBufferState(ctx, true)
			}
		}

		if endOfStreamReceived && len(bufferQueue) > 0 {
			log.Debugf("endOfStreamReceived=true but bufferQueue has %d items, forcing flush. current content piece: %s", len(bufferQueue), buffer)
			if len(buffer) == 0 {
				proxywasm.InjectEncodedDataToFilterChain(bytes.Join(bufferQueue, []byte("")), endOfStreamReceived)
				bufferQueue = clearBufferState(ctx, false)
				proxywasm.ResumeHttpResponse()
				return
			}

			ctx.SetContext("during_call", true)
			checkService := config.GetResponseCheckService(consumer)
			path, headers, body := common.GenerateRequestForText(config, config.Action, checkService, buffer, sessionID)
			err := config.Client.Post(path, headers, body, callback, config.Timeout)
			if err != nil {
				log.Errorf("failed call the safe check service: %v", err)
				proxywasm.InjectEncodedDataToFilterChain(bytes.Join(bufferQueue, []byte("")), true)
				bufferQueue = clearBufferState(ctx, false)
				proxywasm.ResumeHttpResponse()
			}
		}
	}
	if !ctx.GetContext("risk_detected").(bool) {
		unifiedChunk := wrapper.UnifySSEChunk(data)
		hasTrailingSeparator := bytes.HasSuffix(unifiedChunk, []byte("\n\n"))
		trimmedChunk := bytes.TrimSpace(unifiedChunk)
		chunks := bytes.Split(trimmedChunk, []byte("\n\n"))
		// Filter out empty chunks
		nonEmptyChunks := make([][]byte, 0, len(chunks))
		for _, chunk := range chunks {
			if len(chunk) > 0 {
				nonEmptyChunks = append(nonEmptyChunks, chunk)
			}
		}
		// Restore separators
		for i := range len(nonEmptyChunks) - 1 {
			nonEmptyChunks[i] = append(nonEmptyChunks[i], []byte("\n\n")...)
		}
		if hasTrailingSeparator && len(nonEmptyChunks) > 0 {
			nonEmptyChunks[len(nonEmptyChunks)-1] = append(nonEmptyChunks[len(nonEmptyChunks)-1], []byte("\n\n")...)
		}
		for _, chunk := range nonEmptyChunks {
			ctx.PushBuffer(chunk)
		}
		// for _, chunk := range bytes.Split(bytes.TrimSpace(wrapper.UnifySSEChunk(data)), []byte("\n\n")) {
		// 	ctx.PushBuffer([]byte(string(chunk) + "\n\n"))
		// }
		ctx.SetContext("end_of_stream_received", endOfStream)
		log.Debugf("bufferQueue size: %d, buffer_rune_len: %d, endOfStreamReceived:%t,during_call:%t", ctx.BufferQueueSize(), len([]rune(buffer)), endOfStream, ctx.GetContext("during_call").(bool))
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
