package llm

import (
    "context"
    "fmt"

    "github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
)

const (
    PROVIDER_TYPE_OPENAI = "openai"
    // future: dashscope, qwen, etc.
)

// Provider defines common interface for large language model providers.
// GenerateCompletion returns generated text for given prompt.
// Implementations may extend to support chat style requests, streaming, etc.
// For the current use-cases within Higress RAG server a simple prompt-response
// suffices.
//
// NOTE: Additional method signatures can be added later without breaking
// existing callers by creating wrapper interfaces.
//
//go:generate mockgen -source=provider.go -destination=provider_mock.go -package=llm
// (The go:generate comment enables future unit testing generation without
// affecting production builds.)

type Provider interface {
    // GetProviderType returns unique provider type constant, e.g. "openai".
    GetProviderType() string
    // GenerateCompletion generates text based on the given prompt.
    GenerateCompletion(ctx context.Context, prompt string) (string, error)
}

// providerInitializer mirrors pattern used by embedding provider package.
// It creates concrete Provider instances based on configuration.
// This allows lazy registration of multiple providers without introducing an
// explicit switch statement later.
type providerInitializer interface {
    CreateProvider(config.LLMConfig) (Provider, error)
}

var (
    providerInitializers = map[string]providerInitializer{
        PROVIDER_TYPE_OPENAI: &openAIProviderInitializer{},
    }
)

// NewLLMProvider instantiates a concrete Provider according to configuration.
func NewLLMProvider(cfg config.LLMConfig) (Provider, error) {
    initializer, ok := providerInitializers[cfg.Provider]
    if !ok {
        return nil, fmt.Errorf("no initializer found for llm provider type: %s", cfg.Provider)
    }
    return initializer.CreateProvider(cfg)
}