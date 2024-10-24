package main

import (
	"fmt"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/config"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func processSSEMessage(ctx wrapper.HttpContext, c config.PluginConfig, sseMessage string, log wrapper.Log) string {
	subMessages := strings.Split(sseMessage, "\n")
	var message string
	for _, msg := range subMessages {
		if strings.HasPrefix(msg, "data: ") {
			message = msg
			break
		}
	}
	if len(message) < 6 {
		log.Warnf("invalid message: %s", message)
		return ""
	}

	// skip the prefix "data:"
	bodyJson := message[5:]
	// Extract values from JSON fields
	responseBody := gjson.Get(bodyJson, c.CacheStreamValueFrom)
	toolCalls := gjson.Get(bodyJson, c.CacheToolCallsFrom)

	if toolCalls.Exists() {
		// TODO: Temporarily store the tool_calls value in the context for processing
		ctx.SetContext(TOOL_CALLS_CONTEXT_KEY, toolCalls.String())
	}

	// Check if the ResponseBody field exists
	if !responseBody.Exists() {
		// Return an empty string if we cannot extract the content
		log.Warnf("cannot extract content from message: %s", message)
		return ""
	} else {
		tempContentI := ctx.GetContext(CACHE_CONTENT_CONTEXT_KEY)

		// If there is no content in the cache, initialize and set the content
		if tempContentI == nil {
			content := responseBody.String()
			ctx.SetContext(CACHE_CONTENT_CONTEXT_KEY, content)
			return content
		}

		// Update the content in the cache
		appendMsg := responseBody.String()
		content := tempContentI.(string) + appendMsg
		ctx.SetContext(CACHE_CONTENT_CONTEXT_KEY, content)
		return content
	}

}

// Handles partial chunks of data when the full response is not received yet.
func handlePartialChunk(ctx wrapper.HttpContext, c config.PluginConfig, chunk []byte, log wrapper.Log) {
	stream := ctx.GetContext(STREAM_CONTEXT_KEY)

	if stream == nil {
		tempContentI := ctx.GetContext(CACHE_CONTENT_CONTEXT_KEY)
		if tempContentI == nil {
			ctx.SetContext(CACHE_CONTENT_CONTEXT_KEY, chunk)
		} else {
			tempContent := append(tempContentI.([]byte), chunk...)
			ctx.SetContext(CACHE_CONTENT_CONTEXT_KEY, tempContent)
		}
	} else {
		partialMessage := appendPartialMessage(ctx, chunk)
		messages := strings.Split(string(partialMessage), "\n\n")
		for _, msg := range messages[:len(messages)-1] {
			processSSEMessage(ctx, c, msg, log)
		}
		savePartialMessage(ctx, partialMessage, messages)
	}
}

// Appends the partial message chunks
func appendPartialMessage(ctx wrapper.HttpContext, chunk []byte) []byte {
	partialMessageI := ctx.GetContext(PARTIAL_MESSAGE_CONTEXT_KEY)
	if partialMessageI != nil {
		return append(partialMessageI.([]byte), chunk...)
	}
	return chunk
}

// Saves the remaining partial message chunk
func savePartialMessage(ctx wrapper.HttpContext, partialMessage []byte, messages []string) {
	if len(messages) == 0 {
		ctx.SetContext(PARTIAL_MESSAGE_CONTEXT_KEY, nil)
		return
	}

	if !strings.HasSuffix(string(partialMessage), "\n\n") {
		ctx.SetContext(PARTIAL_MESSAGE_CONTEXT_KEY, []byte(messages[len(messages)-1]))
	} else {
		ctx.SetContext(PARTIAL_MESSAGE_CONTEXT_KEY, nil)
	}
}

// Processes a non-empty data chunk and returns the parsed value or an error
func processNonEmptyChunk(ctx wrapper.HttpContext, c config.PluginConfig, chunk []byte, log wrapper.Log) (string, error) {
	stream := ctx.GetContext(STREAM_CONTEXT_KEY)
	var value string

	if stream == nil {
		body := appendFinalBody(ctx, chunk)
		bodyJson := gjson.ParseBytes(body)
		value = bodyJson.Get(c.CacheValueFrom).String()

		if value == "" {
			return "", fmt.Errorf("failed to parse value from response body: %s", body)
		}
	} else {
		value, err := processFinalStreamMessage(ctx, c, log, chunk)
		if err != nil {
			return "", err
		}
		return value, nil
	}

	return value, nil
}

func processEmptyChunk(ctx wrapper.HttpContext, c config.PluginConfig, chunk []byte, log wrapper.Log) (string, error) {
	tempContentI := ctx.GetContext(CACHE_CONTENT_CONTEXT_KEY)
	if tempContentI == nil {
		return string(chunk), nil
	}
	value, ok := tempContentI.([]byte)
	if !ok {
		return "", fmt.Errorf("invalid type for tempContentI")
	}
	return string(value), nil
}

// Appends the final body chunk to the existing body content
func appendFinalBody(ctx wrapper.HttpContext, chunk []byte) []byte {
	tempContentI := ctx.GetContext(CACHE_CONTENT_CONTEXT_KEY)
	if tempContentI != nil {
		return append(tempContentI.([]byte), chunk...)
	}
	return chunk
}

// Processes the final SSE message chunk
func processFinalStreamMessage(ctx wrapper.HttpContext, c config.PluginConfig, log wrapper.Log, chunk []byte) (string, error) {
	var lastMessage []byte
	partialMessageI := ctx.GetContext(PARTIAL_MESSAGE_CONTEXT_KEY)

	if partialMessageI != nil {
		lastMessage = append(partialMessageI.([]byte), chunk...)
	} else {
		lastMessage = chunk
	}

	if !strings.HasSuffix(string(lastMessage), "\n\n") {
		log.Warnf("[onHttpResponseBody] invalid lastMessage: %s", lastMessage)
		return "", fmt.Errorf("invalid lastMessage format")
	}

	lastMessage = lastMessage[:len(lastMessage)-2] // Remove the last \n\n
	return processSSEMessage(ctx, c, string(lastMessage), log), nil
}
