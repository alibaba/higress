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
// 使用分层压缩策略和内容感知的压缩决策
func (c *ProviderConfig) CompressContextInRequest(ctx wrapper.HttpContext, body []byte) ([]byte, error) {
	if c.memoryService == nil || !c.memoryService.IsEnabled() {
		return body, nil
	}

	log.Debugf("[CompressContext] starting context compression and pre-retrieval with layered strategy")

	request := &chatCompletionRequest{}
	if err := json.Unmarshal(body, request); err != nil {
		return body, fmt.Errorf("failed to unmarshal request: %v", err)
	}

	// 检测是否为Agent请求和Agent模式
	isAgent := c.isAgentRequest(request)
	agentMode := "normal"
	if isAgent && c.memoryService != nil {
		if redisService, ok := c.memoryService.(*redisMemoryService); ok {
			agentMode = redisService.config.AgentMode
		}
	}

	log.Debugf("[CompressContext] Request type: Agent=%v, AgentMode=%s", isAgent, agentMode)

	// 初始化分层压缩策略（仅在启用时）
	var layeredStrategy *LayeredCompressionStrategy
	useLayeredCompression := false
	if c.contextCompression != nil && c.contextCompression.EnableLayeredCompression {
		useLayeredCompression = true
		layeredStrategy = NewLayeredCompressionStrategy()
		log.Debugf("[CompressContext] Layered compression strategy enabled")
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
	compressionStats := map[string]int{} // 统计各类型的压縩情况

	// 初始化会话上下文链，跟踪一个会话中所有工具调用的顺序和依赖关系
	sessionToolChain := &SessionToolCallChain{
		ToolCalls:  []ToolCallContextInfo{},
		ContextIds: make(map[string]int),
	}
	// 保存每个工具调用的上下文信息，以持待后续工具调用参考
	toolContextMap := make(map[string]ToolCallContextInfo) // contextId -> tool info

	for i, msg := range request.Messages {
		// Only compress messages with tool or function role
		if msg.Role != "tool" && msg.Role != "function" {
			newMessages = append(newMessages, msg)
			continue
		}

		contentStr := msg.StringContent()
		contentSize := len(contentStr)

		// Step 2.1: 决定是否压缩
		var shouldCompress bool
		var compressionReason string
		var dataLayer string

		if redisService, ok := c.memoryService.(*redisMemoryService); ok {
			if useLayeredCompression && layeredStrategy != nil {
				// 使用分层压缩策略
				useTokenBased := redisService.config.UseTokenBasedCompression
				preserveKeyInfo := redisService.config.PreserveKeyInfo

				decision := layeredStrategy.DecideCompression(
					contentStr,
					useTokenBased,
					agentMode,
					preserveKeyInfo)

				shouldCompress = decision.ShouldCompress
				compressionReason = decision.Reason
				dataLayer = c.contentLayerString(decision.DataLayer)

				log.Debugf("[CompressContext] Message %d: type analysis: %v, layer: %s, strategy: %v, should_compress: %v",
					i, "analysis", dataLayer, decision.Strategy, shouldCompress)

				// 记录统计信息
				strategyStr := c.compressionStrategyString(decision.Strategy)
				compressionStats[strategyStr]++

				// 如果不压缩，记录原因
				if !shouldCompress {
					if useTokenBased {
						tokens := calculateTokensDeepSeekFromString(contentStr)
						log.Debugf("[CompressContext] Message %d: skipped (tokens: %d, layer: %s, reason: %s)", i, tokens, dataLayer, compressionReason)
					} else {
						log.Debugf("[CompressContext] Message %d: skipped (size: %d bytes, layer: %s, reason: %s)", i, contentSize, dataLayer, compressionReason)
					}
					newMessages = append(newMessages, msg)
					continue
				}
			} else {
				// 使用原有逻辑（向后兼容）
				if isAgent {
					// Agent模式：使用Agent感知的压缩判断
					shouldCompress = c.shouldCompressForAgent(contentStr)
					compressionReason = "Agent-aware compression"
					dataLayer = "N/A"
					log.Debugf("[CompressContext] Agent request detected, using Agent-aware compression strategy")
				} else {
					// 标准模式：使用原有逻辑
					shouldCompress = redisService.ShouldCompressByContent(contentStr)
					compressionReason = "Content-based compression"
					dataLayer = "N/A"
				}

				// 日志记录
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

				compressionStats["standard"]++
			}
		}

		// Step 2.2: 保存上下文到Redis
		// 改进的会话级别管理：获取上下文会话ID
		sessionId, err := c.memoryService.GetOrCreateSessionId(ctx)
		if err != nil {
			log.Warnf("[CompressContext] failed to get or create session id: %v, falling back to default", err)
		}

		// 提取工具名称和参数信息（为了提供给LLM摘要器）
		toolName := msg.Name     // 工具输出消息的Name为工具名称
		toolCallId := msg.Id     // 工具输出消息的Id是工具调用ID
		var toolArgs string = "" // 工具调用的参数（从之前的工具调用消息中提取）

		// 尝试从之前的工具调用消息中找到对应的工具调用
		for j := i - 1; j >= 0; j-- {
			prevMsg := request.Messages[j]
			if prevMsg.Role == "assistant" && len(prevMsg.ToolCalls) > 0 {
				for _, tc := range prevMsg.ToolCalls {
					if tc.Function.Name == toolName {
						toolArgs = tc.Function.Arguments
						break
					}
				}
				if toolArgs != "" {
					break
				}
			}
		}

		// 使用改进的会话级保存或带工具上下文的保存
		var contextId string
		if toolName != "" && toolArgs != "" {
			// 有工具上下文，尝试使用带工具上下文的保存
			contextId, err = c.memoryService.SaveContextWithSession(ctx, contentStr, sessionId, toolCallId)
			if err != nil {
				log.Errorf("[CompressContext] failed to save context for message %d: %v, falling back to original content", i, err)
				// Graceful degradation: if compression fails, keep original content
				newMessages = append(newMessages, msg)
				continue
			}
			log.Debugf("[CompressContext] Message %d: saved context with tool context, tool: %s, callId: %s", i, toolName, toolCallId)
		} else {
			// 没有工具上下文，使用基本的会话级保存
			contextId, err = c.memoryService.SaveContextWithSession(ctx, contentStr, sessionId, "")
			if err != nil {
				log.Errorf("[CompressContext] failed to save context for message %d: %v, falling back to original content", i, err)
				// Graceful degradation: if compression fails, keep original content
				newMessages = append(newMessages, msg)
				continue
			}
		}

		// 跟踪工具调用链：保存当前工具的上下文信息
		toolInfo := ToolCallContextInfo{
			ContextId:   contextId,
			ToolName:    toolName,
			ToolCallId:  toolCallId,
			ToolArgs:    toolArgs,
			ToolOutput:  contentStr,
			SessionId:   sessionId,
			Order:       len(sessionToolChain.ToolCalls),
			DependsOnId: "",
		}
		// 版定会话中前一个工具调用的contextId（如果存在）
		if len(sessionToolChain.ToolCalls) > 0 {
			toolInfo.DependsOnId = sessionToolChain.ToolCalls[len(sessionToolChain.ToolCalls)-1].ContextId
		}
		sessionToolChain.ToolCalls = append(sessionToolChain.ToolCalls, toolInfo)
		toolContextMap[contextId] = toolInfo
		sessionToolChain.ContextIds[contextId] = len(sessionToolChain.ToolCalls) - 1

		// Step 2.3: 构建压缩消息
		compressedContent := fmt.Sprintf("[Context stored with ID: %s]", contextId)
		keyInfoStr := ""

		// 提取关键信息（如果启用）
		if c.contextCompression != nil && c.contextCompression.PreserveKeyInfo {
			if useLayeredCompression && layeredStrategy != nil {
				// 使用智能关键信息提取
				analyzer := NewContentAnalyzer()
				contentType, confidence := analyzer.AnalyzeContent(contentStr)
				extractor := NewSmartKeyExtractor()
				keyInfoStr = extractor.ExtractSmartKeyInfo(contentStr, contentType)
				log.Debugf("[CompressContext] Message %d: smart key extraction (type: %v, confidence: %d%%): %s",
					i, c.contentTypeString(contentType), confidence, keyInfoStr)
			} else {
				// 使用原有的关键信息提取
				keyInfoStr = extractKeyInfo(contentStr)
				log.Debugf("[CompressContext] preserved key info for message %d: %s", i, keyInfoStr)
			}

			if keyInfoStr != "" {
				compressedContent = fmt.Sprintf("[Context ID: %s]\nKey Info: %s", contextId, keyInfoStr)
			}
		}

		// Step 2.4: 替换消息内容
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
		savedBytes := contentSize - len(compressedMsg.StringContent())
		totalSavedBytes += savedBytes

		log.Infof("[CompressContext] Message %d: compressed (original: %d bytes, compressed: %d bytes, saved: %d bytes, layer: %s, reason: %s, context_id: %s)",
			i, contentSize, len(compressedMsg.StringContent()), savedBytes, dataLayer, compressionReason, contextId)
	}

	if len(compressedIds) == 0 && len(needRetrievalIds) == 0 {
		log.Debugf("[CompressContext] no messages compressed or retrieved")
		return body, nil
	}

	// Store compressed context IDs for later use
	if len(compressedIds) > 0 {
		ctx.SetContext(ctxKeyCompressedContextIds, compressedIds)
		ctx.SetContext(ctxKeyCompressionEnabled, true)
		// 保存会话级工具调用链，以便检索时恢复调用关系
		ctx.SetContext("session_tool_chain", sessionToolChain)
		ctx.SetContext("tool_context_map", toolContextMap)
		log.Infof("[CompressContext] total saved bytes: %d, compressed %d messages, tool chain length: %d", totalSavedBytes, len(compressedIds), len(sessionToolChain.ToolCalls))
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

	// 尝试从上下文中获取工具调用链信息，以恢复调用关系
	var toolChain *SessionToolCallChain
	if chainInterface := ctx.GetContext("session_tool_chain"); chainInterface != nil {
		if chain, ok := chainInterface.(*SessionToolCallChain); ok {
			toolChain = chain
		}
	}

	if toolChain != nil && len(toolChain.ToolCalls) > 0 {
		// 存在完整的工具调用链，盒示了其之一关系
		summaryBuilder.WriteString("\n### 工具调用顺序与依赖\n")
		for i, toolCall := range toolChain.ToolCalls {
			if i > 0 {
				summaryBuilder.WriteString("  ⮇\ufe0f\n")
			}
			summaryBuilder.WriteString(fmt.Sprintf("  第%d个: %s (ID: %s)\n", i+1, toolCall.ToolName, toolCall.ToolCallId))
		}
		summaryBuilder.WriteString("\n")
	}

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
// 支持基于完整工具调用链的摘要生成，以恢复上下文依赖关系
func (c *ProviderConfig) batchRetrieveContextSummariesConcurrent(ctx wrapper.HttpContext, toolCalls []toolCall) ([]chatMessage, error) {
	if len(toolCalls) == 0 {
		return nil, nil
	}

	// 从上下文中获取工具调用链信息以支持完整的摘要生成
	var toolChain *SessionToolCallChain
	var toolContextMap map[string]ToolCallContextInfo
	if chainInterface := ctx.GetContext("session_tool_chain"); chainInterface != nil {
		if chain, ok := chainInterface.(*SessionToolCallChain); ok {
			toolChain = chain
		}
	}
	if mapInterface := ctx.GetContext("tool_context_map"); mapInterface != nil {
		if m, ok := mapInterface.(map[string]ToolCallContextInfo); ok {
			toolContextMap = m
		}
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
				// 尝试使用简单摘要，或者基于完整调用链生成详细摘要
				var summary string
				var err error

				// 优先需要从头到尾需要一个按顺序的上下文ID映射，以便查找前一个工具
				// 当前：ReadContextSummary 整就返回存储的摘要
				// 未来改进：如果存在完整的工具调用链，可以动态生成摘要
				summary, err = c.memoryService.ReadContextSummary(ctx, contextId)
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
				// 记录措述生成情况，檀包括是否通过二段工具调用链输求的
				var chainInfo string
				if toolChain != nil && len(toolChain.ToolCalls) > 1 {
					chainInfo = fmt.Sprintf(", tool_chain_length=%d", len(toolChain.ToolCalls))
				}
				if toolContextMap != nil && len(toolContextMap) > 0 {
					log.Infof("[batchRetrieveContextSummariesConcurrent] retrieved summary for tool call %s, length: %d%s, total tools: %d", res.toolCall.Id, len(res.summary), chainInfo, len(toolContextMap))
				} else {
					log.Infof("[batchRetrieveContextSummariesConcurrent] retrieved summary for tool call %s, length: %d%s", res.toolCall.Id, len(res.summary), chainInfo)
				}
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
	filePathRegex := regexp.MustCompile(`(?:^|[\s
])(/[^\s
]+|\.\/[^\s
]+|[A-Z]:\\[^\s
]+)`)
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

// contentLayerString 返回数据优先级层级的字符串表示
func (c *ProviderConfig) contentLayerString(dl DataLayer) string {
	switch dl {
	case DataLayerCritical:
		return "Critical"
	case DataLayerImportant:
		return "Important"
	case DataLayerNormal:
		return "Normal"
	case DataLayerLow:
		return "Low"
	default:
		return "Unknown"
	}
}

// compressionStrategyString 返回压缩策略的字符串表示
func (c *ProviderConfig) compressionStrategyString(cs CompressionStrategy) string {
	switch cs {
	case CompressionStrategyNone:
		return "none"
	case CompressionStrategyConservative:
		return "conservative"
	case CompressionStrategyNormal:
		return "normal"
	case CompressionStrategyAggressive:
		return "aggressive"
	default:
		return "unknown"
	}
}

// contentTypeString 返回内容类型的字符串表示
func (c *ProviderConfig) contentTypeString(ct ContentType) string {
	switch ct {
	case ContentTypeMaze:
		return "Maze"
	case ContentTypeCode:
		return "Code"
	case ContentTypeJSON:
		return "JSON"
	case ContentTypeStructuredData:
		return "StructuredData"
	case ContentTypeText:
		return "Text"
	default:
		return "Unknown"
	}
}

// splitByNewline 分割字符串
func splitByNewline(s string) []string {
	return strings.Split(s, "\n")
}

// trimSpace 修剪空格
func trimSpace(s string) string {
	// 简单实现：去掉前后空格
	var start, end int
	for start = 0; start < len(s) && (s[start] == ' ' || s[start] == '\t'); start++ {
	}
	for end = len(s); end > start && (s[end-1] == ' ' || s[end-1] == '\t'); end-- {
	}
	if start >= end {
		return ""
	}
	return s[start:end]
}
