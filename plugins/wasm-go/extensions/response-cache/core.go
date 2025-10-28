package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/response-cache/config"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/resp"
)

// CheckCacheForKey checks if the key is in the cache, or triggers similarity search if not found.
func CheckCacheForKey(key string, ctx wrapper.HttpContext, c config.PluginConfig) error {
	activeCacheProvider := c.GetCacheProvider()
	if activeCacheProvider == nil {
		return logAndReturnError("[CheckCacheForKey] no cache provider configured")
	}

	queryKey := activeCacheProvider.GetCacheKeyPrefix() + key
	log.Debugf("[%s] [CheckCacheForKey] querying cache with key: %s", PLUGIN_NAME, queryKey)

	err := activeCacheProvider.Get(queryKey, func(response resp.Value) {
		handleCacheResponse(key, response, ctx, c)
	})

	if err != nil {
		log.Errorf("[%s] [CheckCacheForKey] failed to retrieve key: %s from cache, error: %v", PLUGIN_NAME, key, err)
		return err
	}

	return nil
}

// handleCacheResponse processes cache response and handles cache hits and misses.
func handleCacheResponse(key string, response resp.Value, ctx wrapper.HttpContext, c config.PluginConfig) {
	if err := response.Error(); err == nil && !response.IsNull() {
		log.Infof("[%s] cache hit for key: %s", PLUGIN_NAME, key)
		processCacheHit(key, response.String(), ctx, c)
		return
	}

	log.Infof("[%s] [handleCacheResponse] cache miss for key: %s", PLUGIN_NAME, key)
	if err := response.Error(); err != nil {
		log.Errorf("[%s] [handleCacheResponse] error retrieving key: %s from cache, error: %v", PLUGIN_NAME, key, err)
	}
	proxywasm.ResumeHttpRequest()
}

// processCacheHit handles a successful cache hit.
func processCacheHit(key string, response string, ctx wrapper.HttpContext, c config.PluginConfig) {
	if strings.TrimSpace(response) == "" {
		log.Warnf("[%s] [processCacheHit] cached response for key %s is empty", PLUGIN_NAME, key)
		proxywasm.ResumeHttpRequest()
		return
	}

	log.Debugf("[%s] [processCacheHit] cached response for key %s: %s", PLUGIN_NAME, key, response)

	ctx.SetContext(CACHE_KEY_CONTEXT_KEY, nil)

	contentType := fmt.Sprintf("%s", c.CacheValueFromBodyType)
	headers := [][2]string{
		{"content-type", contentType},
		{"x-cache-status", "hit"},
	}

	proxywasm.SendHttpResponseWithDetail(200, "response-cache.hit", headers, []byte(response), -1)

}

// logAndReturnError logs an error and returns it.
func logAndReturnError(message string) error {
	message = fmt.Sprintf("[%s] %s", PLUGIN_NAME, message)
	log.Errorf(message)
	return errors.New(message)
}

// Caches the response value
func cacheResponse(ctx wrapper.HttpContext, c config.PluginConfig, key string, value string) {
	if strings.TrimSpace(value) == "" {
		log.Warnf("[%s] [cacheResponse] cached value for key %s is empty", PLUGIN_NAME, key)
		return
	}

	activeCacheProvider := c.GetCacheProvider()
	if activeCacheProvider != nil {
		queryKey := activeCacheProvider.GetCacheKeyPrefix() + key
		_ = activeCacheProvider.Set(queryKey, value, nil)
		log.Debugf("[%s] [cacheResponse] cache set success, key: %s, length of value: %d", PLUGIN_NAME, queryKey, len(value))
	}
}
