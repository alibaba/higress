package provider

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
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
			log.Errorf("[CompressContext] pre-retrieval failed: %v, continuing with partial retrieval", err)
			// Continue processing even if some contexts fail to retrieve (graceful degradation)
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
		// 使用更准确的基于内容的判断（支持token计算和Agent模式）
		if redisService, ok := c.memoryService.(*redisMemoryService); ok {
			// 检测是否为Agent请求
			isAgent := c.isAgentRequest(request)

			var shouldCompress bool
			if isAgent {
				// Agent模式：使用Agent感知的压缩判断
				shouldCompress = c.shouldCompressForAgent(contentStr)
				log.Debugf("[CompressContext] Agent request detected, using Agent-aware compression strategy")
			} else {
				// 标准模式：使用原有逻辑
				shouldCompress = redisService.ShouldCompressByContent(contentStr)
			}

			if !shouldCompress {
				if redisService.config.UseTokenBasedCompression {
					tokenCount := calculateTokensDeepSeekFromString(contentStr)
					log.Debugf("[CompressContext] skipping message %d, token count %d below threshold (agent: %v)", i, tokenCount, isAgent)
				} else {
					log.Debugf("[CompressContext] skipping message %d, size %d below threshold (agent: %v)", i, contentSize, isAgent)
				}
				newMessages = append(newMessages, msg)
				continue
			}
		}

		// Save context to Redis with error handling and graceful degradation
		contextId, err := c.memoryService.SaveContext(ctx, contentStr)
		if err != nil {
			log.Errorf("[CompressContext] failed to save context for message %d: %v, falling back to original content", i, err)
			// Graceful degradation: if compression fails, keep original content
			// This ensures the request can still proceed even if Redis is unavailable
			newMessages = append(newMessages, msg)
			continue
		}

		// Build compressed message with optional key info
		compressedContent := fmt.Sprintf("[Context stored with ID: %s]", contextId)
		if c.contextCompression != nil && c.contextCompression.PreserveKeyInfo {
			keyInfo := extractKeyInfo(contentStr)
			if keyInfo != "" {
				compressedContent = fmt.Sprintf("[Context ID: %s]\nKey Info: %s", contextId, keyInfo)
				log.Debugf("[CompressContext] preserved key info for message %d: %s", i, keyInfo)
			}
		}

		// Replace message content with context reference
		compressedMsg := chatMessage{
			Role:    msg.Role,
			Content: compressedContent,
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

// TransformResponseBody implements TransformResponseBodyHandler interface
// This method processes response body and handles memory retrieval when read_memory calls are detected
func (c *ProviderConfig) TransformResponseBody(ctx wrapper.HttpContext, apiName ApiName, body []byte) ([]byte, error) {
	// Only process chat completion API responses
	if apiName != ApiNameChatCompletion {
		return body, nil
	}

	// Check if compression is enabled
	if !c.IsCompressionEnabled() {
		return body, nil
	}

	// Get original request body from context
	originalRequestBody, ok := ctx.GetContext(CtxRequestBody).([]byte)
	if !ok {
		// If original request body is not available, try to get from context
		originalRequestBody = body
	}

	// Process response for memory retrieval
	needRetrieval, err := c.ProcessResponseForMemoryRetrieval(ctx, body, originalRequestBody)
	if err != nil {
		log.Errorf("[TransformResponseBody] failed to process memory retrieval: %v", err)
		// Return original body on error (graceful degradation)
		return body, nil
	}

	if needRetrieval {
		// Check if auto-retrieve is enabled
		if c.contextCompression != nil && !c.contextCompression.AutoRetrieve {
			// Auto-retrieve disabled, let agent call read_memory actively
			log.Debugf("[TransformResponseBody] auto-retrieve disabled, agent should call read_memory")
			return body, nil
		}

		// If memory retrieval was processed, return modified response
		// The response now includes tool responses in the message content
		modifiedBody, err := c.buildResponseWithToolResults(ctx, body)
		if err != nil {
			log.Errorf("[TransformResponseBody] failed to build response with tool results: %v", err)
			return body, nil
		}
		return modifiedBody, nil
	}

	return body, nil
}

// ProcessResponseForMemoryRetrieval processes responses that require automatic memory retrieval
// Production-grade implementation: automatically injects tool responses into the response
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

	// Batch retrieve context summaries concurrently (使用摘要而非完整内容)
	toolResponses, err := c.batchRetrieveContextSummariesConcurrent(ctx, memoryToolCalls)
	if err != nil {
		log.Errorf("[ProcessResponseForMemoryRetrieval] failed to retrieve context summaries: %v", err)
		// Continue with partial results
	}

	// Store tool responses in context for response building
	ctx.SetContext(ctxKeyOriginalToolCalls, memoryToolCalls)
	ctx.SetContext("tool_responses", toolResponses)

	log.Infof("[ProcessResponseForMemoryRetrieval] retrieved %d context summaries successfully", len(toolResponses))
	return true, nil
}

// buildResponseWithToolResults builds a response that includes tool results summaries
// This allows the LLM to continue processing with the retrieved context summaries
func (c *ProviderConfig) buildResponseWithToolResults(ctx wrapper.HttpContext, originalBody []byte) ([]byte, error) {
	response := &chatCompletionResponse{}
	if err := json.Unmarshal(originalBody, response); err != nil {
		return originalBody, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// Get tool responses (summaries) from context
	toolResponses, ok := ctx.GetContext("tool_responses").([]chatMessage)
	if !ok || len(toolResponses) == 0 {
		return originalBody, nil
	}

	// Build summary text from all tool responses
	var summaryBuilder strings.Builder
	summaryBuilder.WriteString(fmt.Sprintf("\n\n[已检索 %d 个上下文摘要]:\n", len(toolResponses)))

	for i, toolResp := range toolResponses {
		if i > 0 {
			summaryBuilder.WriteString("\n---\n")
		}
		summaryBuilder.WriteString(fmt.Sprintf("上下文 %d (ID: %s):\n", i+1, toolResp.Id))
		contentStr := toolResp.StringContent()
		if len(contentStr) > 0 {
			summaryBuilder.WriteString(contentStr)
		} else {
			summaryBuilder.WriteString("(摘要为空)")
		}
	}

	// Append summaries to assistant's message content
	if len(response.Choices) > 0 && response.Choices[0].Message != nil {
		summaryText := summaryBuilder.String()
		currentContent := ""
		if response.Choices[0].Message.Content != nil {
			if str, ok := response.Choices[0].Message.Content.(string); ok {
				currentContent = str
			}
		}
		if currentContent != "" {
			response.Choices[0].Message.Content = currentContent + summaryText
		} else {
			response.Choices[0].Message.Content = summaryText
		}
		log.Infof("[buildResponseWithToolResults] appended %d context summaries to response", len(toolResponses))
	}

	// Serialize modified response
	modifiedBody, err := json.Marshal(response)
	if err != nil {
		return originalBody, fmt.Errorf("failed to marshal modified response: %v", err)
	}

	return modifiedBody, nil
}

// batchRetrieveContextsConcurrent retrieves multiple contexts concurrently with timeout control
func (c *ProviderConfig) batchRetrieveContextsConcurrent(ctx wrapper.HttpContext, toolCalls []toolCall) ([]chatMessage, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}

	// Use a channel to collect results
	type result struct {
		toolCall toolCall
		content  string
		err      error
	}

	resultChan := make(chan result, len(toolCalls))
	var wg sync.WaitGroup

	// Concurrent retrieval with timeout
	timeout := 5 * time.Second
	done := make(chan struct{})

	// Start retrieval goroutines
	for i := range toolCalls {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			tc := toolCalls[idx]
			args := gjson.Parse(tc.Function.Arguments)
			contextId := args.Get("context_id").String()
			if contextId == "" {
				resultChan <- result{
					toolCall: tc,
					content:  "",
					err:      fmt.Errorf("empty context_id in tool call %s", tc.Id),
				}
				return
			}

			// Read context from Redis with timeout protection
			contentChan := make(chan string, 1)
			errChan := make(chan error, 1)

			go func() {
				content, err := c.memoryService.ReadContext(ctx, contextId)
				if err != nil {
					errChan <- err
					return
				}
				contentChan <- content
			}()

			select {
			case content := <-contentChan:
				resultChan <- result{
					toolCall: tc,
					content:  content,
					err:      nil,
				}
			case err := <-errChan:
				log.Errorf("[batchRetrieveContextsConcurrent] failed to retrieve context %s: %v", contextId, err)
				resultChan <- result{
					toolCall: tc,
					content:  fmt.Sprintf("Error: Failed to retrieve context %s", contextId),
					err:      err,
				}
			case <-time.After(timeout):
				log.Errorf("[batchRetrieveContextsConcurrent] timeout retrieving context %s", contextId)
				resultChan <- result{
					toolCall: tc,
					content:  fmt.Sprintf("Error: Timeout retrieving context %s", contextId),
					err:      fmt.Errorf("timeout"),
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(done)
	}()

	// Collect results
	toolResponses := make([]chatMessage, 0, len(toolCalls))
	resultsCollected := 0

collectLoop:
	for resultsCollected < len(toolCalls) {
		select {
		case res := <-resultChan:
			resultsCollected++
			if res.err == nil || res.content != "" {
				toolMsg := chatMessage{
					Role:    "tool",
					Content: res.content,
					Id:      res.toolCall.Id,
				}
				toolResponses = append(toolResponses, toolMsg)
				log.Infof("[batchRetrieveContextsConcurrent] retrieved context for tool call %s, length: %d", res.toolCall.Id, len(res.content))
			}
		case <-done:
			// All goroutines completed
			break collectLoop
		case <-time.After(timeout + 1*time.Second):
			// Safety timeout
			log.Warnf("[batchRetrieveContextsConcurrent] safety timeout reached, collected %d/%d results", resultsCollected, len(toolCalls))
			break collectLoop
		}
	}

	return toolResponses, nil
}

// batchRetrieveContextSummariesConcurrent retrieves multiple context summaries concurrently with timeout control
// This method uses summaries instead of full content to reduce token usage
func (c *ProviderConfig) batchRetrieveContextSummariesConcurrent(ctx wrapper.HttpContext, toolCalls []toolCall) ([]chatMessage, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}

	// Use a channel to collect results
	type result struct {
		toolCall toolCall
		summary  string
		err      error
	}

	resultChan := make(chan result, len(toolCalls))
	var wg sync.WaitGroup

	// Concurrent retrieval with timeout
	timeout := 5 * time.Second
	done := make(chan struct{})

	// Start retrieval goroutines
	for i := range toolCalls {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			tc := toolCalls[idx]
			args := gjson.Parse(tc.Function.Arguments)
			contextId := args.Get("context_id").String()
			if contextId == "" {
				resultChan <- result{
					toolCall: tc,
					summary:  "",
					err:      fmt.Errorf("empty context_id in tool call %s", tc.Id),
				}
				return
			}

			// Read context summary from Redis with timeout protection
			summaryChan := make(chan string, 1)
			errChan := make(chan error, 1)

			go func() {
				// 使用摘要接口而非完整内容
				summary, err := c.memoryService.ReadContextSummary(ctx, contextId)
				if err != nil {
					errChan <- err
					return
				}
				summaryChan <- summary
			}()

			select {
			case summary := <-summaryChan:
				resultChan <- result{
					toolCall: tc,
					summary:  summary,
					err:      nil,
				}
			case err := <-errChan:
				log.Errorf("[batchRetrieveContextSummariesConcurrent] failed to retrieve summary for context %s: %v", contextId, err)
				resultChan <- result{
					toolCall: tc,
					summary:  fmt.Sprintf("Error: Failed to retrieve summary for context %s", contextId),
					err:      err,
				}
			case <-time.After(timeout):
				log.Errorf("[batchRetrieveContextSummariesConcurrent] timeout retrieving summary for context %s", contextId)
				resultChan <- result{
					toolCall: tc,
					summary:  fmt.Sprintf("Error: Timeout retrieving summary for context %s", contextId),
					err:      fmt.Errorf("timeout"),
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(done)
	}()

	// Collect results
	toolResponses := make([]chatMessage, 0, len(toolCalls))
	resultsCollected := 0

collectLoop:
	for resultsCollected < len(toolCalls) {
		select {
		case res := <-resultChan:
			resultsCollected++
			if res.err == nil || res.summary != "" {
				toolMsg := chatMessage{
					Role:    "tool",
					Content: res.summary,
					Id:      res.toolCall.Id,
				}
				toolResponses = append(toolResponses, toolMsg)
				log.Infof("[batchRetrieveContextSummariesConcurrent] retrieved summary for tool call %s, length: %d", res.toolCall.Id, len(res.summary))
			}
		case <-done:
			// All goroutines completed
			break collectLoop
		case <-time.After(timeout + 1*time.Second):
			// Safety timeout
			log.Warnf("[batchRetrieveContextSummariesConcurrent] safety timeout reached, collected %d/%d results", resultsCollected, len(toolCalls))
			break collectLoop
		}
	}

	return toolResponses, nil
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
// Optimized with concurrent retrieval for better performance
func (c *ProviderConfig) preRetrieveContexts(ctx wrapper.HttpContext, request *chatCompletionRequest, contextIds []string) error {
	if len(contextIds) == 0 {
		return nil
	}

	// Concurrent batch retrieval
	contextMap := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup
	timeout := 5 * time.Second

	// Retrieve contexts concurrently
	for _, contextId := range contextIds {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			// Read context with timeout protection
			contentChan := make(chan string, 1)
			errChan := make(chan error, 1)

			go func() {
				content, err := c.memoryService.ReadContext(ctx, id)
				if err != nil {
					errChan <- err
					return
				}
				contentChan <- content
			}()

			select {
			case content := <-contentChan:
				mu.Lock()
				contextMap[id] = content
				mu.Unlock()
				log.Infof("[preRetrieveContexts] retrieved context %s, length: %d", id, len(content))
			case err := <-errChan:
				log.Errorf("[preRetrieveContexts] failed to retrieve context %s: %v", id, err)
				// Continue with other contexts (graceful degradation)
			case <-time.After(timeout):
				log.Errorf("[preRetrieveContexts] timeout retrieving context %s", id)
			}
		}(contextId)
	}

	// Wait for all retrievals to complete
	wg.Wait()

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
				} else {
					log.Warnf("[preRetrieveContexts] context %s not found in retrieved map", contextId)
				}
			}
		}
	}

	return nil
}

// isAgentRequest 检测是否为Agent请求
// Agent请求通常包含工具调用或工具定义
func (c *ProviderConfig) isAgentRequest(request *chatCompletionRequest) bool {
	// 检测是否有工具定义
	if len(request.Tools) > 0 {
		return true
	}

	// 检测消息历史中是否有工具调用
	for _, msg := range request.Messages {
		// 检测是否有工具调用
		if len(msg.ToolCalls) > 0 {
			return true
		}
		// 检测是否是工具结果消息
		if msg.Role == "tool" || msg.Role == "function" {
			return true
		}
	}

	return false
}

// extractKeyInfo 从内容中提取关键信息
// 用于在压缩时保留重要信息
func extractKeyInfo(content string) string {
	var keyInfo []string

	// 提取文件路径（Unix和Windows路径）
	filePathRegex := regexp.MustCompile(`(?:^|[\s\n])(/[^\s\n]+|\./[^\s\n]+|[A-Z]:\\[^\s\n]+)`)
	filePaths := filePathRegex.FindAllString(content, 5) // 最多5个路径
	if len(filePaths) > 0 {
		uniquePaths := uniqueStrings(filePaths)
		keyInfo = append(keyInfo, "Files: "+strings.Join(uniquePaths, ", "))
	}

	// 提取命令执行状态
	if strings.Contains(strings.ToLower(content), "success") {
		keyInfo = append(keyInfo, "Status: success")
	}
	if strings.Contains(strings.ToLower(content), "error") || strings.Contains(strings.ToLower(content), "failed") {
		keyInfo = append(keyInfo, "Status: error")
	}

	// 提取重要的数字结果（可能是行号、大小等）
	numberRegex := regexp.MustCompile(`\b\d{3,}\b`)  // 3位以上的数字
	numbers := numberRegex.FindAllString(content, 3) // 最多3个
	if len(numbers) > 0 {
		keyInfo = append(keyInfo, "Numbers: "+strings.Join(numbers, ", "))
	}

	// 提取URL
	urlRegex := regexp.MustCompile(`https?://[^\s\n]+`)
	urls := urlRegex.FindAllString(content, 3) // 最多3个
	if len(urls) > 0 {
		keyInfo = append(keyInfo, "URLs: "+strings.Join(urls, ", "))
	}

	if len(keyInfo) == 0 {
		return ""
	}

	return strings.Join(keyInfo, "; ")
}

// uniqueStrings 去除字符串切片中的重复项
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, s := range strs {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// shouldCompressForAgent 判断Agent模式下是否应该压缩
func (c *ProviderConfig) shouldCompressForAgent(content string) bool {
	if c.contextCompression == nil {
		return true
	}

	// 保守模式：提高压缩阈值
	if c.contextCompression.AgentMode == "conservative" {
		if c.contextCompression.UseTokenBasedCompression {
			tokenCount := calculateTokensDeepSeekFromString(content)
			// 保守模式：阈值提高2倍
			threshold := c.contextCompression.CompressionTokenThreshold
			if threshold == 0 {
				threshold = calculateTokensDeepSeek(1000)
			}
			return tokenCount > threshold*2
		} else {
			// 使用字节数判断
			threshold := c.contextCompression.CompressionBytesThreshold
			if threshold == 0 {
				threshold = 1000
			}
			return len(content) > threshold*2
		}
	}

	// 激进模式或默认：使用原有逻辑
	return true
}
