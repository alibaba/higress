package tool_search

import (
	"context"
	"fmt"
	"time"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

// EmbeddingClient handles vector embedding generation using OpenAI-compatible APIs
type EmbeddingClient struct {
	client     *openai.Client
	model      string
	dimensions int
}

// NewEmbeddingClient creates a new EmbeddingClient instance for OpenAI-compatible APIs
func NewEmbeddingClient(apiKey, baseURL, model string, dimensions int) *EmbeddingClient {
	api.LogInfof("Creating EmbeddingClient with baseURL: %s, model: %s, dimensions: %d", baseURL, model, dimensions)

	// Create client with timeout
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
		option.WithRequestTimeout(30*time.Second),
	)

	return &EmbeddingClient{
		client:     &client,
		model:      model,
		dimensions: dimensions,
	}
}

// GetEmbedding generates vector embedding for the given text
func (e *EmbeddingClient) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	api.LogInfof("Generating embedding for text (length: %d)", len(text))
	api.LogDebugf("Using model: %s, dimensions: %d", e.model, e.dimensions)

	// Add timeout to context if not already present
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	params := openai.EmbeddingNewParams{
		Model: e.model,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
		Dimensions:     openai.Int(int64(e.dimensions)),
		EncodingFormat: openai.EmbeddingNewParamsEncodingFormatFloat,
	}

	api.LogDebugf("Calling OpenAI-compatible API for embedding generation")
	embeddingResp, err := e.client.Embeddings.New(ctx, params)
	if err != nil {
		api.LogErrorf("OpenAI-compatible API call failed: %v", err)
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(embeddingResp.Data) == 0 {
		api.LogErrorf("Empty embedding response from API")
		return nil, fmt.Errorf("empty embedding response")
	}

	api.LogDebugf("Successfully received embedding from API")
	api.LogDebugf("Response data length: %d, embedding dimension: %d", len(embeddingResp.Data), len(embeddingResp.Data[0].Embedding))

	// Convert []float64 to []float32
	embedding := make([]float32, len(embeddingResp.Data[0].Embedding))
	for i, v := range embeddingResp.Data[0].Embedding {
		embedding[i] = float32(v)
	}

	api.LogInfof("Embedding conversion completed, final dimension: %d", len(embedding))
	return embedding, nil
}
