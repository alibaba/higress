package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	cache "github.com/go-pkgz/expirable-cache"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"golang.org/x/time/rate"
	"strings"
	"time"
)

// ip+url限流
func main() {
	wrapper.SetCtx(
		// ip+url限流插件
		"ip-rate-limit-plugin",
		wrapper.ParseConfigBy(parseConfig),
		// 为处理请求头，设置自定义函数
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

func parseConfig(json gjson.Result, config *MyConfig, log wrapper.Log) error {
	config.ttlSecond = json.Get("ttlSecond").Int()
	if config.ttlSecond == 0 {
		config.ttlSecond = 60 //默认60秒
	}
	config.burst = int(json.Get("burst").Int())
	if config.burst == 0 {
		config.burst = 3 //默认3个并发
	}
	return nil
}

const (
	// 错误码
	ERR_CODE uint32 = 429
)

// 自定义插件配置
type MyConfig struct {
	ttlSecond int64 // 限流阻塞时间(秒)
	burst     int   //并发数
}

// Create a custom visitor struct which holds the rate limiter for each
// visitor and the last time that the visitor was seen.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// 它是一个map类型，键的类型是string，值的类型是*visitor指针。
var visitors = make(map[string]*visitor)
var tokenBuckets cache.Cache

// Run a background goroutine to remove old entries from the visitors map.
func init() {
	//go cleanupVisitors() //这行会使用调度器
	tokenBuckets, _ = cache.NewCache()
}

func getVisitor(key string, burst int) *rate.Limiter {
	v, exists := visitors[key]
	if !exists {
		limiter := rate.NewLimiter(1, burst)
		// Include the current time when creating a new visitor.
		visitors[key] = &visitor{limiter, time.Now()}
		return limiter
	}

	// Update the last seen time for the visitor.
	v.lastSeen = time.Now()
	return v.limiter
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config MyConfig, log wrapper.Log) types.Action {
	key := md5String(getClientIP() + "_" + ctx.Host() + "_" + ctx.Path())

	err := fmt.Sprintf("this key address (%s) too many requests,locked (%d) seconds.", key, config.ttlSecond)
	_, exist := tokenBuckets.Get(key)
	if exist {
		proxywasm.SendHttpResponse(ERR_CODE, nil, []byte(err), -1) //直接响应结果
		return types.ActionContinue
	}
	limiter := getVisitor(key, config.burst)
	if false == limiter.Allow() {
		ttl := time.Duration(config.ttlSecond) * time.Second
		tokenBuckets.Set(key, "1", ttl)
		proxywasm.SendHttpResponse(ERR_CODE, nil, []byte(err), -1) //直接响应结果
		return types.ActionContinue
	}

	return types.ActionContinue
}

// md5算法
func md5String(value string) string {
	data := []byte(value)                     // 待计算哈希的数据
	hash := md5.Sum(data)                     // 计算 MD5 哈希值
	hashString := hex.EncodeToString(hash[:]) // 将哈希值转换为字符串
	return hashString
}

// 获取ip地址
func getClientIP() string {

	realIp, _ := proxywasm.GetHttpRequestHeader("real-ip")
	// 优先从EO提供的real-ip请求头中获取IP地址

	if realIp != "" {
		return realIp
	}
	// 优先从X-Forwarded-For请求头中获取IP地址
	xForwardedFor, _ := proxywasm.GetHttpRequestHeader("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		// 返回最后一个IP地址，通常是客户端的IP
		if len(ips) > 0 {
			// 返回第一个IP地址，通常是客户端的IP
			return strings.TrimSpace(ips[0])
		}
	}
	return "127.0.0.1"
}
