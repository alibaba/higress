package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

func decodeChatCompletionRequest(body []byte, request *chatCompletionRequest) error {
	if err := json.Unmarshal(body, request); err != nil {
		return fmt.Errorf("unable to unmarshal request: %v", err)
	}
	if request.Messages == nil || len(request.Messages) == 0 {
		return fmt.Errorf("no message found in the request body: %s", body)
	}
	return nil
}

func replaceJsonRequestBody(request interface{}, log wrapper.Log) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("unable to marshal request: %v", err)
	}
	log.Debugf("request body: %s", string(body))
	err = proxywasm.ReplaceHttpRequestBody(body)
	if err != nil {
		return fmt.Errorf("unable to replace the original request body: %v", err)
	}
	return err
}

func insertContextMessage(request *chatCompletionRequest, content string) {
	fileMessage := chatMessage{
		Role:    roleSystem,
		Content: content,
	}
	var firstNonSystemMessageIndex int
	for i, message := range request.Messages {
		if message.Role != roleSystem {
			firstNonSystemMessageIndex = i
			break
		}
	}
	if firstNonSystemMessageIndex == 0 {
		request.Messages = append([]chatMessage{fileMessage}, request.Messages...)
	} else {
		request.Messages = append(request.Messages[:firstNonSystemMessageIndex], append([]chatMessage{fileMessage}, request.Messages[firstNonSystemMessageIndex:]...)...)
	}
}

func replaceJsonResponseBody(response interface{}, log wrapper.Log) error {
	body, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("unable to marshal response: %v", err)
	}
	log.Debugf("response body: %s", string(body))
	err = proxywasm.ReplaceHttpResponseBody(body)
	if err != nil {
		return fmt.Errorf("unable to replace the original response body: %v", err)
	}
	return err
}

type chatCompletionResponseConverter interface{}

// processStreamEvent 从上下文中取出缓冲区，将新chunk追加到缓冲区，然后处理缓冲区中的完整事件
func processStreamEvent(
	ctx wrapper.HttpContext,
	chunk []byte, isLastChunk bool,
	log wrapper.Log,
	streamResponseCovertFunc chatCompletionResponseConverter) []byte {

	if isLastChunk || len(chunk) == 0 {
		return nil
	}
	// 从上下文中取出缓冲区，将新chunk追加到缓冲区
	newBufferedBody := chunk
	if bufferedBody, has := ctx.GetContext(ctxKeyStreamingBody).([]byte); has {
		newBufferedBody = append(bufferedBody, chunk...)
	}

	// 初始化处理下标，以及将要返回的处理过的chunks
	var newEventPivot = -1
	var outputBuffer []byte

	// 从buffer区取出若干完整的chunk，将其转为openAI格式后返回
	// 处理可能包含多个事件的缓冲区
	for {
		eventStartIndex := bytes.Index(newBufferedBody, []byte(streamDataItemKey))
		if eventStartIndex == -1 {
			break // 没有找到新事件，跳出循环
		}

		// 移除缓冲区前面非事件部分
		newBufferedBody = newBufferedBody[eventStartIndex+len(streamDataItemKey):]

		// 查找事件结束的位置（即下一个事件的开始）
		newEventPivot = bytes.Index(newBufferedBody, []byte("\n\n"))
		if newEventPivot == -1 {
			// 未找到事件结束标识，跳出循环等待更多数据，若是最后一个chunk，不一定有2个换行符
			break
		}

		// 提取并处理一个完整的事件
		eventData := newBufferedBody[:newEventPivot]
		newBufferedBody = newBufferedBody[newEventPivot+2:] // 跳过结束标识

		// 转换并追加到输出缓冲区
		switch fn := streamResponseCovertFunc.(type) {
		case func(ctx wrapper.HttpContext, chunk []byte, log wrapper.Log) *chatCompletionResponse:
			openAIResponse := fn(ctx, eventData, log)
			convertedData, err := appendOpenAIChunk(openAIResponse, log)
			if err != nil {
				log.Errorf("failed to append openAI chunk: %v", err)
			}
			outputBuffer = append(outputBuffer, convertedData...)
		case func(ctx wrapper.HttpContext, chunk []byte, log wrapper.Log) []*chatCompletionResponse:
			openAIResponses := fn(ctx, eventData, log)
			for _, response := range openAIResponses {
				convertedData, err := appendOpenAIChunk(response, log)
				if err != nil {
					log.Errorf("failed to append openAI chunk: %v", err)
				}
				outputBuffer = append(outputBuffer, convertedData...)
			}
		default:
			log.Errorf("unsupported streamResponseCovertFunc type")
			return nil
		}
	}

	// 刷新剩余的不完整事件回到上下文缓冲区以便下次继续处理
	ctx.SetContext(ctxKeyStreamingBody, newBufferedBody)
	log.Debugf("=== modified response chunk: %s", string(outputBuffer))

	return outputBuffer
}

func appendOpenAIChunk(openAIResponse *chatCompletionResponse, log wrapper.Log) ([]byte, error) {
	openAIFormattedChunk, err := json.Marshal(openAIResponse)
	if err != nil {
		log.Errorf("unable to marshal response: %v", err)
		return nil, err
	}

	var responseBuilder strings.Builder
	appendResponse(&responseBuilder, string(openAIFormattedChunk))

	return []byte(responseBuilder.String()), nil
}

func appendResponse(responseBuilder *strings.Builder, responseBody string) {
	responseBuilder.WriteString(fmt.Sprintf("%s %s\n\n", streamDataItemKey, responseBody))
}
