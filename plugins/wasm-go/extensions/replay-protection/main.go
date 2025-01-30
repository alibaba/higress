package main

import (
	"fmt"
	"regexp"

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
	)
}

type ReplayProtectionConfig struct {
	ForceNonce     bool // Whether to enforce nonce verification
	NonceTTL       int  // Expiration time of the nonce (in seconds)
	Redis          RedisConfig
	NonceMinLen    int    // Minimum length of the nonce
	NonceMaxLen    int    // Maximum length of the nonce
	NonceHeader    string // Name of the nonce heade
	ValidateBase64 bool   // Whether to validate base64 encoding format
	RejectCode     uint32 // Response code
	RejectMsg      string // Response body
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

	config.NonceHeader = json.Get("nonce_header").String()
	if config.NonceHeader == "" {
		config.NonceHeader = "X-Higress-Nonce"
	}

	config.ValidateBase64 = json.Get("validate_base64").Bool()

	config.RejectCode = uint32(json.Get("reject_code").Int())
	if config.RejectCode == 0 {
		config.RejectCode = 429
	}

	config.RejectMsg = json.Get("reject_msg").String()
	if config.RejectMsg == "" {
		config.RejectMsg = "Duplicate nonce"
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

	err := config.Redis.client.Init(username, password, timeout)
	if err != nil {
		log.Errorf("Failed to initialize Redis client: %v", err)
		return fmt.Errorf("Redis initialization error: %w", err)
	}
	return nil
}

func validateNonce(nonce string, config *ReplayProtectionConfig) error {
	if len(nonce) < config.NonceMinLen || len(nonce) > config.NonceMaxLen {
		return fmt.Errorf("invalid nonce length: must be between %d and %d",
			config.NonceMinLen, config.NonceMaxLen)
	}
	if config.ValidateBase64 {
		if !regexp.MustCompile(`^[a-zA-Z0-9+/=-]+$`).MatchString(nonce) {
			return fmt.Errorf("invalid nonce format: must be base64 encoded")
		}
	}

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config ReplayProtectionConfig, log wrapper.Log) types.Action {
	nonce, _ := proxywasm.GetHttpRequestHeader(config.NonceHeader)
	if config.ForceNonce && nonce == "" {
		// In force mode, reject the request if the nonce header is missing
		log.Warnf("Missing nonce header")
		proxywasm.SendHttpResponse(400, nil, []byte("Missing nonce header"), -1)
		return types.ActionPause
	}

	// If there is no nonce, pass through directly (when not in force mode)
	if nonce == "" {
		return types.ActionContinue
	}

	if err := validateNonce(nonce, &config); err != nil {
		log.Warnf("Invalid nonce: %v", err)
		proxywasm.SendHttpResponse(400, nil, []byte("Invalid nonce"), -1)
		return types.ActionPause
	}

	redisKey := fmt.Sprintf("%s:%s", config.Redis.keyPrefix, nonce)

	// Check if the nonce already exists
	err := config.Redis.client.SetNX(redisKey, "1", config.NonceTTL, func(response resp.Value) {
		if response.Error() != nil {
			log.Errorf("Redis error: %v", response.Error())
			proxywasm.SendHttpResponse(500, nil, []byte("Internal Server Error"), -1)
			return
		} else if response.Integer() == 1 {
			// SETNX successful, pass the request
			proxywasm.ResumeHttpRequest()
			return
		} else {
			// Nonce already exists, reject the request
			log.Warnf("Duplicate nonce detected: %s", nonce)
			proxywasm.SendHttpResponse(
				config.RejectCode,
				nil,
				[]byte(fmt.Sprintf("%s: %s", config.RejectMsg, nonce)),
				-1,
			)
		}
	})

	if err != nil {
		log.Errorf("Redis connection failed: %v", err)
		proxywasm.SendHttpResponse(500, nil, []byte("Internal Server Error"), -1)
		return types.ActionPause
	}
	return types.ActionContinue
}
