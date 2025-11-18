package tool_search

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/vectordb"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

// DBClient handles Milvus database connections and operations
type DBClient struct {
	vectorDB     vectordb.VectorStoreProvider
	config       *config.VectorDBConfig
	tableName    string
	gatewayID    string
	reconnect    chan struct{}
	stop         chan struct{}
	panicCount   int32
	dimensions   int
	embeddingAPI string
}

// ToolRecord represents a tool record in the database
type ToolRecord struct {
	ID         string                 `json:"id"`
	ServerName string                 `json:"server_name"`
	Name       string                 `json:"name"`
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata"`
	GatewayID  string                 `json:"gateway_id"`
}

// NewDBClient creates a new DBClient instance
func NewDBClient(host string, port int, database, username, password, tableName, gatewayID string, stop chan struct{}) *DBClient {
	api.LogInfof("Creating DBClient with tableName: %s, gatewayID: %s", tableName, gatewayID)

	// Parse DSN to extract Milvus configuration
	cfg := &config.VectorDBConfig{
		Provider:   "milvus",
		Host:       host,
		Port:       port,
		Database:   database,
		Collection: tableName,
		Username:   username,
		Password:   password,
	}

	client := &DBClient{
		config:     cfg,
		tableName:  tableName,
		gatewayID:  gatewayID,
		reconnect:  make(chan struct{}, 1),
		stop:       stop,
		dimensions: 1024, // Default dimensions
	}

	// Start reconnection goroutine
	go client.reconnectLoop()

	// Try initial connection
	if err := client.connect(); err != nil {
		api.LogErrorf("Initial database connection failed: %v", err)
	}

	return client
}

func (c *DBClient) connect() error {
	api.LogInfo("Connecting to Milvus database")

	// Create Milvus provider
	provider, err := vectordb.NewVectorDBProvider(c.config, c.dimensions)
	if err != nil {
		return fmt.Errorf("failed to create Milvus provider: %w", err)
	}

	c.vectorDB = provider
	api.LogInfo("Milvus database connected successfully")
	return nil
}

func (c *DBClient) reconnectLoop() {
	defer func() {
		if r := recover(); r != nil {
			api.LogErrorf("Recovered from panic in reconnectLoop: %v", r)

			// Increment panic counter
			atomic.AddInt32(&c.panicCount, 1)

			// If panic count exceeds threshold, stop trying to reconnect
			if atomic.LoadInt32(&c.panicCount) > 3 {
				api.LogErrorf("Too many panics in reconnectLoop, stopping reconnection attempts")
				return
			}

			// Wait for a while before restarting
			time.Sleep(5 * time.Second)

			// Restart the reconnect loop
			go c.reconnectLoop()
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			api.LogInfof("Database connection closed")
			return
		case <-ticker.C:
			if c.vectorDB == nil {
				api.LogInfo("Attempting to reconnect to database")
				if err := c.connect(); err != nil {
					api.LogErrorf("Database reconnection failed: %v", err)
				} else {
					api.LogInfof("Database reconnected successfully")
					atomic.StoreInt32(&c.panicCount, 0)
				}
			}
		case <-c.reconnect:
			api.LogInfo("Reconnection signal received")
			if err := c.connect(); err != nil {
				api.LogErrorf("Database reconnection failed: %v", err)
			} else {
				api.LogInfof("Database reconnected successfully")
				atomic.StoreInt32(&c.panicCount, 0)
			}
		}
	}
}

func (c *DBClient) reconnectIfDbEmpty() error {
	if c.vectorDB == nil {
		api.LogWarn("Database is not connected, attempting to reconnect")
		select {
		case c.reconnect <- struct{}{}:
		default:
		}
		return fmt.Errorf("database is not connected, attempting to reconnect")
	}
	return nil
}

// Ping checks database connectivity
func (c *DBClient) Ping() error {
	if c.vectorDB == nil {
		return fmt.Errorf("database connection is nil")
	}
	return nil
}

func (c *DBClient) SearchTools(query string, vector []float32, topK int) ([]ToolRecord, error) {
	api.LogInfof("Performing vector search for query: '%s', topK: %d", query, topK)
	if err := c.reconnectIfDbEmpty(); err != nil {
		return nil, err
	}

	// For Milvus, we'll perform vector search directly
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Perform vector search
	searchOptions := &schema.SearchOptions{
		TopK: topK,
	}

	results, err := c.vectorDB.SearchDocs(ctx, vector, searchOptions)
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
			GatewayID: c.gatewayID,
		}

		if name, ok := doc.Metadata["name"].(string); ok {
			tool.Name = name
		}

		records = append(records, tool)
	}

	api.LogInfof("Vector search completed, found %d results", len(records))
	return records, nil
}

// GetAllTools retrieves all tools from the database
func (c *DBClient) GetAllTools() ([]ToolRecord, error) {
	api.LogInfof("Executing GetAllTools query from collection: %s", c.tableName)

	if err := c.reconnectIfDbEmpty(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Retrieve all documents
	const maxToolsLimit = 1000
	docs, err := c.vectorDB.ListDocs(ctx, maxToolsLimit)
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
			GatewayID: c.gatewayID,
		}

		if name, ok := doc.Metadata["name"].(string); ok {
			tool.Name = name
		}

		tools = append(tools, tool)
	}

	api.LogInfof("GetAllTools query completed, found %d tools", len(tools))
	return tools, nil
}
