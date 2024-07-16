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
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func main() {
	wrapper.SetCtx(
		"cluster-key-rate-limit",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
	)
}

const (
	ClusterRateLimitFormat string = "higress-cluster-key-rate-limit:%s:%s:%s:%s" // redis key为前缀:限流规则名称:限流类型:限流key名称:限流key对应的实际值
	FixedWindowScript      string = `
    	local ttl = redis.call('ttl', KEYS[1])
    	if ttl < 0 then
        	redis.call('set', KEYS[1], ARGV[1] - 1, 'EX', ARGV[2])
        	return {ARGV[1], ARGV[1] - 1, ARGV[2]}
    	end
    	return {ARGV[1], redis.call('incrby', KEYS[1], -1), ttl}
	`

	LimitContextKey string = "LimitContext" // 限流上下文信息

	ConsumerHeader string = "x-mse-consumer" // LimitByConsumer从该request header获取consumer的名字
	CookieHeader   string = "cookie"

	RateLimitLimitHeader     string = "X-RateLimit-Limit"     // 限制的总请求数
	RateLimitRemainingHeader string = "X-RateLimit-Remaining" // 剩余还可以发送的请求数
	RateLimitResetHeader     string = "X-RateLimit-Reset"     // 限流重置时间（触发限流时返回）
)

type LimitContext struct {
	count     int
	remaining int
	reset     int
}

func parseConfig(json gjson.Result, config *ClusterKeyRateLimitConfig, log wrapper.Log) error {
	err := initRedisClusterClient(json, config)
	if err != nil {
		return err
	}
	err = parseClusterKeyRateLimitConfig(json, config)
	if err != nil {
		return err
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config ClusterKeyRateLimitConfig, log wrapper.Log) types.Action {
	// 判断是否命中限流规则
	val, ruleItem, configItem := checkRequestAgainstLimitRule(ctx, config.ruleItems, log)
	if ruleItem == nil || configItem == nil {
		return types.ActionContinue
	}

	// 构建redis限流key和参数
	limitKey := fmt.Sprintf(ClusterRateLimitFormat, config.ruleName, ruleItem.limitType, ruleItem.key, val)
	keys := []interface{}{limitKey}
	args := []interface{}{configItem.count, configItem.timeWindow}
	// 执行限流逻辑
	err := config.redisClient.Eval(FixedWindowScript, 1, keys, args, func(response resp.Value) {
		defer func() {
			_ = proxywasm.ResumeHttpRequest()
		}()
		resultArray := response.Array()
		if len(resultArray) != 3 {
			log.Errorf("redis response parse error, response: %v", response)
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
		}
	})
	if err != nil {
		log.Errorf("redis call failed: %v", err)
		return types.ActionContinue
	}
	return types.ActionPause
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config ClusterKeyRateLimitConfig, log wrapper.Log) types.Action {
	limitContext, ok := ctx.GetContext(LimitContextKey).(LimitContext)
	if !ok {
		return types.ActionContinue
	}
	if config.showLimitQuotaHeader {
		_ = proxywasm.ReplaceHttpResponseHeader(RateLimitLimitHeader, strconv.Itoa(limitContext.count))
		_ = proxywasm.ReplaceHttpResponseHeader(RateLimitRemainingHeader, strconv.Itoa(limitContext.remaining))
	}
	return types.ActionContinue
}

func checkRequestAgainstLimitRule(ctx wrapper.HttpContext, ruleItems []LimitRuleItem, log wrapper.Log) (string, *LimitRuleItem, *LimitConfigItem) {
	for _, rule := range ruleItems {
		val, ruleItem, configItem := hitRateRuleItem(ctx, rule, log)
		if ruleItem != nil && configItem != nil {
			return val, ruleItem, configItem
		}
	}
	return "", nil, nil
}

func hitRateRuleItem(ctx wrapper.HttpContext, rule LimitRuleItem, log wrapper.Log) (string, *LimitRuleItem, *LimitConfigItem) {
	switch rule.limitType {
	// 根据HTTP请求头限流
	case limitByHeaderType, limitByPerHeaderType:
		val, err := proxywasm.GetHttpRequestHeader(rule.key)
		if err != nil {
			return logDebugAndReturnEmpty(log, "failed to get request header %s: %v", rule.key, err)
		}
		return val, &rule, findMatchingItem(rule.limitType, rule.configItems, val)
	// 根据HTTP请求参数限流
	case limitByParamType, limitByPerParamType:
		parse, err := url.Parse(ctx.Path())
		if err != nil {
			return logDebugAndReturnEmpty(log, "failed to parse request path: %v", err)
		}
		query, err := url.ParseQuery(parse.RawQuery)
		if err != nil {
			return logDebugAndReturnEmpty(log, "failed to parse query params: %v", err)
		}
		val, ok := query[rule.key]
		if !ok {
			return logDebugAndReturnEmpty(log, "request param %s is empty", rule.key)
		}
		return val[0], &rule, findMatchingItem(rule.limitType, rule.configItems, val[0])
	// 根据consumer限流
	case limitByConsumerType, limitByPerConsumerType:
		val, err := proxywasm.GetHttpRequestHeader(ConsumerHeader)
		if err != nil {
			return logDebugAndReturnEmpty(log, "failed to get request header %s: %v", ConsumerHeader, err)
		}
		return val, &rule, findMatchingItem(rule.limitType, rule.configItems, val)
	// 根据cookie中key值限流
	case limitByCookieType, limitByPerCookieType:
		cookie, err := proxywasm.GetHttpRequestHeader(CookieHeader)
		if err != nil {
			return logDebugAndReturnEmpty(log, "failed to get request cookie : %v", err)
		}
		val := extractCookieValueByKey(cookie, rule.key)
		if val == "" {
			return logDebugAndReturnEmpty(log, "cookie key '%s' extracted from cookie '%s' is empty.", rule.key, cookie)
		}
		return val, &rule, findMatchingItem(rule.limitType, rule.configItems, val)
	// 根据客户端IP限流
	case limitByPerIpType:
		realIp, err := getDownStreamIp(rule)
		if err != nil {
			log.Warnf("failed to get down stream ip: %v", err)
			return "", &rule, nil
		}
		for _, item := range rule.configItems {
			if _, found, _ := item.ipNet.Get(realIp); !found {
				continue
			}
			return realIp.String(), &rule, &item
		}
	}
	return "", nil, nil
}

func logDebugAndReturnEmpty(log wrapper.Log, errMsg string, args ...interface{}) (string, *LimitRuleItem, *LimitConfigItem) {
	log.Debugf(errMsg, args...)
	return "", nil, nil
}

func findMatchingItem(limitType limitRuleItemType, items []LimitConfigItem, key string) *LimitConfigItem {
	for _, item := range items {
		// per类型,检查allType和regexpType
		if limitType == limitByPerHeaderType ||
			limitType == limitByPerParamType ||
			limitType == limitByPerConsumerType ||
			limitType == limitByPerCookieType {
			if item.configType == allType || (item.configType == regexpType && item.regexp.MatchString(key)) {
				return &item
			}
		}
		// 其他类型,直接比较key
		if item.key == key {
			return &item
		}
	}
	return nil
}

func getDownStreamIp(rule LimitRuleItem) (net.IP, error) {
	var (
		realIpStr string
		err       error
	)
	if rule.limitByPerIp.sourceType == HeaderSourceType {
		realIpStr, err = proxywasm.GetHttpRequestHeader(rule.limitByPerIp.headerName)
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
	ip := parseIP(realIpStr)
	realIP := net.ParseIP(ip)
	if realIP == nil {
		return nil, fmt.Errorf("invalid ip[%s]", ip)
	}
	return realIP, nil
}

func rejected(config ClusterKeyRateLimitConfig, context LimitContext) {
	headers := make(map[string][]string)
	headers[RateLimitResetHeader] = []string{strconv.Itoa(context.reset)}
	if config.showLimitQuotaHeader {
		headers[RateLimitLimitHeader] = []string{strconv.Itoa(context.count)}
		headers[RateLimitRemainingHeader] = []string{strconv.Itoa(0)}
	}
	_ = proxywasm.SendHttpResponseWithDetail(
		config.rejectedCode, "cluster-key-rate-limit.rejected", reconvertHeaders(headers), []byte(config.rejectedMsg), -1)
}
