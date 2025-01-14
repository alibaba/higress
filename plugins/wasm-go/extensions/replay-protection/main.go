package main

import (
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

func main() {
	wrapper.SetCtx(
		"replay-protection",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
	)
}

type ReplayProtectionConfig struct {
	ForceNonce  bool // 是否启用强制 nonce 校验
	NonceTTL    int  // Nonce 的过期时间（单位：秒）
	Redis       RedisConfig
	NonceMinLen int // nonce 最小长度
	NonceMaxLen int // nonce 最大长度
}

type RedisConfig struct {
	client    wrapper.RedisClient
	keyPrefix string
}

func parseConfig(json gjson.Result, config *ReplayProtectionConfig, log wrapper.Log) error {
	redisConfig := json.Get("redis")
	if !redisConfig.Exists() {
		return fmt.Errorf("missing redis config")
	}

	serviceName := redisConfig.Get("serviceName").String()
	if serviceName == "" {
		return fmt.Errorf("redis service name is required")
	}

	servicePort := redisConfig.Get("servicePort").Int()
	if servicePort == 0 {
		servicePort = 6379
	}

	username := redisConfig.Get("username").String()
	password := redisConfig.Get("password").String()
	timeout := redisConfig.Get("timeout").Int()
	if timeout == 0 {
		timeout = 1000
	}

	keyPrefix := redisConfig.Get("keyPrefix").String()
	if keyPrefix == "" {
		keyPrefix = "replay-protection"
	}
	config.Redis.keyPrefix = keyPrefix

	config.ForceNonce = json.Get("force_nonce").Bool()
	config.NonceTTL = int(json.Get("nonce_ttl").Int())
	if config.NonceTTL < 1 || config.NonceTTL > 1800 {
		config.NonceTTL = 900
	}

	config.Redis.client = wrapper.NewRedisClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: servicePort,
	})

	config.NonceMinLen = int(json.Get("nonce_min_length").Int())
	if config.NonceMinLen == 0 {
		config.NonceMinLen = 8
	}

	config.NonceMaxLen = int(json.Get("nonce_max_length").Int())
	if config.NonceMaxLen == 0 {
		config.NonceMaxLen = 128
	}

	return config.Redis.client.Init(username, password, timeout)
}

func validateNonce(nonce string, config *ReplayProtectionConfig) error {
	if len(nonce) < config.NonceMinLen || len(nonce) > config.NonceMaxLen {
		return fmt.Errorf("invalid nonce length: must be between %d and %d",
			config.NonceMinLen, config.NonceMaxLen)
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9+/=-]+$`).MatchString(nonce) {
		return fmt.Errorf("invalid nonce format: must be base64 encoded")
	}

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config ReplayProtectionConfig, log wrapper.Log) types.Action {
	nonce, _ := proxywasm.GetHttpRequestHeader("x-apigw-nonce")
	if config.ForceNonce && nonce == "" {
		// 强制模式下，缺失 nonce 拒绝请求
		log.Warnf("Missing nonce header")
		proxywasm.SendHttpResponse(400, nil, []byte("Missing nonce header"), -1)
		return types.ActionPause
	}

	// 如果没有 nonce，直接放行（非强制模式时）
	if nonce == "" {
		return types.ActionContinue
	}

	if err := validateNonce(nonce, &config); err != nil {
		log.Warnf("Invalid nonce: %v", err)
		proxywasm.SendHttpResponse(429, nil, []byte("Invalid nonce"), -1)
		return types.ActionPause
	}

	redisKey := fmt.Sprintf("%s:%s", config.Redis.keyPrefix, nonce)

	// 校验 nonce 是否已存在
	err := config.Redis.client.Get(redisKey, func(response resp.Value) {
		if response.Error() != nil {
			log.Errorf("Redis error: %v", response.Error())
			proxywasm.ResumeHttpRequest()
		} else if response.String() == "" {
			// nonce 不存在：存储 nonce 并设置过期时间
			err := config.Redis.client.SetEx(redisKey, "1", config.NonceTTL, func(response resp.Value) {
				if response.Error() != nil {
					log.Errorf("Redis error: %v", response.Error())
				}
				proxywasm.ResumeHttpRequest()
			})
			if err != nil {
				log.Errorf("Failed to set nonce in Redis: %v", err)
				proxywasm.ResumeHttpRequest()
			}
		} else {
			// nonce 已存在：拒绝请求
			log.Warnf("Duplicate nonce detected: %s", nonce)
			proxywasm.SendHttpResponse(429, nil, []byte("Request replay detected"), -1)
		}
	})

	if err != nil {
		log.Errorf("Redis connection failed: %v", err)
		proxywasm.SendHttpResponse(500, nil, []byte("Internal Server Error"), -1)
		return types.ActionPause
	}
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config ReplayProtectionConfig, log wrapper.Log) types.Action {
	nonce, _ := proxywasm.GetHttpRequestHeader("x-apigw-nonce")
	if nonce != "" {
		proxywasm.AddHttpResponseHeader("x-apigw-nonce", nonce)
	}
	return types.ActionContinue
}
