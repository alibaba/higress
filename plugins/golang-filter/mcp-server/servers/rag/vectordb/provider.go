package vectordb

import (
	"context"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
)

// Provider types constants
const (
	PROVIDER_TYPE_CHROMA        = "chroma"
	PROVIDER_TYPE_PINECONE      = "pinecone"
	PROVIDER_TYPE_WEAVIATE      = "weaviate"
	PROVIDER_TYPE_QDRANT        = "qdrant"
	PROVIDER_TYPE_MILVUS        = "milvus"
	PROVIDER_TYPE_FAISS         = "faiss"
	PROVIDER_TYPE_ELASTICSEARCH = "elasticsearch"
)

// VectorStoreBase defines the base interface for vector store implementations
type VectorStoreProvider interface {
	// CreateVectorStore creates a new vector store
	CreateCollection(ctx context.Context, dim int) error

	// DropVectorStore drops the vector store
	DropCollection(ctx context.Context) error

	// AddDoc adds documents to the vector store
	AddDoc(ctx context.Context, docs []schema.Document) error

	// DeleteDoc deletes documents by filename from the vector store
	DeleteDoc(ctx context.Context, id string) error

	// UpdateDoc updates documents in the vector store
	UpdateDoc(ctx context.Context, docs []schema.Document) error

	// SearchDocs searches for similar documents in the vector store
	SearchDocs(ctx context.Context, vector []float32, options *schema.SearchOptions) ([]schema.SearchResult, error)

	// DeleteDocs deletes documents by IDs from the vector store
	DeleteDocs(ctx context.Context, ids []string) error

	// ListDocs lists documents in the vector store
	ListDocs(ctx context.Context, limit int) ([]schema.Document, error)

	// GetProviderType returns the type of the vector store provider
	GetProviderType() string
}

// VectorDBProviderInitializer defines the interface for vector database provider initializers
type VectorDBProviderInitializer interface {
	// CreateProvider creates a new vector database provider instance
	CreateProvider(cfg *config.VectorDBConfig, dim int) (VectorStoreProvider, error)
}

var (
	vectorDBProviderInitializers = map[string]VectorDBProviderInitializer{
		PROVIDER_TYPE_MILVUS: &milvusProviderInitializer{},
	}
)

// CreateVectorDBProvider creates a vector database provider instance
func NewVectorDBProvider(cfg *config.VectorDBConfig, dim int) (VectorStoreProvider, error) {
	initializer, exists := vectorDBProviderInitializers[cfg.Provider]
	if !exists {
		return nil, fmt.Errorf("unknown vector database provider: %s", cfg.Provider)
	}
	// Create provider
	return initializer.CreateProvider(cfg, dim)
}
