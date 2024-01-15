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
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"strings"
	"sync"
)

const (
	CacheStrategyMemory = "memory"
	CacheStrategyDisk   = "disk"

	CacheMethodGET  = "GET"
	CacheMethodHEAD = "HEAD"
	CacheMethodPOST = "POST"
	CacheMethodPUT  = "PUT"
	CacheMethodALL  = "ALL"

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

	CacheHttpStatusCodeOk           = 200
	CacheHttpStatusCodeNotModified  = 304
	CacheHttpStatusCodeBadRequest   = 400
	CacheHttpStatusCodeUnauthorized = 401

	DefaultCacheTTL   = "300s"
	DefaultMemorySize = "50m"
	DefaultDiskSize   = "1g"
	DefaultDiskPath   = "/tmp/cache"
)

func main() {
	wrapper.SetCtx(
		"request-block",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
	)
}

// ProxyCacheConfig is the config for proxy-cache extension
type ProxyCacheConfig struct {
	Zones                []CacheZone
	CacheTTL             string
	CacheMethod          []string
	CacheKey             []string
	CacheHttpStatusCodes []uint32
}

type ProxyCache struct {
	config ProxyCacheConfig
	memory *MemoryCache
	disk   *DiskCache
	Lock   *sync.Mutex
}

type MemoryCache struct {
	Data map[string][]byte
}

type DiskCache struct {
	RootDir string
}

type CacheZone struct {
	CacheStrategy string
	Name          string
	MemorySize    string
	DiskSize      string
	DiskPath      string
}

func NewDefaultProxyCache() *ProxyCacheConfig {
	return &ProxyCacheConfig{
		CacheTTL:             DefaultCacheTTL,
		CacheMethod:          []string{CacheMethodGET, CacheMethodHEAD},
		CacheKey:             []string{CacheKeyHost, CacheKeyPath},
		CacheHttpStatusCodes: []uint32{CacheHttpStatusCodeOk},
		Zones:                make([]CacheZone, 0),
	}
}

func NewDefaultCacheZone() []CacheZone {
	return []CacheZone{
		{
			CacheStrategy: CacheStrategyMemory,
			Name:          "default",
			MemorySize:    DefaultMemorySize,
		},
		{
			CacheStrategy: CacheStrategyDisk,
			Name:          "default",
			DiskSize:      DefaultDiskSize,
			DiskPath:      DefaultDiskPath,
		},
	}
}

func parseConfig(json gjson.Result, config *ProxyCacheConfig, log wrapper.Log) error {
	config = NewDefaultProxyCache()
	if json.Get("cache_ttl").Exists() {
		cacheTTL := json.Get("cache_ttl").String()
		cacheTTL = strings.Replace(cacheTTL, " ", "", -1)
		config.CacheTTL = cacheTTL
	}
	if json.Get("cache_method").Exists() {
		cacheMethod := json.Get("cache_method").String()
		cacheMethod = strings.ToUpper(cacheMethod)
		config.CacheMethod = cacheMethod
	}
	if json.Get("cache_key").Exists() {
		cacheKeyArray := json.Get("cache_key").Array()
		for _, item := range cacheKeyArray {
			config.CacheKey = append(config.CacheKey, item.String())
		}
	}
	if json.Get("cache_http_status_codes").Exists() {
		cacheHttpStatusCodesArray := json.Get("cache_http_status_codes").Array()
		for _, item := range cacheHttpStatusCodesArray {
			config.CacheHttpStatusCodes = append(config.CacheHttpStatusCodes, uint32(item.Int()))
		}
	}
	if json.Get("zones").Exists() {
		zonesArray := json.Get("zones").Array()
		for _, item := range zonesArray {
			zone := CacheZone{}
			if item.Get("cache_strategy").Exists() {
				cacheStrategy := item.Get("cache_strategy").String()
				cacheStrategy = strings.ToLower(cacheStrategy)
				zone.CacheStrategy = cacheStrategy
			}
			if item.Get("name").Exists() {
				zone.Name = item.Get("name").String()
			}
			if item.Get("memory_size").Exists() {
				zone.MemorySize = item.Get("memory_size").String()
			}
			if item.Get("disk_size").Exists() {
				zone.DiskSize = item.Get("disk_size").String()
			}
			if item.Get("disk_path").Exists() {
				zone.DiskPath = item.Get("disk_path").String()
			}
			config.Zones = append(config.Zones, zone)
		}
	}
	if len(config.Zones) == 0 {
		config.Zones = NewDefaultCacheZone()
	}
	return nil
}

func onHttpRequestBody(ctx wrapper.HttpContext, config ProxyCacheConfig, body []byte, log wrapper.Log) types.Action {

}

func (p ProxyCacheConfig) containCacheMethod(method string) bool {
	for _, m := range p.CacheMethod {
		if m == method {
			return true
		}
	}
	return false
}
