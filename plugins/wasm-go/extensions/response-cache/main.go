// 这个文件中主要将OnHttpRequestHeaders、OnHttpRequestBody、OnHttpResponseHeaders、OnHttpResponseBody这四个函数实现
// 其中的缓存思路调用cache.go中的逻辑
package main

import (
	"strconv"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/response-cache/config"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

const (
	PLUGIN_NAME                 = "response-cache"
	CACHE_KEY_CONTEXT_KEY       = "cacheKey"
	SKIP_CACHE_HEADER           = "x-higress-skip-response-cache"

	DEFAULT_MAX_BODY_BYTES uint32 = 10 * 1024 * 1024
)

func main() {
	// CreateClient()
	wrapper.SetCtx(
		PLUGIN_NAME,
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

func parseConfig(json gjson.Result, c *config.PluginConfig, log wrapper.Log) error {
	c.FromJson(json, log)
	if err := c.Validate(); err != nil {
		return err
	}

	if err := c.Complete(log); err != nil {
		log.Errorf("complete config failed: %v", err)
		return err
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, c config.PluginConfig, log wrapper.Log) types.Action {
	skipCache, _ := proxywasm.GetHttpRequestHeader(SKIP_CACHE_HEADER)
	if skipCache == "on" {
		ctx.SetContext(SKIP_CACHE_HEADER, struct{}{})
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}

	// cache from request header
	if c.CacheKeyFromHeader != "" {
		key, _ := proxywasm.GetHttpRequestHeader((c.CacheKeyFromHeader))
		if key == "" {
			log.Warnf("[onHttpRequestHeaders] cache key from header: %s is empty, skip cache", c.CacheKeyFromHeader)
			ctx.DontReadRequestBody()
			return types.ActionContinue
		}
		log.Debugf("[onHttpRequestHeaders] cache key from request header: %s, key: %s", c.CacheKeyFromHeader, key);

		ctx.SetContext(CACHE_KEY_CONTEXT_KEY, key)

		if err := CheckCacheForKey(key, ctx, c, log); err != nil {
			log.Errorf("[onHttpRequestHeaders] check cache for key: %s failed, error: %v", key, err)
		}
		_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}

	// cache from request body but does not have a body or not json format
	contentType, _ := proxywasm.GetHttpRequestHeader("content-type")

	if contentType == "" ||  !strings.Contains(contentType, "application/json") {
		log.Warnf("[onHttpRequestHeaders] content is not json, can't process: %s", contentType)
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}

	ctx.SetRequestBodyBufferLimit(DEFAULT_MAX_BODY_BYTES)

	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	// The request has a body and requires delaying the header transmission until a cache miss occurs,
	// at which point the header should be sent.
	return types.HeaderStopIteration
}

func onHttpRequestBody(ctx wrapper.HttpContext, c config.PluginConfig, body []byte, log wrapper.Log) types.Action {
	var key string
	if c.CacheKeyFromBody != "" {
		bodyJson := gjson.ParseBytes(body)
		
		log.Debugf("[onHttpRequestBody] cache key from requestBody: %s", c.CacheKeyFromBody)

		key = bodyJson.Get(c.CacheKeyFromBody).String()
	
		if key == "" {
			log.Debug("[onHttpRequestBody] parse key from request body failed")
			ctx.DontReadResponseBody()
			return types.ActionContinue
		}
	} else {
		key = string(body)
		log.Debugf("[onHttpRequestBody] cache key from requestWholeBody.")
	}

	log.Debugf("[onHttpRequestBody] key: %s", key)
	ctx.SetContext(CACHE_KEY_CONTEXT_KEY, key)

	if err := CheckCacheForKey(key, ctx, c, log); err != nil {
		log.Errorf("[onHttpRequestBody] check cache for key: %s failed, error: %v", key, err)
		return types.ActionContinue
	}

	return types.ActionPause
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, c config.PluginConfig, log wrapper.Log) types.Action {
	status, err := proxywasm.GetHttpResponseHeader(":status")
	if err != nil {
		log.Errorf("[onHttpResponseBody] unable to load :status header from response: %v", err)
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}

	// 状态码判断
	found := false
	respCode, _ := strconv.Atoi(status)
	for _, element := range c.CacheResponseCode {
		if element == int32(respCode) {
			found = true
			break
		}
	}
	if !found {
		log.Infof("[onHttpResponseBody] status not allow to cached: %s",status)
		proxywasm.AddHttpResponseHeader("x-cache-status", "skip")
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}
	
	skipCache := ctx.GetContext(SKIP_CACHE_HEADER)
	if skipCache != nil {
		proxywasm.AddHttpResponseHeader("x-cache-status", "skip")
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}
	if ctx.GetContext(CACHE_KEY_CONTEXT_KEY) != nil {
		proxywasm.AddHttpResponseHeader("x-cache-status", "miss")
	}
	ctx.SetResponseBodyBufferLimit(DEFAULT_MAX_BODY_BYTES)
	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, c config.PluginConfig, body []byte, log wrapper.Log) types.Action {

	key := ctx.GetContext(CACHE_KEY_CONTEXT_KEY)
	if key == nil {
		log.Debug("[onHttpResponseBody] key is nil, skip cache")
		return types.ActionContinue
	}

	var value string
	if c.CacheValueFromBody != "" {
		if strings.Contains(c.CacheValueFromBodyType, "application/json") {
				//cache json parse response body
				bodyJson := gjson.ParseBytes(body)
				if !bodyJson.Exists() {
					log.Warnf("[onHttpResponseBody] parse json from non json response body: %s", body)
					return types.ActionContinue
				}
				value = bodyJson.Get(c.CacheValueFromBody).String()
				if strings.TrimSpace(value) == "" {
					log.Warnf("[onHttpResponseBody] parse value from response body failed, body:%s", body)
					return types.ActionContinue
				}
		}
		//If there are other body types, add a parsing process here.
	} else {
		value = string(body)
	}

	cacheResponse(ctx, c, key.(string), value, log)
	return types.ActionContinue

}
