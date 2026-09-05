package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	wasmtest "github.com/higress-group/wasm-go/pkg/test"
	"github.com/stretchr/testify/require"
)

// 请求体流式转换的集成测试：通过宿主模拟器按块喂请求体，验证
// 提交点前 Pause / 提交点后 Continue、回落到全量路径、提交点后失败、透传等控制流。
// 转换结果与官方实现的等价性由 provider 包里的差分测试负责，这里只验集成层的行为。

var streamingClaudeConfig = json.RawMessage(`{"provider":{"type":"claude","apiTokens":["sk-test"],"modelMapping":{"m":"claude-3"}}}`)
var streamingGenericConfig = json.RawMessage(`{"provider":{"type":"generic","genericHost":"generic.example.com","apiTokens":["t"]}}`)

func streamingRequestHeaders(path string) [][2]string {
	return [][2]string{
		{":authority", "example.com"},
		{":path", path},
		{":method", "POST"},
		{"Content-Type", "application/json"},
		{"Content-Length", "1"},
	}
}

// feedChunks 按 chunk 大小把 body 喂给插件，返回每块的动作与拼接后的上游 body。
func feedChunks(host wasmtest.TestHost, body []byte, chunk int) (actions []types.Action, upstream []byte) {
	for i := 0; i < len(body); i += chunk {
		j := i + chunk
		if j > len(body) {
			j = len(body)
		}
		a := host.CallOnHttpStreamingRequestBody(body[i:j], j == len(body))
		actions = append(actions, a)
		if a == types.ActionContinue {
			upstream = append(upstream, host.GetRequestBody()...)
		}
	}
	return
}

func requestHeader(host wasmtest.TestHost, name string) string {
	for _, h := range host.GetRequestHeaders() {
		if strings.EqualFold(h[0], name) {
			return h[1]
		}
	}
	return ""
}

func TestStreamingRequest_ClaudeSdkOrder(t *testing.T) {
	wasmtest.RunTest(t, func(t *testing.T) {
		host, status := wasmtest.NewTestHost(streamingClaudeConfig)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		require.Equal(t, types.HeaderStopIteration, host.CallOnHttpRequestHeaders(streamingRequestHeaders("/v1/chat/completions")))

		// 官方 SDK 的字段顺序：messages 在前，model / stream 在末尾；content 200KB 远超提交点
		big := strings.Repeat("y", 200000)
		body := `{"messages":[{"role":"system","content":"S"},{"role":"user","content":"` + big + `"}],"model":"m","stream":true,"max_tokens":50}`
		actions, upstream := feedChunks(host, []byte(body), 4096)

		// 提交点（64KB）之前一律 Pause，之后 Continue
		pauses := 0
		for _, a := range actions {
			if a == types.ActionPause {
				pauses++
			} else {
				break
			}
		}
		// 跨过提交点的那一块本身就返回 Continue：之前的块全部 Pause，累计恰好落在 64KB 前后
		require.Less(t, pauses*4096, 64<<10, "提交点前应一直 Pause")
		require.GreaterOrEqual(t, (pauses+1)*4096, 64<<10, "越过提交点就应放行，不该多攒")
		require.Equal(t, types.ActionContinue, actions[len(actions)-1])
		require.Greater(t, len(actions)-pauses, 1, "提交点后应逐块 Continue，而不是攒到最后")

		var out map[string]any
		require.NoError(t, json.Unmarshal(upstream, &out), "上游 body 应是合法 JSON: %s", truncate(upstream))
		require.Equal(t, "claude-3", out["model"])
		require.Equal(t, "S", out["system"])
		require.Equal(t, true, out["stream"])
		require.Equal(t, float64(50), out["max_tokens"])
		msgs := out["messages"].([]any)
		require.Len(t, msgs, 1)
		require.Equal(t, "user", msgs[0].(map[string]any)["role"])
		require.Len(t, msgs[0].(map[string]any)["content"].(string), 200000)
	})
}

func TestStreamingRequest_AcceptHeaderWhenStreamKnownEarly(t *testing.T) {
	wasmtest.RunTest(t, func(t *testing.T) {
		host, status := wasmtest.NewTestHost(streamingClaudeConfig)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		host.CallOnHttpRequestHeaders(streamingRequestHeaders("/v1/chat/completions"))

		body := `{"model":"m","stream":true,"messages":[{"role":"user","content":"` + strings.Repeat("y", 200000) + `"}]}`
		_, upstream := feedChunks(host, []byte(body), 4096)
		require.Equal(t, "text/event-stream", requestHeader(host, "Accept"), "stream 在提交点前可知时应改写 Accept")
		require.True(t, json.Valid(upstream))
	})
}

func TestStreamingRequest_FallbackBeforeCommit(t *testing.T) {
	wasmtest.RunTest(t, func(t *testing.T) {
		host, status := wasmtest.NewTestHost(streamingClaudeConfig)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		host.CallOnHttpRequestHeaders(streamingRequestHeaders("/v1/chat/completions"))

		// claude_content_blocks 流式未复刻，且出现在提交点之前 → 全程 Pause，末块走官方全量路径
		body := `{"model":"m","messages":[{"role":"user","claude_content_blocks":[{"type":"text","text":"B"}],"content":"` + strings.Repeat("y", 200000) + `"}]}`
		actions, upstream := feedChunks(host, []byte(body), 4096)
		for _, a := range actions[:len(actions)-1] {
			require.Equal(t, types.ActionPause, a)
		}
		require.Equal(t, types.ActionContinue, actions[len(actions)-1])
		var out map[string]any
		require.NoError(t, json.Unmarshal(upstream, &out))
		msgs := out["messages"].([]any)
		blocks := msgs[0].(map[string]any)["content"].([]any)
		require.Equal(t, "B", blocks[0].(map[string]any)["text"], "回落后应是官方对 claude_content_blocks 的映射")
		require.Equal(t, "claude-3", out["model"])
	})
}

func TestStreamingRequest_FailAfterCommit(t *testing.T) {
	wasmtest.RunTest(t, func(t *testing.T) {
		host, status := wasmtest.NewTestHost(streamingClaudeConfig)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		host.CallOnHttpRequestHeaders(streamingRequestHeaders("/v1/chat/completions"))

		// 不支持的形态出现在 200KB 之后：字节已放行，只能失败
		body := `{"model":"m","messages":[{"role":"user","content":"` + strings.Repeat("y", 200000) + `"},{"role":"user","content":"U","claude_content_blocks":[{"type":"text","text":"B"}]}]}`
		actions, _ := feedChunks(host, []byte(body), 4096)
		require.Equal(t, types.ActionPause, actions[len(actions)-1])
		resp := host.GetLocalResponse()
		require.NotNil(t, resp, "应发出本地 500 应答")
		require.Equal(t, uint32(500), resp.StatusCode)
		require.Contains(t, string(resp.Data), "bailed after commit")
	})
}

func TestStreamingRequest_GenericPassthrough(t *testing.T) {
	wasmtest.RunTest(t, func(t *testing.T) {
		host, status := wasmtest.NewTestHost(streamingGenericConfig)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		host.CallOnHttpRequestHeaders(streamingRequestHeaders("/v1/chat/completions"))

		body := "this is not even json " + strings.Repeat("z", 100000)
		actions, upstream := feedChunks(host, []byte(body), 4096)
		for _, a := range actions {
			require.Equal(t, types.ActionContinue, a, "generic 应逐块直接放行")
		}
		require.Equal(t, body, string(upstream))
	})
}

func TestStreamingRequest_SingleChunk(t *testing.T) {
	wasmtest.RunTest(t, func(t *testing.T) {
		host, status := wasmtest.NewTestHost(streamingClaudeConfig)
		defer host.Reset()
		require.Equal(t, types.OnPluginStartStatusOK, status)
		host.CallOnHttpRequestHeaders(streamingRequestHeaders("/v1/chat/completions"))

		body := `{"model":"m","messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"f","parameters":{"type":"object"}}}]}`
		require.Equal(t, types.ActionContinue, host.CallOnHttpStreamingRequestBody([]byte(body), true))
		var out map[string]any
		require.NoError(t, json.Unmarshal(host.GetRequestBody(), &out))
		require.Equal(t, "claude-3", out["model"])
		require.Equal(t, float64(4096), out["max_tokens"])
		require.Len(t, out["tools"].([]any), 1)
	})
}

func truncate(b []byte) string {
	if len(b) > 200 {
		return string(b[:200]) + "…"
	}
	return string(b)
}
