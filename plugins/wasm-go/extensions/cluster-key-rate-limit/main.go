// Copyright (c) 2024 Alibaba Group Holding Ltd.
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
	"net"
	"net/url"
	"strconv"
	"strings"

	"cluster-key-rate-limit/config"
	"cluster-key-rate-limit/util"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"cluster-key-rate-limit",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
	)
}

const (
	// ClusterKeyPrefix 集群限流插件在 Redis 中 key 的统一前缀
	ClusterKeyPrefix = "higress-cluster-key-rate-limit"
	// ClusterGlobalRateLimitFormat  全局限流模式 redis key 为 ClusterKeyPrefix:限流规则名称:global_threshold:时间窗口:窗口内限流数
	ClusterGlobalRateLimitFormat = ClusterKeyPrefix + ":%s:global_threshold:%d:%d"
	// ClusterRateLimitFormat 规则限流模式 redis key 为 ClusterKeyPrefix:限流规则名称:限流类型:时间窗口:窗口内限流数:限流key名称:限流key对应的实际值
	ClusterRateLimitFormat = ClusterKeyPrefix + ":%s:%s:%d:%d:%s:%s"
	FixedWindowScript      = `
    	local ttl = redis.call('ttl', KEYS[1])
    	if ttl < 0 then
        	redis.call('set', KEYS[1], ARGV[1] - 1, 'EX', ARGV[2])
        	return {ARGV[1], ARGV[1] - 1, ARGV[2]}
    	end
    	return {ARGV[1], redis.call('incrby', KEYS[1], -1), ttl}
	`

	LimitContextKey = "LimitContext" // 限流上下文信息

	CookieHeader = "cookie"

	RateLimitLimitHeader     = "X-RateLimit-Limit"     // 限制的总请求数
	RateLimitRemainingHeader = "X-RateLimit-Remaining" // 剩余还可以发送的请求数
	RateLimitResetHeader     = "X-RateLimit-Reset"     // 限流重置时间（触发限流时返回）
)

type LimitContext struct {
	count     int
	remaining int
	reset     int
}

func parseConfig(json gjson.Result, cfg *config.ClusterKeyRateLimitConfig) error {
	err := config.InitRedisClusterClient(json, cfg)
	if err != nil {
		return err
	}
	err = config.ParseClusterKeyRateLimitConfig(json, cfg)
	if err != nil {
		return err
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config config.ClusterKeyRateLimitConfig) types.Action {
	limitKey, count, timeWindow := "", int64(0), int64(0)

	if config.GlobalThreshold != nil {
		// 全局限流模式
		limitKey = fmt.Sprintf(ClusterGlobalRateLimitFormat, config.RuleName, config.GlobalThreshold.TimeWindow, config.GlobalThreshold.Count)
		count = config.GlobalThreshold.Count
		timeWindow = config.GlobalThreshold.TimeWindow
	} else {
		// 规则限流模式
		val, ruleItem, configItem := checkRequestAgainstLimitRule(ctx, config.RuleItems)
		if ruleItem == nil || configItem == nil {
			// 没有匹配到限流规则直接返回
			return types.ActionContinue
		}

		limitKey = fmt.Sprintf(ClusterRateLimitFormat, config.RuleName, ruleItem.LimitType, configItem.TimeWindow, configItem.Count, ruleItem.Key, val)
		count = configItem.Count
		timeWindow = configItem.TimeWindow
	}

	// 执行限流逻辑
	keys := []interface{}{limitKey}
	args := []interface{}{count, timeWindow}
	err := config.RedisClient.Eval(FixedWindowScript, 1, keys, args, func(response resp.Value) {
		resultArray := response.Array()
		if len(resultArray) != 3 {
			log.Errorf("redis response parse error, response: %v", response)
			proxywasm.ResumeHttpRequest()
			return
		}
		context := LimitContext{
			count:     resultArray[0].Integer(),
			remaining: resultArray[1].Integer(),
			reset:     resultArray[2].Integer(),
		}
		if context.remaining < 0 {
			// 触发限流
			rejected(config, context)
		} else {
			ctx.SetContext(LimitContextKey, context)
			proxywasm.ResumeHttpRequest()
		}
	})

	if err != nil {
		log.Errorf("redis call failed: %v", err)
		return types.ActionContinue
	}
	return types.ActionPause
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config config.ClusterKeyRateLimitConfig) types.Action {
	limitContext, ok := ctx.GetContext(LimitContextKey).(LimitContext)
	if !ok {
		return types.ActionContinue
	}
	if config.ShowLimitQuotaHeader {
		_ = proxywasm.ReplaceHttpResponseHeader(RateLimitLimitHeader, strconv.Itoa(limitContext.count))
		_ = proxywasm.ReplaceHttpResponseHeader(RateLimitRemainingHeader, strconv.Itoa(limitContext.remaining))
	}
	return types.ActionContinue
}

func checkRequestAgainstLimitRule(ctx wrapper.HttpContext, ruleItems []config.LimitRuleItem) (string, *config.LimitRuleItem, *config.LimitConfigItem) {
	if len(ruleItems) > 0 {
		for _, rule := range ruleItems {
			val, ruleItem, configItem := hitRateRuleItem(ctx, rule)
			if ruleItem != nil && configItem != nil {
				return val, ruleItem, configItem
			}
		}
	}
	return "", nil, nil
}

func hitRateRuleItem(ctx wrapper.HttpContext, rule config.LimitRuleItem) (string, *config.LimitRuleItem, *config.LimitConfigItem) {
	switch rule.LimitType {
	// 根据HTTP请求头限流
	case config.LimitByHeaderType, config.LimitByPerHeaderType:
		val, err := proxywasm.GetHttpRequestHeader(rule.Key)
		if err != nil {
			return logDebugAndReturnEmpty("failed to get request header %s: %v", rule.Key, err)
		}
		return val, &rule, findMatchingItem(rule.LimitType, rule.ConfigItems, val)
	// 根据HTTP请求参数限流
	case config.LimitByParamType, config.LimitByPerParamType:
		parse, err := url.Parse(ctx.Path())
		if err != nil {
			return logDebugAndReturnEmpty("failed to parse request path: %v", err)
		}
		query, err := url.ParseQuery(parse.RawQuery)
		if err != nil {
			return logDebugAndReturnEmpty("failed to parse query params: %v", err)
		}
		val, ok := query[rule.Key]
		if !ok {
			return logDebugAndReturnEmpty("request param %s is empty", rule.Key)
		}
		return val[0], &rule, findMatchingItem(rule.LimitType, rule.ConfigItems, val[0])
	// 根据consumer限流
	case config.LimitByConsumerType, config.LimitByPerConsumerType:
		val, err := proxywasm.GetHttpRequestHeader(config.ConsumerHeader)
		if err != nil {
			return logDebugAndReturnEmpty("failed to get request header %s: %v", config.ConsumerHeader, err)
		}
		return val, &rule, findMatchingItem(rule.LimitType, rule.ConfigItems, val)
	// 根据cookie中key值限流
	case config.LimitByCookieType, config.LimitByPerCookieType:
		cookie, err := proxywasm.GetHttpRequestHeader(CookieHeader)
		if err != nil {
			return logDebugAndReturnEmpty("failed to get request cookie : %v", err)
		}
		val := util.ExtractCookieValueByKey(cookie, rule.Key)
		if val == "" {
			return logDebugAndReturnEmpty("cookie key '%s' extracted from cookie '%s' is empty.", rule.Key, cookie)
		}
		return val, &rule, findMatchingItem(rule.LimitType, rule.ConfigItems, val)
	// 根据客户端IP限流
	case config.LimitByPerIpType:
		realIp, err := getDownStreamIp(rule)
		if err != nil {
			log.Warnf("failed to get down stream ip: %v", err)
			return "", &rule, nil
		}
		for _, item := range rule.ConfigItems {
			if _, found, _ := item.IpNet.Get(realIp); !found {
				continue
			}
			return realIp.String(), &rule, &item
		}
	}
	return "", nil, nil
}

func logDebugAndReturnEmpty(errMsg string, args ...interface{}) (string, *config.LimitRuleItem, *config.LimitConfigItem) {
	log.Debugf(errMsg, args...)
	return "", nil, nil
}

func findMatchingItem(limitType config.LimitRuleItemType, items []config.LimitConfigItem, key string) *config.LimitConfigItem {
	for _, item := range items {
		// per类型,检查allType和regexpType
		if limitType == config.LimitByPerHeaderType ||
			limitType == config.LimitByPerParamType ||
			limitType == config.LimitByPerConsumerType ||
			limitType == config.LimitByPerCookieType {
			if item.ConfigType == config.AllType || (item.ConfigType == config.RegexpType && item.Regexp.MatchString(key)) {
				return &item
			}
		}
		// 其他类型,直接比较key
		if item.Key == key {
			return &item
		}
	}
	return nil
}

func getDownStreamIp(rule config.LimitRuleItem) (net.IP, error) {
	var (
		realIpStr string
		err       error
	)
	if rule.LimitByPerIp.SourceType == config.HeaderSourceType {
		realIpStr, err = proxywasm.GetHttpRequestHeader(rule.LimitByPerIp.HeaderName)
		if err == nil {
			realIpStr = strings.Split(strings.Trim(realIpStr, " "), ",")[0]
		}
	} else {
		var bs []byte
		bs, err = proxywasm.GetProperty([]string{"source", "address"})
		realIpStr = string(bs)
	}
	if err != nil {
		return nil, err
	}
	ip := util.ParseIP(realIpStr)
	realIP := net.ParseIP(ip)
	if realIP == nil {
		return nil, fmt.Errorf("invalid ip[%s]", ip)
	}
	return realIP, nil
}

func rejected(config config.ClusterKeyRateLimitConfig, context LimitContext) {
	headers := make(map[string][]string)
	headers[RateLimitResetHeader] = []string{strconv.Itoa(context.reset)}
	if config.ShowLimitQuotaHeader {
		headers[RateLimitLimitHeader] = []string{strconv.Itoa(context.count)}
		headers[RateLimitRemainingHeader] = []string{strconv.Itoa(0)}
	}
	_ = proxywasm.SendHttpResponseWithDetail(
		config.RejectedCode, "cluster-key-rate-limit.rejected", util.ReconvertHeaders(headers), []byte(config.RejectedMsg), -1)
}
