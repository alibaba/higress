package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

// ClaudeToOpenAIConverter converts Claude protocol requests to OpenAI protocol
type ClaudeToOpenAIConverter struct{}

// ConvertClaudeRequestToOpenAI converts a Claude chat completion request to OpenAI format
func (c *ClaudeToOpenAIConverter) ConvertClaudeRequestToOpenAI(body []byte) ([]byte, error) {
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
		openaiMsg := chatMessage{
			Role: claudeMsg.Role,
		}

		// Handle different content types
		switch content := claudeMsg.Content.(type) {
		case string:
			// Simple text content
			openaiMsg.Content = content
		case []claudeChatMessageContent:
			// Multi-modal content
			var openaiContents []chatMessageContent
			for _, claudeContent := range content {
				switch claudeContent.Type {
				case "text":
					openaiContents = append(openaiContents, chatMessageContent{
						Type: contentTypeText,
						Text: claudeContent.Text,
					})
				case "image":
					if claudeContent.Source != nil {
						if claudeContent.Source.Type == "base64" {
							// Convert base64 image to OpenAI format
							dataUrl := fmt.Sprintf("data:%s;base64,%s", claudeContent.Source.MediaType, claudeContent.Source.Data)
							openaiContents = append(openaiContents, chatMessageContent{
								Type: contentTypeImageUrl,
								ImageUrl: &chatMessageContentImageUrl{
									Url: dataUrl,
								},
							})
						} else if claudeContent.Source.Type == "url" {
							openaiContents = append(openaiContents, chatMessageContent{
								Type: contentTypeImageUrl,
								ImageUrl: &chatMessageContentImageUrl{
									Url: claudeContent.Source.Url,
								},
							})
						}
					}
				}
			}
			openaiMsg.Content = openaiContents
		}

		openaiRequest.Messages = append(openaiRequest.Messages, openaiMsg)
	}

	// Handle system message - Claude has separate system field
	if claudeRequest.System != "" {
		systemMsg := chatMessage{
			Role:    roleSystem,
			Content: claudeRequest.System,
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

	return json.Marshal(openaiRequest)
}

// ConvertOpenAIResponseToClaude converts an OpenAI response back to Claude format
func (c *ClaudeToOpenAIConverter) ConvertOpenAIResponseToClaude(ctx wrapper.HttpContext, body []byte) ([]byte, error) {
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
		Usage: claudeTextGenUsage{
			InputTokens:  openaiResponse.Usage.PromptTokens,
			OutputTokens: openaiResponse.Usage.CompletionTokens,
		},
	}

	// Convert the first choice content
	if len(openaiResponse.Choices) > 0 {
		choice := openaiResponse.Choices[0]
		if choice.Message != nil {
			content := claudeTextGenContent{
				Type: "text",
				Text: choice.Message.StringContent(),
			}
			claudeResponse.Content = []claudeTextGenContent{content}
		}

		// Convert finish reason
		if choice.FinishReason != nil {
			claudeFinishReason := openAIFinishReasonToClaude(*choice.FinishReason)
			claudeResponse.StopReason = &claudeFinishReason
		}
	}

	return json.Marshal(claudeResponse)
}

// ConvertOpenAIStreamResponseToClaude converts OpenAI streaming response to Claude format
func (c *ClaudeToOpenAIConverter) ConvertOpenAIStreamResponseToClaude(ctx wrapper.HttpContext, chunk []byte) ([]byte, error) {
	// For streaming responses, we need to handle the Server-Sent Events format
	lines := strings.Split(string(chunk), "\n")
	var result strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Skip [DONE] messages
			if data == "[DONE]" {
				continue
			}

			var openaiStreamResponse chatCompletionResponse
			if err := json.Unmarshal([]byte(data), &openaiStreamResponse); err != nil {
				log.Errorf("unable to unmarshal openai stream response: %v", err)
				continue
			}

			// Convert to Claude streaming format
			claudeStreamResponse := c.buildClaudeStreamResponse(ctx, &openaiStreamResponse)
			if claudeStreamResponse != nil {
				responseData, err := json.Marshal(claudeStreamResponse)
				if err != nil {
					log.Errorf("unable to marshal claude stream response: %v", err)
					continue
				}
				result.WriteString(fmt.Sprintf("data: %s\n\n", responseData))
			}
		}
	}

	return []byte(result.String()), nil
}

// buildClaudeStreamResponse builds a Claude streaming response from OpenAI streaming response
func (c *ClaudeToOpenAIConverter) buildClaudeStreamResponse(ctx wrapper.HttpContext, openaiResponse *chatCompletionResponse) *claudeTextGenStreamResponse {
	if len(openaiResponse.Choices) == 0 {
		return nil
	}

	choice := openaiResponse.Choices[0]

	// Determine the response type based on the content
	if choice.Delta != nil && choice.Delta.Content != "" {
		// Content delta
		if deltaContent, ok := choice.Delta.Content.(string); ok {
			return &claudeTextGenStreamResponse{
				Type:  "content_block_delta",
				Index: choice.Index,
				Delta: &claudeTextGenDelta{
					Type: "text_delta",
					Text: deltaContent,
				},
			}
		}
	} else if choice.FinishReason != nil {
		// Message completed
		claudeFinishReason := openAIFinishReasonToClaude(*choice.FinishReason)
		return &claudeTextGenStreamResponse{
			Type:  "message_delta",
			Index: choice.Index,
			Delta: &claudeTextGenDelta{
				Type:       "message_delta",
				StopReason: &claudeFinishReason,
			},
			Usage: &claudeTextGenUsage{
				InputTokens:  openaiResponse.Usage.PromptTokens,
				OutputTokens: openaiResponse.Usage.CompletionTokens,
			},
		}
	} else if choice.Delta != nil && choice.Delta.Role != "" {
		// Message start
		return &claudeTextGenStreamResponse{
			Type:  "message_start",
			Index: choice.Index,
			Message: &claudeTextGenResponse{
				Id:    openaiResponse.Id,
				Type:  "message",
				Role:  "assistant",
				Model: openaiResponse.Model,
				Usage: claudeTextGenUsage{
					InputTokens:  openaiResponse.Usage.PromptTokens,
					OutputTokens: 0,
				},
			},
		}
	}

	return nil
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
