package embedding

import (
	"context"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
)

// Provider type constants for different embedding services
const (
	// DashScope embedding service
	PROVIDER_TYPE_DASHSCOPE   = "dashscope"
	// TextIn embedding service
	PROVIDER_TYPE_TEXTIN      = "textin"
	// Cohere embedding service
	PROVIDER_TYPE_COHERE      = "cohere"
	// OpenAI embedding service
	PROVIDER_TYPE_OPENAI      = "openai"
	// Ollama embedding service
	PROVIDER_TYPE_OLLAMA      = "ollama"
	// HuggingFace embedding service
	PROVIDER_TYPE_HUGGINGFACE = "huggingface"
	// XFYun embedding service
	PROVIDER_TYPE_XFYUN       = "xfyun"
	// Azure embedding service
	PROVIDER_TYPE_AZURE       = "azure"
)

// Factory interface for creating Provider instances
type providerInitializer interface {
	// Creates a new Provider with the given configuration
	CreateProvider(config.EmbeddingConfig) (Provider, error)
}

// Maps provider types to their initializers
var (
	providerInitializers = map[string]providerInitializer{
		PROVIDER_TYPE_DASHSCOPE: &dashScopeProviderInitializer{},
		PROVIDER_TYPE_OPENAI:    &openAIProviderInitializer{},
	}
)

// Provider defines the interface for embedding services
type Provider interface {
	// Returns the provider type identifier
	GetProviderType() string
	// Generates embedding vector for the input text
	// Returns a float32 array representing the embedding vector
	GetEmbedding(ctx context.Context, queryString string) ([]float32, error)
}

// Creates a new embedding Provider based on the configuration
// Returns error if provider type is not supported
func NewEmbeddingProvider(config config.EmbeddingConfig) (Provider, error) {
	initializer, ok := providerInitializers[config.Provider]
	if !ok {
		return nil, fmt.Errorf("no initializer found for provider type: %s", config.Provider)
	}
	return initializer.CreateProvider(config)
}
