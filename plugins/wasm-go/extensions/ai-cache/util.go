package main

import (
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/config"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func processSSEMessage(ctx wrapper.HttpContext, config config.PluginConfig, sseMessage string, log wrapper.Log) string {
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
	responseBody := gjson.Get(bodyJson, config.CacheStreamValueFrom)
	toolCalls := gjson.Get(bodyJson, config.CacheToolCallsFrom)

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
