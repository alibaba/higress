package provider

import (
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	// Context keys for compression
	ctxKeyCompressionEnabled    = "compression_enabled"
	ctxKeyCompressedContextIds  = "compressed_context_ids"
	ctxKeyMemoryToolInjected    = "memory_tool_injected"
	ctxKeyOriginalToolCalls     = "original_tool_calls"
	ctxKeyNeedAutoRetrieve      = "need_auto_retrieve"
	ctxKeyAutoRetrieveContextId = "auto_retrieve_context_id"

	// Memory tool definitions
	MemoryToolSaveContext = "save_context"
	MemoryToolReadMemory  = "read_memory"
)

// CompressContextInRequest 在请求中压缩上下文
func (c *ProviderConfig) CompressContextInRequest(ctx wrapper.HttpContext, body []byte) ([]byte, error) {
	if c.memoryService == nil || !c.memoryService.IsEnabled() {
		return body, nil
	}

	log.Debugf("[CompressContext] starting context compression")

	request := &chatCompletionRequest{}
	if err := json.Unmarshal(body, request); err != nil {
		return body, fmt.Errorf("failed to unmarshal request: %v", err)
	}

	// 查找需要压缩的tool调用结果
	compressedIds := make(map[string]string) // message index -> context_id
	var newMessages []chatMessage
	totalSavedBytes := 0

	for i, msg := range request.Messages {
		// 只压缩 tool 或 function 角色的消息
		if msg.Role != "tool" && msg.Role != "function" {
			newMessages = append(newMessages, msg)
			continue
		}

		contentStr := msg.StringContent()
		contentSize := len(contentStr)

		// 检查是否应该压缩
		if redisService, ok := c.memoryService.(*redisMemoryService); ok {
			if !redisService.ShouldCompress(contentSize) {
				log.Debugf("[CompressContext] skipping message %d, size %d below threshold", i, contentSize)
				newMessages = append(newMessages, msg)
				continue
			}
		}

		// 保存上下文到Redis
		contextId, err := c.memoryService.SaveContext(ctx, contentStr)
		if err != nil {
			log.Errorf("[CompressContext] failed to save context for message %d: %v", i, err)
			newMessages = append(newMessages, msg)
			continue
		}

		// 替换消息内容为上下文引用
		compressedMsg := chatMessage{
			Role:    msg.Role,
			Content: fmt.Sprintf("[Context stored with ID: %s]", contextId),
		}
		if msg.Name != "" {
			compressedMsg.Name = msg.Name
		}
		if msg.Id != "" {
			compressedMsg.Id = msg.Id
		}

		newMessages = append(newMessages, compressedMsg)
		compressedIds[fmt.Sprintf("%d", i)] = contextId
		totalSavedBytes += contentSize - len(compressedMsg.StringContent())

		log.Infof("[CompressContext] compressed message %d, saved %d bytes, context_id: %s", 
			i, contentSize-len(compressedMsg.StringContent()), contextId)
	}

	if len(compressedIds) == 0 {
		log.Debugf("[CompressContext] no messages compressed")
		return body, nil
	}

	// 存储压缩的上下文ID以供后续使用
	ctx.SetContext(ctxKeyCompressedContextIds, compressedIds)
	ctx.SetContext(ctxKeyCompressionEnabled, true)

	log.Infof("[CompressContext] total saved bytes: %d, compressed %d messages", totalSavedBytes, len(compressedIds))

	// 更新request的messages
	request.Messages = newMessages

	// 注入内存工具定义
	if err := c.InjectMemoryTools(request); err != nil {
		log.Warnf("[CompressContext] failed to inject memory tools: %v", err)
	} else {
		ctx.SetContext(ctxKeyMemoryToolInjected, true)
	}

	// 重新序列化请求
	modifiedBody, err := json.Marshal(request)
	if err != nil {
		return body, fmt.Errorf("failed to marshal modified request: %v", err)
	}

	return modifiedBody, nil
}

// InjectMemoryTools 注入内存管理工具定义
func (c *ProviderConfig) InjectMemoryTools(request *chatCompletionRequest) error {
	// 定义 save_context 工具
	saveContextTool := tool{
		Type: "function",
		Function: function{
			Name:        MemoryToolSaveContext,
			Description: "Save conversation context or tool output to external memory for later retrieval",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The content to save to memory",
					},
				},
				"required": []string{"content"},
			},
		},
	}

	// 定义 read_memory 工具
	readMemoryTool := tool{
		Type: "function",
		Function: function{
			Name:        MemoryToolReadMemory,
			Description: "Read previously stored context from memory using context ID",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"context_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the context to retrieve",
					},
				},
				"required": []string{"context_id"},
			},
		},
	}

	// 将工具添加到请求中
	if request.Tools == nil {
		request.Tools = make([]tool, 0)
	}

	// 检查工具是否已存在
	hasReadMemory := false
	for _, t := range request.Tools {
		if t.Function.Name == MemoryToolReadMemory {
			hasReadMemory = true
			break
		}
	}

	if !hasReadMemory {
		request.Tools = append(request.Tools, readMemoryTool, saveContextTool)
		log.Debugf("[InjectMemoryTools] injected memory tools into request")
	}

	return nil
}

// HandleMemoryToolCall 处理LLM返回的内存工具调用
func (c *ProviderConfig) HandleMemoryToolCall(ctx wrapper.HttpContext, toolCall toolCall) (string, bool, error) {
	if toolCall.Function.Name != MemoryToolReadMemory {
		return "", false, nil
	}

	log.Debugf("[HandleMemoryToolCall] detected read_memory call: %s", toolCall.Function.Arguments)

	// 解析参数获取context_id
	args := gjson.Parse(toolCall.Function.Arguments)
	contextId := args.Get("context_id").String()
	if contextId == "" {
		return "", true, fmt.Errorf("missing context_id in read_memory call")
	}

	// 从Redis读取上下文
	content, err := c.memoryService.ReadContext(ctx, contextId)
	if err != nil {
		return "", true, fmt.Errorf("failed to read context %s: %v", contextId, err)
	}

	log.Infof("[HandleMemoryToolCall] retrieved context %s, length: %d", contextId, len(content))
	return content, true, nil
}

// ProcessResponseWithMemoryRetrieval 处理需要自动检索内存的响应
func (c *ProviderConfig) ProcessResponseWithMemoryRetrieval(
	ctx wrapper.HttpContext, 
	body []byte,
	originalRequestBody []byte,
) ([]byte, bool, error) {
	if c.memoryService == nil || !c.memoryService.IsEnabled() {
		return body, false, nil
	}

	response := &chatCompletionResponse{}
	if err := json.Unmarshal(body, response); err != nil {
		return body, false, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// 检查是否有read_memory工具调用
	needRetrieval := false
	var contextIdsToRetrieve []string

	for _, choice := range response.Choices {
		if choice.Message != nil {
			for _, toolCall := range choice.Message.ToolCalls {
				if toolCall.Function.Name == MemoryToolReadMemory {
					args := gjson.Parse(toolCall.Function.Arguments)
					contextId := args.Get("context_id").String()
					if contextId != "" {
						contextIdsToRetrieve = append(contextIdsToRetrieve, contextId)
						needRetrieval = true
					}
				}
			}
		}
	}

	if !needRetrieval {
		return body, false, nil
	}

	log.Infof("[ProcessResponseWithMemoryRetrieval] need to retrieve %d contexts", len(contextIdsToRetrieve))

	// 重新构建请求，添加检索到的上下文
	request := &chatCompletionRequest{}
	if err := json.Unmarshal(originalRequestBody, request); err != nil {
		return body, false, fmt.Errorf("failed to unmarshal original request: %v", err)
	}

	// 添加助手的工具调用消息
	assistantMsg := chatMessage{
		Role:      roleAssistant,
		ToolCalls: response.Choices[0].Message.ToolCalls,
	}
	request.Messages = append(request.Messages, assistantMsg)

	// 添加检索到的上下文作为工具响应
	for _, contextId := range contextIdsToRetrieve {
		content, err := c.memoryService.ReadContext(ctx, contextId)
		if err != nil {
			log.Errorf("[ProcessResponseWithMemoryRetrieval] failed to retrieve context %s: %v", contextId, err)
			content = fmt.Sprintf("Error retrieving context: %v", err)
		}

		toolMsg := chatMessage{
			Role:    "tool",
			Content: content,
			Name:    MemoryToolReadMemory,
		}
		request.Messages = append(request.Messages, toolMsg)
	}

	// 重新序列化请求
	newRequestBody, err := json.Marshal(request)
	if err != nil {
		return body, false, fmt.Errorf("failed to marshal new request: %v", err)
	}

	// 标记需要重新请求
	ctx.SetContext(ctxKeyNeedAutoRetrieve, true)
	
	return newRequestBody, true, nil
}

// ProcessStreamingResponseForMemoryCall 处理流式响应中的内存工具调用
func (c *ProviderConfig) ProcessStreamingResponseForMemoryCall(
	ctx wrapper.HttpContext,
	event StreamEvent,
) (bool, error) {
	if c.memoryService == nil || !c.memoryService.IsEnabled() {
		return false, nil
	}

	// 检查事件数据中是否包含read_memory工具调用
	data := event.Data
	if data == "" || data == streamEndDataValue {
		return false, nil
	}

	// 解析delta中的tool_calls
	toolCallsJson := gjson.Get(data, "choices.0.delta.tool_calls")
	if !toolCallsJson.Exists() {
		return false, nil
	}

	// 检查是否有read_memory调用
	for _, tc := range toolCallsJson.Array() {
		funcName := tc.Get("function.name").String()
		if funcName == MemoryToolReadMemory {
			// 累积工具调用信息
			existingToolCalls := ctx.GetStringContext(ctxKeyOriginalToolCalls, "")
			existingToolCalls += tc.String() + ","
			ctx.SetContext(ctxKeyOriginalToolCalls, existingToolCalls)
			
			// 提取context_id
			args := tc.Get("function.arguments").String()
			if args != "" {
				contextId := gjson.Get(args, "context_id").String()
				if contextId != "" {
					ctx.SetContext(ctxKeyAutoRetrieveContextId, contextId)
					log.Infof("[ProcessStreamingResponse] detected read_memory call for context: %s", contextId)
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// GetMemoryService 获取内存服务
func (c *ProviderConfig) GetMemoryService() MemoryService {
	return c.memoryService
}

// IsCompressionEnabled 检查压缩是否启用
func (c *ProviderConfig) IsCompressionEnabled() bool {
	return c.memoryService != nil && c.memoryService.IsEnabled()
}
