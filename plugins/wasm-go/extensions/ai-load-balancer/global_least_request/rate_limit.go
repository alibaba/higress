package global_least_request

import (
	"fmt"

	"github.com/higress-group/wasm-go/pkg/wrapper"
)

func (lb GlobalLeastRequestLoadBalancer) checkRateLimit(hostSelected string, currentCount int64, ctx wrapper.HttpContext, routeName string, clusterName string) bool {
	// 如果没有配置最大请求数，直接通过
	if lb.maxRequestCount <= 0 {
		return true
	}

	// 如果当前请求数大于最大请求数，则限流
	// 注意：Lua脚本已经加了1，所以这里比较的是加1后的值
	if currentCount > lb.maxRequestCount {
		// 恢复 Redis 计数
		lb.redisClient.HIncrBy(fmt.Sprintf(RedisKeyFormat, routeName, clusterName), hostSelected, -1, nil)
		return false
	}

	return true
}
