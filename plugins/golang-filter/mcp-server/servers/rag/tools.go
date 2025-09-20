package rag

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

// HandleCreateKnowledgeFromText 处理从文本创建知识
func HandleCreateKnowledgeFromText(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		text, ok := arguments["text"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid text argument")
		}

		// TODO: 实现从文本创建知识的逻辑
		result := map[string]interface{}{
			"success": true,
			"message": "Knowledge created from text",
			"id":      "knowledge-1",
			"text":    text,
		}

		return buildCallToolResult(result)
	}
}

// HandleListChunks 处理列出知识块
func HandleListChunks(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		chunks, err := ragClient.ListChunks()
		if err != nil {
			return nil, fmt.Errorf("list chunks failed, err: %w", err)
		}

		result := map[string]interface{}{
			"chunks": chunks,
			"total":  len(chunks),
		}

		return buildCallToolResult(result)
	}
}

// HandleDeleteChunk 处理删除知识块
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
			"message": "Chunk deleted",
			"id":      id,
		}

		return buildCallToolResult(result)
	}
}

// HandleCreateSession 处理创建聊天会话
func HandleCreateSession(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// TODO: 实现创建聊天会话的逻辑
		result := map[string]interface{}{
			"session_id": "session-1",
			"created_at": "2024-01-01T00:00:00Z",
		}

		return buildCallToolResult(result)
	}
}

// HandleGetSession 处理获取会话详情
func HandleGetSession(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		sessionId, ok := arguments["session_id"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid session_id argument")
		}

		// TODO: 实现获取会话详情的逻辑
		result := map[string]interface{}{
			"session_id": sessionId,
			"messages":   []interface{}{},
		}

		return buildCallToolResult(result)
	}
}

// HandleListSessions 处理列出会话
func HandleListSessions(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// TODO: 实现列出会话的逻辑
		result := map[string]interface{}{
			"sessions": []interface{}{},
			"total":    0,
		}

		return buildCallToolResult(result)
	}
}

// HandleDeleteSession 处理删除会话
func HandleDeleteSession(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		sessionId, ok := arguments["session_id"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid session_id argument")
		}

		// TODO: 实现删除会话的逻辑
		result := map[string]interface{}{
			"success":    true,
			"message":    "Session deleted",
			"session_id": sessionId,
		}

		return buildCallToolResult(result)
	}
}

// HandleSearch 处理搜索
func HandleSearch(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		query, ok := arguments["query"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid query argument")
		}
		topK, ok := arguments["topk"].(int)
		if !ok {
			topK = 10
		}

		threshold, ok := arguments["threshold"].(float64)
		if !ok {
			threshold = 0.5
		}

		searchResult, err := ragClient.SearchChunks(query, int(topK), threshold)
		if err != nil {
			return nil, fmt.Errorf("search chunks failed, err: %w", err)
		}

		// TODO: 实现搜索的逻辑
		result := map[string]interface{}{
			"results": searchResult,
			"total":   len(searchResult),
			"query":   query,
		}

		return buildCallToolResult(result)
	}
}

// HandleChat 处理聊天
func HandleChat(ragClient *RAGClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		message, ok := arguments["message"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid message argument")
		}

		// TODO: 实现聊天的逻辑
		result := map[string]interface{}{
			"response": "This is a sample response from RAG system",
			"message":  message,
		}

		return buildCallToolResult(result)
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
				"description": "The text content to create knowledge from"
			}
		},
		"required": ["text"]
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
			"message": {
				"type": "string",
				"description": "The chat message"
			},
			"session_id": {
				"type": "string",
				"description": "The session ID (optional)"
			}
		},
		"required": ["message"]
	}`)
}
