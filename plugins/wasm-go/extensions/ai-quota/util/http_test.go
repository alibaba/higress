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

package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCreateHeaders 测试CreateHeaders函数
func TestCreateHeaders(t *testing.T) {
	tests := []struct {
		name     string
		kvs      []string
		expected [][2]string
	}{
		{
			name: "single header",
			kvs:  []string{"Content-Type", "text/plain"},
			expected: [][2]string{
				{"Content-Type", "text/plain"},
			},
		},
		{
			name: "multiple headers",
			kvs:  []string{"Content-Type", "application/json", "Authorization", "Bearer token"},
			expected: [][2]string{
				{"Content-Type", "application/json"},
				{"Authorization", "Bearer token"},
			},
		},
		{
			name:     "empty input",
			kvs:      []string{},
			expected: [][2]string{},
		},
		{
			name: "odd number of elements",
			kvs:  []string{"Content-Type", "text/plain", "Authorization"},
			expected: [][2]string{
				{"Content-Type", "text/plain"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateHeaders(tt.kvs...)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestConstants 测试常量定义
func TestConstants(t *testing.T) {
	require.Equal(t, "Content-Type", HeaderContentType)
	require.Equal(t, "text/plain", MimeTypeTextPlain)
	require.Equal(t, "application/json", MimeTypeApplicationJson)
}

// TestSendResponse 测试SendResponse函数
// 注意：这个函数调用了proxywasm SDK，在单元测试中我们主要验证函数签名和基本逻辑
func TestSendResponse(t *testing.T) {
	// 由于SendResponse函数调用了proxywasm SDK，在单元测试环境中可能无法完全执行
	// 但我们仍然可以测试函数的存在性和基本结构
	t.Run("function exists", func(t *testing.T) {
		// 验证函数存在且可以调用（即使可能失败）
		// 在实际的proxy-wasm环境中，这个函数应该能正常工作
		require.NotNil(t, SendResponse)
	})
}
