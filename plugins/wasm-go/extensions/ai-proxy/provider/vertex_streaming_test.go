package provider

import (
	"testing"

	"github.com/higress-group/wasm-go/pkg/iface"
	"github.com/stretchr/testify/require"
)

type mockHttpContext struct {
	context map[string]interface{}
}

func newMockHttpContext() *mockHttpContext {
	return &mockHttpContext{
		context: map[string]interface{}{
			ctxKeyFinalRequestModel: "gemini-2.5-flash",
		},
	}
}

func (m *mockHttpContext) Scheme() string                           { return "" }
func (m *mockHttpContext) Host() string                             { return "" }
func (m *mockHttpContext) Path() string                             { return "" }
func (m *mockHttpContext) Method() string                           { return "" }
func (m *mockHttpContext) SetContext(key string, value interface{}) { m.context[key] = value }
func (m *mockHttpContext) GetContext(key string) interface{}        { return m.context[key] }
func (m *mockHttpContext) GetBoolContext(key string, defaultValue bool) bool {
	value, ok := m.context[key].(bool)
	if !ok {
		return defaultValue
	}
	return value
}
func (m *mockHttpContext) GetStringContext(key, defaultValue string) string {
	value, ok := m.context[key].(string)
	if !ok {
		return defaultValue
	}
	return value
}
func (m *mockHttpContext) GetByteSliceContext(key string, defaultValue []byte) []byte {
	value, ok := m.context[key].([]byte)
	if !ok {
		return defaultValue
	}
	return value
}
func (m *mockHttpContext) GetUserAttribute(key string) interface{}          { return nil }
func (m *mockHttpContext) SetUserAttribute(key string, value interface{})   {}
func (m *mockHttpContext) SetUserAttributeMap(kvmap map[string]interface{}) {}
func (m *mockHttpContext) GetUserAttributeMap() map[string]interface{}      { return nil }
func (m *mockHttpContext) WriteUserAttributeToLog() error                   { return nil }
func (m *mockHttpContext) WriteUserAttributeToLogWithKey(key string) error  { return nil }
func (m *mockHttpContext) WriteUserAttributeToTrace() error                 { return nil }
func (m *mockHttpContext) DontReadRequestBody()                             {}
func (m *mockHttpContext) DontReadResponseBody()                            {}
func (m *mockHttpContext) BufferRequestBody()                               {}
func (m *mockHttpContext) BufferResponseBody()                              {}
func (m *mockHttpContext) NeedPauseStreamingResponse()                      {}
func (m *mockHttpContext) PushBuffer(buffer []byte)                         {}
func (m *mockHttpContext) PopBuffer() []byte                                { return nil }
func (m *mockHttpContext) BufferQueueSize() int                             { return 0 }
func (m *mockHttpContext) DisableReroute()                                  {}
func (m *mockHttpContext) SetRequestBodyBufferLimit(byteSize uint32)        {}
func (m *mockHttpContext) SetResponseBodyBufferLimit(byteSize uint32)       {}
func (m *mockHttpContext) RouteCall(method, url string, headers [][2]string, body []byte, callback iface.RouteResponseCallback) error {
	return nil
}
func (m *mockHttpContext) GetExecutionPhase() iface.HTTPExecutionPhase { return iface.DecodeData }
func (m *mockHttpContext) HasRequestBody() bool                        { return false }
func (m *mockHttpContext) HasResponseBody() bool                       { return false }
func (m *mockHttpContext) IsWebsocket() bool                           { return false }
func (m *mockHttpContext) IsBinaryRequestBody() bool {
	return false
}
func (m *mockHttpContext) IsBinaryResponseBody() bool { return false }

func TestVertexStreamingResponseBodyBuffersSplitChunks(t *testing.T) {
	provider := &vertexProvider{}
	ctx := newMockHttpContext()

	firstChunk := []byte(`data: {"candidates":[{"content":{"parts":[{"text":"Hel`)
	output, err := provider.OnStreamingResponseBody(ctx, ApiNameChatCompletion, firstChunk, false)
	require.NoError(t, err)
	require.Nil(t, output)
	require.Equal(t, firstChunk, ctx.GetByteSliceContext(ctxKeyStreamingBody, nil))

	secondChunk := []byte("lo\"}],\"role\":\"model\"},\"finishReason\":\"\",\"index\":0}],\"usageMetadata\":{\"promptTokenCount\":9,\"candidatesTokenCount\":5,\"totalTokenCount\":14}}\n")
	output, err = provider.OnStreamingResponseBody(ctx, ApiNameChatCompletion, secondChunk, false)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Contains(t, string(output), "chat.completion.chunk")
	require.Contains(t, string(output), `"content":"Hello"`)
	require.NotContains(t, string(output), "[DONE]")
	require.Nil(t, ctx.GetContext(ctxKeyStreamingBody))
}

func TestVertexStreamingResponseBodyKeepsFinalPayloadBeforeDone(t *testing.T) {
	provider := &vertexProvider{}
	ctx := newMockHttpContext()

	lastChunk := []byte(`data: {"candidates":[{"content":{"parts":[{"text":"Hello!"}],"role":"model"},"finishReason":"STOP","index":0}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":6,"totalTokenCount":15}}`)
	output, err := provider.OnStreamingResponseBody(ctx, ApiNameChatCompletion, lastChunk, true)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Contains(t, string(output), "chat.completion.chunk")
	require.Contains(t, string(output), `"content":"Hello!"`)
	require.Contains(t, string(output), "data: [DONE]")
}
