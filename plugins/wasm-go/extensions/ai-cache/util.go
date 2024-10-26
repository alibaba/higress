package main

import (
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/config"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func handleNonLastChunk(ctx wrapper.HttpContext, c config.PluginConfig, chunk []byte, log wrapper.Log) []byte {
	stream := ctx.GetContext(STREAM_CONTEXT_KEY)
	if stream == nil {
		return handleNonStreamChunk(ctx, c, chunk, log)
	}
	return handleStreamChunk(ctx, c, chunk, log)
}

func handleNonStreamChunk(ctx wrapper.HttpContext, c config.PluginConfig, chunk []byte, log wrapper.Log) []byte {
	tempContentI := ctx.GetContext(CACHE_CONTENT_CONTEXT_KEY)
	if tempContentI == nil {
		ctx.SetContext(CACHE_CONTENT_CONTEXT_KEY, chunk)
		return chunk
	}
	tempContent := tempContentI.([]byte)
	tempContent = append(tempContent, chunk...)
	ctx.SetContext(CACHE_CONTENT_CONTEXT_KEY, tempContent)
	return chunk
}

func handleStreamChunk(ctx wrapper.HttpContext, c config.PluginConfig, chunk []byte, log wrapper.Log) []byte {
	var partialMessage []byte
	partialMessageI := ctx.GetContext(PARTIAL_MESSAGE_CONTEXT_KEY)
	if partialMessageI != nil {
		partialMessage = append(partialMessageI.([]byte), chunk...)
	} else {
		partialMessage = chunk
	}
	messages := strings.Split(string(partialMessage), "\n\n")
	for i, msg := range messages {
		if i < len(messages)-1 {
			processSSEMessage(ctx, c, msg, log)
		}
	}
	if !strings.HasSuffix(string(partialMessage), "\n\n") {
		ctx.SetContext(PARTIAL_MESSAGE_CONTEXT_KEY, []byte(messages[len(messages)-1]))
	} else {
		ctx.SetContext(PARTIAL_MESSAGE_CONTEXT_KEY, nil)
	}
	return chunk
}

func processNonStreamLastChunk(ctx wrapper.HttpContext, c config.PluginConfig, chunk []byte, log wrapper.Log) string {
	var body []byte
	tempContentI := ctx.GetContext(CACHE_CONTENT_CONTEXT_KEY)
	if tempContentI != nil {
		body = append(tempContentI.([]byte), chunk...)
	} else {
		body = chunk
	}
	bodyJson := gjson.ParseBytes(body)
	value := bodyJson.Get(c.CacheValueFrom).String()
	if value == "" {
		log.Warnf("[%s] [processNonStreamLastChunk] parse value from response body failed, body:%s", PLUGIN_NAME, body)
	}
	return value
}

func processStreamLastChunk(ctx wrapper.HttpContext, c config.PluginConfig, chunk []byte, log wrapper.Log) string {
	if len(chunk) > 0 {
		var lastMessage []byte
		partialMessageI := ctx.GetContext(PARTIAL_MESSAGE_CONTEXT_KEY)
		if partialMessageI != nil {
			lastMessage = append(partialMessageI.([]byte), chunk...)
		} else {
			lastMessage = chunk
		}
		if !strings.HasSuffix(string(lastMessage), "\n\n") {
			log.Warnf("[%s] [processStreamLastChunk] invalid lastMessage:%s", PLUGIN_NAME, lastMessage)
			return ""
		}
		lastMessage = lastMessage[:len(lastMessage)-2]
		return processSSEMessage(ctx, c, string(lastMessage), log)
	}
	tempContentI := ctx.GetContext(CACHE_CONTENT_CONTEXT_KEY)
	if tempContentI == nil {
		return ""
	}
	return tempContentI.(string)
}

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
		log.Warnf("[%s] [processSSEMessage] invalid message: %s", PLUGIN_NAME, message)
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
		log.Warnf("[%s] [processSSEMessage] cannot extract content from message: %s", PLUGIN_NAME, message)
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
