package provider

import (
	"encoding/json"
	"fmt"
	"strings"

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
	ctxKeyReRequestBody         = "re_request_body"

	// Memory tool definitions
	MemoryToolSaveContext = "save_context"
	MemoryToolReadMemory  = "read_memory"
)

// CompressContextInRequest 在请求中压缩上下文并智能预检索
func (c *ProviderConfig) CompressContextInRequest(ctx wrapper.HttpContext, body []byte) ([]byte, error) {
	if c.memoryService == nil || !c.memoryService.IsEnabled() {
		return body, nil
	}

	log.Debugf("[CompressContext] starting context compression and pre-retrieval")

	request := &chatCompletionRequest{}
	if err := json.Unmarshal(body, request); err != nil {
		return body, fmt.Errorf("failed to unmarshal request: %v", err)
	}

	// 第一步：检查是否有需要恢复的压缩引用
	needRetrievalIds := c.extractCompressedContextIds(request.Messages)
	if len(needRetrievalIds) > 0 {
		log.Infof("[CompressContext] found %d compressed context references, pre-retrieving...", len(needRetrievalIds))
		if err := c.preRetrieveContexts(ctx, request, needRetrievalIds); err != nil {
			log.Errorf("[CompressContext] pre-retrieval failed: %v", err)
		}
	}

	// 第二步：查找需要压缩的tool调用结果
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

	if len(compressedIds) == 0 && len(needRetrievalIds) == 0 {
		log.Debugf("[CompressContext] no messages compressed or retrieved")
		return body, nil
	}

	// 存储压缩的上下文ID以供后续使用
	if len(compressedIds) > 0 {
		ctx.SetContext(ctxKeyCompressedContextIds, compressedIds)
		ctx.SetContext(ctxKeyCompressionEnabled, true)
		log.Infof("[CompressContext] total saved bytes: %d, compressed %d messages", totalSavedBytes, len(compressedIds))
	}

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

// GetMemoryService 获取内存服务
func (c *ProviderConfig) GetMemoryService() MemoryService {
	return c.memoryService
}

// IsCompressionEnabled 检查压缩是否启用
func (c *ProviderConfig) IsCompressionEnabled() bool {
	return c.memoryService != nil && c.memoryService.IsEnabled()
}

// ProcessResponseForMemoryRetrieval 处理需要自动检索内存的响应
// 这是生产级实现：检测到read_memory调用后，触发异步重新请求
func (c *ProviderConfig) ProcessResponseForMemoryRetrieval(
	ctx wrapper.HttpContext,
	body []byte,
	originalRequestBody []byte,
) (bool, error) {
	if c.memoryService == nil || !c.memoryService.IsEnabled() {
		return false, nil
	}

	response := &chatCompletionResponse{}
	if err := json.Unmarshal(body, response); err != nil {
		return false, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// 检查是否有 read_memory 工具调用
	var memoryToolCalls []toolCall
	for _, choice := range response.Choices {
		if choice.Message != nil {
			for _, toolCall := range choice.Message.ToolCalls {
				if toolCall.Function.Name == MemoryToolReadMemory {
					memoryToolCalls = append(memoryToolCalls, toolCall)
				}
			}
		}
	}

	if len(memoryToolCalls) == 0 {
		return false, nil
	}

	log.Infof("[ProcessResponseForMemoryRetrieval] detected %d read_memory calls, initiating auto-retrieval", len(memoryToolCalls))

	// 存储助手的工具调用消息
	assistantMsg := chatMessage{
		Role:      roleAssistant,
		ToolCalls: response.Choices[0].Message.ToolCalls,
	}
	if response.Choices[0].Message.Content != "" {
		assistantMsg.Content = response.Choices[0].Message.Content
	}

	// 批量检索上下文
	toolResponses := make([]chatMessage, 0, len(memoryToolCalls))
	for _, toolCall := range memoryToolCalls {
		args := gjson.Parse(toolCall.Function.Arguments)
		contextId := args.Get("context_id").String()
		if contextId == "" {
			log.Warnf("[ProcessResponseForMemoryRetrieval] empty context_id in tool call %s", toolCall.Id)
			continue
		}

		// 从Redis读取上下文
		content, err := c.memoryService.ReadContext(ctx, contextId)
		if err != nil {
			log.Errorf("[ProcessResponseForMemoryRetrieval] failed to retrieve context %s: %v", contextId, err)
			content = fmt.Sprintf("Error: Failed to retrieve context %s", contextId)
		}

		// 构建工具响应消息
		toolMsg := chatMessage{
			Role:    "tool",
			Content: content,
			Id:      toolCall.Id, // 使用Id字段关联工具调用
		}
		toolResponses = append(toolResponses, toolMsg)
		log.Infof("[ProcessResponseForMemoryRetrieval] retrieved context %s, length: %d", contextId, len(content))
	}

	// 构建新请求
	request := &chatCompletionRequest{}
	if err := json.Unmarshal(originalRequestBody, request); err != nil {
		return false, fmt.Errorf("failed to unmarshal original request: %v", err)
	}

	// 添加助手消息和工具响应
	request.Messages = append(request.Messages, assistantMsg)
	request.Messages = append(request.Messages, toolResponses...)

	// 序列化新请求
	newRequestBody, err := json.Marshal(request)
	if err != nil {
		return false, fmt.Errorf("failed to marshal new request: %v", err)
	}

	// 存储新请求体以供重新请求使用
	ctx.SetContext(ctxKeyReRequestBody, newRequestBody)
	ctx.SetContext(ctxKeyNeedAutoRetrieve, true)

	log.Infof("[ProcessResponseForMemoryRetrieval] prepared re-request with %d tool responses", len(toolResponses))
	return true, nil
}

// extractCompressedContextIds 从消息中提取压缩的上下文ID引用
func (c *ProviderConfig) extractCompressedContextIds(messages []chatMessage) []string {
	var contextIds []string
	pattern := "[Context stored with ID: "

	for _, msg := range messages {
		content := msg.StringContent()
		// 查找压缩引用模式
		if idx := strings.Index(content, pattern); idx != -1 {
			start := idx + len(pattern)
			end := strings.Index(content[start:], "]")
			if end != -1 {
				contextId := content[start : start+end]
				contextIds = append(contextIds, contextId)
				log.Debugf("[extractCompressedContextIds] found context reference: %s", contextId)
			}
		}
	}

	return contextIds
}

// preRetrieveContexts 预先检索压缩的上下文并恢复到消息中
func (c *ProviderConfig) preRetrieveContexts(ctx wrapper.HttpContext, request *chatCompletionRequest, contextIds []string) error {
	// 批量检索上下文
	contextMap := make(map[string]string)
	for _, contextId := range contextIds {
		content, err := c.memoryService.ReadContext(ctx, contextId)
		if err != nil {
			log.Errorf("[preRetrieveContexts] failed to retrieve context %s: %v", contextId, err)
			continue
		}
		contextMap[contextId] = content
		log.Infof("[preRetrieveContexts] retrieved context %s, length: %d", contextId, len(content))
	}

	// 恢复消息内容
	for i := range request.Messages {
		content := request.Messages[i].StringContent()
		pattern := "[Context stored with ID: "

		if idx := strings.Index(content, pattern); idx != -1 {
			start := idx + len(pattern)
			end := strings.Index(content[start:], "]")
			if end != -1 {
				contextId := content[start : start+end]
				if retrievedContent, ok := contextMap[contextId]; ok {
					// 恢复原始内容
					request.Messages[i].Content = retrievedContent
					log.Infof("[preRetrieveContexts] restored message %d with context %s", i, contextId)
				}
			}
		}
	}

	return nil
}
