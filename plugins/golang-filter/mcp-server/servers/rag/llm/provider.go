package llm

import (
	"context"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
)

const (
	// OpenAI LLM provider
	PROVIDER_TYPE_OPENAI = "openai"
	// More providers can be added (e.g., Qwen)
)

// Provider defines interface for LLM providers with prompt-response pattern.
// Extensible for future chat-style and streaming features.
type Provider interface {
	// Returns provider type for registration and lookup
	GetProviderType() string

	// Generates text response for given prompt
	//
	// ctx: For cancellation and timeout
	// prompt: Input text
	// Returns: Generated response and error if any
	GenerateCompletion(ctx context.Context, prompt string) (string, error)
}

// Factory interface for creating Provider instances
type providerInitializer interface {
	// Creates Provider with given config
	CreateProvider(config.LLMConfig) (Provider, error)
}

// Maps provider types to initializers
var (
	providerInitializers = map[string]providerInitializer{
		PROVIDER_TYPE_OPENAI: &openAIProviderInitializer{},
	}
)

// Creates Provider instance based on config
//
// cfg: Provider config
// Returns: Provider instance and error if any
func NewLLMProvider(cfg config.LLMConfig) (Provider, error) {
	initializer, ok := providerInitializers[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("no initializer found for llm provider type: %s", cfg.Provider)
	}
	return initializer.CreateProvider(cfg)
}
