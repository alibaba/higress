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

	"ai-token-ratelimit/config"
	"ai-token-ratelimit/util"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/tokenusage"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"ai-token-ratelimit",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessStreamingResponseBody(onHttpStreamingBody),
	)
}

const (
	RedisKeyPrefix string = "higress-token-ratelimit"
	// AiTokenGlobalRateLimitFormat  全局限流模式 redis key 为 RedisKeyPrefix:限流规则名称:global_threshold:时间窗口
	AiTokenGlobalRateLimitFormat = RedisKeyPrefix + ":%s:global_threshold:%d"
	// AiTokenRateLimitFormat 规则限流模式 redis key 为 RedisKeyPrefix:限流规则名称:限流类型:时间窗口:限流key名称:限流key对应的实际值
	AiTokenRateLimitFormat        = RedisKeyPrefix + ":%s:%s:%d:%s:%s"
	RequestPhaseFixedWindowScript = `
		local current = redis.call('get', KEYS[1])
		local ttl = redis.call('ttl', KEYS[1])
		local threshold = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
	
    	-- 键不存在时，返回初始状态（计数0，窗口时间为过期时间）
		if not current then
			return {threshold, 0, window}
		end

		-- 修复异常过期时间（确保窗口有效）
		if ttl < 0 then
			ttl = window
		end
	
    	-- 返回窗口状态：阈值、当前计数、剩余时间
		return {threshold, tonumber(current), ttl}
	`
	ResponsePhaseFixedWindowScript = `
        local key = KEYS[1]
        local threshold = tonumber(ARGV[1])
        local window = tonumber(ARGV[2])
        local added = tonumber(ARGV[3])  -- 需要累加的token数量
        
        local current = tonumber(redis.call('get', key) or "0")
        
        -- 只有当前计数未超过阈值时才执行累加
        if current <= threshold then
            current = redis.call('incrby', key, added)
            -- 第一次设置值时初始化过期时间
            if current == added then
                redis.call('expire', key, window)
            else
                -- 非首次设置时检查过期时间，确保窗口有效性
                local ttl = redis.call('ttl', key)
                if ttl < 0 then
                    redis.call('expire', key, window)
                end
            end
        end
        
        -- 返回当前窗口状态：阈值、当前计数、剩余时间
        return {threshold, current, redis.call('ttl', key)}
    `

	LimitRedisContextKey = "LimitRedisContext"

	CookieHeader = "cookie"

	RateLimitResetHeader = "X-TokenRateLimit-Reset" // 限流重置时间（触发限流时返回）

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

func parseConfig(json gjson.Result, cfg *config.AiTokenRateLimitConfig) error {
	err := config.InitRedisClusterClient(json, cfg)
	if err != nil {
		return err
	}
	err = config.ParseAiTokenRateLimitConfig(json, cfg)
	if err != nil {
		return err
	}
	// Metric settings
	cfg.CounterMetrics = make(map[string]proxywasm.MetricCounter)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, cfg config.AiTokenRateLimitConfig) types.Action {
	ctx.DisableReroute()
	limitKey, count, timeWindow := "", int64(0), int64(0)

	if cfg.GlobalThreshold != nil {
		// 全局限流模式
		limitKey = fmt.Sprintf(AiTokenGlobalRateLimitFormat, cfg.RuleName, cfg.GlobalThreshold.TimeWindow)
		count = cfg.GlobalThreshold.Count
		timeWindow = cfg.GlobalThreshold.TimeWindow
	} else {
		// 规则限流模式
		val, ruleItem, configItem := checkRequestAgainstLimitRule(ctx, cfg.RuleItems)
		if ruleItem == nil || configItem == nil {
			// 没有匹配到限流规则直接返回
			return types.ActionContinue
		}

		limitKey = fmt.Sprintf(AiTokenRateLimitFormat, cfg.RuleName, ruleItem.LimitType, configItem.TimeWindow, ruleItem.Key, val)
		count = configItem.Count
		timeWindow = configItem.TimeWindow
	}

	ctx.SetContext(LimitRedisContextKey, LimitRedisContext{
		key:    limitKey,
		count:  count,
		window: timeWindow,
	})

	// 执行限流逻辑
	keys := []interface{}{limitKey}
	args := []interface{}{count, timeWindow}
	err := cfg.RedisClient.Eval(RequestPhaseFixedWindowScript, 1, keys, args, func(response resp.Value) {
		resultArray := response.Array()
		if len(resultArray) != 3 {
			log.Errorf("redis response parse error, response: %v", response)
			proxywasm.ResumeHttpRequest()
			return
		}

		// 获取限流结果
		threshold, current, ttl := resultArray[0].Integer(), resultArray[1].Integer(), resultArray[2].Integer()
		context := LimitContext{
			count:     threshold,
			remaining: threshold - current,
			reset:     ttl,
		}
		if current > threshold {
			// 触发限流
			ctx.SetUserAttribute("token_ratelimit_status", "limited")
			ctx.WriteUserAttributeToLogWithKey(wrapper.AILogKey)
			rejected(cfg, context)
		} else {
			proxywasm.ResumeHttpRequest()
		}
	})
	if err != nil {
		log.Errorf("redis call failed: %v", err)
		return types.ActionContinue
	}
	return types.HeaderStopAllIterationAndWatermark
}

func onHttpStreamingBody(ctx wrapper.HttpContext, cfg config.AiTokenRateLimitConfig, data []byte, endOfStream bool) []byte {
	if usage := tokenusage.GetTokenUsage(ctx, data); usage.TotalToken > 0 {
		ctx.SetContext(tokenusage.CtxKeyInputToken, usage.InputToken)
		ctx.SetContext(tokenusage.CtxKeyOutputToken, usage.OutputToken)
	}
	if endOfStream {
		if ctx.GetContext(tokenusage.CtxKeyInputToken) == nil || ctx.GetContext(tokenusage.CtxKeyOutputToken) == nil {
			return data
		}
		inputToken := ctx.GetContext(tokenusage.CtxKeyInputToken).(int64)
		outputToken := ctx.GetContext(tokenusage.CtxKeyOutputToken).(int64)
		limitRedisContext, ok := ctx.GetContext(LimitRedisContextKey).(LimitRedisContext)
		if !ok {
			return data
		}
		keys := []interface{}{limitRedisContext.key}
		args := []interface{}{limitRedisContext.count, limitRedisContext.window, inputToken + outputToken}
		err := cfg.RedisClient.Eval(ResponsePhaseFixedWindowScript, 1, keys, args, nil)
		if err != nil {
			log.Errorf("redis call failed: %v", err)
		}
	}
	return data
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
		val, err := proxywasm.GetHttpRequestHeader(util.ConsumerHeader)
		if err != nil {
			return logDebugAndReturnEmpty("failed to get request header %s: %v", util.ConsumerHeader, err)
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

func generateMetricName(route, cluster, model, consumer, metricName string) string {
	return fmt.Sprintf("route.%s.upstream.%s.model.%s.consumer.%s.metric.%s", route, cluster, model, consumer, metricName)
}

func rejected(cfg config.AiTokenRateLimitConfig, context LimitContext) {
	headers := make(map[string][]string)
	headers[RateLimitResetHeader] = []string{strconv.Itoa(context.reset)}
	_ = proxywasm.SendHttpResponseWithDetail(
		cfg.RejectedCode, "ai-token-ratelimit.rejected", util.ReconvertHeaders(headers), []byte(cfg.RejectedMsg), -1)

	route, _ := util.GetRouteName()
	cluster, _ := util.GetClusterName()
	consumer, _ := util.GetConsumer()
	cfg.IncrementCounter(generateMetricName(route, cluster, "none", consumer, TokenRateLimitCount), 1)
}
