package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// ClaudeToOpenAIConverter converts Claude protocol requests to OpenAI protocol
type ClaudeToOpenAIConverter struct {
	// State tracking for streaming conversion
	messageStartSent bool
	messageStopSent  bool
	messageId        string
	// Cache stop_reason until we get usage info
	pendingStopReason *string
	// Content block tracking with dynamic index allocation
	nextContentIndex     int
	thinkingBlockIndex   int
	thinkingBlockStarted bool
	thinkingBlockStopped bool
	textBlockIndex       int
	textBlockStarted     bool
	textBlockStopped     bool
	toolBlockIndex       int
	toolBlockStarted     bool
	toolBlockStopped     bool
	// Tool call state tracking
	toolCallStates map[string]*toolCallState
}

// contentConversionResult represents the result of converting Claude content to OpenAI format
type contentConversionResult struct {
	textParts         []string
	toolCalls         []toolCall
	toolResults       []claudeChatMessageContent
	openaiContents    []chatMessageContent
	hasNonTextContent bool
}

// toolCallState tracks the state of a tool call during streaming
type toolCallState struct {
	id              string
	name            string
	argumentsBuffer string
	isComplete      bool
}

// ConvertClaudeRequestToOpenAI converts a Claude chat completion request to OpenAI format
func (c *ClaudeToOpenAIConverter) ConvertClaudeRequestToOpenAI(body []byte) ([]byte, error) {
	log.Debugf("[Claude->OpenAI] Original Claude request body: %s", string(body))

	var claudeRequest claudeTextGenRequest
	if err := json.Unmarshal(body, &claudeRequest); err != nil {
		return nil, fmt.Errorf("unable to unmarshal claude request: %v", err)
	}

	// Convert Claude request to OpenAI format
	openaiRequest := chatCompletionRequest{
		Model:       claudeRequest.Model,
		Stream:      claudeRequest.Stream,
		Temperature: claudeRequest.Temperature,
		TopP:        claudeRequest.TopP,
		MaxTokens:   claudeRequest.MaxTokens,
		Stop:        claudeRequest.StopSequences,
	}

	// Convert messages from Claude format to OpenAI format
	for _, claudeMsg := range claudeRequest.Messages {
		// Handle different content types using the type-safe wrapper
		if claudeMsg.Content.IsString {
			// Simple text content
			openaiMsg := chatMessage{
				Role:    claudeMsg.Role,
				Content: claudeMsg.Content.GetStringValue(),
			}
			openaiRequest.Messages = append(openaiRequest.Messages, openaiMsg)
		} else {
			// Multi-modal content - process with convertContentArray
			conversionResult := c.convertContentArray(claudeMsg.Content.GetArrayValue())

			// Handle tool calls if present
			if len(conversionResult.toolCalls) > 0 {
				// Use tool_calls format (current OpenAI standard)
				openaiMsg := chatMessage{
					Role:      claudeMsg.Role,
					ToolCalls: conversionResult.toolCalls,
				}

				// Add text content if present, otherwise set to null
				if len(conversionResult.textParts) > 0 {
					openaiMsg.Content = strings.Join(conversionResult.textParts, "\n\n")
				} else {
					openaiMsg.Content = nil
				}

				openaiRequest.Messages = append(openaiRequest.Messages, openaiMsg)
			}

			// Handle tool results if present
			if len(conversionResult.toolResults) > 0 {
				for _, toolResult := range conversionResult.toolResults {
					toolMsg := chatMessage{
						Role:       "tool",
						Content:    toolResult.Content,
						ToolCallId: toolResult.ToolUseId,
					}
					openaiRequest.Messages = append(openaiRequest.Messages, toolMsg)
				}
			}

			// Handle regular content if no tool calls or tool results
			if len(conversionResult.toolCalls) == 0 && len(conversionResult.toolResults) == 0 {
				var content interface{}
				if !conversionResult.hasNonTextContent && len(conversionResult.textParts) > 0 {
					// Simple text content
					content = strings.Join(conversionResult.textParts, "\n\n")
				} else {
					// Multi-modal content or empty content
					content = conversionResult.openaiContents
				}

				openaiMsg := chatMessage{
					Role:    claudeMsg.Role,
					Content: content,
				}
				openaiRequest.Messages = append(openaiRequest.Messages, openaiMsg)
			}
		}
	}

	// Handle system message - Claude has separate system field
	systemStr := claudeRequest.System.String()
	if systemStr != "" {
		systemMsg := chatMessage{
			Role:    roleSystem,
			Content: systemStr,
		}
		// Insert system message at the beginning
		openaiRequest.Messages = append([]chatMessage{systemMsg}, openaiRequest.Messages...)
	}

	// Convert tools if present
	for _, claudeTool := range claudeRequest.Tools {
		openaiTool := tool{
			Type: "function",
			Function: function{
				Name:        claudeTool.Name,
				Description: claudeTool.Description,
				Parameters:  claudeTool.InputSchema,
			},
		}
		openaiRequest.Tools = append(openaiRequest.Tools, openaiTool)
	}

	// Convert tool choice if present
	if claudeRequest.ToolChoice != nil {
		if claudeRequest.ToolChoice.Type == "tool" && claudeRequest.ToolChoice.Name != "" {
			openaiRequest.ToolChoice = &toolChoice{
				Type: "function",
				Function: function{
					Name: claudeRequest.ToolChoice.Name,
				},
			}
		} else {
			// For other types like "auto", "none", etc.
			openaiRequest.ToolChoice = claudeRequest.ToolChoice.Type
		}

		// Handle parallel tool calls
		openaiRequest.ParallelToolCalls = !claudeRequest.ToolChoice.DisableParallelToolUse
	}

	// Convert thinking configuration if present
	if claudeRequest.Thinking != nil {
		log.Debugf("[Claude->OpenAI] Found thinking config: type=%s, budget_tokens=%d",
			claudeRequest.Thinking.Type, claudeRequest.Thinking.BudgetTokens)

		if claudeRequest.Thinking.Type == "enabled" {
			openaiRequest.ReasoningMaxTokens = claudeRequest.Thinking.BudgetTokens

			// Set ReasoningEffort based on budget_tokens
			// low: <4096, medium: >=4096 and <16384, high: >=16384
			if claudeRequest.Thinking.BudgetTokens < 4096 {
				openaiRequest.ReasoningEffort = "low"
			} else if claudeRequest.Thinking.BudgetTokens < 16384 {
				openaiRequest.ReasoningEffort = "medium"
			} else {
				openaiRequest.ReasoningEffort = "high"
			}

			log.Debugf("[Claude->OpenAI] Converted thinking config: budget_tokens=%d, reasoning_effort=%s, reasoning_max_tokens=%d",
				claudeRequest.Thinking.BudgetTokens, openaiRequest.ReasoningEffort, openaiRequest.ReasoningMaxTokens)
		}
	} else {
		log.Debugf("[Claude->OpenAI] No thinking config found")
	}

	result, err := json.Marshal(openaiRequest)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal openai request: %v", err)
	}

	log.Debugf("[Claude->OpenAI] Converted OpenAI request body: %s", string(result))
	return result, nil
}

// ConvertOpenAIResponseToClaude converts an OpenAI response back to Claude format
func (c *ClaudeToOpenAIConverter) ConvertOpenAIResponseToClaude(ctx wrapper.HttpContext, body []byte) ([]byte, error) {
	log.Debugf("[OpenAI->Claude] Original OpenAI response body: %s", string(body))

	var openaiResponse chatCompletionResponse
	if err := json.Unmarshal(body, &openaiResponse); err != nil {
		return nil, fmt.Errorf("unable to unmarshal openai response: %v", err)
	}

	// Convert OpenAI response to Claude format
	claudeResponse := claudeTextGenResponse{
		Id:    openaiResponse.Id,
		Type:  "message",
		Role:  "assistant",
		Model: openaiResponse.Model,
	}

	// Only include usage if it's available
	if openaiResponse.Usage != nil {
		claudeResponse.Usage = claudeTextGenUsage{
			InputTokens:  openaiResponse.Usage.PromptTokens,
			OutputTokens: openaiResponse.Usage.CompletionTokens,
		}
	}

	// Convert the first choice content
	if len(openaiResponse.Choices) > 0 {
		choice := openaiResponse.Choices[0]
		if choice.Message != nil {
			var contents []claudeTextGenContent

			// Add reasoning content (thinking) if present - check both reasoning and reasoning_content fields
			var reasoningText string
			if choice.Message.Reasoning != "" {
				reasoningText = choice.Message.Reasoning
			} else if choice.Message.ReasoningContent != "" {
				reasoningText = choice.Message.ReasoningContent
			}

			if reasoningText != "" {
				contents = append(contents, claudeTextGenContent{
					Type:      "thinking",
					Signature: "", // OpenAI doesn't provide signature, use empty string
					Thinking:  reasoningText,
				})
				log.Debugf("[OpenAI->Claude] Added thinking content: %s", reasoningText)
			}

			// Add text content if present
			if choice.Message.StringContent() != "" {
				contents = append(contents, claudeTextGenContent{
					Type: "text",
					Text: choice.Message.StringContent(),
				})
			}

			// Add tool calls if present
			if len(choice.Message.ToolCalls) > 0 {
				for _, toolCall := range choice.Message.ToolCalls {
					if !toolCall.Function.IsEmpty() {
						// Parse arguments from JSON string to map
						var input map[string]interface{}
						if toolCall.Function.Arguments != "" {
							if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &input); err != nil {
								log.Errorf("Failed to parse tool call arguments: %v", err)
								input = map[string]interface{}{}
							}
						} else {
							input = map[string]interface{}{}
						}

						contents = append(contents, claudeTextGenContent{
							Type:  "tool_use",
							Id:    toolCall.Id,
							Name:  toolCall.Function.Name,
							Input: input,
						})
					}
				}
			}

			claudeResponse.Content = contents
		}

		// Convert finish reason
		if choice.FinishReason != nil {
			claudeFinishReason := openAIFinishReasonToClaude(*choice.FinishReason)
			claudeResponse.StopReason = &claudeFinishReason
		}
	}

	result, err := json.Marshal(claudeResponse)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal claude response: %v", err)
	}

	log.Debugf("[OpenAI->Claude] Converted Claude response body: %s", string(result))
	return result, nil
}

// ConvertOpenAIStreamResponseToClaude converts OpenAI streaming response to Claude format
func (c *ClaudeToOpenAIConverter) ConvertOpenAIStreamResponseToClaude(ctx wrapper.HttpContext, chunk []byte) ([]byte, error) {
	log.Debugf("[OpenAI->Claude] Original OpenAI streaming chunk: %s", string(chunk))

	// Initialize tool call states if needed
	if c.toolCallStates == nil {
		c.toolCallStates = make(map[string]*toolCallState)
	}

	// For streaming responses, we need to handle the Server-Sent Events format
	lines := strings.Split(string(chunk), "\n")
	var result strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Handle [DONE] messages
			if data == "[DONE]" {
				log.Debugf("[OpenAI->Claude] Processing [DONE] message, finalizing stream")

				// Send final content_block_stop events for any active blocks
				if c.thinkingBlockStarted && !c.thinkingBlockStopped {
					c.thinkingBlockStopped = true
					log.Debugf("[OpenAI->Claude] Sending final thinking content_block_stop event at index %d", c.thinkingBlockIndex)
					stopEvent := &claudeTextGenStreamResponse{
						Type:  "content_block_stop",
						Index: &c.thinkingBlockIndex,
					}
					stopData, _ := json.Marshal(stopEvent)
					result.WriteString(fmt.Sprintf("data: %s\n\n", stopData))
				}
				if c.textBlockStarted && !c.textBlockStopped {
					c.textBlockStopped = true
					log.Debugf("[OpenAI->Claude] Sending final text content_block_stop event at index %d", c.textBlockIndex)
					stopEvent := &claudeTextGenStreamResponse{
						Type:  "content_block_stop",
						Index: &c.textBlockIndex,
					}
					stopData, _ := json.Marshal(stopEvent)
					result.WriteString(fmt.Sprintf("data: %s\n\n", stopData))
				}
				if c.toolBlockStarted && !c.toolBlockStopped {
					c.toolBlockStopped = true
					log.Debugf("[OpenAI->Claude] Sending final tool content_block_stop event at index %d", c.toolBlockIndex)
					stopEvent := &claudeTextGenStreamResponse{
						Type:  "content_block_stop",
						Index: &c.toolBlockIndex,
					}
					stopData, _ := json.Marshal(stopEvent)
					result.WriteString(fmt.Sprintf("data: %s\n\n", stopData))
				}

				// If we have a pending stop_reason but no usage, send message_delta with just stop_reason
				if c.pendingStopReason != nil {
					log.Debugf("[OpenAI->Claude] Sending final message_delta with pending stop_reason: %s", *c.pendingStopReason)
					messageDelta := &claudeTextGenStreamResponse{
						Type: "message_delta",
						Delta: &claudeTextGenDelta{
							Type:       "message_delta",
							StopReason: c.pendingStopReason,
						},
					}
					stopData, _ := json.Marshal(messageDelta)
					result.WriteString(fmt.Sprintf("data: %s\n\n", stopData))
					c.pendingStopReason = nil
				}

				if c.messageStartSent && !c.messageStopSent {
					c.messageStopSent = true
					log.Debugf("[OpenAI->Claude] Sending final message_stop event")
					messageStopEvent := &claudeTextGenStreamResponse{
						Type: "message_stop",
					}
					stopData, _ := json.Marshal(messageStopEvent)
					result.WriteString(fmt.Sprintf("data: %s\n\n", stopData))
				}

				// Reset all state for next request
				c.messageStartSent = false
				c.messageStopSent = false
				c.messageId = ""
				c.pendingStopReason = nil
				c.nextContentIndex = 0
				c.thinkingBlockIndex = -1
				c.thinkingBlockStarted = false
				c.thinkingBlockStopped = false
				c.textBlockIndex = -1
				c.textBlockStarted = false
				c.textBlockStopped = false
				c.toolBlockIndex = -1
				c.toolBlockStarted = false
				c.toolBlockStopped = false
				c.toolCallStates = make(map[string]*toolCallState)
				log.Debugf("[OpenAI->Claude] Reset converter state for next request")

				continue
			}

			var openaiStreamResponse chatCompletionResponse
			if err := json.Unmarshal([]byte(data), &openaiStreamResponse); err != nil {
				log.Debugf("unable to unmarshal openai stream response: %v, data: %s", err, data)
				continue
			}

			// Convert to Claude streaming format
			claudeStreamResponses := c.buildClaudeStreamResponse(ctx, &openaiStreamResponse)
			log.Debugf("[OpenAI->Claude] Generated %d Claude stream events from OpenAI chunk", len(claudeStreamResponses))

			for i, claudeStreamResponse := range claudeStreamResponses {
				responseData, err := json.Marshal(claudeStreamResponse)
				if err != nil {
					log.Errorf("unable to marshal claude stream response: %v", err)
					continue
				}
				log.Debugf("[OpenAI->Claude] Stream event [%d/%d]: %s", i+1, len(claudeStreamResponses), string(responseData))
				result.WriteString(fmt.Sprintf("data: %s\n\n", responseData))
			}
		}
	}

	claudeChunk := []byte(result.String())
	log.Debugf("[OpenAI->Claude] Converted Claude streaming chunk: %s", string(claudeChunk))
	return claudeChunk, nil
}

// buildClaudeStreamResponse builds Claude streaming responses from OpenAI streaming response
func (c *ClaudeToOpenAIConverter) buildClaudeStreamResponse(ctx wrapper.HttpContext, openaiResponse *chatCompletionResponse) []*claudeTextGenStreamResponse {
	if len(openaiResponse.Choices) == 0 {
		log.Debugf("[OpenAI->Claude] No choices in OpenAI response, skipping")
		return nil
	}

	choice := openaiResponse.Choices[0]
	var responses []*claudeTextGenStreamResponse

	// Log what we're processing
	hasRole := choice.Delta != nil && choice.Delta.Role != ""
	hasContent := choice.Delta != nil && choice.Delta.Content != ""
	hasFinishReason := choice.FinishReason != nil
	hasUsage := openaiResponse.Usage != nil

	log.Debugf("[OpenAI->Claude] Processing OpenAI chunk - Role: %v, Content: %v, FinishReason: %v, Usage: %v",
		hasRole, hasContent, hasFinishReason, hasUsage)

	// Handle message start (only once)
	// Note: OpenRouter may send multiple messages with role but empty content at the start
	// We only send message_start for the first one
	if choice.Delta != nil && choice.Delta.Role != "" && !c.messageStartSent {
		c.messageId = openaiResponse.Id
		c.messageStartSent = true

		message := &claudeTextGenResponse{
			Id:      openaiResponse.Id,
			Type:    "message",
			Role:    "assistant",
			Model:   openaiResponse.Model,
			Content: []claudeTextGenContent{},
		}

		// Only include usage if it's available
		if openaiResponse.Usage != nil {
			message.Usage = claudeTextGenUsage{
				InputTokens:  openaiResponse.Usage.PromptTokens,
				OutputTokens: 0,
			}
		}

		responses = append(responses, &claudeTextGenStreamResponse{
			Type:    "message_start",
			Message: message,
		})

		log.Debugf("[OpenAI->Claude] Generated message_start event for id: %s", openaiResponse.Id)
	} else if choice.Delta != nil && choice.Delta.Role != "" && c.messageStartSent {
		// Skip duplicate role messages from OpenRouter
		log.Debugf("[OpenAI->Claude] Skipping duplicate role message for id: %s", openaiResponse.Id)
	}

	// Handle reasoning content (thinking) first - check both reasoning and reasoning_content fields
	var reasoningText string
	if choice.Delta != nil {
		if choice.Delta.Reasoning != "" {
			reasoningText = choice.Delta.Reasoning
		} else if choice.Delta.ReasoningContent != "" {
			reasoningText = choice.Delta.ReasoningContent
		}
	}

	if reasoningText != "" {
		log.Debugf("[OpenAI->Claude] Processing reasoning content delta: %s", reasoningText)

		// Send content_block_start for thinking only once with dynamic index
		if !c.thinkingBlockStarted {
			c.thinkingBlockIndex = c.nextContentIndex
			c.nextContentIndex++
			c.thinkingBlockStarted = true
			log.Debugf("[OpenAI->Claude] Generated content_block_start event for thinking at index %d", c.thinkingBlockIndex)
			responses = append(responses, &claudeTextGenStreamResponse{
				Type:  "content_block_start",
				Index: &c.thinkingBlockIndex,
				ContentBlock: &claudeTextGenContent{
					Type:      "thinking",
					Signature: "", // OpenAI doesn't provide signature
					Thinking:  "",
				},
			})
		}

		// Send content_block_delta for thinking
		log.Debugf("[OpenAI->Claude] Generated content_block_delta event with thinking: %s", reasoningText)
		responses = append(responses, &claudeTextGenStreamResponse{
			Type:  "content_block_delta",
			Index: &c.thinkingBlockIndex,
			Delta: &claudeTextGenDelta{
				Type: "thinking_delta", // Use thinking_delta for reasoning content
				Text: reasoningText,
			},
		})
	}

	// Handle content
	if choice.Delta != nil && choice.Delta.Content != nil && choice.Delta.Content != "" {
		deltaContent, ok := choice.Delta.Content.(string)
		if !ok {
			log.Debugf("[OpenAI->Claude] Content is not a string: %T", choice.Delta.Content)
			return responses
		}

		log.Debugf("[OpenAI->Claude] Processing content delta: %s", deltaContent)

		// Close thinking content block if it's still open
		if c.thinkingBlockStarted && !c.thinkingBlockStopped {
			c.thinkingBlockStopped = true
			log.Debugf("[OpenAI->Claude] Closing thinking content block before text")
			responses = append(responses, &claudeTextGenStreamResponse{
				Type:  "content_block_stop",
				Index: &c.thinkingBlockIndex,
			})
		}

		// Send content_block_start only once for text content with dynamic index
		if !c.textBlockStarted {
			c.textBlockIndex = c.nextContentIndex
			c.nextContentIndex++
			c.textBlockStarted = true
			log.Debugf("[OpenAI->Claude] Generated content_block_start event for text at index %d", c.textBlockIndex)
			responses = append(responses, &claudeTextGenStreamResponse{
				Type:  "content_block_start",
				Index: &c.textBlockIndex,
				ContentBlock: &claudeTextGenContent{
					Type: "text",
					Text: "",
				},
			})
		}

		// Send content_block_delta
		log.Debugf("[OpenAI->Claude] Generated content_block_delta event with text: %s", deltaContent)
		responses = append(responses, &claudeTextGenStreamResponse{
			Type:  "content_block_delta",
			Index: &c.textBlockIndex,
			Delta: &claudeTextGenDelta{
				Type: "text_delta",
				Text: deltaContent,
			},
		})
	}

	// Handle tool calls in streaming response
	if choice.Delta != nil && len(choice.Delta.ToolCalls) > 0 {
		for _, toolCall := range choice.Delta.ToolCalls {
			if !toolCall.Function.IsEmpty() {
				log.Debugf("[OpenAI->Claude] Processing tool call delta")

				// Get or create tool call state
				state := c.toolCallStates[toolCall.Id]
				if state == nil {
					state = &toolCallState{
						id:              toolCall.Id,
						name:            toolCall.Function.Name,
						argumentsBuffer: "",
						isComplete:      false,
					}
					c.toolCallStates[toolCall.Id] = state
					log.Debugf("[OpenAI->Claude] Created new tool call state for id: %s, name: %s", toolCall.Id, toolCall.Function.Name)
				}

				// Accumulate arguments
				if toolCall.Function.Arguments != "" {
					state.argumentsBuffer += toolCall.Function.Arguments
					log.Debugf("[OpenAI->Claude] Accumulated tool arguments: %s", state.argumentsBuffer)
				}

				// Try to parse accumulated arguments as JSON to check if complete
				var input map[string]interface{}
				if state.argumentsBuffer != "" {
					if err := json.Unmarshal([]byte(state.argumentsBuffer), &input); err == nil {
						// Successfully parsed - arguments are complete
						if !state.isComplete {
							state.isComplete = true
							log.Debugf("[OpenAI->Claude] Tool call arguments complete for %s: %s", state.name, state.argumentsBuffer)

							// Close thinking content block if it's still open
							if c.thinkingBlockStarted && !c.thinkingBlockStopped {
								c.thinkingBlockStopped = true
								log.Debugf("[OpenAI->Claude] Closing thinking content block before tool use")
								responses = append(responses, &claudeTextGenStreamResponse{
									Type:  "content_block_stop",
									Index: &c.thinkingBlockIndex,
								})
							}

							// Close text content block if it's still open
							if c.textBlockStarted && !c.textBlockStopped {
								c.textBlockStopped = true
								log.Debugf("[OpenAI->Claude] Closing text content block before tool use")
								responses = append(responses, &claudeTextGenStreamResponse{
									Type:  "content_block_stop",
									Index: &c.textBlockIndex,
								})
							}

							// Send content_block_start for tool_use only when we have complete arguments with dynamic index
							if !c.toolBlockStarted {
								c.toolBlockIndex = c.nextContentIndex
								c.nextContentIndex++
								c.toolBlockStarted = true
								log.Debugf("[OpenAI->Claude] Generated content_block_start event for tool_use at index %d", c.toolBlockIndex)
								responses = append(responses, &claudeTextGenStreamResponse{
									Type:  "content_block_start",
									Index: &c.toolBlockIndex,
									ContentBlock: &claudeTextGenContent{
										Type:  "tool_use",
										Id:    toolCall.Id,
										Name:  state.name,
										Input: input,
									},
								})
							}
						}
					} else {
						// Still accumulating arguments
						log.Debugf("[OpenAI->Claude] Tool arguments not yet complete, continuing to accumulate: %v", err)
					}
				}
			}
		}
	}

	// Handle finish reason
	if choice.FinishReason != nil {
		claudeFinishReason := openAIFinishReasonToClaude(*choice.FinishReason)
		log.Debugf("[OpenAI->Claude] Processing finish_reason: %s -> %s", *choice.FinishReason, claudeFinishReason)

		// Send content_block_stop for any active content blocks
		if c.thinkingBlockStarted && !c.thinkingBlockStopped {
			c.thinkingBlockStopped = true
			log.Debugf("[OpenAI->Claude] Generated thinking content_block_stop event at index %d", c.thinkingBlockIndex)
			responses = append(responses, &claudeTextGenStreamResponse{
				Type:  "content_block_stop",
				Index: &c.thinkingBlockIndex,
			})
		}
		if c.textBlockStarted && !c.textBlockStopped {
			c.textBlockStopped = true
			log.Debugf("[OpenAI->Claude] Generated text content_block_stop event at index %d", c.textBlockIndex)
			responses = append(responses, &claudeTextGenStreamResponse{
				Type:  "content_block_stop",
				Index: &c.textBlockIndex,
			})
		}
		if c.toolBlockStarted && !c.toolBlockStopped {
			c.toolBlockStopped = true
			log.Debugf("[OpenAI->Claude] Generated tool content_block_stop event at index %d", c.toolBlockIndex)
			responses = append(responses, &claudeTextGenStreamResponse{
				Type:  "content_block_stop",
				Index: &c.toolBlockIndex,
			})
		}

		// Cache stop_reason until we get usage info (Claude protocol requires them together)
		c.pendingStopReason = &claudeFinishReason
		log.Debugf("[OpenAI->Claude] Cached stop_reason: %s, waiting for usage", claudeFinishReason)
	}

	// Handle usage information
	if openaiResponse.Usage != nil && choice.FinishReason == nil {
		log.Debugf("[OpenAI->Claude] Processing usage info - input: %d, output: %d",
			openaiResponse.Usage.PromptTokens, openaiResponse.Usage.CompletionTokens)

		// Send message_delta with both stop_reason and usage (Claude protocol requirement)
		messageDelta := &claudeTextGenStreamResponse{
			Type: "message_delta",
			Delta: &claudeTextGenDelta{
				Type: "message_delta",
			},
			Usage: &claudeTextGenUsage{
				InputTokens:  openaiResponse.Usage.PromptTokens,
				OutputTokens: openaiResponse.Usage.CompletionTokens,
			},
		}

		// Include cached stop_reason if available
		if c.pendingStopReason != nil {
			log.Debugf("[OpenAI->Claude] Combining cached stop_reason %s with usage", *c.pendingStopReason)
			messageDelta.Delta.StopReason = c.pendingStopReason
			c.pendingStopReason = nil // Clear cache
		}

		log.Debugf("[OpenAI->Claude] Generated message_delta event with usage and stop_reason")
		responses = append(responses, messageDelta)

		// Send message_stop after combined message_delta
		if !c.messageStopSent {
			c.messageStopSent = true
			log.Debugf("[OpenAI->Claude] Generated message_stop event")
			responses = append(responses, &claudeTextGenStreamResponse{
				Type: "message_stop",
			})
		}
	}

	return responses
}

// openAIFinishReasonToClaude converts OpenAI finish reason to Claude format
func openAIFinishReasonToClaude(reason string) string {
	switch reason {
	case finishReasonStop:
		return "end_turn"
	case finishReasonLength:
		return "max_tokens"
	case finishReasonToolCall:
		return "tool_use"
	default:
		return reason
	}
}

// convertContentArray converts an array of Claude content to OpenAI content format
func (c *ClaudeToOpenAIConverter) convertContentArray(claudeContents []claudeChatMessageContent) *contentConversionResult {
	result := &contentConversionResult{
		textParts:         []string{},
		toolCalls:         []toolCall{},
		toolResults:       []claudeChatMessageContent{},
		openaiContents:    []chatMessageContent{},
		hasNonTextContent: false,
	}

	for _, claudeContent := range claudeContents {
		switch claudeContent.Type {
		case "text":
			if claudeContent.Text != "" {
				result.textParts = append(result.textParts, claudeContent.Text)
				result.openaiContents = append(result.openaiContents, chatMessageContent{
					Type: contentTypeText,
					Text: claudeContent.Text,
				})
			}
		case "image":
			result.hasNonTextContent = true
			if claudeContent.Source != nil {
				if claudeContent.Source.Type == "base64" {
					// Convert base64 image to OpenAI format
					dataUrl := fmt.Sprintf("data:%s;base64,%s", claudeContent.Source.MediaType, claudeContent.Source.Data)
					result.openaiContents = append(result.openaiContents, chatMessageContent{
						Type: contentTypeImageUrl,
						ImageUrl: &chatMessageContentImageUrl{
							Url: dataUrl,
						},
					})
				} else if claudeContent.Source.Type == "url" {
					result.openaiContents = append(result.openaiContents, chatMessageContent{
						Type: contentTypeImageUrl,
						ImageUrl: &chatMessageContentImageUrl{
							Url: claudeContent.Source.Url,
						},
					})
				}
			}
		case "tool_use":
			result.hasNonTextContent = true
			// Convert Claude tool_use to OpenAI tool_calls format
			if claudeContent.Id != "" && claudeContent.Name != "" {
				// Convert input to JSON string for OpenAI format
				var argumentsStr string
				if claudeContent.Input != nil {
					if argBytes, err := json.Marshal(claudeContent.Input); err == nil {
						argumentsStr = string(argBytes)
					}
				}

				toolCall := toolCall{
					Id:   claudeContent.Id,
					Type: "function",
					Function: functionCall{
						Name:      claudeContent.Name,
						Arguments: argumentsStr,
					},
				}
				result.toolCalls = append(result.toolCalls, toolCall)
				log.Debugf("[Claude->OpenAI] Converted tool_use to tool_call: %s", claudeContent.Name)
			}
		case "tool_result":
			result.hasNonTextContent = true
			// Store tool results for processing
			result.toolResults = append(result.toolResults, claudeContent)
			log.Debugf("[Claude->OpenAI] Found tool_result for tool_use_id: %s", claudeContent.ToolUseId)
		}
	}

	return result
}
