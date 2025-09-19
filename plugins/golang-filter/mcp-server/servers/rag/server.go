package rag

import (
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/mark3labs/mcp-go/mcp"
)

const Version = "1.0.0"

type RAGConfig struct {
	config *config.Config
}

func init() {
	// api.LogInfof("rag server init")
	common.GlobalRegistry.RegisterServer("rag", &RAGConfig{
		config: &config.Config{
			RAG: config.RAGConfig{
				Splitter: config.SplitterConfig{
					Provider:     "nosplitter",
					ChunkSize:    0,
					ChunkOverlap: 0,
				},
				MaxResults: 10,
			},
			Embedding: config.EmbeddingConfig{
				Provider:  "dashscope",
				APIKey:    "",
				BaseURL:   "",
				Model:     "text-embedding-v4",
				Dimension: 1024,
			},
			VectorDB: config.VectorDBConfig{
				Provider:   "milvus",
				Host:       "localhost",
				Port:       6379,
				Database:   "default",
				Collection: "rag",
				Username:   "",
				Password:   "",
			},
		},
	})
}

func (c *RAGConfig) ParseConfig(config map[string]any) error {
	api.LogInfof("start to parse RAG raw config %v", config)
	// 解析RAG配置
	if ragConfig, ok := config["rag"].(map[string]any); ok {
		if splitter, exists := ragConfig["splitter"].(map[string]any); exists {
			if splitterType, exists := splitter["provider"].(string); exists {
				c.config.RAG.Splitter.Provider = splitterType
			}
			if chunkSize, exists := splitter["chunk_size"].(float64); exists {
				c.config.RAG.Splitter.ChunkSize = int(chunkSize)
			}
			if chunkOverlap, exists := splitter["chunk_overlap"].(float64); exists {
				c.config.RAG.Splitter.ChunkOverlap = int(chunkOverlap)
			}
		}
	}

	// 解析Embedding配置
	if embeddingConfig, ok := config["embedding"].(map[string]any); ok {
		if provider, exists := embeddingConfig["provider"].(string); exists {
			c.config.Embedding.Provider = provider
		} else {
			return errors.New("missing embedding provider")
		}

		if apiKey, exists := embeddingConfig["api_key"].(string); exists {
			c.config.Embedding.APIKey = apiKey
		}
		// base_url
		if baseURL, exists := embeddingConfig["base_url"].(string); exists {
			c.config.Embedding.BaseURL = baseURL
		}
		// model
		if model, exists := embeddingConfig["model"].(string); exists {
			c.config.Embedding.Model = model
		}
		// dimension
		if dimension, exists := embeddingConfig["dimension"].(float64); exists {
			c.config.Embedding.Dimension = int(dimension)
		}
	}

	// 解析VectorDB配置
	if vectordbConfig, ok := config["vectordb"].(map[string]any); ok {
		if provider, exists := vectordbConfig["provider"].(string); exists {
			c.config.VectorDB.Provider = provider
		} else {
			return errors.New("missing vectordb provider")
		}
		// host
		if host, exists := vectordbConfig["host"].(string); exists {
			c.config.VectorDB.Host = host
		}
		// port
		if port, exists := vectordbConfig["port"].(float64); exists {
			c.config.VectorDB.Port = int(port)
		}
		// db_name
		if dbName, exists := vectordbConfig["database"].(string); exists {
			c.config.VectorDB.Database = dbName
		}
		// collection
		if collection, exists := vectordbConfig["collection"].(string); exists {
			c.config.VectorDB.Collection = collection
		}

		// username
		if username, exists := vectordbConfig["username"].(string); exists {
			c.config.VectorDB.Username = username
		}
		// password
		if password, exists := vectordbConfig["password"].(string); exists {
			c.config.VectorDB.Password = password
		}
	}

	api.LogInfof("RAG Config ParseConfig Done: %+v", c.config)
	return nil
}

func (c *RAGConfig) NewServer(serverName string) (*common.MCPServer, error) {
	api.LogInfof("start to new rag server and register tools")
	mcpServer := common.NewMCPServer(
		serverName,
		Version,
		common.WithInstructions("This is a RAG (Retrieval-Augmented Generation) server for knowledge management and intelligent Q&A"),
	)

	// 创建RAG客户端（这里可以根据配置初始化各种客户端）
	ragClient, err := NewRAGClient(c.config)
	if err != nil {
		return nil, fmt.Errorf("create rag client failed, err: %w", err)
	}

	// 添加知识管理工具
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("create-chunk-from-text", "Create chunks from text content", GetCreateChunkFromTextSchema()),
		HandleCreateKnowledgeFromText(ragClient),
	)

	// 添加块管理工具
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("list-chunks", "List all chunks ", GetListChunksSchema()),
		HandleListChunks(ragClient),
	)
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("delete-chunk", "Delete specific chunk by ID", GetDeleteChunkSchema()),
		HandleDeleteChunk(ragClient),
	)

	// 添加搜索工具
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("search-chunk", "Search chunks with query", GetSearchSchema()),
		HandleSearch(ragClient),
	)

	// 添加会话管理工具
	// mcpServer.AddTool(
	// 	mcp.NewToolWithRawSchema("create-session", "Create a new chat session", GetCreateSessionSchema()),
	// 	HandleCreateSession(ragClient),
	// )
	// mcpServer.AddTool(
	// 	mcp.NewToolWithRawSchema("get-session", "Get session details by ID", GetGetSessionSchema()),
	// 	HandleGetSession(ragClient),
	// )
	// mcpServer.AddTool(
	// 	mcp.NewToolWithRawSchema("list-sessions", "List all chat sessions", GetListSessionsSchema()),
	// 	HandleListSessions(ragClient),
	// )
	// mcpServer.AddTool(
	// 	mcp.NewToolWithRawSchema("delete-session", "Delete session by ID", GetDeleteSessionSchema()),
	// 	HandleDeleteSession(ragClient),
	// )

	// 添加聊天工具
	// mcpServer.AddTool(
	// 	mcp.NewToolWithRawSchema("chat", "Chat with RAG system", GetChatSchema()),
	// 	HandleChat(ragClient),
	// )
	api.LogInfof("start to new rag server and register tools done")
	return mcpServer, nil
}
