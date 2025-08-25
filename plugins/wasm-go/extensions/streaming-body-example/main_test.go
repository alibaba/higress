// Copyright (c) 2022 Alibaba Group Holding Ltd.
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

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

func TestOnHttpRequestBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(nil)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)

		// 先调用请求头处理
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "example.com"},
			{":path", "/test"},
			{":method", "POST"},
		})
		require.Equal(t, types.ActionContinue, action)

		// 测试单个请求体块
		t.Run("single chunk", func(t *testing.T) {
			chunk := []byte("Hello, World!")
			action := host.CallOnHttpStreamingRequestBody(chunk, false)
			require.Equal(t, types.ActionContinue, action)

			modifiedChunk := host.GetRequestBody()
			// 验证返回的内容是固定的 "test\n"
			expected := []byte("test\n")
			require.Equal(t, expected, modifiedChunk)
		})

		// 测试多个请求体块
		t.Run("multiple chunks", func(t *testing.T) {
			chunk1 := []byte("First chunk")
			chunk2 := []byte("Second chunk")
			chunk3 := []byte("Third chunk")

			// 处理第一个块（不是最后一个）
			action := host.CallOnHttpStreamingRequestBody(chunk1, false)
			require.Equal(t, types.ActionContinue, action)

			modifiedChunk1 := host.GetRequestBody()
			require.Equal(t, []byte("test\n"), modifiedChunk1)

			// 处理第二个块（不是最后一个）
			action = host.CallOnHttpStreamingRequestBody(chunk2, false)
			require.Equal(t, types.ActionContinue, action)

			modifiedChunk2 := host.GetRequestBody()
			require.Equal(t, []byte("test\n"), modifiedChunk2)

			// 处理最后一个块
			action = host.CallOnHttpStreamingRequestBody(chunk3, true)
			require.Equal(t, types.ActionContinue, action)

			modifiedChunk3 := host.GetRequestBody()
			require.Equal(t, []byte("test\n"), modifiedChunk3)
		})

		// 测试空请求体
		t.Run("empty chunk", func(t *testing.T) {
			emptyChunk := []byte("")
			action := host.CallOnHttpStreamingRequestBody(emptyChunk, true)
			require.Equal(t, types.ActionContinue, action)

			modifiedChunk := host.GetRequestBody()
			// 即使输入为空，也应该返回固定的 "test\n"
			expected := []byte("test\n")
			require.Equal(t, expected, modifiedChunk)
		})

		// 测试大请求体块
		t.Run("large chunk", func(t *testing.T) {
			largeChunk := make([]byte, 1000)
			for i := range largeChunk {
				largeChunk[i] = byte(i % 256)
			}

			action := host.CallOnHttpStreamingRequestBody(largeChunk, false)
			require.Equal(t, types.ActionContinue, action)
			modifiedChunk := host.GetRequestBody()

			// 无论输入多大，都应该返回固定的 "test\n"
			expected := []byte("test\n")
			require.Equal(t, expected, modifiedChunk)
		})

		host.CompleteHttp()
	})
}

func TestOnHttpResponseBody(t *testing.T) {
	test.RunTest(t, func(t *testing.T) {
		host, status := test.NewTestHost(nil)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)

		// 先调用请求头处理
		action := host.CallOnHttpRequestHeaders([][2]string{
			{":authority", "example.com"},
			{":path", "/test"},
			{":method", "GET"},
		})
		require.Equal(t, types.ActionContinue, action)

		// 再调用响应头处理
		action = host.CallOnHttpResponseHeaders([][2]string{
			{":status", "200"},
			{"content-type", "text/plain"},
		})
		require.Equal(t, types.ActionContinue, action)

		// 测试单个响应体块
		t.Run("single chunk", func(t *testing.T) {
			chunk := []byte("Original response content")
			action := host.CallOnHttpStreamingResponseBody(chunk, false)
			require.Equal(t, types.ActionContinue, action)
			modifiedChunk := host.GetResponseBody()

			// 验证返回的内容是固定的 "test\n"
			expected := []byte("test\n")
			require.Equal(t, expected, modifiedChunk)
		})

		// 测试多个响应体块
		t.Run("multiple chunks", func(t *testing.T) {
			chunk1 := []byte("Response chunk 1")
			chunk2 := []byte("Response chunk 2")
			chunk3 := []byte("Response chunk 3")

			// 处理第一个块（不是最后一个）
			action := host.CallOnHttpStreamingResponseBody(chunk1, false)
			require.Equal(t, types.ActionContinue, action)
			modifiedChunk1 := host.GetResponseBody()
			require.Equal(t, []byte("test\n"), modifiedChunk1)

			// 处理第二个块（不是最后一个）
			action = host.CallOnHttpStreamingResponseBody(chunk2, false)
			require.Equal(t, types.ActionContinue, action)
			modifiedChunk2 := host.GetResponseBody()
			require.Equal(t, []byte("test\n"), modifiedChunk2)

			// 处理最后一个块
			action = host.CallOnHttpStreamingResponseBody(chunk3, true)
			require.Equal(t, types.ActionContinue, action)
			modifiedChunk3 := host.GetResponseBody()
			require.Equal(t, []byte("test\n"), modifiedChunk3)
		})

		// 测试空响应体
		t.Run("empty chunk", func(t *testing.T) {
			emptyChunk := []byte("")
			action := host.CallOnHttpStreamingResponseBody(emptyChunk, true)
			require.Equal(t, types.ActionContinue, action)
			modifiedChunk := host.GetResponseBody()

			// 即使输入为空，也应该返回固定的 "test\n"
			expected := []byte("test\n")
			require.Equal(t, expected, modifiedChunk)
		})

		// 测试大响应体块
		t.Run("large chunk", func(t *testing.T) {
			largeChunk := make([]byte, 2000)
			for i := range largeChunk {
				largeChunk[i] = byte(i % 256)
			}

			action := host.CallOnHttpStreamingResponseBody(largeChunk, false)
			require.Equal(t, types.ActionContinue, action)
			modifiedChunk := host.GetResponseBody()

			// 无论输入多大，都应该返回固定的 "test\n"
			expected := []byte("test\n")
			require.Equal(t, expected, modifiedChunk)
		})

		host.CompleteHttp()
	})
}
