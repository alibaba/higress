package rag

import (
	"fmt"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/embedding"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/textsplitter"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/vectordb"
	"github.com/google/uuid"
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

func (r *RAGClient) ListKnowledge() ([]schema.Knowledge, error) {

	knowledges, err := r.vectordbProvider.ListKnowledge(context.Background(), MAX_LIST_KNOWLEDGE_ROW_COUNT)
	if err != nil {
		return nil, fmt.Errorf("list knowledge failed, err: %w", err)
	}
	return knowledges, nil
}

func (r *RAGClient) GetKnowledge(id string) (*schema.Knowledge, error) {
	knowledge, err := r.vectordbProvider.GetKnowledge(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("get knowledge failed, err: %w", err)
	}
	return knowledge, nil
}

// DeleteKnowledge 删除知识
func (r *RAGClient) DeleteKnowledge(id string) error {
	if err := r.vectordbProvider.DeleteKnowledge(context.Background(), id); err != nil {
		return fmt.Errorf("delete knowledge failed, err: %w", err)
	}
	return nil
}

// ListChunks 根据知识ID列出文档块，按 DocumentIndex 升序返回
func (r *RAGClient) ListChunks(knowledgeID string) ([]schema.Document, error) {
	docs, err := r.vectordbProvider.ListDocs(context.Background(), knowledgeID, MAX_LIST_DOCUMENT_ROW_COUNT)
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

func (r *RAGClient) CreateKnowledgeFromText(text string, name string) (*schema.Knowledge, error) {
	// start to splitter text
	knowledge := schema.Knowledge{
		ID:               uuid.New().String(),
		Name:             name,
		SourceURL:        "",
		Status:           "pending",
		FileSize:         int64(len(text)),
		EnableMultimodel: false,
		Metadata:         map[string]any{},
		ChunkCount:       0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := r.vectordbProvider.CreateKnowledge(context.Background(), knowledge); err != nil {
		return nil, fmt.Errorf("create knowledge failed, err: %w", err)
	}

	return &knowledge, nil
}

func (r *RAGClient) handleKnowledgeFromText(text string, knowledge schema.Knowledge) (*schema.Knowledge, error) {

	docs, err := textsplitter.CreateDocuments(r.textSplitter, []string{text}, make([]map[string]any, 0))
	if err != nil {
		return nil, fmt.Errorf("create documents failed, err: %w", err)
	}

	for _, doc := range docs {
		doc.Metadata["knowledge_id"] = knowledge.ID
		// handle embedding
		embedding, err := r.embeddingProvider.GetEmbedding(context.Background(), doc.Content)
		if err != nil {
			return nil, fmt.Errorf("create embedding failed, err: %w", err)
		}
		doc.Vector = embedding
	}

	if err := r.vectordbProvider.AddDoc(context.Background(), knowledge.ID, docs); err != nil {

	}

	knowledge.ChunkCount = len(docs)

	// update knowledge
	if err := r.vectordbProvider.UpdateKnowledge(context.Background(), knowledge); err != nil {
		return nil, fmt.Errorf("update knowledge failed, err: %w", err)
	}
	return &knowledge, nil

}
