package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/packages/param"
)

const (
	OPENAI_DEFAULT_MODEL = "gpt-4o"
)

type OpenAIProvider struct {
	client      *openai.Client
	model       string
	temperature float64
	maxTokens   int
}

type openAIProviderInitializer struct{}

func (i *openAIProviderInitializer) validateConfig(cfg *config.LLMConfig) error {
	if cfg.APIKey == "" {
		return errors.New("[openai llm] apiKey is required")
	}
	if cfg.Model == "" {
		cfg.Model = OPENAI_DEFAULT_MODEL
	}

	if cfg.Temperature <= 0 || cfg.Temperature > 2 {
		cfg.Temperature = 0.5
	}

	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 2048
	}
	return nil
}

func (i *openAIProviderInitializer) CreateProvider(cfg config.LLMConfig) (Provider, error) {
	if err := i.validateConfig(&cfg); err != nil {
		return nil, err
	}
	// Create OpenAI client
	var clientOptions []option.RequestOption
	clientOptions = append(clientOptions, option.WithAPIKey(cfg.APIKey))

	// If a custom baseURL is set, use it
	if cfg.BaseURL != "" {
		clientOptions = append(clientOptions, option.WithBaseURL(cfg.BaseURL))
	}

	// Create OpenAI client
	client := openai.NewClient(clientOptions...)

	return &OpenAIProvider{
		client:      &client,
		model:       cfg.Model,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
	}, nil
}

// GenerateCompletion implements Provider interface.
func (o *OpenAIProvider) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	// Create chat request
	params := openai.ChatCompletionNewParams{
		Model: o.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	}

	// Set optional parameters
	if o.temperature > 0 {
		temperature := float64(o.temperature)
		params.Temperature = param.Opt[float64]{Value: temperature}
	}

	if o.maxTokens > 0 {
		maxTokens := int64(o.maxTokens)
		params.MaxTokens = param.Opt[int64]{Value: maxTokens}
	}

	// Send request
	response, err := o.client.Chat.Completions.New(ctx, params)
	if err != nil {
		// Handle error
		return "", fmt.Errorf("openai llm error: %w", err)
	}

	// Check response
	if len(response.Choices) == 0 {
		return "", errors.New("openai llm: empty choices")
	}

	// Return generated content
	return response.Choices[0].Message.Content, nil
}

func (o *OpenAIProvider) GetProviderType() string {
	return PROVIDER_TYPE_OPENAI
}
