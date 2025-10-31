package provider

import (
	"encoding/json"
	"testing"
)

func TestCompressContextInRequest(t *testing.T) {
	// 创建mock配置
	config := &ProviderConfig{
		contextCompression: &ContextCompressionConfig{
			Enabled:                   true,
			CompressionBytesThreshold: 100,
			MemoryTTL:                 3600,
		},
	}

	// 创建禁用的内存服务用于测试
	config.memoryService = &disabledMemoryService{}

	// 创建测试请求
	requestBody := `{
		"model": "gpt-4",
		"messages": [
			{
				"role": "user",
				"content": "Hello"
			},
			{
				"role": "tool",
				"content": "Short content"
			}
		]
	}`

	// 测试压缩功能（应该跳过因为服务被禁用）
	result, err := config.CompressContextInRequest(nil, []byte(requestBody))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证返回的是原始body
	if string(result) != requestBody {
		t.Error("expected original body when service is disabled")
	}
}

func TestExtractCompressedContextIds(t *testing.T) {
	config := &ProviderConfig{}

	messages := []chatMessage{
		{
			Role:    "user",
			Content: "Hello",
		},
		{
			Role:    "tool",
			Content: "[Context stored with ID: abc123def456]",
		},
		{
			Role:    "tool",
			Content: "Normal content [Context stored with ID: xyz789] more content",
		},
	}

	ids := config.extractCompressedContextIds(messages)

	if len(ids) != 2 {
		t.Errorf("expected 2 context IDs, got %d", len(ids))
	}

	expectedIds := []string{"abc123def456", "xyz789"}
	for i, id := range ids {
		if id != expectedIds[i] {
			t.Errorf("expected ID %s, got %s", expectedIds[i], id)
		}
	}
}

func TestInjectMemoryTools(t *testing.T) {
	config := &ProviderConfig{}

	request := &chatCompletionRequest{
		Model: "gpt-4",
		Messages: []chatMessage{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
	}

	err := config.InjectMemoryTools(request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证工具已注入
	if len(request.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(request.Tools))
	}

	// 验证工具名称
	hasReadMemory := false
	hasSaveContext := false
	for _, tool := range request.Tools {
		if tool.Function.Name == MemoryToolReadMemory {
			hasReadMemory = true
		}
		if tool.Function.Name == MemoryToolSaveContext {
			hasSaveContext = true
		}
	}

	if !hasReadMemory {
		t.Error("read_memory tool not found")
	}
	if !hasSaveContext {
		t.Error("save_context tool not found")
	}
}

func TestHandleMemoryToolCall(t *testing.T) {
	config := &ProviderConfig{
		memoryService: &disabledMemoryService{},
	}

	// 测试非read_memory调用
	otherToolCall := toolCall{
		Function: functionCall{
			Name:      "other_tool",
			Arguments: `{"param": "value"}`,
		},
	}

	content, isMemoryCall, err := config.HandleMemoryToolCall(nil, otherToolCall)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isMemoryCall {
		t.Error("expected false for non-memory tool call")
	}
	if content != "" {
		t.Error("expected empty content for non-memory tool call")
	}

	// 测试read_memory调用
	readMemoryCall := toolCall{
		Function: functionCall{
			Name:      MemoryToolReadMemory,
			Arguments: `{"context_id": "test123"}`,
		},
	}

	_, isMemoryCall, err = config.HandleMemoryToolCall(nil, readMemoryCall)
	if !isMemoryCall {
		t.Error("expected true for read_memory tool call")
	}
	// 应该返回错误因为服务被禁用
	if err == nil {
		t.Error("expected error when service is disabled")
	}
}

func TestCompressContextFullWorkflow(t *testing.T) {
	// 创建完整的请求体
	request := chatCompletionRequest{
		Model: "gpt-4",
		Messages: []chatMessage{
			{
				Role:    "user",
				Content: "Hello",
			},
			{
				Role:    "assistant",
				Content: "Hi, I'll search for information.",
				ToolCalls: []toolCall{
					{
						Id:   "call_123",
						Type: "function",
						Function: functionCall{
							Name:      "search",
							Arguments: `{"query":"test"}`,
						},
					},
				},
			},
			{
				Role:    "tool",
				Content: "[Context stored with ID: compressed123]",
				Name:    "search",
			},
			{
				Role:    "user",
				Content: "Summarize the results",
			},
		},
	}

	config := &ProviderConfig{
		memoryService: &disabledMemoryService{},
	}

	// 测试提取压缩的context ID
	ids := config.extractCompressedContextIds(request.Messages)
	if len(ids) != 1 {
		t.Errorf("expected 1 compressed context ID, got %d", len(ids))
	}
	if ids[0] != "compressed123" {
		t.Errorf("expected context ID 'compressed123', got '%s'", ids[0])
	}

	// 验证请求可以被正确序列化
	_, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
}

func TestProcessResponseForMemoryRetrieval(t *testing.T) {
	config := &ProviderConfig{
		memoryService: &disabledMemoryService{},
	}

	// 测试无工具调用的响应
	responseNoToolCall := `{
		"id": "chatcmpl-123",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "Hello!"
				},
				"finish_reason": "stop"
			}
		]
	}`

	requestBody := []byte(`{"model":"gpt-4","messages":[]}`)
	needRetrieval, err := config.ProcessResponseForMemoryRetrieval(nil, []byte(responseNoToolCall), requestBody)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if needRetrieval {
		t.Error("expected false for response without tool calls")
	}

	// 测试有read_memory调用的响应
	responseWithMemoryCall := `{
		"id": "chatcmpl-456",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"tool_calls": [
						{
							"id": "call_123",
							"type": "function",
							"function": {
								"name": "read_memory",
								"arguments": "{\"context_id\":\"abc123\"}"
							}
						}
					]
				},
				"finish_reason": "tool_calls"
			}
		]
	}`

	// 这个测试会失败因为服务被禁用，但应该检测到需要检索
	needRetrieval, err = config.ProcessResponseForMemoryRetrieval(nil, []byte(responseWithMemoryCall), requestBody)
	// 服务禁用时应该返回false
	if needRetrieval {
		t.Error("expected false when service is disabled")
	}
}
