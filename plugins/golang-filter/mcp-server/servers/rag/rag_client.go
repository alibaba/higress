package rag

import (
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/embedding"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/vectordb"
	"oras.land/oras-go/pkg/context"
)

const (
	MAX_LIST_KNOWLEDGE_ROW_COUNT = 1000
)

// RAGClient RAG客户端
type RAGClient struct {
	config *config.Config
	// 这里可以添加LLM客户端、向量数据库客户端等
	vectordbProvider  vectordb.VectorStoreProvider
	embeddingProvider embedding.Provider
}

// NewRAGClient 创建新的RAG客户端
func NewRAGClient(config *config.Config) (*RAGClient, error) {
	ragclient := &RAGClient{
		config: config,
	}
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

func (r *RAGClient) ListKnowledge() ([]schema.Knowledge, error) {

	knowledges, err := r.vectordbProvider.ListKnowledge(context.Background(), MAX_LIST_KNOWLEDGE_ROW_COUNT)
	if err != nil {
		return nil, fmt.Errorf("list knowledge failed, err: %w", err)
	}
	return knowledges, nil
}
