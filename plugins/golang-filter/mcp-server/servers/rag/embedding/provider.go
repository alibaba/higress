package embedding

import (
	"context"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
)

const (
	PROVIDER_TYPE_DASHSCOPE   = "dashscope"
	PROVIDER_TYPE_TEXTIN      = "textin"
	PROVIDER_TYPE_COHERE      = "cohere"
	PROVIDER_TYPE_OPENAI      = "openai"
	PROVIDER_TYPE_OLLAMA      = "ollama"
	PROVIDER_TYPE_HUGGINGFACE = "huggingface"
	PROVIDER_TYPE_XFYUN       = "xfyun"
	PROVIDER_TYPE_AZURE       = "azure"
)

type providerInitializer interface {
	CreateProvider(config.EmbeddingConfig) (Provider, error)
}

var (
	providerInitializers = map[string]providerInitializer{
		PROVIDER_TYPE_DASHSCOPE: &dashScopeProviderInitializer{},
		// PROVIDER_TYPE_TEXTIN:      &textInProviderInitializer{},
		// PROVIDER_TYPE_COHERE:      &cohereProviderInitializer{},
		PROVIDER_TYPE_OPENAI: &openAIProviderInitializer{},
		// PROVIDER_TYPE_OLLAMA:      &ollamaProviderInitializer{},
		// PROVIDER_TYPE_HUGGINGFACE: &huggingfaceProviderInitializer{},
		// PROVIDER_TYPE_XFYUN:       &xfyunProviderInitializer{},
		// PROVIDER_TYPE_AZURE:       &azureProviderInitializer{},
	}
)

type Provider interface {
	GetProviderType() string
	GetEmbedding(ctx context.Context, queryString string) ([]float64, error)
}

func NewEmbeddingProvider(config config.EmbeddingConfig) (Provider, error) {
	initializer, ok := providerInitializers[config.Provider]
	if !ok {
		return nil, fmt.Errorf("no initializer found for provider type: %s", config.Provider)
	}
	return initializer.CreateProvider(config)
}
