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

// milvusProviderInitializer Milvus提供者初始化器
type milvusProviderInitializer struct{}

// InitConfig 初始化配置
func (m *milvusProviderInitializer) InitConfig(cfg *config.VectorDBConfig) error {
	if cfg.Provider != MILVUS_PROVIDER_TYPE {
		return fmt.Errorf("provider type mismatch: expected %s, got %s", MILVUS_PROVIDER_TYPE, cfg.Provider)
	}

	// 设置默认值
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

// ValidateConfig 验证配置
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

// CreateProvider 创建Milvus提供者
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

// MilvusProvider Milvus向量数据库提供者
type MilvusProvider struct {
	client     client.Client
	config     *config.VectorDBConfig
	Collection string
}

// NewMilvusProvider 创建新的Milvus提供者
func NewMilvusProvider(cfg *config.VectorDBConfig, dim int) (VectorStoreProvider, error) {
	// 创建Milvus客户端
	connectParam := client.Config{
		Address: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
	}

	connectParam.DBName = cfg.Database
	// 如果有用户名和密码，添加认证
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

// CreateCollection 创建集合
func (m *MilvusProvider) CreateCollection(ctx context.Context, dim int) error {
	// 检查集合是否存在
	document_exists, err := m.client.HasCollection(ctx, m.Collection)
	if err != nil {
		return fmt.Errorf("failed to check %s collection existence: %w", m.Collection, err)
	}

	if !document_exists {
		fmt.Printf("create collection %s\n", m.Collection)
		// 创建schema
		schema := entity.NewSchema().
			WithName(m.Collection).
			WithDescription("Knowledge document collection").
			WithAutoID(false).
			WithDynamicFieldEnabled(false)

		// 添加字段 - 根据 schema.Document 结构
		// 主键字段 - ID
		pkField := entity.NewField().
			WithName("id").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(256).
			WithIsPrimaryKey(true).
			WithIsAutoID(false)
		schema.WithField(pkField)

		// 内容字段 - Content
		contentField := entity.NewField().
			WithName("content").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(8192)
		schema.WithField(contentField)

		// 向量字段 - Vector
		vectorField := entity.NewField().
			WithName("vector").
			WithDataType(entity.FieldTypeFloatVector).
			WithDim(int64(dim))
		schema.WithField(vectorField)

		// 元数据字段 - Metadata
		metadataField := entity.NewField().
			WithName("metadata").
			WithDataType(entity.FieldTypeJSON)
		schema.WithField(metadataField)

		// 创建时间字段 - CreatedAt (存储为Unix时间戳)
		createdAtField := entity.NewField().
			WithName("created_at").
			WithDataType(entity.FieldTypeInt64)
		schema.WithField(createdAtField)

		// 稀疏向量字段 (如果SDK支持)
		// sparseVectorField := entity.NewField().
		// 	WithName("sparse_vector").
		// 	WithDataType(entity.FieldTypeSparseFloatVector)
		// schema.WithField(sparseVectorField)

		// 创建集合
		err = m.client.CreateCollection(ctx, schema, entity.DefaultShardNumber)
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}

		// 创建向量索引
		vectorIndex, err := entity.NewIndexHNSW(entity.IP, 8, 64)
		if err != nil {
			return fmt.Errorf("failed to create vector index: %w", err)
		}

		err = m.client.CreateIndex(ctx, m.Collection, "vector", vectorIndex, false, client.WithIndexName("vector_index"))
		if err != nil {
			return fmt.Errorf("failed to create vector index: %w", err)
		}

		// 创建稀疏向量索引 (如果需要)
		// sparseIndex, err := entity.NewIndexSparseInverted(entity.IP, 0.3)
		// if err != nil {
		// 	return fmt.Errorf("failed to create sparse index: %w", err)
		// }
		// err = m.client.CreateIndex(ctx, m.collection, "sparse_vector", sparseIndex, false, entity.WithIndexName("sparse_vector_index"))
		// if err != nil {
		// 	return fmt.Errorf("failed to create sparse vector index: %w", err)
		// }

	}

	// 加载集合
	err = m.client.LoadCollection(ctx, m.Collection, false)
	if err != nil {
		return fmt.Errorf("failed to load document collection: %w", err)
	}
	return nil
}

// DropCollection 删除集合
func (m *MilvusProvider) DropCollection(ctx context.Context) error {
	// 检查集合是否存在
	exists, err := m.client.HasCollection(ctx, m.Collection)
	if err != nil {
		return fmt.Errorf("failed to check %s collection existence: %w", m.Collection, err)
	}
	if !exists {
		return fmt.Errorf("collection %s does not exist", m.Collection)
	}
	// 删除集合
	err = m.client.DropCollection(ctx, m.Collection)
	if err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}
	return nil
}

// AddDoc 添加文档到向量数据库
func (m *MilvusProvider) AddDoc(ctx context.Context, docs []schema.Document) error {
	if len(docs) == 0 {
		return nil
	}
	// 准备数据
	ids := make([]string, len(docs))
	contents := make([]string, len(docs))
	vectors := make([][]float32, len(docs))
	metadatas := make([][]byte, len(docs))
	createdAts := make([]int64, len(docs))

	for i, doc := range docs {
		ids[i] = doc.ID
		contents[i] = doc.Content

		// 转换向量类型
		vectorFloat32 := make([]float32, len(doc.Vector))
		for j, v := range doc.Vector {
			vectorFloat32[j] = float32(v)
		}
		vectors[i] = vectorFloat32

		// 序列化元数据
		metadataBytes, err := json.Marshal(doc.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata for doc %s: %w", doc.ID, err)
		}
		metadatas[i] = metadataBytes

		createdAts[i] = doc.CreatedAt.UnixMilli()
	}

	// 构建插入数据
	columns := []entity.Column{
		entity.NewColumnVarChar("id", ids),
		entity.NewColumnVarChar("content", contents),
		entity.NewColumnFloatVector("vector", len(vectors[0]), vectors),
		entity.NewColumnJSONBytes("metadata", metadatas),
		entity.NewColumnInt64("created_at", createdAts),
	}

	// 插入数据
	_, err := m.client.Insert(ctx, m.Collection, "", columns...)
	if err != nil {
		return fmt.Errorf("failed to insert documents: %w", err)
	}

	// 刷新数据
	err = m.client.Flush(ctx, m.Collection, false)
	if err != nil {
		return fmt.Errorf("failed to flush collection: %w", err)
	}

	return nil
}

// DeleteDoc 删除文档 - 根据文件名删除所有相关文档
func (m *MilvusProvider) DeleteDoc(ctx context.Context, id string) error {
	// 构建删除表达式 - 根据元数据中的文件名
	expr := fmt.Sprintf(`id == "%s"`, id)
	// 删除数据
	err := m.client.Delete(ctx, m.Collection, "", expr)
	if err != nil {
		return fmt.Errorf("failed to delete documents for id %s: %w", id, err)
	}

	// 刷新数据
	err = m.client.Flush(ctx, m.Collection, false)
	if err != nil {
		return fmt.Errorf("failed to flush collection after delete: %w", err)
	}

	return nil
}

// UpdateDoc 更新文档 - 先删除再添加
func (m *MilvusProvider) UpdateDoc(ctx context.Context, docs []schema.Document) error {
	// 先删除现有文档
	ids := make([]string, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
	}
	if err := m.DeleteDocs(ctx, ids); err != nil {
		return fmt.Errorf("failed to delete existing documents: %w", err)
	}
	// 添加新文档
	if err := m.AddDoc(ctx, docs); err != nil {
		return fmt.Errorf("failed to add new documents: %w", err)
	}

	return nil
}

// SearchDocs 搜索相似文档
func (m *MilvusProvider) SearchDocs(ctx context.Context, vector []float32, options *schema.SearchOptions) ([]schema.SearchResult, error) {
	if options == nil {
		options = &schema.SearchOptions{TopK: 10}
	}
	// 构建搜索参数
	sp, _ := entity.NewIndexHNSWSearchParam(16)
	// 构建过滤表达式
	expr := ""
	searchResults, err := m.client.Search(
		ctx,
		m.Collection,
		[]string{}, // 分区名
		expr,       // 过滤表达式
		[]string{"id", "content", "metadata", "created_at"}, // 输出字段
		[]entity.Vector{entity.FloatVector(vector)},
		"vector",  // anns_field
		entity.IP, // metric_type
		options.TopK,
		sp,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	// // 解析结果
	var results []schema.SearchResult
	for _, result := range searchResults {
		for i := 0; i < result.ResultCount; i++ {
			id, _ := result.IDs.Get(i)
			score := result.Scores[i]
			// 获取字段数据
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

// DeleteDocs 删除文档
func (m *MilvusProvider) DeleteDocs(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// 构建删除表达式
	// expr := fmt.Sprintf("id in [%s]", joinStrings(ids, ","))
	// Milvus expects string values to be quoted within the expression, otherwise the parser will
	// treat the hyphen inside UUID as a minus operator and raise a parse error.
	quotedIDs := make([]string, len(ids))
	for i, id := range ids {
		quotedIDs[i] = fmt.Sprintf("\"%s\"", id)
	}
	expr := fmt.Sprintf("id in [%s]", strings.Join(quotedIDs, ","))

	// 删除数据
	err := m.client.Delete(ctx, m.Collection, "", expr)
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}
	// 刷新数据
	err = m.client.Flush(ctx, m.Collection, false)
	if err != nil {
		return fmt.Errorf("failed to flush collection after delete: %w", err)
	}

	return nil
}

func (m *MilvusProvider) ListDocs(ctx context.Context, limit int) ([]schema.Document, error) {
	// 构建查询表达式
	expr := ""
	// 查询所有相关文档
	queryResult, err := m.client.Query(
		ctx,
		m.Collection,
		[]string{}, // 分区
		expr,       // 过滤条件
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

	// 解析查询结果
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

// GetProviderType 获取提供者类型
func (m *MilvusProvider) GetProviderType() string {
	return MILVUS_PROVIDER_TYPE
}

// Close 关闭连接
func (m *MilvusProvider) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// joinStrings joins a slice of strings with the given separator.
func joinStrings(elems []string, sep string) string {
	return strings.Join(elems, sep)
}
