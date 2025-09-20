package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/common"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
)

const (
	OPENAI_CHAT_ENDPOINT = "/chat/completions"
	OPENAI_DEFAULT_MODEL = "gpt-3.5-turbo"
)

// openAI specific configuration captured after initialization.
type openAIProviderConfig struct {
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
}

type openAIProviderInitializer struct{}

var openAIConfig openAIProviderConfig

func (i *openAIProviderInitializer) initConfig(c config.LLMConfig) {
	openAIConfig.apiKey = c.APIKey
	openAIConfig.baseURL = c.BaseURL
	openAIConfig.model = c.Model
	if openAIConfig.model == "" {
		openAIConfig.model = OPENAI_DEFAULT_MODEL
	}
	if openAIConfig.baseURL == "" {
		openAIConfig.baseURL = "https://api.openai.com/v1" // default public endpoint
	}
	openAIConfig.maxTokens = c.MaxTokens
	openAIConfig.temperature = c.Temperature
}

func (i *openAIProviderInitializer) validateConfig() error {
	if openAIConfig.apiKey == "" {
		return errors.New("[openai llm] apiKey is required")
	}
	return nil
}

func (i *openAIProviderInitializer) CreateProvider(cfg config.LLMConfig) (Provider, error) {
	i.initConfig(cfg)
	if err := i.validateConfig(); err != nil {
		return nil, err
	}
	headers := map[string]string{
		"Authorization": "Bearer " + openAIConfig.apiKey,
		"Content-Type":  "application/json",
	}
	client := common.NewHTTPClient(openAIConfig.baseURL, headers)
	return &OpenAIProvider{client: client, cfg: openAIConfig}, nil
}

type OpenAIProvider struct {
	client *common.HTTPClient
	cfg    openAIProviderConfig
}

type openAIChatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatCompletionResponse struct {
	ID      string                               `json:"id"`
	Object  string                               `json:"object"`
	Choices []openAIChatCompletionResponseChoice `json:"choices"`
	Error   *openAIError                         `json:"error,omitempty"`
}

type openAIChatCompletionResponseChoice struct {
	Index        int               `json:"index"`
	Message      openAIChatMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
	Param   string `json:"param"`
}

// GenerateCompletion implements Provider interface.
func (o *OpenAIProvider) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	req := openAIChatCompletionRequest{
		Model: o.cfg.model,
		Messages: []openAIChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: o.cfg.temperature,
		MaxTokens:   o.cfg.maxTokens,
	}

	body, err := o.client.Post(OPENAI_CHAT_ENDPOINT, req)
	if err != nil {
		return "", fmt.Errorf("openai llm post error: %w", err)
	}

	var resp openAIChatCompletionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("openai llm unmarshal error: %w", err)
	}

	if resp.Error != nil {
		return "", fmt.Errorf("openai llm api error: %s - %s", resp.Error.Type, resp.Error.Message)
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("openai llm: empty choices")
	}

	return resp.Choices[0].Message.Content, nil
}

func (o *OpenAIProvider) GetProviderType() string {
	return PROVIDER_TYPE_OPENAI
}
