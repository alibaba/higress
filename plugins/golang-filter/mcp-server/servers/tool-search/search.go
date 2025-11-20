package tool_search

import (
	"context"
	"fmt"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

// SearchService handles tool search operations
type SearchService struct {
	milvusProvider  *MilvusVectorStoreProvider
	config          *config.VectorDBConfig
	tableName       string
	gatewayID       string
	dimensions      int
	embeddingClient *EmbeddingClient
}

// NewSearchService creates a new SearchService instance
func NewSearchService(host string, port int, database, username, password, tableName, gatewayID string, embeddingClient *EmbeddingClient, dimensions int) *SearchService {
	// Create Milvus configuration
	cfg := &config.VectorDBConfig{
		Provider:   "milvus",
		Host:       host,
		Port:       port,
		Database:   database,
		Collection: tableName,
		Username:   username,
		Password:   password,
	}

	// Create Milvus provider
	provider, err := NewMilvusVectorStoreProvider(cfg, dimensions)
	if err != nil {
		api.LogErrorf("Failed to create Milvus provider: %v", err)
		return nil
	}

	return &SearchService{
		milvusProvider:  provider,
		config:          cfg,
		tableName:       tableName,
		gatewayID:       gatewayID,
		dimensions:      dimensions,
		embeddingClient: embeddingClient,
	}
}

// ToolSearchResult represents the result of a tool search
type ToolSearchResult struct {
	Tools []ToolDefinition `json:"tools"`
}

// ToolDefinition represents a tool definition in the search result
type ToolDefinition map[string]interface{}

// SearchTools performs semantic search for tools
func (s *SearchService) SearchTools(ctx context.Context, query string, topK int) (*ToolSearchResult, error) {
	api.LogInfof("Starting tool search for query: '%s', topK: %d", query, topK)

	// Generate vector embedding for the query
	vector, err := s.embeddingClient.GetEmbedding(ctx, query)
	if err != nil {
		api.LogErrorf("Failed to generate embedding for query '%s': %v", query, err)
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	api.LogInfof("Embedding generated successfully, vector dimension: %d", len(vector))

	// Perform vector search
	records, err := s.searchToolsInDB(query, vector, topK)
	if err != nil {
		api.LogErrorf("Failed to search tools: %v", err)
		return nil, fmt.Errorf("failed to search tools: %w", err)
	}

	api.LogInfof("Vector search completed, found %d records", len(records))

	return s.convertRecordsToResult(records), nil
}

// convertRecordsToResult converts database records to tool search result
func (s *SearchService) convertRecordsToResult(records []ToolRecord) *ToolSearchResult {
	api.LogInfof("Converting %d records to tool definitions", len(records))

	tools := make([]ToolDefinition, 0, len(records))
	for i, record := range records {
		var tool ToolDefinition

		// Use metadata if available
		if len(record.Metadata) > 0 {
			tool = record.Metadata
			api.LogDebugf("Successfully parsed metadata for tool %s", record.Name)
		} else {
			api.LogDebugf("No metadata found for tool %s, using basic definition", record.Name)
			// If no metadata, create a basic tool definition
			tool = ToolDefinition{
				"name":        record.Name,
				"description": record.Content,
			}
		}

		// Update the name to include server name
		tool["name"] = fmt.Sprintf("%s", record.Name)

		tools = append(tools, tool)

		api.LogDebugf("Tool %d: %s - %s", i+1, tool["name"], record.Content)
	}

	api.LogInfof("Successfully converted %d tools", len(tools))
	return &ToolSearchResult{Tools: tools}
}

// GetAllTools retrieves all available tools
func (s *SearchService) GetAllTools() (*ToolSearchResult, error) {
	api.LogInfo("Retrieving all tools")
	records, err := s.getAllToolsFromDB()
	if err != nil {
		api.LogErrorf("Failed to get all tools: %v", err)
		return nil, fmt.Errorf("failed to get all tools: %w", err)
	}

	api.LogInfof("Found %d tools in database", len(records))

	// Convert records to tool definitions
	tools := make([]ToolDefinition, 0, len(records))
	for _, record := range records {
		var tool ToolDefinition

		// Use metadata if available
		if len(record.Metadata) > 0 {
			tool = record.Metadata
			api.LogDebugf("Successfully parsed metadata for tool %s", record.Name)
		} else {
			api.LogDebugf("No metadata found for tool %s, using basic definition", record.Name)
			// If no metadata, create a basic tool definition
			tool = ToolDefinition{
				"name":        record.Name,
				"description": record.Content,
			}
		}

		// Update the name to include server name
		tool["name"] = fmt.Sprintf("%s", record.Name)

		tools = append(tools, tool)
	}

	api.LogInfof("Successfully converted %d tools", len(tools))
	return &ToolSearchResult{Tools: tools}, nil
}

// ToolRecord represents a tool record in the database
type ToolRecord struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	GatewayID string                 `json:"gateway_id"`
}

func (s *SearchService) searchToolsInDB(query string, vector []float32, topK int) ([]ToolRecord, error) {
	api.LogInfof("Performing vector search for query: '%s', topK: %d", query, topK)

	// For Milvus, we'll perform vector search directly
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Perform vector search
	searchOptions := &schema.SearchOptions{
		TopK: topK,
	}

	results, err := s.milvusProvider.SearchDocs(ctx, vector, searchOptions)
	if err != nil {
		api.LogErrorf("Vector search failed: %v", err)
		return nil, fmt.Errorf("failed to perform vector search: %w", err)
	}

	// Convert results to ToolRecords
	var records []ToolRecord
	for _, result := range results {
		doc := result.Document
		tool := ToolRecord{
			ID:        doc.ID,
			Content:   doc.Content,
			Metadata:  doc.Metadata,
			GatewayID: s.gatewayID,
		}

		if name, ok := doc.Metadata["name"].(string); ok {
			tool.Name = name
		}

		records = append(records, tool)
	}

	api.LogInfof("Vector search completed, found %d results", len(records))
	return records, nil
}

// getAllToolsFromDB retrieves all tools from the database
func (s *SearchService) getAllToolsFromDB() ([]ToolRecord, error) {
	api.LogInfof("Executing GetAllTools query from collection: %s", s.tableName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Retrieve all documents without limit
	docs, err := s.milvusProvider.ListAllDocs(ctx)
	if err != nil {
		api.LogErrorf("Failed to list documents: %v", err)
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	// Convert documents to ToolRecords
	var tools []ToolRecord
	for _, doc := range docs {
		tool := ToolRecord{
			ID:        doc.ID,
			Content:   doc.Content,
			Metadata:  doc.Metadata,
			GatewayID: s.gatewayID,
		}

		if name, ok := doc.Metadata["name"].(string); ok {
			tool.Name = name
		}

		tools = append(tools, tool)
	}

	api.LogInfof("GetAllTools query completed, found %d tools", len(tools))
	return tools, nil
}
