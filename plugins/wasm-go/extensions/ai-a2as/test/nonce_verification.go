package test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 测试配置：启用Nonce验证
var nonceVerificationConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"protocol": "openai",
		"authenticatedPrompts": map[string]interface{}{
			"enabled":                 true,
			"mode":                    "simple",
			"signatureHeader":         "Signature",
			"sharedSecret":            "test-secret-key-12345",
			"algorithm":               "hmac-sha256",
			"allowUnsigned":           false,
			"enableNonceVerification": true,
			"nonceHeader":             "X-A2AS-Nonce",
			"nonceExpiry":             300,
			"nonceMinLength":          16,
		},
	})
	return data
}()

// 测试配置：自定义Nonce头
var customNonceHeaderConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"protocol": "openai",
		"authenticatedPrompts": map[string]interface{}{
			"enabled":                 true,
			"mode":                    "simple",
			"signatureHeader":         "Signature",
			"sharedSecret":            "test-secret-key-12345",
			"enableNonceVerification": true,
			"nonceHeader":             "X-Custom-Nonce",
			"nonceExpiry":             300,
			"nonceMinLength":          20,
		},
	})
	return data
}()

// 测试配置：短过期时间
var shortExpiryConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"protocol": "openai",
		"authenticatedPrompts": map[string]interface{}{
			"enabled":                 true,
			"mode":                    "simple",
			"signatureHeader":         "Signature",
			"sharedSecret":            "test-secret-key",
			"enableNonceVerification": true,
			"nonceHeader":             "X-A2AS-Nonce",
			"nonceExpiry":             1, // 1秒过期
			"nonceMinLength":          16,
		},
	})
	return data
}()

// 测试配置：禁用Nonce验证
var nonceDisabledConfig = func() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"protocol": "openai",
		"authenticatedPrompts": map[string]interface{}{
			"enabled":                 true,
			"mode":                    "simple",
			"signatureHeader":         "Signature",
			"sharedSecret":            "test-secret-key",
			"enableNonceVerification": false, // 禁用
			"nonceHeader":             "X-A2AS-Nonce",
		},
	})
	return data
}()

// computeSimpleSignature 计算简单HMAC签名
func computeSimpleSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func RunNonceVerificationParseConfigTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试1：基本配置解析
		t.Run("parse nonce verification config", func(t *testing.T) {
			host, status := test.NewTestHost(nonceVerificationConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			reqBody := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
			signature := computeSimpleSignature("test-secret-key-12345", reqBody)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-A2AS-Nonce", "valid-nonce-0001234567890"},
				{"Signature", signature},
			})

			action := host.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionContinue, action)
		})

		// 测试2：自定义Nonce头配置
		t.Run("custom nonce header", func(t *testing.T) {
			host, status := test.NewTestHost(customNonceHeaderConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			reqBody := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
			signature := computeSimpleSignature("test-secret-key-12345", reqBody)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-Custom-Nonce", "custom-nonce-123456789012345678"},
				{"Signature", signature},
			})

			action := host.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionContinue, action)
		})

		// 测试3：禁用Nonce验证
		t.Run("nonce verification disabled", func(t *testing.T) {
			host, status := test.NewTestHost(nonceDisabledConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			reqBody := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
			signature := computeSimpleSignature("test-secret-key", reqBody)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				// 没有Nonce头也应该成功
				{"Signature", signature},
			})

			action := host.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionContinue, action)
		})
	})
}

func RunNonceVerificationTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试1：有效的Nonce验证成功
		t.Run("valid nonce passes verification", func(t *testing.T) {
			host, status := test.NewTestHost(nonceVerificationConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			nonce := fmt.Sprintf("nonce-%d-1234567890", time.Now().UnixNano())
			reqBody := []byte(`{"messages":[{"role":"user","content":"test message"}]}`)
			signature := computeSimpleSignature("test-secret-key-12345", reqBody)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-A2AS-Nonce", nonce},
				{"Signature", signature},
			})

			action := host.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionContinue, action, "有效的Nonce应该通过验证")
		})

		// 测试2：重放攻击检测
		t.Run("replay attack detection", func(t *testing.T) {
			nonce := fmt.Sprintf("replay-nonce-%d", time.Now().UnixNano())
			reqBody := []byte(`{"messages":[{"role":"user","content":"first request"}]}`)
			signature := computeSimpleSignature("test-secret-key-12345", reqBody)

			// 第一次请求
			host1, status := test.NewTestHost(nonceVerificationConfig)
			defer host1.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			_ = host1.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-A2AS-Nonce", nonce},
				{"Signature", signature},
			})

			action := host1.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionContinue, action, "第一次请求应该成功")

			// 第二次使用相同的Nonce（重放攻击）
			// 注意：由于每个测试都有独立的host，nonceStore不共享
			// 这个测试验证的是同一个host实例内的重放检测
			_ = host1.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-A2AS-Nonce", nonce},
				{"Signature", signature},
			})

			action2 := host1.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionPause, action2, "重放攻击应该被阻止")
		})

		// 测试3：Nonce太短
		t.Run("nonce too short", func(t *testing.T) {
			host, status := test.NewTestHost(nonceVerificationConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			shortNonce := "short-nonce" // 少于16字符
			reqBody := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
			signature := computeSimpleSignature("test-secret-key-12345", reqBody)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-A2AS-Nonce", shortNonce},
				{"Signature", signature},
			})

			action := host.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionPause, action, "太短的Nonce应该被拒绝")
		})

		// 测试4：缺少Nonce头
		t.Run("missing nonce header", func(t *testing.T) {
			host, status := test.NewTestHost(nonceVerificationConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			reqBody := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
			signature := computeSimpleSignature("test-secret-key-12345", reqBody)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				// 缺少X-A2AS-Nonce头
				{"Signature", signature},
			})

			action := host.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionPause, action, "缺少Nonce应该被拒绝")
		})

		// 测试5：多个请求使用不同的Nonce
		t.Run("multiple requests with different nonces", func(t *testing.T) {
			host, status := test.NewTestHost(nonceVerificationConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			for i := 0; i < 5; i++ {
				nonce := fmt.Sprintf("concurrent-nonce-%d-%d", time.Now().UnixNano(), i)
				reqBody := []byte(fmt.Sprintf(`{"messages":[{"role":"user","content":"request %d"}]}`, i))
				signature := computeSimpleSignature("test-secret-key-12345", reqBody)

				_ = host.CallOnHttpRequestHeaders([][2]string{
					{":authority", "example.com"},
					{":path", "/v1/chat/completions"},
					{":method", "POST"},
					{"Content-Type", "application/json"},
					{"X-A2AS-Nonce", nonce},
					{"Signature", signature},
				})

				action := host.CallOnHttpRequestBody(reqBody)
				require.Equal(t, types.ActionContinue, action, fmt.Sprintf("第%d个请求应该成功", i+1))
			}
		})

		// 测试6：自定义最小长度验证 - 满足要求
		t.Run("custom nonce min length - valid", func(t *testing.T) {
			host, status := test.NewTestHost(customNonceHeaderConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			nonce := "nonce-12345678901234567890" // 27字符，满足20的要求
			reqBody := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
			signature := computeSimpleSignature("test-secret-key-12345", reqBody)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-Custom-Nonce", nonce},
				{"Signature", signature},
			})

			action := host.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionContinue, action, "满足自定义最小长度应该成功")
		})

		// 测试6b：自定义最小长度验证 - 不满足要求
		t.Run("custom nonce min length - invalid", func(t *testing.T) {
			host, status := test.NewTestHost(customNonceHeaderConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			shortNonce := "nonce-1234567890" // 17字符，不满足20的要求
			reqBody := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
			signature := computeSimpleSignature("test-secret-key-12345", reqBody)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-Custom-Nonce", shortNonce},
				{"Signature", signature},
			})

			action := host.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionPause, action, "不满足最小长度应该被拒绝")
		})

		// 测试7：空Nonce值
		t.Run("empty nonce value", func(t *testing.T) {
			host, status := test.NewTestHost(nonceVerificationConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			reqBody := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
			signature := computeSimpleSignature("test-secret-key-12345", reqBody)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-A2AS-Nonce", ""}, // 空值
				{"Signature", signature},
			})

			action := host.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionPause, action, "空Nonce应该被拒绝")
		})

		// 测试8：超长Nonce（应该被接受）
		t.Run("very long nonce accepted", func(t *testing.T) {
			host, status := test.NewTestHost(nonceVerificationConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			// 生成一个很长的Nonce
			longNonce := ""
			for i := 0; i < 100; i++ {
				longNonce += "a"
			}

			reqBody := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
			signature := computeSimpleSignature("test-secret-key-12345", reqBody)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-A2AS-Nonce", longNonce},
				{"Signature", signature},
			})

			action := host.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionContinue, action, "超长Nonce应该被接受")
		})

		// 测试9：特殊字符Nonce
		t.Run("nonce with special characters", func(t *testing.T) {
			host, status := test.NewTestHost(nonceVerificationConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			specialNonce := "nonce-!@#$%^&*()_+-=[]{}|;:,.<>?/~`"
			reqBody := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
			signature := computeSimpleSignature("test-secret-key-12345", reqBody)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-A2AS-Nonce", specialNonce},
				{"Signature", signature},
			})

			action := host.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionContinue, action, "包含特殊字符的Nonce应该被接受")
		})

		// 测试10：UUID格式的Nonce
		t.Run("uuid format nonce", func(t *testing.T) {
			host, status := test.NewTestHost(nonceVerificationConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			uuidNonce := "550e8400-e29b-41d4-a716-446655440000"
			reqBody := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
			signature := computeSimpleSignature("test-secret-key-12345", reqBody)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-A2AS-Nonce", uuidNonce},
				{"Signature", signature},
			})

			action := host.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionContinue, action, "UUID格式的Nonce应该被接受")
		})
	})
}

// RunNonceExpiryTests 测试Nonce过期相关功能
func RunNonceExpiryTests(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		// 测试：Nonce过期后可以重用
		t.Run("expired nonce can be reused", func(t *testing.T) {
			host, status := test.NewTestHost(shortExpiryConfig)
			defer host.Reset()
			require.Equal(t, types.OnPluginStartStatusOK, status)

			nonce := fmt.Sprintf("expiry-test-%d", time.Now().UnixNano())
			reqBody := []byte(`{"messages":[{"role":"user","content":"first"}]}`)
			signature := computeSimpleSignature("test-secret-key", reqBody)

			// 第一次请求
			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-A2AS-Nonce", nonce},
				{"Signature", signature},
			})

			action := host.CallOnHttpRequestBody(reqBody)
			require.Equal(t, types.ActionContinue, action, "第一次请求应该成功")

			// 等待过期（1秒 + 一点缓冲）
			time.Sleep(2 * time.Second)

			// 过期后应该可以重用
			reqBody2 := []byte(`{"messages":[{"role":"user","content":"second"}]}`)
			signature2 := computeSimpleSignature("test-secret-key", reqBody2)

			_ = host.CallOnHttpRequestHeaders([][2]string{
				{":authority", "example.com"},
				{":path", "/v1/chat/completions"},
				{":method", "POST"},
				{"Content-Type", "application/json"},
				{"X-A2AS-Nonce", nonce},
				{"Signature", signature2},
			})

			action2 := host.CallOnHttpRequestBody(reqBody2)
			require.Equal(t, types.ActionContinue, action2, "过期后应该可以重用Nonce")
		})
	})
}
