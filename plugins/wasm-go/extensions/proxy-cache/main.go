// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"higress/plugins/wasm-go/extensions/proxy-cache/cache"
	"strconv"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

const (
	CacheStrategyMemory = "memory"
	CacheStrategyDisk   = "disk"

	CacheMethodGET   = "GET"
	CacheMethodHEAD  = "HEAD"
	CacheMethodPURGE = "PURGE"

	CacheKeyHost       = "$host"
	CacheKeyScheme     = "$scheme"
	CacheKeyPath       = "$path"
	CacheKeyQuery      = "$query"
	CacheKeyMethod     = "$method"
	CacheKeyUserAgent  = "$user-agent"
	CacheKeyAccept     = "$accept"
	CacheKeyAcceptLang = "$accept-language"
	CacheKeyAcceptEnc  = "$accept-encoding"
	CacheKeyCookie     = "$cookie"
	CacheKeyReferer    = "$referer"

	CacheHttpStatusCodeOk = 200

	DefaultCacheTTL   = "300s"
	DefaultMemorySize = "50m"
)

func main() {
	wrapper.SetCtx(
		"proxy-cache",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
		wrapper.ParseOverrideConfigBy(parseGlobalConfig, parseOverrideRuleConfig),
	)
}

// ProxyCacheConfig is the config for proxy-cache extension
type ProxyCacheConfig struct {
	// CacheStrategy is the cache strategy, memory or disk
	CacheStrategy string
	// MemorySize is the memory size for memory cache or disk cache
	MemorySize string
	// DiskSize is the disk size for disk cache
	DiskSize string
	// RootDir is the root directory for disk cache
	RootDir string
	// CacheTTL is the ttl for cache
	CacheTTL string
	// CacheMethod is the method for cache
	CacheMethod []string
	// CacheKey is the key for cache
	CacheKey []string
	// CacheHttpStatusCodes is the http status codes for cache
	CacheHttpStatusCodes []uint32
	// actualTTL is the actual ttl by calculate CacheTTL
	actualTTL int
	// actualMemorySize is the actual memory size by calculate MemorySize
	actualCacheKey string
	// cache is the cache instance
	cache cache.Cache
}

func NewDefaultProxyCache() *ProxyCacheConfig {
	// default config is memory cache, current DiskSize and RootDir are empty
	// if CacheStrategy is disk, disk size and disk path must be set
	return &ProxyCacheConfig{
		CacheTTL:             DefaultCacheTTL,
		CacheMethod:          []string{CacheMethodGET, CacheMethodHEAD},
		CacheKey:             []string{CacheKeyHost, CacheKeyPath},
		CacheHttpStatusCodes: []uint32{CacheHttpStatusCodeOk},
		CacheStrategy:        CacheStrategyMemory,
		MemorySize:           DefaultMemorySize,
	}
}

func parseConfig(json gjson.Result, config *ProxyCacheConfig, log wrapper.Log) error {
	// create default config
	config = NewDefaultProxyCache()

	// get cache ttl
	if json.Get("cache_ttl").Exists() {
		cacheTTL := json.Get("cache_ttl").String()
		cacheTTL = strings.Replace(cacheTTL, " ", "", -1)
		config.CacheTTL = cacheTTL
	}
	// get cache method
	if json.Get("cache_method").Exists() {
		cacheMethodArray := json.Get("cache_method").Array()
		for _, item := range cacheMethodArray {
			cacheMethod := item.String()
			cacheMethod = strings.ToUpper(cacheMethod)
			config.CacheMethod = append(config.CacheMethod, cacheMethod)
		}
	}
	// get cache key
	if json.Get("cache_key").Exists() {
		cacheKeyArray := json.Get("cache_key").Array()
		for _, item := range cacheKeyArray {
			config.CacheKey = append(config.CacheKey, item.String())
		}
	}
	// get cache http status codes
	if json.Get("cache_http_status_codes").Exists() {
		cacheHttpStatusCodesArray := json.Get("cache_http_status_codes").Array()
		for _, item := range cacheHttpStatusCodesArray {
			config.CacheHttpStatusCodes = append(config.CacheHttpStatusCodes, uint32(item.Int()))
		}
	}
	// get cache strategy
	if json.Get("cache_strategy").Exists() {
		config.CacheStrategy = json.Get("cache_strategy").String()
	}
	// get memory size
	if json.Get("memory_size").Exists() {
		config.MemorySize = json.Get("memory_size").String()
	}
	// get disk size and disk path
	if json.Get("disk_size").Exists() {
		config.DiskSize = json.Get("disk_size").String()
	}
	if json.Get("disk_path").Exists() {
		config.RootDir = json.Get("disk_path").String()
	}

	// validate config
	err := validation(config, log)
	if err != nil {
		return err
	}

	// calculate actual ttl
	ttl, err := calculateTTL(config.CacheTTL)
	if err != nil {
		return err
	}

	// create cache
	switch config.CacheStrategy {
	case CacheStrategyMemory:
		memorySize, err := calculate(config.MemorySize)
		if err != nil {
			return err
		}
		config.cache, err = cache.NewMemoryCache(memorySize, ttl)
		if err != nil {
			return err
		}
	case CacheStrategyDisk:
		diskSize, err := calculate(config.DiskSize)
		if err != nil {
			return err
		}
		memorySize, err := calculate(config.MemorySize)
		if err != nil {
			return err
		}
		config.cache, err = cache.NewDiskCache(cache.DiskCacheOptions{
			RootDir:     config.RootDir,
			DiskLimit:   diskSize,
			MemoryLimit: memorySize,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func validation(config *ProxyCacheConfig, log wrapper.Log) error {
	if config.CacheStrategy != CacheStrategyMemory && config.CacheStrategy != CacheStrategyDisk {
		log.Errorf("invalid cache strategy: %s", config.CacheStrategy)
		return types.ErrorStatusBadArgument
	}
	if config.CacheStrategy == CacheStrategyDisk {
		if config.DiskSize == "" {
			log.Error("disk size is empty")
			return types.ErrorStatusBadArgument
		}
		if config.RootDir == "" {
			log.Error("disk path is empty")
			return types.ErrorStatusBadArgument
		}
	}
	if config.CacheTTL == "" {
		log.Error("cache ttl is empty")
		return types.ErrorStatusBadArgument
	}
	if config.MemorySize == "" {
		log.Error("memory size is empty")
		return types.ErrorStatusBadArgument
	}
	if config.CacheMethod == nil {
		log.Error("cache method is empty")
		return types.ErrorStatusBadArgument
	}
	if config.CacheKey == nil {
		log.Error("cache key is empty")
		return types.ErrorStatusBadArgument
	}
	if config.CacheHttpStatusCodes == nil {
		log.Error("cache http status codes is empty")
		return types.ErrorStatusBadArgument
	}
	return nil
}

func calculate(size string) (int, error) {
	size = strings.Replace(size, " ", "", -1)
	if strings.HasSuffix(size, "k") || strings.HasSuffix(size, "K") {
		size = strings.TrimSuffix(size, "k")
		size = strings.TrimSuffix(size, "K")
		sizeInt, err := strconv.Atoi(size)
		if err != nil {
			return 0, err
		}
		return sizeInt * 1024, nil
	}
	if strings.HasSuffix(size, "m") || strings.HasSuffix(size, "M") {
		size = strings.TrimSuffix(size, "m")
		size = strings.TrimSuffix(size, "M")
		sizeInt, err := strconv.Atoi(size)
		if err != nil {
			return 0, err
		}
		return sizeInt * 1024 * 1024, nil
	}
	if strings.HasSuffix(size, "g") || strings.HasSuffix(size, "G") {
		size = strings.TrimSuffix(size, "g")
		size = strings.TrimSuffix(size, "G")
		sizeInt, err := strconv.Atoi(size)
		if err != nil {
			return 0, err
		}
		return sizeInt * 1024 * 1024 * 1024, nil
	}
	sizeInt, err := strconv.Atoi(size)
	if err != nil {
		return 0, err
	}
	return sizeInt, nil
}

func calculateTTL(ttl string) (int, error) {
	ttl = strings.Replace(ttl, " ", "", -1)
	if strings.HasSuffix(ttl, "s") || strings.HasSuffix(ttl, "S") {
		ttl = strings.TrimSuffix(ttl, "s")
		ttl = strings.TrimSuffix(ttl, "S")
		ttlInt, err := strconv.Atoi(ttl)
		if err != nil {
			return 0, err
		}
		return ttlInt, nil
	}
	if strings.HasSuffix(ttl, "m") || strings.HasSuffix(ttl, "M") {
		ttl = strings.TrimSuffix(ttl, "m")
		ttl = strings.TrimSuffix(ttl, "M")
		ttlInt, err := strconv.Atoi(ttl)
		if err != nil {
			return 0, err
		}
		return ttlInt * 60, nil
	}
	if strings.HasSuffix(ttl, "h") || strings.HasSuffix(ttl, "H") {
		ttl = strings.TrimSuffix(ttl, "h")
		ttl = strings.TrimSuffix(ttl, "H")
		ttlInt, err := strconv.Atoi(ttl)
		if err != nil {
			return 0, err
		}
		return ttlInt * 60 * 60, nil
	}
	if strings.HasSuffix(ttl, "d") || strings.HasSuffix(ttl, "D") {
		ttl = strings.TrimSuffix(ttl, "d")
		ttl = strings.TrimSuffix(ttl, "D")
		ttlInt, err := strconv.Atoi(ttl)
		if err != nil {
			return 0, err
		}
		return ttlInt * 60 * 60 * 24, nil
	}
	ttlInt, err := strconv.Atoi(ttl)
	if err != nil {
		return 0, err
	}
	return ttlInt, nil
}

func onHttpRequestBody(ctx wrapper.HttpContext, config ProxyCacheConfig, body []byte, log wrapper.Log) types.Action {
	// add response header
	err := proxywasm.AddHttpResponseHeader("Higress-Cache", "MISS")
	if err != nil {
		log.Errorf("add response header failed: %v", err)
		return types.ActionContinue
	}

	// find value from cache, if found, return it
	// if not found, continue to process request
	cacheKeyList := make([]string, 0)
	for _, cacheKey := range config.CacheKey {
		switch cacheKey {
		case CacheKeyHost:
			host := ctx.Host()
			cacheKey = CacheKeyHost + host
		case CacheKeyScheme:
			scheme := ctx.Scheme()
			cacheKey = CacheKeyScheme + scheme
		case CacheKeyPath:
			path := ctx.Path()
			cacheKey = CacheKeyPath + path
		case CacheKeyQuery:
			query, err := proxywasm.GetHttpRequestHeader(":query")
			if err != nil {
				log.Error("parse request query failed")
				return types.ActionContinue
			}
			cacheKey = CacheKeyQuery + query
		case CacheKeyMethod:
			method := ctx.Method()
			cacheKey = CacheKeyMethod + method
		case CacheKeyUserAgent:
			userAgent, err := proxywasm.GetHttpRequestHeader("user-agent")
			if err != nil {
				log.Error("parse request user-agent failed")
				return types.ActionContinue
			}
			cacheKey = CacheKeyUserAgent + userAgent
		case CacheKeyAccept:
			accept, err := proxywasm.GetHttpRequestHeader("accept")
			if err != nil {
				log.Error("parse request accept failed")
				return types.ActionContinue
			}
			cacheKey = CacheKeyAccept + accept
		case CacheKeyAcceptLang:
			acceptLang, err := proxywasm.GetHttpRequestHeader("accept-language")
			if err != nil {
				log.Error("parse request accept-language failed")
				return types.ActionContinue
			}
			cacheKey = CacheKeyAcceptLang + acceptLang
		case CacheKeyAcceptEnc:
			acceptEnc, err := proxywasm.GetHttpRequestHeader("accept-encoding")
			if err != nil {
				log.Error("parse request accept-encoding failed")
				return types.ActionContinue
			}
			cacheKey = CacheKeyAcceptEnc + acceptEnc
		case CacheKeyCookie:
			cookie, err := proxywasm.GetHttpRequestHeader("cookie")
			if err != nil {
				log.Error("parse request cookie failed")
				return types.ActionContinue
			}
			cacheKey = CacheKeyCookie + cookie
		case CacheKeyReferer:
			referer, err := proxywasm.GetHttpRequestHeader("referer")
			if err != nil {
				log.Error("parse request referer failed")
				return types.ActionContinue
			}
			cacheKey = CacheKeyReferer + referer
		default:
			log.Errorf("invalid cache key: %s", cacheKey)
		}
		cacheKeyList = append(cacheKeyList, cacheKey)
	}
	// TODO: join them or use multi key?
	cacheKey := strings.Join(cacheKeyList, "-")
	config.actualCacheKey = cacheKey
	value, ok := config.cache.Get(cacheKey)
	if ok {
		// if method is PURGE, delete cache
		if ctx.Method() == CacheMethodPURGE {
			err := config.cache.Delete(cacheKey)
			if err != nil {
				log.Errorf("delete cache failed: %v", err)
				return types.ActionContinue
			}
			return types.ActionContinue
		}
		// update response header
		err := proxywasm.AddHttpResponseHeader("Higress-Cache", "HIT")
		if err != nil {
			log.Errorf("add response header failed: %v", err)
			return types.ActionContinue
		}
		_ = proxywasm.SendHttpResponse(200, nil, value, -1)
		return types.ActionPause
	}
	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config ProxyCacheConfig, body []byte, log wrapper.Log) types.Action {
	// cache response body
	// if response status code is not 200, do not cache it
	status, err := proxywasm.GetHttpResponseHeader(":status")
	if err != nil {
		log.Error("parse response status code failed")
		return types.ActionContinue
	}
	// convert status code to uint32
	statusCode, err := strconv.Atoi(status)
	if err != nil {
		log.Errorf("convert status code to uint32 failed: %v", err)
		return types.ActionContinue
	}
	if !config.containCacheHttpStatus(uint32(statusCode)) {
		return types.ActionContinue
	}
	// if request method is not GET or HEAD, do not cache it
	if !config.containCacheMethod(ctx.Method()) {
		return types.ActionContinue
	}
	// check actual cache key
	if config.actualCacheKey == "" {
		log.Error("actual cache key is empty")
		return types.ActionContinue
	}
	// cache response body
	err = config.cache.Set(config.actualCacheKey, body)
	if err != nil {
		log.Errorf("cache response body failed: %v", err)
		return types.ActionContinue
	}
	return types.ActionContinue
}

func (p ProxyCacheConfig) containCacheMethod(method string) bool {
	for _, m := range p.CacheMethod {
		if m == method {
			return true
		}
	}
	return false
}

func (p ProxyCacheConfig) containCacheHttpStatus(statusCode uint32) bool {
	for _, code := range p.CacheHttpStatusCodes {
		if code == statusCode {
			return true
		}
	}
	return false
}

func parseGlobalConfig(json gjson.Result, global *ProxyCacheConfig, log wrapper.Log) error {
	// if switch memory cache to disk or disk cache to memory, need to clean cache
	if json.Get("cache_strategy").Exists() && global.CacheStrategy != json.Get("cache_strategy").String() {
		if global.cache != nil {
			err := global.cache.Clean()
			if err != nil {
				log.Errorf("clean cache failed: %v", err)
				return err
			}
		}
	}
	return parseConfig(json, global, log)
}

func parseOverrideRuleConfig(json gjson.Result, global ProxyCacheConfig, config *ProxyCacheConfig, log wrapper.Log) error {
	*config = global
	return nil
}

// TODO: when delete wasm plugin, clean cache, how to do?
