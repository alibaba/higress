package rag

import (
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/mark3labs/mcp-go/mcp"
)

const Version = "1.0.0"

type RAGConfig struct {
	config *config.Config
}

func init() {
	common.GlobalRegistry.RegisterServer("rag", &RAGConfig{
		config: &config.Config{
			RAG: config.RAGConfig{
				Splitter: config.SplitterConfig{
					Provider:     "recursive",
					ChunkSize:    500,
					ChunkOverlap: 50,
				},
				Threshold: 0.5,
				TopK:      10,
			},
			LLM: config.LLMConfig{
				Provider:    "openai",
				APIKey:      "",
				BaseURL:     "",
				Model:       "gpt-4o",
				Temperature: 0.5,
				MaxTokens:   2048,
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
	// Parse RAG configuration
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
		if threshold, exists := ragConfig["threshold"].(float64); exists {
			c.config.RAG.Threshold = threshold
		}
		if topK, exists := ragConfig["top_k"].(float64); exists {
			c.config.RAG.TopK = int(topK)
		}
	}

	// Parse Embedding configuration
	if embeddingConfig, ok := config["embedding"].(map[string]any); ok {
		if provider, exists := embeddingConfig["provider"].(string); exists {
			c.config.Embedding.Provider = provider
		} else {
			return errors.New("missing embedding provider")
		}

		if apiKey, exists := embeddingConfig["api_key"].(string); exists {
			c.config.Embedding.APIKey = apiKey
		}
		if baseURL, exists := embeddingConfig["base_url"].(string); exists {
			c.config.Embedding.BaseURL = baseURL
		}
		if model, exists := embeddingConfig["model"].(string); exists {
			c.config.Embedding.Model = model
		}
		if dimension, exists := embeddingConfig["dimension"].(float64); exists {
			c.config.Embedding.Dimension = int(dimension)
		}
	}

	// Parse VectorDB configuration
	if vectordbConfig, ok := config["vectordb"].(map[string]any); ok {
		if provider, exists := vectordbConfig["provider"].(string); exists {
			c.config.VectorDB.Provider = provider
		} else {
			return errors.New("missing vectordb provider")
		}
		if host, exists := vectordbConfig["host"].(string); exists {
			c.config.VectorDB.Host = host
		}
		if port, exists := vectordbConfig["port"].(float64); exists {
			c.config.VectorDB.Port = int(port)
		}
		if dbName, exists := vectordbConfig["database"].(string); exists {
			c.config.VectorDB.Database = dbName
		}
		if collection, exists := vectordbConfig["collection"].(string); exists {
			c.config.VectorDB.Collection = collection
		}
		if username, exists := vectordbConfig["username"].(string); exists {
			c.config.VectorDB.Username = username
		}
		if password, exists := vectordbConfig["password"].(string); exists {
			c.config.VectorDB.Password = password
		}
	}

	return nil
}

func (c *RAGConfig) NewServer(serverName string) (*common.MCPServer, error) {
	mcpServer := common.NewMCPServer(
		serverName,
		Version,
		common.WithInstructions("This is a RAG (Retrieval-Augmented Generation) server for knowledge management and intelligent Q&A"),
	)

	// Initialize RAG client with configuration
	ragClient, err := NewRAGClient(c.config)
	if err != nil {
		return nil, fmt.Errorf("create rag client failed, err: %w", err)
	}

	// Knowledge Base Management Tools
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("create-chunks-from-text", "Process and segment input text into semantic chunks for knowledge base ingestion", GetCreateChunkFromTextSchema()),
		HandleCreateChunkFromText(ragClient),
	)

	// Chunk Management Tools
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("list-chunks", "Retrieve and display all knowledge chunks in the database", GetListChunksSchema()),
		HandleListChunks(ragClient),
	)
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("delete-chunk", "Remove a specific knowledge chunk from the database using its unique identifier", GetDeleteChunkSchema()),
		HandleDeleteChunk(ragClient),
	)

	// Semantic Search Tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("search-chunks", "Perform semantic search across knowledge chunks using natural language query", GetSearchSchema()),
		HandleSearch(ragClient),
	)

	// Intelligent Q&A Tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("chat", "Generate contextually relevant responses using RAG system with LLM integration", GetChatSchema()),
		HandleChat(ragClient),
	)

	return mcpServer, nil
}
