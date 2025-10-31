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

// CompressContextInRequest compresses context in request with intelligent pre-retrieval
func (c *ProviderConfig) CompressContextInRequest(ctx wrapper.HttpContext, body []byte) ([]byte, error) {
	if c.memoryService == nil || !c.memoryService.IsEnabled() {
		return body, nil
	}

	log.Debugf("[CompressContext] starting context compression and pre-retrieval")

	request := &chatCompletionRequest{}
	if err := json.Unmarshal(body, request); err != nil {
		return body, fmt.Errorf("failed to unmarshal request: %v", err)
	}

	// Step 1: Check if there are compressed references that need to be restored
	needRetrievalIds := c.extractCompressedContextIds(request.Messages)
	if len(needRetrievalIds) > 0 {
		log.Infof("[CompressContext] found %d compressed context references, pre-retrieving...", len(needRetrievalIds))
		if err := c.preRetrieveContexts(ctx, request, needRetrievalIds); err != nil {
			log.Errorf("[CompressContext] pre-retrieval failed: %v", err)
		}
	}

	// Step 2: Find tool call results that need to be compressed
	compressedIds := make(map[string]string) // message index -> context_id
	var newMessages []chatMessage
	totalSavedBytes := 0

	for i, msg := range request.Messages {
		// Only compress messages with tool or function role
		if msg.Role != "tool" && msg.Role != "function" {
			newMessages = append(newMessages, msg)
			continue
		}

		contentStr := msg.StringContent()
		contentSize := len(contentStr)

		// Check if compression should be applied
		if redisService, ok := c.memoryService.(*redisMemoryService); ok {
			if !redisService.ShouldCompress(contentSize) {
				log.Debugf("[CompressContext] skipping message %d, size %d below threshold", i, contentSize)
				newMessages = append(newMessages, msg)
				continue
			}
		}

		// Save context to Redis
		contextId, err := c.memoryService.SaveContext(ctx, contentStr)
		if err != nil {
			log.Errorf("[CompressContext] failed to save context for message %d: %v", i, err)
			newMessages = append(newMessages, msg)
			continue
		}

		// Replace message content with context reference
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

	// Store compressed context IDs for later use
	if len(compressedIds) > 0 {
		ctx.SetContext(ctxKeyCompressedContextIds, compressedIds)
		ctx.SetContext(ctxKeyCompressionEnabled, true)
		log.Infof("[CompressContext] total saved bytes: %d, compressed %d messages", totalSavedBytes, len(compressedIds))
	}

	// Update request messages
	request.Messages = newMessages

	// Inject memory tool definitions
	if err := c.InjectMemoryTools(request); err != nil {
		log.Warnf("[CompressContext] failed to inject memory tools: %v", err)
	} else {
		ctx.SetContext(ctxKeyMemoryToolInjected, true)
	}

	// Re-serialize request
	modifiedBody, err := json.Marshal(request)
	if err != nil {
		return body, fmt.Errorf("failed to marshal modified request: %v", err)
	}

	return modifiedBody, nil
}

// InjectMemoryTools injects memory management tool definitions
func (c *ProviderConfig) InjectMemoryTools(request *chatCompletionRequest) error {
	// Define save_context tool
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

	// Define read_memory tool
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

	// Add tools to request
	if request.Tools == nil {
		request.Tools = make([]tool, 0)
	}

	// Check if tools already exist
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

// HandleMemoryToolCall handles memory tool calls returned by LLM
func (c *ProviderConfig) HandleMemoryToolCall(ctx wrapper.HttpContext, toolCall toolCall) (string, bool, error) {
	if toolCall.Function.Name != MemoryToolReadMemory {
		return "", false, nil
	}

	log.Debugf("[HandleMemoryToolCall] detected read_memory call: %s", toolCall.Function.Arguments)

	// Parse arguments to get context_id
	args := gjson.Parse(toolCall.Function.Arguments)
	contextId := args.Get("context_id").String()
	if contextId == "" {
		return "", true, fmt.Errorf("missing context_id in read_memory call")
	}

	// Read context from Redis
	content, err := c.memoryService.ReadContext(ctx, contextId)
	if err != nil {
		return "", true, fmt.Errorf("failed to read context %s: %v", contextId, err)
	}

	log.Infof("[HandleMemoryToolCall] retrieved context %s, length: %d", contextId, len(content))
	return content, true, nil
}

// GetMemoryService returns the memory service
func (c *ProviderConfig) GetMemoryService() MemoryService {
	return c.memoryService
}

// IsCompressionEnabled checks if compression is enabled
func (c *ProviderConfig) IsCompressionEnabled() bool {
	return c.memoryService != nil && c.memoryService.IsEnabled()
}

// ProcessResponseForMemoryRetrieval processes responses that require automatic memory retrieval
// Production-grade implementation: triggers async re-request when read_memory call is detected
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

	// Check for read_memory tool calls
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

	// Store assistant's tool call message
	assistantMsg := chatMessage{
		Role:      roleAssistant,
		ToolCalls: response.Choices[0].Message.ToolCalls,
	}
	if response.Choices[0].Message.Content != "" {
		assistantMsg.Content = response.Choices[0].Message.Content
	}

	// Batch retrieve contexts
	toolResponses := make([]chatMessage, 0, len(memoryToolCalls))
	for _, toolCall := range memoryToolCalls {
		args := gjson.Parse(toolCall.Function.Arguments)
		contextId := args.Get("context_id").String()
		if contextId == "" {
			log.Warnf("[ProcessResponseForMemoryRetrieval] empty context_id in tool call %s", toolCall.Id)
			continue
		}

		// Read context from Redis
		content, err := c.memoryService.ReadContext(ctx, contextId)
		if err != nil {
			log.Errorf("[ProcessResponseForMemoryRetrieval] failed to retrieve context %s: %v", contextId, err)
			content = fmt.Sprintf("Error: Failed to retrieve context %s", contextId)
		}

		// Build tool response message
		toolMsg := chatMessage{
			Role:    "tool",
			Content: content,
			Id:      toolCall.Id, // Use Id field to associate with tool call
		}
		toolResponses = append(toolResponses, toolMsg)
		log.Infof("[ProcessResponseForMemoryRetrieval] retrieved context %s, length: %d", contextId, len(content))
	}

	// Build new request
	request := &chatCompletionRequest{}
	if err := json.Unmarshal(originalRequestBody, request); err != nil {
		return false, fmt.Errorf("failed to unmarshal original request: %v", err)
	}

	// Add assistant message and tool responses
	request.Messages = append(request.Messages, assistantMsg)
	request.Messages = append(request.Messages, toolResponses...)

	// Serialize new request
	newRequestBody, err := json.Marshal(request)
	if err != nil {
		return false, fmt.Errorf("failed to marshal new request: %v", err)
	}

	// Store new request body for re-request use
	ctx.SetContext(ctxKeyReRequestBody, newRequestBody)
	ctx.SetContext(ctxKeyNeedAutoRetrieve, true)

	log.Infof("[ProcessResponseForMemoryRetrieval] prepared re-request with %d tool responses", len(toolResponses))
	return true, nil
}

// extractCompressedContextIds extracts compressed context ID references from messages
func (c *ProviderConfig) extractCompressedContextIds(messages []chatMessage) []string {
	var contextIds []string
	pattern := "[Context stored with ID: "

	for _, msg := range messages {
		content := msg.StringContent()
		// Find compressed reference pattern
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

// preRetrieveContexts pre-retrieves compressed contexts and restores them to messages
func (c *ProviderConfig) preRetrieveContexts(ctx wrapper.HttpContext, request *chatCompletionRequest, contextIds []string) error {
	// Batch retrieve contexts
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

	// Restore message content
	for i := range request.Messages {
		content := request.Messages[i].StringContent()
		pattern := "[Context stored with ID: "

		if idx := strings.Index(content, pattern); idx != -1 {
			start := idx + len(pattern)
			end := strings.Index(content[start:], "]")
			if end != -1 {
				contextId := content[start : start+end]
				if retrievedContent, ok := contextMap[contextId]; ok {
					// Restore original content
					request.Messages[i].Content = retrievedContent
					log.Infof("[preRetrieveContexts] restored message %d with context %s", i, contextId)
				}
			}
		}
	}

	return nil
}
