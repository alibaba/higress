package rag

import (
	"fmt"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/embedding"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/textsplitter"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/vectordb"
	"github.com/distribution/distribution/v3/uuid"
	"oras.land/oras-go/pkg/context"
)

const (
	MAX_LIST_KNOWLEDGE_ROW_COUNT = 1000
	MAX_LIST_DOCUMENT_ROW_COUNT  = 1000
)

// RAGClient RAG客户端
type RAGClient struct {
	config *config.Config
	// 这里可以添加LLM客户端、向量数据库客户端等
	vectordbProvider  vectordb.VectorStoreProvider
	embeddingProvider embedding.Provider
	textSplitter      textsplitter.TextSplitter
}

// NewRAGClient 创建新的RAG客户端
func NewRAGClient(config *config.Config) (*RAGClient, error) {
	ragclient := &RAGClient{
		config: config,
	}

	textSplitter, err := textsplitter.NewTextSplitter(&config.RAG.Splitter)
	if err != nil {
		return nil, fmt.Errorf("create text splitter failed, err: %w", err)
	}
	ragclient.textSplitter = textSplitter

	embeddingProvider, err := embedding.NewEmbeddingProvider(ragclient.config.Embedding)
	if err != nil {
		return nil, fmt.Errorf("create embedding provider failed, err: %w", err)
	}
	ragclient.embeddingProvider = embeddingProvider

	demoVector, err := embeddingProvider.GetEmbedding(context.Background(), "初始化")
	if err != nil {
		return nil, fmt.Errorf("create init embedding failed, err: %w", err)
	}
	dim := len(demoVector)

	provider, err := vectordb.NewVectorDBProvider(&ragclient.config.VectorDB, dim)
	if err != nil {
		return nil, fmt.Errorf("create vector store provider failed, err: %w", err)
	}
	ragclient.vectordbProvider = provider

	return ragclient, nil
}

// ListChunks 根据知识ID列出文档块，按 DocumentIndex 升序返回
func (r *RAGClient) ListChunks() ([]schema.Document, error) {
	docs, err := r.vectordbProvider.ListDocs(context.Background(), MAX_LIST_DOCUMENT_ROW_COUNT)
	if err != nil {
		return nil, fmt.Errorf("list chunks failed, err: %w", err)
	}
	return docs, nil
}

// DeleteChunk 删除指定文档块
func (r *RAGClient) DeleteChunk(id string) error {
	if err := r.vectordbProvider.DeleteDocs(context.Background(), []string{id}); err != nil {
		return fmt.Errorf("delete chunk failed, err: %w", err)
	}
	return nil
}

func (r *RAGClient) CreateChunkFromText(text string, chunkName string) ([]schema.Document, error) {

	docs, err := textsplitter.CreateDocuments(r.textSplitter, []string{text}, make([]map[string]any, 0))
	if err != nil {
		return nil, fmt.Errorf("create documents failed, err: %w", err)
	}

	results := make([]schema.Document, 0, len(docs))

	for chunkIndex, doc := range docs {
		doc.ID = uuid.Generate().String()
		doc.Metadata["chunk_index"] = chunkIndex
		doc.Metadata["chunk_name"] = chunkName
		doc.Metadata["chunk_size"] = len(doc.Content)
		// handle embedding
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

// search 搜索文档块
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
