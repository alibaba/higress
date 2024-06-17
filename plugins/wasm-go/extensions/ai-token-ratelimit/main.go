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

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

func main() {
	wrapper.SetCtx(
		"ai-token-ratelimit",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onHttpStreamingBody),
	)
}

const (
	LimitRedisContextKey          string = "LimitRedisContext"
	ClusterRateLimitFormat        string = "higress-token-ratelimit:%s:%s"
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
	RateLimitLimitHeader     string = "X-RateLimit-Limit"     // 限制的总请求数
	RateLimitRemainingHeader string = "X-RateLimit-Remaining" // 剩余还可以发送的请求数
	RateLimitResetHeader     string = "X-RateLimit-Reset"     // 限流重置时间（触发限流时返回）
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

func parseConfig(json gjson.Result, config *ClusterKeyRateLimitConfig, log wrapper.Log) error {
	err := initRedisClusterClient(json, config)
	if err != nil {
		return err
	}
	err = parseClusterKeyRateLimitConfig(json, config, log)
	if err != nil {
		return err
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config ClusterKeyRateLimitConfig, log wrapper.Log) types.Action {
	// 判断是否命中限流规则
	key, limitItem := hitRateLimitRule(ctx, config, log)
	if limitItem == nil {
		return types.ActionContinue
	}

	// 构建redis限流key和参数
	limitKey := fmt.Sprintf(ClusterRateLimitFormat, config.ruleName, key)
	keys := []interface{}{limitKey}
	args := []interface{}{limitItem.count, limitItem.timeWindow}

	limitRedisContext := LimitRedisContext{
		key:    limitKey,
		count:  limitItem.count,
		window: limitItem.timeWindow,
	}
	ctx.SetContext(LimitRedisContextKey, limitRedisContext)

	// 执行限流逻辑
	err := config.redisClient.Eval(RequestPhaseFixedWindowScript, 1, keys, args, func(response resp.Value) {
		defer proxywasm.ResumeHttpRequest()
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
		}
	})
	if err != nil {
		log.Errorf("redis call failed: %v", err)
		return types.ActionContinue
	}
	return types.ActionPause
}

func onHttpStreamingBody(ctx wrapper.HttpContext, config ClusterKeyRateLimitConfig, data []byte, endOfStream bool, log wrapper.Log) []byte {
	if !endOfStream {
		return data
	}
	inputTokenStr, err := proxywasm.GetProperty([]string{"filter_state", "wasm.input_token"})
	if err != nil {
		return data
	}
	outputTokenStr, err := proxywasm.GetProperty([]string{"filter_state", "wasm.output_token"})
	if err != nil {
		return data
	}
	inputToken, err := strconv.Atoi(string(inputTokenStr))
	if err != nil {
		return data
	}
	outputToken, err := strconv.Atoi(string(outputTokenStr))
	if err != nil {
		return data
	}
	limitRedisContext, ok := ctx.GetContext(LimitRedisContextKey).(LimitRedisContext)
	if !ok {
		return data
	}
	keys := []interface{}{limitRedisContext.key}
	args := []interface{}{limitRedisContext.count, limitRedisContext.window, inputToken + outputToken}

	err = config.redisClient.Eval(ResponsePhaseFixedWindowScript, 1, keys, args, func(response resp.Value) {
		if response.Error() != nil {
			log.Errorf("call Eval error: %v", response.Error())
		}
		proxywasm.ResumeHttpResponse()
	})
	if err != nil {
		log.Errorf("redis call failed: %v", err)
		return data
	} else {
		return data
	}
}

func hitRateLimitRule(ctx wrapper.HttpContext, config ClusterKeyRateLimitConfig, log wrapper.Log) (string, *LimitItem) {
	switch config.limitType {
	case limitByHeaderType:
		headerVal, err := proxywasm.GetHttpRequestHeader(config.limitByHeader)
		if err != nil {
			log.Debugf("failed to get request header %s: %v", config.limitByHeader, err)
			return "", nil
		}
		return headerVal, findMatchingItem(config.limitItems, headerVal)
	case limitByParamType:
		parse, _ := url.Parse(ctx.Path())
		query, _ := url.ParseQuery(parse.RawQuery)
		val, ok := query[config.limitByParam]
		if !ok {
			log.Debugf("request param %s is empty", config.limitByParam)
			return "", nil
		} else {
			return val[0], findMatchingItem(config.limitItems, val[0])
		}
	case limitByPerIpType:
		realIp, err := getDownStreamIp(config)
		if err != nil {
			log.Warnf("failed to get down stream ip: %v", err)
			return "", nil
		}
		for _, item := range config.limitItems {
			if _, found, _ := item.ipNet.Get(realIp); !found {
				continue
			}
			return realIp.String(), &item
		}
	}
	return "", nil
}

func findMatchingItem(items []LimitItem, key string) *LimitItem {
	for _, item := range items {
		if item.key == key {
			return &item
		}
	}
	return nil
}

func getDownStreamIp(config ClusterKeyRateLimitConfig) (net.IP, error) {
	var (
		realIpStr string
		err       error
	)
	if config.limitByPerIp.sourceType == HeaderSourceType {
		realIpStr, err = proxywasm.GetHttpRequestHeader(config.limitByPerIp.headerName)
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
	_ = proxywasm.SendHttpResponse(
		config.rejectedCode, reconvertHeaders(headers), []byte(config.rejectedMsg), -1)
}
