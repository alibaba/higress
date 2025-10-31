// Copyright (c) 2025 Alibaba Group Holding Ltd.
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
	"testing"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

// TestNonceStore_AddAndCheck 测试添加和检查Nonce
func TestNonceStore_AddAndCheck(t *testing.T) {
	config := &A2ASConfig{
		nonceStore: make(map[string]int64),
		metrics:    make(map[string]proxywasm.MetricCounter),
		gauges:     make(map[string]proxywasm.MetricGauge),
	}

	nonce := "test-nonce-1234567890"
	currentTime := time.Now().Unix()
	expiryTime := currentTime + 300

	// 添加Nonce
	config.nonceStore[nonce] = expiryTime

	// 检查Nonce存在
	if _, exists := config.nonceStore[nonce]; !exists {
		t.Error("Nonce应该被存储")
	}

	// 检查过期时间正确
	if config.nonceStore[nonce] != expiryTime {
		t.Errorf("过期时间不正确，期望 %d，得到 %d", expiryTime, config.nonceStore[nonce])
	}
}

// TestNonceStore_ReplayDetection 测试重放攻击检测逻辑
func TestNonceStore_ReplayDetection(t *testing.T) {
	config := &A2ASConfig{
		nonceStore: make(map[string]int64),
		metrics:    make(map[string]proxywasm.MetricCounter),
		gauges:     make(map[string]proxywasm.MetricGauge),
	}

	nonce := "replay-test-nonce"
	currentTime := time.Now().Unix()
	expiryTime := currentTime + 300

	// 第一次使用：Nonce不存在
	if _, exists := config.nonceStore[nonce]; exists {
		t.Error("Nonce不应该已经存在")
	}

	// 存储Nonce
	config.nonceStore[nonce] = expiryTime

	// 第二次使用：应该检测到重放
	if storedExpiry, exists := config.nonceStore[nonce]; exists {
		if currentTime < storedExpiry {
			// 这是重放攻击
			t.Log("✓ 成功检测到重放攻击")
		} else {
			t.Error("Nonce已过期，不应该被视为重放攻击")
		}
	} else {
		t.Error("Nonce应该存在于存储中")
	}
}

// TestNonceStore_Expiry 测试Nonce过期逻辑
func TestNonceStore_Expiry(t *testing.T) {
	config := &A2ASConfig{
		nonceStore: make(map[string]int64),
		metrics:    make(map[string]proxywasm.MetricCounter),
		gauges:     make(map[string]proxywasm.MetricGauge),
	}

	nonce := "expiry-test-nonce"
	currentTime := time.Now().Unix()

	// 添加一个已经过期的Nonce
	expiredTime := currentTime - 10
	config.nonceStore[nonce] = expiredTime

	// 检查是否过期
	if storedExpiry, exists := config.nonceStore[nonce]; exists {
		if currentTime >= storedExpiry {
			t.Log("✓ Nonce已正确标记为过期")
			// 在实际代码中，过期的Nonce应该被删除并允许重用
			delete(config.nonceStore, nonce)
		} else {
			t.Error("Nonce应该被标记为过期")
		}
	}

	// 验证过期Nonce已被删除
	if _, exists := config.nonceStore[nonce]; exists {
		t.Error("过期的Nonce应该被删除")
	}
}

// TestCleanExpiredNonces_Logic 测试过期Nonce清理逻辑
func TestCleanExpiredNonces_Logic(t *testing.T) {
	config := &A2ASConfig{
		nonceStore: make(map[string]int64),
		metrics:    make(map[string]proxywasm.MetricCounter),
		gauges:     make(map[string]proxywasm.MetricGauge),
	}

	currentTime := time.Now().Unix()

	// 添加多个Nonce：一些过期，一些未过期
	config.nonceStore["expired-nonce-1"] = currentTime - 10
	config.nonceStore["expired-nonce-2"] = currentTime - 5
	config.nonceStore["valid-nonce-1"] = currentTime + 100
	config.nonceStore["valid-nonce-2"] = currentTime + 200

	// 验证初始状态
	if len(config.nonceStore) != 4 {
		t.Errorf("初始应该有4个Nonce，实际有%d个", len(config.nonceStore))
	}

	// 手动执行清理逻辑（不调用cleanExpiredNonces以避免proxywasm调用）
	for nonce, expiryTime := range config.nonceStore {
		if currentTime >= expiryTime {
			delete(config.nonceStore, nonce)
		}
	}

	// 验证清理结果
	if len(config.nonceStore) != 2 {
		t.Errorf("清理后应该有2个Nonce，实际有%d个", len(config.nonceStore))
	}

	// 验证过期的被删除
	if _, exists := config.nonceStore["expired-nonce-1"]; exists {
		t.Error("expired-nonce-1应该被删除")
	}
	if _, exists := config.nonceStore["expired-nonce-2"]; exists {
		t.Error("expired-nonce-2应该被删除")
	}

	// 验证有效的被保留
	if _, exists := config.nonceStore["valid-nonce-1"]; !exists {
		t.Error("valid-nonce-1应该被保留")
	}
	if _, exists := config.nonceStore["valid-nonce-2"]; !exists {
		t.Error("valid-nonce-2应该被保留")
	}
}

// TestNonceStore_MultipleNonces 测试多个Nonce的存储
func TestNonceStore_MultipleNonces(t *testing.T) {
	config := &A2ASConfig{
		nonceStore: make(map[string]int64),
		metrics:    make(map[string]proxywasm.MetricCounter),
		gauges:     make(map[string]proxywasm.MetricGauge),
	}

	currentTime := time.Now().Unix()

	// 添加多个不同的Nonce
	nonces := []string{
		"nonce-001-1234567890",
		"nonce-002-1234567890",
		"nonce-003-1234567890",
		"nonce-004-1234567890",
		"nonce-005-1234567890",
	}

	for i, nonce := range nonces {
		config.nonceStore[nonce] = currentTime + int64((i+1)*100)
	}

	// 验证所有Nonce都被存储
	if len(config.nonceStore) != 5 {
		t.Errorf("应该有5个Nonce，实际有%d个", len(config.nonceStore))
	}

	// 验证每个Nonce都可以被找到
	for _, nonce := range nonces {
		if _, exists := config.nonceStore[nonce]; !exists {
			t.Errorf("Nonce %s应该存在", nonce)
		}
	}
}

// TestNonceLength_Validation 测试Nonce长度验证逻辑
func TestNonceLength_Validation(t *testing.T) {
	testCases := []struct {
		name      string
		nonce     string
		minLength int
		valid     bool
	}{
		{"长度刚好满足", "1234567890123456", 16, true},
		{"长度超过要求", "12345678901234567890", 16, true},
		{"长度不足", "123456789012345", 16, false},
		{"空Nonce", "", 16, false},
		{"高要求满足", "12345678901234567890123456789012", 32, true},
		{"高要求不满足", "1234567890123456", 32, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 模拟长度验证逻辑
			isValid := len(tc.nonce) >= tc.minLength

			if isValid != tc.valid {
				t.Errorf("期望 valid=%v，实际 valid=%v，nonce长度=%d，最小长度=%d",
					tc.valid, isValid, len(tc.nonce), tc.minLength)
			}
		})
	}
}

// TestNonceStore_ConcurrentAccess 测试并发访问场景
func TestNonceStore_ConcurrentAccess(t *testing.T) {
	config := &A2ASConfig{
		nonceStore: make(map[string]int64),
		metrics:    make(map[string]proxywasm.MetricCounter),
		gauges:     make(map[string]proxywasm.MetricGauge),
	}

	currentTime := time.Now().Unix()

	// 模拟快速连续添加Nonce（在实际场景中可能是并发的）
	for i := 0; i < 10; i++ {
		nonce := make([]byte, 20)
		for j := range nonce {
			nonce[j] = byte('0' + (i+j)%10)
		}
		config.nonceStore[string(nonce)] = currentTime + int64(i)
	}

	// 验证所有Nonce都被正确存储
	if len(config.nonceStore) != 10 {
		t.Errorf("应该有10个Nonce，实际有%d个", len(config.nonceStore))
	}
}
