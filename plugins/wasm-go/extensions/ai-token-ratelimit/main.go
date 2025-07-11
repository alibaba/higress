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
	"bytes"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

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
		"ai-token-ratelimit",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onHttpStreamingBody),
	)
}

const (
	ClusterRateLimitFormat        string = "higress-token-ratelimit:%s:%s:%d:%d:%s:%s" // ruleName, limitType, timewindow, windowsize, key, val
	RequestPhaseFixedWindowScript string = `
	local ttl = redis.call('ttl', KEYS[1])
	if ttl < 0 then
	redis.call('set', KEYS[1], ARGV[1], 'EX', ARGV[2])
	return {ARGV[1], ARGV[1], ARGV[2]}
	end
	return {ARGV[1], redis.call('get', KEYS[1]), ttl}
	`
	ResponsePhaseFixedWindowScript string = `
	local ttl = redis.call('ttl', KEYS[1])
	if ttl < 0 then
	redis.call('set', KEYS[1], ARGV[1]-ARGV[3], 'EX', ARGV[2])
	return {ARGV[1], ARGV[1]-ARGV[3], ARGV[2]}
	end
	return {ARGV[1], redis.call('decrby', KEYS[1], ARGV[3]), ttl}
	`

	LimitRedisContextKey string = "LimitRedisContext"

	ConsumerHeader string = "x-mse-consumer" // LimitByConsumer从该request header获取consumer的名字
	CookieHeader   string = "cookie"

	RateLimitLimitHeader     string = "X-TokenRateLimit-Limit"     // 限制的总请求数
	RateLimitRemainingHeader string = "X-TokenRateLimit-Remaining" // 剩余还可以发送的请求数
	RateLimitResetHeader     string = "X-TokenRateLimit-Reset"     // 限流重置时间（触发限流时返回）

	TokenRateLimitCount = "token_ratelimit_count" // metric name
)

type LimitContext struct {
	count     int
	remaining int
	reset     int
}

type LimitRedisContext struct {
	key    string
	count  int64
	window int64
}

func parseConfig(json gjson.Result, config *ClusterKeyRateLimitConfig, log log.Log) error {
	err := initRedisClusterClient(json, config, log)
	if err != nil {
		return err
	}
	err = parseClusterKeyRateLimitConfig(json, config)
	if err != nil {
		return err
	}
	// Metric settings
	config.counterMetrics = make(map[string]proxywasm.MetricCounter)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config ClusterKeyRateLimitConfig, log log.Log) types.Action {
	// 判断是否命中限流规则
	val, ruleItem, configItem := checkRequestAgainstLimitRule(ctx, config.ruleItems, log)
	if ruleItem == nil || configItem == nil {
		return types.ActionContinue
	}

	// 构建redis限流key和参数
	limitKey := fmt.Sprintf(ClusterRateLimitFormat, config.ruleName, ruleItem.limitType, configItem.timeWindow, configItem.count, ruleItem.key, val)
	keys := []interface{}{limitKey}
	args := []interface{}{configItem.count, configItem.timeWindow}

	limitRedisContext := LimitRedisContext{
		key:    limitKey,
		count:  configItem.count,
		window: configItem.timeWindow,
	}
	ctx.SetContext(LimitRedisContextKey, limitRedisContext)

	// 执行限流逻辑
	err := config.redisClient.Eval(RequestPhaseFixedWindowScript, 1, keys, args, func(response resp.Value) {
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
			ctx.SetUserAttribute("token_ratelimit_status", "limited")
			ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
			rejected(config, context)
		} else {
			proxywasm.ResumeHttpRequest()
		}
	})
	if err != nil {
		log.Errorf("redis call failed: %v", err)
		return types.ActionContinue
	}
	return types.ActionPause
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config ClusterKeyRateLimitConfig, data []byte, endOfStream bool, log log.Log) []byte {
	var inputToken, outputToken int64
	if inputToken, outputToken, ok := getUsage(data); ok {
		ctx.SetContext("input_token", inputToken)
		ctx.SetContext("output_token", outputToken)
	}
	if endOfStream {
		if ctx.GetContext("input_token") == nil || ctx.GetContext("output_token") == nil {
			return data
		}
		inputToken = ctx.GetContext("input_token").(int64)
		outputToken = ctx.GetContext("output_token").(int64)
		limitRedisContext, ok := ctx.GetContext(LimitRedisContextKey).(LimitRedisContext)
		if !ok {
			return data
		}
		keys := []interface{}{limitRedisContext.key}
		args := []interface{}{limitRedisContext.count, limitRedisContext.window, inputToken + outputToken}
		err := config.redisClient.Eval(ResponsePhaseFixedWindowScript, 1, keys, args, nil)
		if err != nil {
			log.Errorf("redis call failed: %v", err)
		}
	}
	return data
}

func getUsage(data []byte) (inputTokenUsage int64, outputTokenUsage int64, ok bool) {
	chunks := bytes.Split(bytes.TrimSpace(data), []byte("\n\n"))
	for _, chunk := range chunks {
		// the feature strings are used to identify the usage data, like:
		// {"model":"gpt2","usage":{"prompt_tokens":1,"completion_tokens":1}}
		if !bytes.Contains(chunk, []byte("prompt_tokens")) || !bytes.Contains(chunk, []byte("completion_tokens")) {
			continue
		}
		inputTokenObj := gjson.GetBytes(chunk, "usage.prompt_tokens")
		outputTokenObj := gjson.GetBytes(chunk, "usage.completion_tokens")
		if inputTokenObj.Exists() && outputTokenObj.Exists() {
			inputTokenUsage = inputTokenObj.Int()
			outputTokenUsage = outputTokenObj.Int()
			ok = true
			return
		}
	}
	return
}

func checkRequestAgainstLimitRule(ctx wrapper.HttpContext, ruleItems []LimitRuleItem, log log.Log) (string, *LimitRuleItem, *LimitConfigItem) {
	for _, rule := range ruleItems {
		val, ruleItem, configItem := hitRateRuleItem(ctx, rule, log)
		if ruleItem != nil && configItem != nil {
			return val, ruleItem, configItem
		}
	}
	return "", nil, nil
}

func hitRateRuleItem(ctx wrapper.HttpContext, rule LimitRuleItem, log log.Log) (string, *LimitRuleItem, *LimitConfigItem) {
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

func logDebugAndReturnEmpty(log log.Log, errMsg string, args ...interface{}) (string, *LimitRuleItem, *LimitConfigItem) {
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

func (config *ClusterKeyRateLimitConfig) incrementCounter(metricName string, inc uint64) {
	if inc == 0 {
		return
	}
	counter, ok := config.counterMetrics[metricName]
	if !ok {
		counter = proxywasm.DefineCounterMetric(metricName)
		config.counterMetrics[metricName] = counter
	}
	counter.Increment(inc)
}

func generateMetricName(route, cluster, model, consumer, metricName string) string {
	return fmt.Sprintf("route.%s.upstream.%s.model.%s.consumer.%s.metric.%s", route, cluster, model, consumer, metricName)
}

func getRouteName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"route_name"}); err != nil {
		return "-", err
	} else {
		return string(raw), nil
	}
}

func getClusterName() (string, error) {
	if raw, err := proxywasm.GetProperty([]string{"cluster_name"}); err != nil {
		return "-", err
	} else {
		return string(raw), nil
	}
}

func getConsumer() (string, error) {
	if consumer, err := proxywasm.GetHttpRequestHeader(ConsumerHeader); err != nil {
		return "none", err
	} else {
		return consumer, nil
	}
}

func rejected(config ClusterKeyRateLimitConfig, context LimitContext) {
	headers := make(map[string][]string)
	headers[RateLimitResetHeader] = []string{strconv.Itoa(context.reset)}
	_ = proxywasm.SendHttpResponseWithDetail(
		config.rejectedCode, "ai-token-ratelimit.rejected", reconvertHeaders(headers), []byte(config.rejectedMsg), -1)

	route, _ := getRouteName()
	cluster, _ := getClusterName()
	consumer, _ := getConsumer()
	config.incrementCounter(generateMetricName(route, cluster, "none", consumer, TokenRateLimitCount), 1)
}
