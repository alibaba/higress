package rag

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// HandleCreateChunkFromText handles the creation of knowledge chunks from text input
func HandleCreateChunkFromText(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		text, ok1 := arguments["text"].(string)
		title, ok2 := arguments["title"].(string)
		if !ok1 {
			return nil, fmt.Errorf("invalid text argument")
		}
		if !ok2 {
			return nil, fmt.Errorf("invalid title argument")
		}
		// Create knowledge chunks
		docs, err := ragClient.CreateChunkFromText(text, title)
		if err != nil {
			return nil, fmt.Errorf("create chunk failed, err: %w", err)
		}

		result := map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("chunks created from text, title: %s", title),
			"data":    docs,
		}

		return buildCallToolResult(result)
	}
}

// HandleListChunks handles the listing of knowledge chunks
func HandleListChunks(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		chunks, err := ragClient.ListChunks()
		if err != nil {
			return nil, fmt.Errorf("list chunks failed, err: %w", err)
		}
		return buildCallToolResult(chunks)
	}
}

// HandleDeleteChunk handles the deletion of a knowledge chunk
func HandleDeleteChunk(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		id, ok := arguments["id"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid id argument")
		}

		if err := ragClient.DeleteChunk(id); err != nil {
			return nil, fmt.Errorf("delete chunk failed, err: %w", err)
		}

		result := map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("chunk deleted, id: %s", id),
		}

		return buildCallToolResult(result)
	}
}

// HandleCreateSession handles the creation of a chat session
func HandleCreateSession(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result := map[string]interface{}{
			"session_id": "session-1",
			"created_at": "2024-01-01T00:00:00Z",
		}

		return buildCallToolResult(result)
	}
}

// HandleGetSession handles retrieving session details
func HandleGetSession(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		sessionId, ok := arguments["session_id"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid session_id argument")
		}

		result := map[string]interface{}{
			"session_id": sessionId,
			"messages":   []interface{}{},
		}

		return buildCallToolResult(result)
	}
}

// HandleListSessions handles listing all sessions
func HandleListSessions(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result := map[string]interface{}{
			"sessions": []interface{}{},
			"total":    0,
		}

		return buildCallToolResult(result)
	}
}

// HandleDeleteSession handles the deletion of a session
func HandleDeleteSession(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		sessionId, ok := arguments["session_id"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid session_id argument")
		}

		result := map[string]interface{}{
			"success":    true,
			"message":    "Session deleted",
			"session_id": sessionId,
		}

		return buildCallToolResult(result)
	}
}

// HandleSearch handles semantic search functionality
func HandleSearch(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		query, ok := arguments["query"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid query argument")
		}
		topK, ok := arguments["topk"].(int)
		if !ok {
			topK = ragClient.config.RAG.TopK
		}

		threshold, ok := arguments["threshold"].(float64)
		if !ok {
			threshold = ragClient.config.RAG.Threshold
		}

		searchResult, err := ragClient.SearchChunks(query, int(topK), threshold)
		if err != nil {
			return nil, fmt.Errorf("search chunks failed, err: %w", err)
		}
		return buildCallToolResult(searchResult)
	}
}

// HandleChat handles chat interactions using LLM
func HandleChat(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		query, ok := arguments["query"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid query argument")
		}
		// check llm provider
		if ragClient.llmProvider == nil {
			return nil, fmt.Errorf("llm provider is empty, please check the llm configuration")
		}
		// Generate response using RAGClient's LLM
		reply, err := ragClient.Chat(query)
		if err != nil {
			return nil, fmt.Errorf("chat failed, err: %w", err)
		}

		return buildCallToolResult(reply)
	}
}

// buildCallToolResult builds the call tool result
func buildCallToolResult(results any) (*mcp.CallToolResult, error) {
	jsonData, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}, nil
}

// Schema functions

// GetCreateChunkFromTextSchema returns the schema for create chunk from text tool
func GetCreateChunkFromTextSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"text": {
				"type": "string",
				"description": "The text content to create chunks from"
			},
			"title": {
				"type": "string",
				"description": "The title of text content"
			}
		},
		"required": ["text", "title"]
	}`)
}

// GetListKnowledgeSchema returns the schema for list knowledge tool
func GetListKnowledgeSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {}
	}`)
}

// GetGetKnowledgeSchema returns the schema for get knowledge tool
func GetGetKnowledgeSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {
				"type": "string",
				"description": "The knowledge ID"
			}
		},
		"required": ["id"]
	}`)
}

// GetDeleteKnowledgeSchema returns the schema for delete knowledge tool
func GetDeleteKnowledgeSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {
				"type": "string",
				"description": "The knowledge ID to delete"
			}
		},
		"required": ["id"]
	}`)
}

// GetListChunksSchema returns the schema for list chunks tool
func GetListChunksSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {}
	}`)
}

// GetDeleteChunkSchema returns the schema for delete chunk tool
func GetDeleteChunkSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {
				"type": "string",
				"description": "The chunk ID to delete"
			}
		},
		"required": ["id"]
	}`)
}

// GetCreateSessionSchema returns the schema for create session tool
func GetCreateSessionSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {}
	}`)
}

// GetGetSessionSchema returns the schema for get session tool
func GetGetSessionSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"session_id": {
				"type": "string",
				"description": "The session ID"
			}
		},
		"required": ["session_id"]
	}`)
}

// GetListSessionsSchema returns the schema for list sessions tool
func GetListSessionsSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {}
	}`)
}

// GetDeleteSessionSchema returns the schema for delete session tool
func GetDeleteSessionSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"session_id": {
				"type": "string",
				"description": "The session ID to delete"
			}
		},
		"required": ["session_id"]
	}`)
}

// GetSearchSchema returns the schema for search tool
func GetSearchSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The search query"
			},
			"topk": {
                "type": "integer",
                "description": "The number of top results to return (optional, default 10)"
            },
            "threshold": {
                "type": "number",
                "description": "The relevance score threshold for filtering results (optional, default 0.5)"
            }
		},
		"required": ["query"]
	}`)
}

// GetChatSchema returns the schema for chat tool
func GetChatSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "User query"
			}
		},
		"required": ["query"]
	}`)
}

// Enhanced Tools for Advanced RAG Features

// HandleHybridSearch handles hybrid search using both vector and BM25 retrieval
func HandleHybridSearch(enhancedClient *EnhancedRAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		query, ok := arguments["query"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid query argument")
		}

		// Parse optional parameters
		topK := enhancedClient.config.RAG.TopK
		if tk, ok := arguments["topk"].(float64); ok {
			topK = int(tk)
		}

		threshold := enhancedClient.config.RAG.Threshold
		if th, ok := arguments["threshold"].(float64); ok {
			threshold = th
		}

		fusionMethod := "rrf"
		if fm, ok := arguments["fusion_method"].(string); ok {
			fusionMethod = fm
		}

		vectorWeight := 0.6
		if vw, ok := arguments["vector_weight"].(float64); ok {
			vectorWeight = vw
		}

		bm25Weight := 0.4
		if bw, ok := arguments["bm25_weight"].(float64); ok {
			bm25Weight = bw
		}

		// Create search options
		options := &fusion.HybridSearchOptions{
			VectorTopK:   topK * 2,
			BM25TopK:     topK * 2,
			FinalTopK:    topK,
			VectorWeight: vectorWeight,
			BM25Weight:   bm25Weight,
			MinScore:     threshold,
			EnableVector: true,
			EnableBM25:   true,
			FusionOptions: fusion.DefaultFusionOptions(),
		}

		// Set fusion method
		switch fusionMethod {
		case "rrf":
			options.FusionMethod = fusion.RRFFusion
		case "weighted":
			options.FusionMethod = fusion.WeightedFusion
		case "borda":
			options.FusionMethod = fusion.BordaFusion
		default:
			options.FusionMethod = fusion.RRFFusion
		}

		results, err := enhancedClient.HybridSearch(ctx, query, options)
		if err != nil {
			return nil, fmt.Errorf("hybrid search failed: %w", err)
		}

		return buildCallToolResult(map[string]interface{}{
			"query":   query,
			"results": results,
			"options": options,
		})
	}
}

// HandleEnhancedSearch handles enhanced search with query enhancement and post-processing
func HandleEnhancedSearch(enhancedClient *EnhancedRAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		query, ok := arguments["query"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid query argument")
		}

		topK := enhancedClient.config.RAG.TopK
		if tk, ok := arguments["topk"].(float64); ok {
			topK = int(tk)
		}

		threshold := enhancedClient.config.RAG.Threshold
		if th, ok := arguments["threshold"].(float64); ok {
			threshold = th
		}

		result, err := enhancedClient.EnhancedSearch(ctx, query, topK, threshold)
		if err != nil {
			return nil, fmt.Errorf("enhanced search failed: %w", err)
		}

		return buildCallToolResult(result)
	}
}

// HandleCRAGSearch handles Corrective RAG search with web augmentation
func HandleCRAGSearch(enhancedClient *EnhancedRAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		query, ok := arguments["query"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid query argument")
		}

		topK := enhancedClient.config.RAG.TopK
		if tk, ok := arguments["topk"].(float64); ok {
			topK = int(tk)
		}

		result, err := enhancedClient.CRAGSearch(ctx, query, topK)
		if err != nil {
			return nil, fmt.Errorf("CRAG search failed: %w", err)
		}

		return buildCallToolResult(result)
	}
}

// HandleEnhancedChat handles enhanced chat with all advanced features
func HandleEnhancedChat(enhancedClient *EnhancedRAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		query, ok := arguments["query"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid query argument")
		}

		if enhancedClient.llmProvider == nil {
			return nil, fmt.Errorf("llm provider is empty, please check the llm configuration")
		}

		result, err := enhancedClient.EnhancedChat(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("enhanced chat failed: %w", err)
		}

		return buildCallToolResult(result)
	}
}

// HandleQueryEnhancement handles query enhancement operations
func HandleQueryEnhancement(enhancedClient *EnhancedRAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		query, ok := arguments["query"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid query argument")
		}

		// Parse enhancement options
		options := &queryenhancement.EnhancementOptions{
			EnableRewrite:             true,
			EnableExpansion:           true,
			EnableDecomposition:       false,
			EnableIntentClassification: true,
			MaxRewriteCount:           3,
			MaxExpansionTerms:         10,
			MaxSubQueries:             5,
		}

		if enableRewrite, ok := arguments["enable_rewrite"].(bool); ok {
			options.EnableRewrite = enableRewrite
		}
		if enableExpansion, ok := arguments["enable_expansion"].(bool); ok {
			options.EnableExpansion = enableExpansion
		}
		if enableDecomposition, ok := arguments["enable_decomposition"].(bool); ok {
			options.EnableDecomposition = enableDecomposition
		}
		if enableIntent, ok := arguments["enable_intent"].(bool); ok {
			options.EnableIntentClassification = enableIntent
		}

		result, err := enhancedClient.queryEnhancer.EnhanceQuery(ctx, query, options)
		if err != nil {
			return nil, fmt.Errorf("query enhancement failed: %w", err)
		}

		return buildCallToolResult(result)
	}
}

// HandleSearchComparison handles comparison between different search methods
func HandleSearchComparison(enhancedClient *EnhancedRAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		query, ok := arguments["query"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid query argument")
		}

		topK := enhancedClient.config.RAG.TopK
		if tk, ok := arguments["topk"].(float64); ok {
			topK = int(tk)
		}

		comparison, err := enhancedClient.hybridSearchProvider.CompareSearchMethods(ctx, query, topK)
		if err != nil {
			return nil, fmt.Errorf("search comparison failed: %w", err)
		}

		return buildCallToolResult(comparison)
	}
}

// Enhanced Tool Schemas

// GetHybridSearchSchema returns the schema for hybrid search tool
func GetHybridSearchSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The search query"
			},
			"topk": {
				"type": "integer",
				"description": "The number of top results to return (optional, default 10)"
			},
			"threshold": {
				"type": "number",
				"description": "The relevance score threshold for filtering results (optional, default 0.5)"
			},
			"fusion_method": {
				"type": "string",
				"description": "Fusion method: rrf, weighted, borda (optional, default rrf)",
				"enum": ["rrf", "weighted", "borda"]
			},
			"vector_weight": {
				"type": "number",
				"description": "Weight for vector search results (optional, default 0.6)"
			},
			"bm25_weight": {
				"type": "number",
				"description": "Weight for BM25 search results (optional, default 0.4)"
			}
		},
		"required": ["query"]
	}`)
}

// GetEnhancedSearchSchema returns the schema for enhanced search tool
func GetEnhancedSearchSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The search query"
			},
			"topk": {
				"type": "integer",
				"description": "The number of top results to return (optional, default 10)"
			},
			"threshold": {
				"type": "number",
				"description": "The relevance score threshold for filtering results (optional, default 0.5)"
			}
		},
		"required": ["query"]
	}`)
}

// GetCRAGSearchSchema returns the schema for CRAG search tool
func GetCRAGSearchSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The search query"
			},
			"topk": {
				"type": "integer",
				"description": "The number of top results to return (optional, default 10)"
			}
		},
		"required": ["query"]
	}`)
}

// GetEnhancedChatSchema returns the schema for enhanced chat tool
func GetEnhancedChatSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "User query for enhanced chat with advanced RAG features"
			}
		},
		"required": ["query"]
	}`)
}

// GetQueryEnhancementSchema returns the schema for query enhancement tool
func GetQueryEnhancementSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The query to enhance"
			},
			"enable_rewrite": {
				"type": "boolean",
				"description": "Enable query rewriting (optional, default true)"
			},
			"enable_expansion": {
				"type": "boolean",
				"description": "Enable query expansion (optional, default true)"
			},
			"enable_decomposition": {
				"type": "boolean",
				"description": "Enable query decomposition (optional, default false)"
			},
			"enable_intent": {
				"type": "boolean",
				"description": "Enable intent classification (optional, default true)"
			}
		},
		"required": ["query"]
	}`)
}

// GetSearchComparisonSchema returns the schema for search comparison tool
func GetSearchComparisonSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The search query to compare across different methods"
			},
			"topk": {
				"type": "integer",
				"description": "The number of top results to return for each method (optional, default 10)"
			}
		},
		"required": ["query"]
	}`)
}
