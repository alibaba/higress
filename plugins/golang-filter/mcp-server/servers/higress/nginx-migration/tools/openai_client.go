// Package tools provides nginx configuration migration tools for Higress
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIClient implements LLMClient interface using OpenAI API
type OpenAIClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey, baseURL string) *OpenAIClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// OpenAIRequest represents the request structure for OpenAI API
type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents the response structure from OpenAI API
type OpenAIResponse struct {
	Choices []Choice     `json:"choices"`
	Error   *OpenAIError `json:"error,omitempty"`
}

// Choice represents a choice in the response
type Choice struct {
	Message Message `json:"message"`
}

// OpenAIError represents an error from OpenAI API
type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// NginxAnalysis represents LLM analysis of nginx configuration
type NginxAnalysis struct {
	Complexity      string   `json:"complexity"`
	SecurityIssues  []string `json:"security_issues"`
	PerformanceTips []string `json:"performance_tips"`
	MigrationNotes  []string `json:"migration_notes"`
}

// GenerateResponse implements LLMClient interface
func (c *OpenAIClient) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	request := OpenAIRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert DevOps engineer specializing in Nginx and Higress configurations. Provide clear, actionable advice.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	response, err := c.makeRequest(ctx, "/chat/completions", request)
	if err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices received")
	}

	return response.Choices[0].Message.Content, nil
}

// AnalyzeNginxConfig implements LLMClient interface
func (c *OpenAIClient) AnalyzeNginxConfig(ctx context.Context, config string) (*NginxAnalysis, error) {
	prompt := fmt.Sprintf(`
Analyze this Nginx configuration and provide a structured analysis:

%s

Please analyze:
1. Complexity level (Simple/Medium/Complex)
2. Security issues (list any potential security concerns)
3. Performance tips (suggestions for optimization)
4. Migration notes (specific considerations for migrating to Higress)

Respond in JSON format:
{
  "complexity": "Simple/Medium/Complex",
  "security_issues": ["issue1", "issue2"],
  "performance_tips": ["tip1", "tip2"],
  "migration_notes": ["note1", "note2"]
}`, config)

	response, err := c.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Parse the JSON response
	var analysis NginxAnalysis
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		// If JSON parsing fails, create a basic analysis
		analysis = NginxAnalysis{
			Complexity:      "Medium",
			SecurityIssues:  []string{"Unable to parse AI response"},
			PerformanceTips: []string{"Review configuration manually"},
			MigrationNotes:  []string{"Standard migration process"},
		}
	}

	return &analysis, nil
}

// SuggestOptimizations implements LLMClient interface
func (c *OpenAIClient) SuggestOptimizations(ctx context.Context, config *NginxConfig) ([]string, error) {
	configJSON, _ := json.MarshalIndent(config, "", "  ")

	prompt := fmt.Sprintf(`
Based on this parsed Nginx configuration, suggest 5-7 specific optimizations before migrating to Higress:

%s

Focus on:
- Security improvements
- Performance optimizations
- Higress-specific considerations
- Best practices

Provide suggestions as a numbered list.`, string(configJSON))

	response, err := c.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Parse the numbered list into individual suggestions
	lines := strings.Split(response, "\n")
	var suggestions []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && (strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "2.") ||
			strings.HasPrefix(line, "3.") || strings.HasPrefix(line, "4.") ||
			strings.HasPrefix(line, "5.") || strings.HasPrefix(line, "6.") ||
			strings.HasPrefix(line, "7.")) {
			// Remove the number prefix
			if idx := strings.Index(line, "."); idx != -1 && idx+1 < len(line) {
				suggestion := strings.TrimSpace(line[idx+1:])
				if suggestion != "" {
					suggestions = append(suggestions, suggestion)
				}
			}
		}
	}

	if len(suggestions) == 0 {
		suggestions = []string{"No specific suggestions generated", "Review configuration manually"}
	}

	return suggestions, nil
}

// makeRequest makes an HTTP request to the OpenAI API
func (c *OpenAIClient) makeRequest(ctx context.Context, endpoint string, request interface{}) (*OpenAIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", response.Error.Message)
	}

	return &response, nil
}
