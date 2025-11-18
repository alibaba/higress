package tool_search

import (
	"context"
	"fmt"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

// SearchService handles tool search operations
type SearchService struct {
	dbClient        *DBClient
	embeddingClient *EmbeddingClient
}

// NewSearchService creates a new SearchService instance
func NewSearchService(dbClient *DBClient, embeddingClient *EmbeddingClient) *SearchService {
	return &SearchService{
		dbClient:        dbClient,
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
	records, err := s.dbClient.SearchTools(query, vector, topK)
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
			api.LogDebugf("Successfully parsed metadata for tool %s___%s", record.ServerName, record.Name)
		} else {
			api.LogDebugf("No metadata found for tool %s___%s, using basic definition", record.ServerName, record.Name)
			// If no metadata, create a basic tool definition
			tool = ToolDefinition{
				"name":        record.Name,
				"description": record.Content,
			}
		}

		// Update the name to include server name
		tool["name"] = fmt.Sprintf("%s___%s", record.ServerName, record.Name)

		tools = append(tools, tool)

		if i < 3 { // Log first 3 tools for debugging
			api.LogDebugf("Tool %d: %s - %s", i+1, tool["name"], record.Content)
		}
	}

	api.LogInfof("Successfully converted %d tools", len(tools))
	return &ToolSearchResult{Tools: tools}
}

// GetAllTools retrieves all available tools
func (s *SearchService) GetAllTools() (*ToolSearchResult, error) {
	api.LogInfo("Retrieving all tools")
	records, err := s.dbClient.GetAllTools()
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
			api.LogDebugf("Successfully parsed metadata for tool %s___%s", record.ServerName, record.Name)
		} else {
			api.LogDebugf("No metadata found for tool %s___%s, using basic definition", record.ServerName, record.Name)
			// If no metadata, create a basic tool definition
			tool = ToolDefinition{
				"name":        record.Name,
				"description": record.Content,
			}
		}

		// Update the name to include server name
		tool["name"] = fmt.Sprintf("%s___%s", record.ServerName, record.Name)

		tools = append(tools, tool)
	}

	api.LogInfof("Successfully converted %d tools", len(tools))
	return &ToolSearchResult{Tools: tools}, nil
}
