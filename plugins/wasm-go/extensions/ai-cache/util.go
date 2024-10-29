package main

import (
	"fmt"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/config"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func handleNonLastChunk(ctx wrapper.HttpContext, c config.PluginConfig, chunk []byte, log wrapper.Log) error {
	stream := ctx.GetContext(STREAM_CONTEXT_KEY)
	err := error(nil)
	if stream == nil {
		err = handleNonStreamChunk(ctx, c, chunk, log)
	} else {
		err = handleStreamChunk(ctx, c, chunk, log)
	}
	return err
}

func handleNonStreamChunk(ctx wrapper.HttpContext, c config.PluginConfig, chunk []byte, log wrapper.Log) error {
	tempContentI := ctx.GetContext(CACHE_CONTENT_CONTEXT_KEY)
	if tempContentI == nil {
		ctx.SetContext(CACHE_CONTENT_CONTEXT_KEY, chunk)
		return nil
	}
	tempContent := tempContentI.([]byte)
	tempContent = append(tempContent, chunk...)
	ctx.SetContext(CACHE_CONTENT_CONTEXT_KEY, tempContent)
	return nil
}

func handleStreamChunk(ctx wrapper.HttpContext, c config.PluginConfig, chunk []byte, log wrapper.Log) error {
	var partialMessage []byte
	partialMessageI := ctx.GetContext(PARTIAL_MESSAGE_CONTEXT_KEY)
	log.Debugf("[handleStreamChunk] cache content: %v", ctx.GetContext(CACHE_CONTENT_CONTEXT_KEY))
	if partialMessageI != nil {
		partialMessage = append(partialMessageI.([]byte), chunk...)
	} else {
		partialMessage = chunk
	}
	messages := strings.Split(string(partialMessage), "\n\n")
	for i, msg := range messages {
		if i < len(messages)-1 {
			_, err := processSSEMessage(ctx, c, msg, log)
			if err != nil {
				return fmt.Errorf("[handleStreamChunk] processSSEMessage failed, error: %v", err)
			}
		}
	}
	if !strings.HasSuffix(string(partialMessage), "\n\n") {
		ctx.SetContext(PARTIAL_MESSAGE_CONTEXT_KEY, []byte(messages[len(messages)-1]))
	} else {
		ctx.SetContext(PARTIAL_MESSAGE_CONTEXT_KEY, nil)
	}
	return nil
}

func processNonStreamLastChunk(ctx wrapper.HttpContext, c config.PluginConfig, chunk []byte, log wrapper.Log) (string, error) {
	var body []byte
	tempContentI := ctx.GetContext(CACHE_CONTENT_CONTEXT_KEY)
	if tempContentI != nil {
		body = append(tempContentI.([]byte), chunk...)
	} else {
		body = chunk
	}
	bodyJson := gjson.ParseBytes(body)
	value := bodyJson.Get(c.CacheValueFrom).String()
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("[processNonStreamLastChunk] parse value from response body failed, body:%s", body)
	}
	return value, nil
}

func processStreamLastChunk(ctx wrapper.HttpContext, c config.PluginConfig, chunk []byte, log wrapper.Log) (string, error) {
	if len(chunk) > 0 {
		var lastMessage []byte
		partialMessageI := ctx.GetContext(PARTIAL_MESSAGE_CONTEXT_KEY)
		if partialMessageI != nil {
			lastMessage = append(partialMessageI.([]byte), chunk...)
		} else {
			lastMessage = chunk
		}
		if !strings.HasSuffix(string(lastMessage), "\n\n") {
			return "", fmt.Errorf("[processStreamLastChunk] invalid lastMessage:%s", lastMessage)
		}
		lastMessage = lastMessage[:len(lastMessage)-2]
		value, err := processSSEMessage(ctx, c, string(lastMessage), log)
		if err != nil {
			return "", fmt.Errorf("[processStreamLastChunk] processSSEMessage failed, error: %v", err)
		}
		return value, nil
	}
	tempContentI := ctx.GetContext(CACHE_CONTENT_CONTEXT_KEY)
	if tempContentI == nil {
		return "", nil
	}
	return tempContentI.(string), nil
}

func processSSEMessage(ctx wrapper.HttpContext, c config.PluginConfig, sseMessage string, log wrapper.Log) (string, error) {
	subMessages := strings.Split(sseMessage, "\n")
	var message string
	for _, msg := range subMessages {
		if strings.HasPrefix(msg, "data:") {
			message = msg
			break
		}
	}
	if len(message) < 6 {
		return "", fmt.Errorf("[processSSEMessage] invalid message: %s", message)
	}

	// skip the prefix "data:"
	bodyJson := message[5:]

	if strings.TrimSpace(bodyJson) == "[DONE]" {
		return "", nil
	}

	// Extract values from JSON fields
	responseBody := gjson.Get(bodyJson, c.CacheStreamValueFrom)
	toolCalls := gjson.Get(bodyJson, c.CacheToolCallsFrom)

	if toolCalls.Exists() {
		// TODO: Temporarily store the tool_calls value in the context for processing
		ctx.SetContext(TOOL_CALLS_CONTEXT_KEY, toolCalls.String())
	}

	// Check if the ResponseBody field exists
	if !responseBody.Exists() {
		if ctx.GetContext(CACHE_CONTENT_CONTEXT_KEY) != nil {
			log.Debugf("[processSSEMessage] unable to extract content from message; cache content is not nil: %s", message)
			return "", nil
		}
		return "", fmt.Errorf("[processSSEMessage] unable to extract content from message; cache content is nil: %s", message)
	} else {
		tempContentI := ctx.GetContext(CACHE_CONTENT_CONTEXT_KEY)

		// If there is no content in the cache, initialize and set the content
		if tempContentI == nil {
			content := responseBody.String()
			ctx.SetContext(CACHE_CONTENT_CONTEXT_KEY, content)
			return content, nil
		}

		// Update the content in the cache
		appendMsg := responseBody.String()
		content := tempContentI.(string) + appendMsg
		ctx.SetContext(CACHE_CONTENT_CONTEXT_KEY, content)
		return content, nil
	}
}
