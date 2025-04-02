package provider

import (
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
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

func decodeEmbeddingsRequest(body []byte, request *embeddingsRequest) error {
	if err := json.Unmarshal(body, request); err != nil {
		return fmt.Errorf("unable to unmarshal request: %v", err)
	}
	return nil
}

func replaceJsonRequestBody(request interface{}) error {
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

func replaceRequestBody(body []byte) error {
	log.Debugf("request body: %s", string(body))
	err := proxywasm.ReplaceHttpRequestBody(body)
	if err != nil {
		return fmt.Errorf("unable to replace the original request body: %v", err)
	}
	return nil
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

func ReplaceResponseBody(body []byte) error {
	log.Debugf("response body: %s", string(body))
	err := proxywasm.ReplaceHttpResponseBody(body)
	if err != nil {
		return fmt.Errorf("unable to replace the original response body: %v", err)
	}
	return nil
}
