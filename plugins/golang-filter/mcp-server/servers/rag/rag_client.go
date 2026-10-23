package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/embedding"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/llm"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/textsplitter"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/vectordb"
	"github.com/distribution/distribution/v3/uuid"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

const (
	MAX_LIST_KNOWLEDGE_ROW_COUNT = 1000
	MAX_LIST_DOCUMENT_ROW_COUNT  = 1000
)

// RAGClient represents the RAG (Retrieval-Augmented Generation) client
type RAGClient struct {
	config            *config.Config
	vectordbProvider  vectordb.VectorStoreProvider
	embeddingProvider embedding.Provider
	textSplitter      textsplitter.TextSplitter
	llmProvider       llm.Provider
}

// NewRAGClient creates a new RAG client instance
func NewRAGClient(config *config.Config) (*RAGClient, error) {
	api.LogDebugf("RAG NewRAGClient: %+v", config)
	ragclient := &RAGClient{
		config: config,
	}
	textSplitter, err := textsplitter.NewTextSplitter(&config.RAG.Splitter)
	if err != nil {
		return nil, fmt.Errorf("create text splitter failed, err: %w", err)
	}
	ragclient.textSplitter = textSplitter

	api.LogDebugf("RAG New Embedding Provider: %+v", ragclient.config.Embedding)
	embeddingProvider, err := embedding.NewEmbeddingProvider(ragclient.config.Embedding)
	if err != nil {
		return nil, fmt.Errorf("create embedding provider failed, err: %w", err)
	}
	ragclient.embeddingProvider = embeddingProvider

	api.LogDebugf("RAG New LLM Provider: %+v", ragclient.config.LLM)
	if ragclient.config.LLM.Provider == "" {
		ragclient.llmProvider = nil
	} else {
		llmProvider, err := llm.NewLLMProvider(ragclient.config.LLM)
		if err != nil {
			return nil, fmt.Errorf("create llm provider failed, err: %w", err)
		}
		ragclient.llmProvider = llmProvider
	}

	api.LogDebugf("RAG New VectorDB Provider: %+v", ragclient.config.VectorDB)
	dim := ragclient.config.Embedding.Dimensions
	provider, err := vectordb.NewVectorDBProvider(&ragclient.config.VectorDB, dim)
	if err != nil {
		return nil, fmt.Errorf("create vector store provider failed, err: %w", err)
	}
	ragclient.vectordbProvider = provider
	return ragclient, nil
}

// ListChunks lists document chunks by knowledge ID, returns in ascending order of DocumentIndex
func (r *RAGClient) ListChunks() ([]schema.Document, error) {
	docs, err := r.vectordbProvider.ListDocs(context.Background(), MAX_LIST_DOCUMENT_ROW_COUNT)
	if err != nil {
		return nil, fmt.Errorf("list chunks failed, err: %w", err)
	}
	return docs, nil
}

// DeleteChunk deletes a specific document chunk
func (r *RAGClient) DeleteChunk(id string) error {
	if err := r.vectordbProvider.DeleteDocs(context.Background(), []string{id}); err != nil {
		return fmt.Errorf("delete chunk failed, err: %w", err)
	}
	return nil
}

func (r *RAGClient) CreateChunkFromText(text string, title string) ([]schema.Document, error) {

	docs, err := textsplitter.CreateDocuments(r.textSplitter, []string{text}, make([]map[string]any, 0))
	if err != nil {
		return nil, fmt.Errorf("create documents failed, err: %w", err)
	}

	results := make([]schema.Document, 0, len(docs))

	for chunkIndex, doc := range docs {
		doc.ID = uuid.Generate().String()
		doc.Metadata["chunk_index"] = chunkIndex
		doc.Metadata["chunk_title"] = title
		doc.Metadata["chunk_size"] = len(doc.Content)
		// Generate embedding for the document
		embedding, err := r.embeddingProvider.GetEmbedding(context.Background(), doc.Content)
		if err != nil {
			return nil, fmt.Errorf("create embedding failed, err: %w", err)
		}
		doc.Vector = embedding
		doc.CreatedAt = time.Now()
		results = append(results, doc)
	}

	if err := r.vectordbProvider.AddDoc(context.Background(), results); err != nil {
		return nil, fmt.Errorf("add documents failed, err: %w", err)
	}

	return results, nil
}

// SearchChunks searches for document chunks
func (r *RAGClient) SearchChunks(query string, topK int, threshold float64) ([]schema.SearchResult, error) {

	vector, err := r.embeddingProvider.GetEmbedding(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("create embedding failed, err: %w", err)
	}
	options := &schema.SearchOptions{
		TopK:      topK,
		Threshold: threshold,
	}
	docs, err := r.vectordbProvider.SearchDocs(context.Background(), vector, options)
	if err != nil {
		return nil, fmt.Errorf("search chunks failed, err: %w", err)
	}
	return docs, nil
}

// Chat generates a response using LLM
func (r *RAGClient) Chat(query string) (string, error) {
	if r.llmProvider == nil {
		return "", fmt.Errorf("llm provider not initialized")
	}

	docs, err := r.SearchChunks(query, r.config.RAG.TopK, r.config.RAG.Threshold)
	if err != nil {
		return "", fmt.Errorf("search chunks failed, err: %w", err)
	}

	contexts := make([]string, 0, len(docs))
	for _, doc := range docs {
		contexts = append(contexts, strings.ReplaceAll(doc.Document.Content, "\n", " "))
	}

	prompt := llm.BuildPrompt(query, contexts, "\n\n")
	resp, err := r.llmProvider.GenerateCompletion(context.Background(), prompt)
	if err != nil {
		return "", fmt.Errorf("generate completion failed, err: %w", err)
	}
	return resp, nil
}
