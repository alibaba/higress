package tool_search

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

type MilvusVectorStoreProvider struct {
	client     client.Client
	collection string
	dimensions int
}

func NewMilvusVectorStoreProvider(cfg *config.VectorDBConfig, dimensions int) (*MilvusVectorStoreProvider, error) {
	connectParam := client.Config{
		Address: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
	}
	connectParam.DBName = cfg.Database

	// 添加认证信息（如果提供）
	if cfg.Username != "" && cfg.Password != "" {
		connectParam.Username = cfg.Username
		connectParam.Password = cfg.Password
	}

	milvusClient, err := client.NewClient(context.Background(), connectParam)
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus client: %w", err)
	}

	return &MilvusVectorStoreProvider{
		client:     milvusClient,
		collection: cfg.Collection,
		dimensions: dimensions,
	}, nil
}

func (c *MilvusVectorStoreProvider) ListAllDocs(ctx context.Context) ([]schema.Document, error) {

	expr := ""

	outputFields := []string{"id", "content", "metadata", "created_at"}

	queryResult, err := c.client.Query(
		ctx,
		c.collection,
		[]string{}, // partitions
		expr,       // filter condition
		outputFields,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query all documents: %w", err)
	}

	if len(queryResult) == 0 {
		return []schema.Document{}, nil
	}

	rowCount := queryResult[0].Len()
	documents := make([]schema.Document, 0, rowCount)

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

func (c *MilvusVectorStoreProvider) SearchDocs(ctx context.Context, vector []float32, options *schema.SearchOptions) ([]schema.SearchResult, error) {
	if options == nil {
		options = &schema.SearchOptions{TopK: 10}
	}

	sp, err := entity.NewIndexHNSWSearchParam(16) // 默认 HNSW 搜索参数
	if err != nil {
		return nil, fmt.Errorf("failed to build search param: %w", err)
	}

	outputFields := []string{"id", "content", "metadata"}
	searchResults, err := c.client.Search(
		ctx,
		c.collection,
		[]string{},   // partition names
		"",           // filter expression
		outputFields, // output fields
		[]entity.Vector{entity.FloatVector(vector)},
		"vector",  // anns_field
		entity.IP, // metric_type
		options.TopK,
		sp,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	var results []schema.SearchResult
	for _, result := range searchResults {
		for i := 0; i < result.ResultCount; i++ {
			id, _ := result.IDs.Get(i)
			score := result.Scores[i]

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

func (c *MilvusVectorStoreProvider) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
