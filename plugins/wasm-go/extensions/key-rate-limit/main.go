// Copyright (c) 2023 Alibaba Group Holding Ltd.
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
	"fmt"
	"net/url"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	tokenbucket "github.com/kubeservice-stack/common/pkg/ratelimiter/token"
	"github.com/pkg/errors"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

var (
	ruleSet         bool            // 插件是否至少在一个 domain 或 route 上生效
	protectionSpace = "MSE Gateway" // 认证失败时，返回响应头 WWW-Authenticate: Key Rate realm=MSE Gateway
)

const (
	SecondNano = 1000 * 1000 * 1000
	MinuteNano = SecondNano * 60
	HourNano   = MinuteNano * 60
	DayNano    = HourNano * 24
)

func main() {
	wrapper.SetCtx(
		"key-rate-auth",
		wrapper.ParseOverrideConfigBy(parseGlobalConfig, parseOverrideRuleConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type LimitKey struct {
	// @Title key名称
	// @Title en-US key name
	// @Description 限流器名称。
	// @Description en-US The name of the ratelimit.
	key string `yaml:"key"`

	// @Title 每秒请求数
	// @Title en-US Number of requests allowed per second
	// @Description 允许每秒请求次数.
	// @Description en-US Number of requests allowed per second.
	queryPerSecond uint64 `yaml:"query_per_second"`

	// @Title 每分钟请求数
	// @Title en-US Number of requests allowed per minute
	// @Description 允许每分钟请求次数.
	// @Description en-US Number of requests allowed per minute.
	queryPerMinute uint64 `yaml:"query_per_minute"`

	// @Title 每小时请求数
	// @Title en-US Number of requests allowed per hour
	// @Description 允许每小时请求次数.
	// @Description en-US Number of requests allowed per hour.
	queryPerHour uint64 `yaml:"query_per_hour"`

	// @Title 每天请求数
	// @Title en-US Number of requests allowed per day
	// @Description 允许每天请求次数.
	// @Description en-US Number of requests allowed per day.
	queryPerDay uint64 `yaml:"query_per_day"`
}

// @Name key-rate-auth
// @Category auth
// @Phase AUTHN
// @Priority 322
// @Title zh-CN Key Rate Auth
// @Description zh-CN 本插件实现了基于特定键值实现限流功能。键值来源可以是 URL 参数、HTTP 请求头.
// @Description en-US This plugin implements a rate-limiting function based on specific key-values. The key-values may come from URL parameters or HTTP headers.
// @IconUrl https://img.alicdn.com/imgextra/i4/O1CN01BPFGlT1pGZ2VDLgaH_!!6000000005333-2-tps-42-42.png
// @Version 1.0.0
//
// @Contact.name Higress Team
// @Contact.url http://higress.io/
// @Contact.email admin@higress.io
//
// @Example
// global_auth: false
// limit_keys:
//   - key: keyname1
//     query_per_second: 10
//   - key: keyname2
//     query_per_hour: 10000
// limit_by_header: true
// @End
type KeyRateConfig struct {
	// @Title 是否开启全局认证
	// @Title en-US Enable Global Auth
	// @Description 若不开启全局认证，则全局配置只提供凭证信息。只有在域名或路由上进行了配置才会启用认证。
	// @Description en-US If set to false, only consumer info will be accepted from the global config. Auth feature shall only be enabled if the corresponding domain or route is configured.
	// @Scope GLOBAL
	globalAuth *bool `yaml:"global_auth"`

	// @Title 限流器实例列表
	// @Title en-US ratelimt instance List
	// @Description 配置匹配键值后的限流次数.
	// @Description en-US Rate-limiting thresholds when matching specific key-values.
	// @Scope GLOBAL
	limitKeys []LimitKey `yaml:"limit_keys"`

	// @Title 来源于URL参数键值
	// @Title en-US the API Key from the URL parameters.
	// @Description 如果配置 true 时，网关会尝试从 URL 参数中解析键值.
	// @Description en-US When configured true, the gateway will try to parse the API Key from the URL parameters.
	// @Scope GLOBAL
	limitByParam string `yaml:"limit_by_param,omitempty"`

	// @Title 键值源于Header参数键值
	// @Title en-US the API Key from the HTTP headers.
	// @Description 如果配置 true 时，网关会尝试从 HTTP 请求头中解析键值.
	// @Description en-US When configured true, the gateway will try to parse the API Key from the HTTP headers.
	// @Scope GLOBAL
	limitByHeader string `yaml:"limit_by_header,omitempty"`

	// @Title 授权访问的调用方列表
	// @Title en-US Allowed Consumers
	// @Description 对于匹配上述条件的请求，允许访问的调用方列表。
	// @Description en-US Consumers to be allowed for matched requests.
	allow []string `yaml:"allow"`

	ratelimter2name map[string]*tokenbucket.TokenBucket `yaml:"-"`
}

func parseGlobalConfig(json gjson.Result, global *KeyRateConfig, log wrapper.Log) error {
	// log.Debug("global config")
	ruleSet = false
	global.ratelimter2name = make(map[string]*tokenbucket.TokenBucket)

	limitkeys := json.Get("limit_keys")
	if !limitkeys.Exists() {
		return errors.New("limit_keys is required")
	}
	if len(limitkeys.Array()) == 0 {
		return errors.New("limit_keys cannot be empty")
	}

	for _, item := range limitkeys.Array() {
		key := item.Get("key")
		if !key.Exists() || key.String() == "" {
			return errors.New("limit_keys key name is required")
		}
		// key duplicate
		if _, ok := global.ratelimter2name[key.String()]; ok {
			return errors.Errorf("duplicate limit_keys key name: %s", key.String())
		}

		query_per_second := item.Get("query_per_second")
		query_per_minute := item.Get("query_per_minute")
		query_per_hour := item.Get("query_per_hour")
		query_per_day := item.Get("query_per_day")
		if !query_per_second.Exists() && !query_per_minute.Exists() &&
			!query_per_hour.Exists() && query_per_day.Exists() {
			return errors.New("must one of query_per_second/query_per_minute/query_per_hour/query_per_day required")
		}
		v, idx, ok := CheckRequest(query_per_day.Int(), query_per_hour.Int(), query_per_minute.Int(), query_per_second.Int())
		if !ok {
			return errors.New("just one of query_per_second/query_per_minute/query_per_hour/query_per_day required")
		}

		limitkey := LimitKey{
			key:            key.String(),
			queryPerDay:    Uint64(query_per_day.Exists(), uint64(query_per_day.Int()), 0),
			queryPerHour:   Uint64(query_per_hour.Exists(), uint64(query_per_hour.Int()), 0),
			queryPerMinute: Uint64(query_per_minute.Exists(), uint64(query_per_minute.Int()), 0),
			queryPerSecond: Uint64(query_per_second.Exists(), uint64(query_per_second.Int()), 0),
		}
		global.limitKeys = append(global.limitKeys, limitkey)
		global.ratelimter2name[key.String()] = tokenbucket.New(key.String(), v, time.Duration(idx))
	}

	globalAuth := json.Get("global_auth")
	if globalAuth.Exists() {
		ga := globalAuth.Bool()
		global.globalAuth = &ga
	}

	// limit_by_param or limit_by_header
	inquery := json.Get("limit_by_param")
	inheader := json.Get("limit_by_header")
	if !inheader.Exists() && !inquery.Exists() {
		return errors.New("must one of limit_by_header/limit_by_param required")
	}

	if inquery.Exists() {
		global.limitByParam = inquery.String()
	}
	if inheader.Exists() {
		global.limitByHeader = inheader.String()
	}
	return nil
}

func parseOverrideRuleConfig(json gjson.Result, global KeyRateConfig, config *KeyRateConfig, log wrapper.Log) error {
	log.Debug("domain/route config")
	// override config via global
	*config = global

	allow := json.Get("allow")
	if !allow.Exists() {
		return errors.New("allow is required")
	}
	if len(allow.Array()) == 0 {
		return errors.New("allow cannot be empty")
	}

	for _, item := range allow.Array() {
		config.allow = append(config.allow, item.String())
	}
	ruleSet = true

	return nil
}

// key-rate-auth 插件认证逻辑：
// - global_auth == true 开启全局生效：
//   - 若当前 domain/route 未配置 allow 列表，即未配置该插件：则在所有 consumers 中查找，如果找到则认证通过，否则认证失败 (1*)
//   - 若当前 domain/route 配置了该插件：则在 allow 列表中查找，如果找到则认证通过，否则认证失败
//
// - global_auth == false 非全局生效：(2*)
//   - 若当前 domain/route 未配置该插件：则直接放行
//   - 若当前 domain/route 配置了该插件：则在 allow 列表中查找，如果找到则认证通过，否则认证失败
func onHttpRequestHeaders(ctx wrapper.HttpContext, config KeyRateConfig, log wrapper.Log) types.Action {
	var (
		noAllow            = len(config.allow) == 0 // 未配置 allow 列表，表示插件在该 domain/route 未生效
		globalAuthNoSet    = config.globalAuth == nil
		globalAuthSetFalse = !globalAuthNoSet && !*config.globalAuth
	)
	// log.Debugf("global auth set: %t", !globalAuthNoSet)
	// log.Debugf("rule set: %t", ruleSet)
	// log.Debugf("config: %+v", config)

	// 不需要认证而直接放行的情况：
	// - global_auth == false 且 当前 domain/route 未配置该插件
	// - global_auth 未设置 且 有至少一个 domain/route 配置该插件 且 当前 domain/route 未配置该插件
	if globalAuthSetFalse || (globalAuthNoSet && ruleSet) {
		if noAllow {
			log.Info("key rate authorization is not required")
			return types.ActionContinue
		}
	}

	// 以下为需要认证的情况：
	key := ""
	if config.limitByHeader != "" {
		value, err := proxywasm.GetHttpRequestHeader(config.limitByHeader)
		if err == nil && value != "" {
			key = value
		}
	} else if config.limitByParam != "" {
		requestUrl, _ := proxywasm.GetHttpRequestHeader(":path")
		url, _ := url.Parse(requestUrl)
		queryValues := url.Query()
		values, ok := queryValues[config.limitByParam]
		if ok && len(values) > 0 {
			key = values[0]
		}
	}

	if key == "" {
		log.Info("Not found limit key in request")
		return types.ActionContinue
	}

	ratelimit, ok := config.ratelimter2name[key]
	if !ok {
		log.Info("Not found key rate authorization")
		return types.ActionContinue
	}

	if ratelimit.Limit() {
		return TooManyRequest()
	}
	return authenticated()
}

func TooManyRequest() types.Action {
	_ = proxywasm.SendHttpResponse(429, WWWAuthenticateHeader(protectionSpace),
		[]byte("Too many requests"), -1)
	return types.ActionContinue
}

func authenticated() types.Action {
	return types.ActionContinue
}

func WWWAuthenticateHeader(realm string) [][2]string {
	return [][2]string{
		{"WWW-Authenticate", fmt.Sprintf("Key rate realm=%s", realm)},
	}
}

func Uint64(expr bool, a, b uint64) uint64 {
	if expr {
		return a
	}
	return b
}

func CheckRequest(a, b, c, d int64) (uint64, int64, bool) {
	values := []int64{a, b, c, d}
	indexs := []int64{SecondNano, SecondNano, MinuteNano, DayNano}
	count := 0
	ret := uint64(0)
	idx := 0
	for index, value := range values {
		if value > 0 {
			count++
			idx = index
			ret = uint64(value)
		}
		if count >= 2 {
			return 0, 0, false
		}
	}
	if count <= 0 {
		return 0, 0, false
	}
	return ret, indexs[idx], true
}
