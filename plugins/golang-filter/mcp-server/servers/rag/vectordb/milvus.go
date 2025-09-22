package vectordb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
	MILVUS_DUMMY_DIM     = 8
	MILVUS_PROVIDER_TYPE = "milvus"
)

// MilvusProviderInitializer initializes the Milvus vector store provider
type milvusProviderInitializer struct{}

// InitConfig initializes the configuration with default values if not set
func (m *milvusProviderInitializer) InitConfig(cfg *config.VectorDBConfig) error {
	if cfg.Provider != MILVUS_PROVIDER_TYPE {
		return fmt.Errorf("provider type mismatch: expected %s, got %s", MILVUS_PROVIDER_TYPE, cfg.Provider)
	}

	// Set default values
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == 0 {
		cfg.Port = 19530
	}
	if cfg.Database == "" {
		cfg.Database = "default"
	}

	if cfg.Collection == "" {
		cfg.Collection = schema.DEFAULT_DOCUMENT_COLLECTION
	}

	return nil
}

// ValidateConfig validates the configuration parameters
func (m *milvusProviderInitializer) ValidateConfig(cfg *config.VectorDBConfig) error {
	if cfg.Host == "" {
		return fmt.Errorf("milvus host is required")
	}
	if cfg.Port <= 0 {
		return fmt.Errorf("milvus port must be positive")
	}

	if cfg.Database == "" {
		return fmt.Errorf("milvus database is required")
	}

	if cfg.Collection == "" {
		return fmt.Errorf("milvus document collection is required")
	}
	return nil
}

// CreateProvider creates a new Milvus vector store provider instance
func (m *milvusProviderInitializer) CreateProvider(cfg *config.VectorDBConfig, dim int) (VectorStoreProvider, error) {
	if err := m.InitConfig(cfg); err != nil {
		return nil, err
	}
	if err := m.ValidateConfig(cfg); err != nil {
		return nil, err
	}
	provider, err := NewMilvusProvider(cfg, dim)
	return provider, err
}

// MilvusProvider implements the vector store provider interface for Milvus
type MilvusProvider struct {
	client     client.Client
	config     *config.VectorDBConfig
	Collection string
}

// NewMilvusProvider creates a new instance of MilvusProvider
func NewMilvusProvider(cfg *config.VectorDBConfig, dim int) (VectorStoreProvider, error) {
	// Create Milvus client
	connectParam := client.Config{
		Address: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
	}

	connectParam.DBName = cfg.Database
	// Add authentication if credentials are provided
	if cfg.Username != "" && cfg.Password != "" {
		connectParam.Username = cfg.Username
		connectParam.Password = cfg.Password
	}

	milvusClient, err := client.NewClient(context.Background(), connectParam)
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus client: %w", err)
	}

	provider := &MilvusProvider{
		client:     milvusClient,
		config:     cfg,
		Collection: cfg.Collection,
	}

	ctx := context.Background()
	if err := provider.CreateCollection(ctx, dim); err != nil {
		return nil, err
	}
	return provider, nil
}

// CreateCollection creates a new collection with the specified dimension
func (m *MilvusProvider) CreateCollection(ctx context.Context, dim int) error {
	// Check if collection exists
	document_exists, err := m.client.HasCollection(ctx, m.Collection)
	if err != nil {
		return fmt.Errorf("failed to check %s collection existence: %w", m.Collection, err)
	}

	if !document_exists {
		fmt.Printf("create collection %s\n", m.Collection)
		// Create schema
		schema := entity.NewSchema().
			WithName(m.Collection).
			WithDescription("Knowledge document collection").
			WithAutoID(false).
			WithDynamicFieldEnabled(false)

		// Add fields based on schema.Document structure
		// Primary key field - ID
		pkField := entity.NewField().
			WithName("id").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(256).
			WithIsPrimaryKey(true).
			WithIsAutoID(false)
		schema.WithField(pkField)

		// Content field
		contentField := entity.NewField().
			WithName("content").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(8192)
		schema.WithField(contentField)

		// Vector field
		vectorField := entity.NewField().
			WithName("vector").
			WithDataType(entity.FieldTypeFloatVector).
			WithDim(int64(dim))
		schema.WithField(vectorField)

		// Metadata field
		metadataField := entity.NewField().
			WithName("metadata").
			WithDataType(entity.FieldTypeJSON)
		schema.WithField(metadataField)

		// CreatedAt field (stored as Unix timestamp)
		createdAtField := entity.NewField().
			WithName("created_at").
			WithDataType(entity.FieldTypeInt64)
		schema.WithField(createdAtField)

		// Create collection
		err = m.client.CreateCollection(ctx, schema, entity.DefaultShardNumber)
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}

		// Create vector index
		vectorIndex, err := entity.NewIndexHNSW(entity.IP, 8, 64)
		if err != nil {
			return fmt.Errorf("failed to create vector index: %w", err)
		}

		err = m.client.CreateIndex(ctx, m.Collection, "vector", vectorIndex, false, client.WithIndexName("vector_index"))
		if err != nil {
			return fmt.Errorf("failed to create vector index: %w", err)
		}
	}

	// Load collection
	err = m.client.LoadCollection(ctx, m.Collection, false)
	if err != nil {
		return fmt.Errorf("failed to load document collection: %w", err)
	}
	return nil
}

// DropCollection removes the collection from the database
func (m *MilvusProvider) DropCollection(ctx context.Context) error {
	// Check if collection exists
	exists, err := m.client.HasCollection(ctx, m.Collection)
	if err != nil {
		return fmt.Errorf("failed to check %s collection existence: %w", m.Collection, err)
	}
	if !exists {
		return fmt.Errorf("collection %s does not exist", m.Collection)
	}
	// Drop collection
	err = m.client.DropCollection(ctx, m.Collection)
	if err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}
	return nil
}

// AddDoc adds documents to the vector database
func (m *MilvusProvider) AddDoc(ctx context.Context, docs []schema.Document) error {
	if len(docs) == 0 {
		return nil
	}
	// Prepare data
	ids := make([]string, len(docs))
	contents := make([]string, len(docs))
	vectors := make([][]float32, len(docs))
	metadatas := make([][]byte, len(docs))
	createdAts := make([]int64, len(docs))

	for i, doc := range docs {
		ids[i] = doc.ID
		contents[i] = doc.Content

		// Convert vector type
		vectorFloat32 := make([]float32, len(doc.Vector))
		for j, v := range doc.Vector {
			vectorFloat32[j] = float32(v)
		}
		vectors[i] = vectorFloat32

		// Serialize metadata
		metadataBytes, err := json.Marshal(doc.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata for doc %s: %w", doc.ID, err)
		}
		metadatas[i] = metadataBytes

		createdAts[i] = doc.CreatedAt.UnixMilli()
	}

	// Build insert data
	columns := []entity.Column{
		entity.NewColumnVarChar("id", ids),
		entity.NewColumnVarChar("content", contents),
		entity.NewColumnFloatVector("vector", len(vectors[0]), vectors),
		entity.NewColumnJSONBytes("metadata", metadatas),
		entity.NewColumnInt64("created_at", createdAts),
	}

	// Insert data
	_, err := m.client.Insert(ctx, m.Collection, "", columns...)
	if err != nil {
		return fmt.Errorf("failed to insert documents: %w", err)
	}

	// Flush data
	err = m.client.Flush(ctx, m.Collection, false)
	if err != nil {
		return fmt.Errorf("failed to flush collection: %w", err)
	}

	return nil
}

// DeleteDoc deletes a document by its ID
func (m *MilvusProvider) DeleteDoc(ctx context.Context, id string) error {
	// Build delete expression
	expr := fmt.Sprintf(`id == "%s"`, id)
	// Delete data
	err := m.client.Delete(ctx, m.Collection, "", expr)
	if err != nil {
		return fmt.Errorf("failed to delete documents for id %s: %w", id, err)
	}

	// Flush data
	err = m.client.Flush(ctx, m.Collection, false)
	if err != nil {
		return fmt.Errorf("failed to flush collection after delete: %w", err)
	}

	return nil
}

// UpdateDoc updates documents by first deleting existing ones and then adding new ones
func (m *MilvusProvider) UpdateDoc(ctx context.Context, docs []schema.Document) error {
	// Delete existing documents
	ids := make([]string, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
	}
	if err := m.DeleteDocs(ctx, ids); err != nil {
		return fmt.Errorf("failed to delete existing documents: %w", err)
	}
	// Add new documents
	if err := m.AddDoc(ctx, docs); err != nil {
		return fmt.Errorf("failed to add new documents: %w", err)
	}

	return nil
}

// SearchDocs performs similarity search for documents
func (m *MilvusProvider) SearchDocs(ctx context.Context, vector []float32, options *schema.SearchOptions) ([]schema.SearchResult, error) {
	if options == nil {
		options = &schema.SearchOptions{TopK: 10}
	}
	// Build search parameters
	sp, _ := entity.NewIndexHNSWSearchParam(16)
	// Build filter expression
	expr := ""
	searchResults, err := m.client.Search(
		ctx,
		m.Collection,
		[]string{}, // partition names
		expr,       // filter expression
		[]string{"id", "content", "metadata", "created_at"}, // output fields
		[]entity.Vector{entity.FloatVector(vector)},
		"vector",  // anns_field
		entity.IP, // metric_type
		options.TopK,
		sp,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	// Parse results
	var results []schema.SearchResult
	for _, result := range searchResults {
		for i := 0; i < result.ResultCount; i++ {
			id, _ := result.IDs.Get(i)
			score := result.Scores[i]
			// Get field data
			var content string
			var metadata map[string]interface{}

			for _, field := range result.Fields {
				switch field.Name() {
				case "content":
					if contentCol, ok := field.(*entity.ColumnVarChar); ok {
						if contentVal, err := contentCol.Get(i); err == nil {
							if contentStr, ok := contentVal.(string); ok {
								content = contentStr
							}
						}
					}
				case "metadata":
					if metaCol, ok := field.(*entity.ColumnJSONBytes); ok {
						if metaVal, err := metaCol.Get(i); err == nil {
							if metaBytes, ok := metaVal.([]byte); ok {
								if err := json.Unmarshal(metaBytes, &metadata); err != nil {
									metadata = make(map[string]interface{})
								}
							}
						}
					}
				}
			}

			searchResult := schema.SearchResult{
				Document: schema.Document{
					ID:       fmt.Sprintf("%s", id),
					Content:  content,
					Metadata: metadata,
				},
				Score: float64(score),
			}
			results = append(results, searchResult)
		}
	}
	return results, nil
}

// DeleteDocs deletes multiple documents by their IDs
func (m *MilvusProvider) DeleteDocs(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Build delete expression
	// Milvus expects string values to be quoted within the expression, otherwise the parser will
	// treat the hyphen inside UUID as a minus operator and raise a parse error.
	quotedIDs := make([]string, len(ids))
	for i, id := range ids {
		quotedIDs[i] = fmt.Sprintf("\"%s\"", id)
	}
	expr := fmt.Sprintf("id in [%s]", strings.Join(quotedIDs, ","))

	// Delete data
	err := m.client.Delete(ctx, m.Collection, "", expr)
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}
	// Flush data
	err = m.client.Flush(ctx, m.Collection, false)
	if err != nil {
		return fmt.Errorf("failed to flush collection after delete: %w", err)
	}

	return nil
}

// ListDocs retrieves all documents with optional limit
func (m *MilvusProvider) ListDocs(ctx context.Context, limit int) ([]schema.Document, error) {
	// Build query expression
	expr := ""
	// Query all relevant documents
	queryResult, err := m.client.Query(
		ctx,
		m.Collection,
		[]string{}, // partitions
		expr,       // filter condition
		[]string{"id", "content", "metadata", "created_at"},
		client.WithOffset(0), client.WithLimit(int64(limit)),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}

	if len(queryResult) == 0 {
		return []schema.Document{}, nil
	}

	rowCount := queryResult[0].Len()
	documents := make([]schema.Document, 0, rowCount)

	// Parse query results
	for i := 0; i < rowCount; i++ {
		var (
			id        string
			content   string
			metadata  map[string]interface{}
			createdAt int64
		)

		for _, col := range queryResult {
			switch col.Name() {
			case "id":
				if v, err := col.(*entity.ColumnVarChar).Get(i); err == nil {
					id = v.(string)
				}
			case "content":
				if v, err := col.(*entity.ColumnVarChar).Get(i); err == nil {
					content = v.(string)
				}
			case "metadata":
				if v, err := col.(*entity.ColumnJSONBytes).Get(i); err == nil {
					if bytes, ok := v.([]byte); ok {
						_ = json.Unmarshal(bytes, &metadata)
					}
				}
			case "created_at":
				if v, err := col.(*entity.ColumnInt64).Get(i); err == nil {
					createdAt = v.(int64)
				}
			}
		}

		doc := schema.Document{
			ID:        id,
			Content:   content,
			Metadata:  metadata,
			CreatedAt: time.UnixMilli(createdAt),
		}
		documents = append(documents, doc)
	}
	return documents, nil
}

// GetProviderType returns the provider type identifier
func (m *MilvusProvider) GetProviderType() string {
	return MILVUS_PROVIDER_TYPE
}

// Close closes the connection to the Milvus server
func (m *MilvusProvider) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// joinStrings joins a slice of strings with the given separator
func joinStrings(elems []string, sep string) string {
	return strings.Join(elems, sep)
}
