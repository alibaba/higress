package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"hmac-auth-apisix/config"
)

const (
	// 认证涉及的请求头
	authorizationHeader = "Authorization"
	dateHeader          = "Date"
	digestHeader        = "Digest"
	// 认证通过后在请求头 consumerHeader 中添加消费者信息
	consumerHeader = "X-Mse-Consumer"

	signaturePrefix       = "Signature "
	errorResponseTemplate = `{"message":"client request can't be validated: %s"}`
)

var (
	// 使用正则表达式匹配 key="value" 格式
	fieldRegex = regexp.MustCompile(`(\w+)="([^"]*)"`)
)

func main() {}

func init() {
	wrapper.SetCtx(
		"hmac-auth-apisix",
		wrapper.ParseOverrideConfig(config.ParseGlobalConfig, config.ParseOverrideRuleConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessRequestBody(onHttpRequestBody),
	)
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, cfg config.HmacAuthConfig) types.Action {
	var (
		// 未配置 allow 列表，表示插件在该 domain/route 未生效
		noAllow            = len(cfg.Allow) == 0
		globalAuthNoSet    = cfg.GlobalAuth == nil
		globalAuthSetTrue  = !globalAuthNoSet && *cfg.GlobalAuth
		globalAuthSetFalse = !globalAuthNoSet && !*cfg.GlobalAuth
		ruleSet            = config.RuleSet
	)

	// 不需要认证而直接放行的情况：
	// - global_auth == false 且 当前 domain/route 未配置该插件
	// - global_auth 未设置 且 有至少一个 domain/route 配置该插件 且 当前 domain/route 未配置该插件
	if globalAuthSetFalse || (globalAuthNoSet && ruleSet) {
		if noAllow {
			log.Info("authorization is not required")
			ctx.DontReadRequestBody()
			return types.ActionContinue
		}
	}
	// 提取 HMAC 字段和消费者信息
	hmacParams, err := retrieveHmacFieldsAndConsumer(cfg)
	if err != nil {
		// 只有在完全无法解析认证信息时才考虑匿名消费者
		if cfg.AnonymousConsumer != "" {
			ctx.DontReadRequestBody()
			setConsumerHeader(cfg.AnonymousConsumer)
			return types.ActionContinue
		}
		return sendUnauthorizedResponse(err.Error())
	}

	if globalAuthSetTrue && !noAllow { // 全局生效，但当前 domain/route 配置了 allow 列表
		if !contains(cfg.Allow, hmacParams.ConsumerName) {
			log.Warnf("consumer %q is not allowed", hmacParams.ConsumerName)
			return sendUnauthorizedResponse("consumer '" + hmacParams.ConsumerName + "' is not allowed")
		}
	} else if globalAuthSetFalse || (globalAuthNoSet && ruleSet) { // 非全局生效
		if !noAllow && !contains(cfg.Allow, hmacParams.ConsumerName) { // 配置了 allow 列表且当前消费者不在 allow 列表中
			log.Warnf("consumer %q is not allowed", hmacParams.ConsumerName)
			return sendUnauthorizedResponse("consumer '" + hmacParams.ConsumerName + "' is not allowed")
		}
	}

	// 校验时间偏差
	if cfg.ClockSkew > 0 {
		if err := validateClockSkew(cfg.ClockSkew); err != nil {
			return sendUnauthorizedResponse(err.Error())
		}
	}

	// 验证算法是否允许
	if !contains(cfg.AllowedAlgorithms, hmacParams.Algorithm) {
		return sendUnauthorizedResponse("Invalid algorithm")
	}

	// 验证签名头
	if len(cfg.SignedHeaders) > 0 {
		if len(hmacParams.Headers) == 0 {
			return sendUnauthorizedResponse("headers missing")
		}

		// 检查所有配置的签名头是否都在签名中
		signedHeadersMap := make(map[string]bool)
		for _, header := range hmacParams.Headers {
			signedHeadersMap[header] = true
		}

		for _, requiredHeader := range cfg.SignedHeaders {
			if !signedHeadersMap[requiredHeader] {
				return sendUnauthorizedResponse("expected header \"" + requiredHeader + "\" missing in signing")
			}
		}
	}

	// 验证 HMAC 签名
	if err := validateSignature(hmacParams, cfg); err != nil {
		return sendUnauthorizedResponse(err.Error())
	}

	// 验证成功，设置消费者信息
	setConsumerHeader(hmacParams.ConsumerName)

	// 如果需要隐藏凭证
	if cfg.HideCredentials {
		proxywasm.RemoveHttpRequestHeader(authorizationHeader)
	}

	// 如果有请求体且需要验证请求体，进入 onHttpRequestBody 方法
	if wrapper.HasRequestBody() && cfg.ValidateRequestBody {
		return types.HeaderStopIteration
	}
	ctx.DontReadRequestBody()
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, cfg config.HmacAuthConfig, body []byte) types.Action {
	if cfg.ValidateRequestBody {
		digestHeaderVal, _ := proxywasm.GetHttpRequestHeader(digestHeader)
		if digestHeaderVal == "" {
			return sendUnauthorizedResponse("Invalid digest")
		}

		// 计算请求体的 SHA-256 摘要
		hash := sha256.Sum256(body)
		encodedDigest := base64.StdEncoding.EncodeToString(hash[:])
		digestCreated := "SHA-256=" + encodedDigest

		// 比较请求头中的 Digest 和服务端计算的摘要
		if digestCreated != digestHeaderVal {
			log.Warnf("Request body digest validation failed. Expected: %s, Got: %s, Body size: %d bytes",
				digestCreated, digestHeaderVal, len(body))
			return sendUnauthorizedResponse("Invalid digest")
		}
	}
	return types.ActionContinue
}

// HmacParams 存储从 Authorization 头解析出的 HMAC 参数
type HmacParams struct {
	KeyId        string
	Algorithm    string
	Signature    string
	Headers      []string
	ConsumerName string
}

// retrieveHmacFieldsAndConsumer 从 Authorization 头中提取 HMAC 参数和消费者信息
func retrieveHmacFieldsAndConsumer(cfg config.HmacAuthConfig) (*HmacParams, error) {
	hmacParams := &HmacParams{}

	// 获取 Authorization 头
	authString, err := proxywasm.GetHttpRequestHeader(authorizationHeader)
	if err != nil {
		if err == types.ErrorStatusNotFound {
			return nil, fmt.Errorf("missing Authorization header")
		}
		return nil, err
	}

	// 检查是否以 "Signature " 开头
	if !strings.HasPrefix(authString, signaturePrefix) {
		return nil, fmt.Errorf("Authorization header does not start with 'Signature '")
	}

	// 使用正则表达式解析字段，跳过 "Signature " 前缀
	matches := fieldRegex.FindAllStringSubmatch(authString[len(signaturePrefix):], -1)

	for _, match := range matches {
		if len(match) == 3 {
			key := match[1]
			value := match[2]

			switch key {
			case "keyId":
				hmacParams.KeyId = value
			case "algorithm":
				hmacParams.Algorithm = value
			case "signature":
				hmacParams.Signature = value
			case "headers":
				// 分割 headers 字段
				if value != "" {
					hmacParams.Headers = strings.Split(value, " ")
				}
			}
		}
	}

	// 验证必要字段
	if hmacParams.KeyId == "" || hmacParams.Signature == "" {
		return nil, fmt.Errorf("keyId or signature missing")
	}

	if hmacParams.Algorithm == "" {
		return nil, fmt.Errorf("algorithm missing")
	}

	// 根据 keyId 查找消费者名称
	consumerName := ""
	found := false
	for _, consumer := range cfg.Consumers {
		if consumer.AccessKey == hmacParams.KeyId {
			consumerName = consumer.Name
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("Invalid keyId")
	}

	hmacParams.ConsumerName = consumerName
	return hmacParams, nil
}

// validateClockSkew 检查时间偏差
func validateClockSkew(clockSkew int) error {
	dateHeaderVal, _ := proxywasm.GetHttpRequestHeader(dateHeader)
	if dateHeaderVal == "" {
		return fmt.Errorf("Date header missing. failed to validate clock skew")
	}

	// 解析 GMT 格式时间
	dateTime, err := time.Parse("Mon, 02 Jan 2006 15:04:05 GMT", dateHeaderVal)
	if err != nil {
		return fmt.Errorf("Invalid GMT format time")
	}

	// 计算时间差
	currentTime := time.Now()
	diff := math.Abs(float64(currentTime.Unix() - dateTime.Unix()))

	// 检查是否超过 clock_skew
	if int(diff) > clockSkew {
		return fmt.Errorf("Clock skew exceeded")
	}

	return nil
}

// validateSignature 验证签名
func validateSignature(hmacParams *HmacParams, cfg config.HmacAuthConfig) error {
	// 根据 keyId 查找对应的 secretKey
	secretKey := ""
	found := false
	for _, consumer := range cfg.Consumers {
		if consumer.AccessKey == hmacParams.KeyId {
			secretKey = consumer.SecretKey
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("Invalid keyId")
	}

	// 生成 HMAC 签名
	signingString, err := generateSigningString(hmacParams)
	if err != nil {
		return fmt.Errorf("Failed to generate signing string")
	}
	expectedSignature, err := generateHmacSignature(secretKey, hmacParams.Algorithm, signingString)
	if err != nil {
		return err
	}

	// 比较签名
	if hmacParams.Signature != expectedSignature {
		log.Warnf("Signature validation failed. Algorithm: %s, Expected: %s, Got: %s, Signing String: %s",
			hmacParams.Algorithm, expectedSignature, hmacParams.Signature, signingString)
		return fmt.Errorf("Invalid signature")
	}

	return nil
}

// generateSigningString 生成签名字符串
func generateSigningString(hmacParams *HmacParams) (string, error) {
	var signingStringItems []string
	signingStringItems = append(signingStringItems, hmacParams.KeyId)

	// 获取请求方法和路径
	requestMethod, err := proxywasm.GetHttpRequestHeader(":method")
	if err != nil {
		requestMethod = "GET"
	}

	requestURI, err := proxywasm.GetHttpRequestHeader(":path")
	if err != nil || requestURI == "" {
		requestURI = "/"
	}

	if len(hmacParams.Headers) > 0 {
		for _, h := range hmacParams.Headers {
			if h == "@request-target" {
				requestTarget := requestMethod + " " + requestURI
				signingStringItems = append(signingStringItems, requestTarget)
			} else {
				headerValue, err := proxywasm.GetHttpRequestHeader(h)
				if err == nil {
					signingStringItems = append(signingStringItems, h+": "+headerValue)
				}
			}
		}
	}

	signingString := strings.Join(signingStringItems, "\n") + "\n"
	return signingString, nil
}

// generateHmacSignature 生成 HMAC 签名
func generateHmacSignature(secretKey, algorithm, message string) (string, error) {
	var mac hash.Hash

	switch algorithm {
	case "hmac-sha1":
		mac = hmac.New(sha1.New, []byte(secretKey))
	case "hmac-sha256":
		mac = hmac.New(sha256.New, []byte(secretKey))
	case "hmac-sha512":
		mac = hmac.New(sha512.New, []byte(secretKey))
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	mac.Write([]byte(message))
	signature := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(signature), nil
}

func sendUnauthorizedResponse(message string) types.Action {
	errorResponse := fmt.Sprintf(errorResponseTemplate, message)
	proxywasm.SendHttpResponse(401, nil, []byte(errorResponse), -1)
	return types.ActionContinue
}

func setConsumerHeader(name string) {
	_ = proxywasm.AddHttpRequestHeader(consumerHeader, name)
}

func contains(arr []string, item string) bool {
	for _, i := range arr {
		if i == item {
			return true
		}
	}
	return false
}
