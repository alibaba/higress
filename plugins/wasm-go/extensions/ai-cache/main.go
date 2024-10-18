// 这个文件中主要将OnHttpRequestHeaders、OnHttpRequestBody、OnHttpResponseHeaders、OnHttpResponseBody这四个函数实现
// 其中的缓存思路调用cache.go中的逻辑，然后cache.go中的逻辑会调用textEmbeddingProvider和vectorStoreProvider中的逻辑（实例）
package main

import (
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/config"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

const (
	PLUGIN_NAME                 = "ai-cache"
	CACHE_KEY_CONTEXT_KEY       = "cacheKey"
	CACHE_KEY_EMBEDDING_KEY     = "cacheKeyEmbedding"
	CACHE_CONTENT_CONTEXT_KEY   = "cacheContent"
	PARTIAL_MESSAGE_CONTEXT_KEY = "partialMessage"
	TOOL_CALLS_CONTEXT_KEY      = "toolCalls"
	STREAM_CONTEXT_KEY          = "stream"
)

func main() {
	// CreateClient()
	wrapper.SetCtx(
		PLUGIN_NAME,
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onHttpResponseBody),
	)
}

func parseConfig(json gjson.Result, config *config.PluginConfig, log wrapper.Log) error {
	// config.EmbeddingProviderConfig.FromJson(json.Get("embeddingProvider"))
	// config.VectorDatabaseProviderConfig.FromJson(json.Get("vectorBaseProvider"))
	// config.RedisConfig.FromJson(json.Get("redis"))
	config.FromJson(json)
	if err := config.Validate(); err != nil {
		return err
	}
	// 注意，在 parseConfig 阶段初始化 client 会出错，比如 docker compose 中的 redis 就无法使用
	if err := config.Complete(log); err != nil {
		log.Errorf("complete config failed: %v", err)
		return err
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log) types.Action {
	// 这段代码是为了测试，在 parseConfig 阶段初始化 client 会出错，比如 docker compose 中的 redis 就无法使用
	// 但是在 onHttpRequestHeaders 中可以连接到 redis、
	// 修复需要修改 envoy
	// ----------------------------------------------------------------------------
	// if err := config.Complete(log); err != nil {
	// 	log.Errorf("complete config failed:%v", err)
	// }
	// activeCacheProvider := config.GetCacheProvider()
	// if err := activeCacheProvider.Init("", "", 2000); err != nil {
	// 	log.Errorf("init redis failed:%v", err)
	// }
	// activeCacheProvider.Set("test", "test", func(response resp.Value) {})
	// log.Warnf("redis init success")
	// ----------------------------------------------------------------------------

	contentType, _ := proxywasm.GetHttpRequestHeader("content-type")
	// The request does not have a body.
	if contentType == "" {
		return types.ActionContinue
	}
	if !strings.Contains(contentType, "application/json") {
		log.Warnf("content is not json, can't process: %s", contentType)
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	// The request has a body and requires delaying the header transmission until a cache miss occurs,
	// at which point the header should be sent.
	return types.HeaderStopIteration
}

func onHttpRequestBody(ctx wrapper.HttpContext, config config.PluginConfig, body []byte, log wrapper.Log) types.Action {

	bodyJson := gjson.ParseBytes(body)
	// TODO: It may be necessary to support stream mode determination for different LLM providers.
	stream := false
	if bodyJson.Get("stream").Bool() {
		stream = true
		ctx.SetContext(STREAM_CONTEXT_KEY, struct{}{})
	}

	var key string
	if config.CacheKeyStrategy == "lastQuestion" {
		key = bodyJson.Get("messages.@reverse.0.content").String()
	} else if config.CacheKeyStrategy == "allQuestions" {
		// Retrieve all user messages and concatenate them
		messages := bodyJson.Get("messages").Array()
		var userMessages []string
		for _, msg := range messages {
			if msg.Get("role").String() == "user" {
				userMessages = append(userMessages, msg.Get("content").String())
			}
		}
		key = strings.Join(userMessages, " ")
	} else if config.CacheKeyStrategy == "disable" {
		log.Debugf("[onHttpRequestBody] cache key strategy is disabled")
		ctx.DontReadRequestBody()
		return types.ActionContinue
	} else {
		log.Warnf("[onHttpRequestBody] unknown cache key strategy: %s", config.CacheKeyStrategy)
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}

	ctx.SetContext(CACHE_KEY_CONTEXT_KEY, key)
	log.Debugf("[onHttpRequestBody] key: %s", key)
	if key == "" {
		log.Debug("[onHttpRequestBody] parse key from request body failed")
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}

	if err := CheckCacheForKey(key, ctx, config, log, stream, true); err != nil {
		log.Errorf("check cache for key: %s failed, error: %v", key, err)
		return types.ActionContinue
	}

	return types.ActionPause
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log) types.Action {
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	if strings.Contains(contentType, "text/event-stream") {
		ctx.SetContext(STREAM_CONTEXT_KEY, struct{}{})
	}
	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config config.PluginConfig, chunk []byte, isLastChunk bool, log wrapper.Log) []byte {
	log.Debugf("[onHttpResponseBody] escaped chunk: %q", string(chunk))
	log.Debugf("[onHttpResponseBody] isLastChunk: %v", isLastChunk)

	if ctx.GetContext(TOOL_CALLS_CONTEXT_KEY) != nil {
		return chunk
	}

	key := ctx.GetContext(CACHE_KEY_CONTEXT_KEY)
	if key == nil {
		log.Debug("[onHttpResponseBody] key is nil, bypass caching")
		return chunk
	}

	if !isLastChunk {
		handlePartialChunk(ctx, config, chunk, log)
		return chunk
	}

	// Handle last chunk
	var value string
	var err error

	if len(chunk) > 0 {
		value, err = processNonEmptyChunk(ctx, config, chunk, log)
	} else {
		value, err = processEmptyChunk(ctx, config, chunk, log)
	}

	if err != nil {
		log.Warnf("[onHttpResponseBody] failed to process chunk: %v", err)
		return chunk
	}
	// Cache the final value
	cacheResponse(ctx, config, key.(string), value, log)

	// Handle embedding upload if available
	uploadEmbeddingAndAnswer(ctx, config, key.(string), value, log)

	return chunk
}
