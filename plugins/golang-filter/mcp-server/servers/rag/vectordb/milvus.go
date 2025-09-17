package vectordb

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
	if cfg.KnowledgeCollection == "" {
		cfg.KnowledgeCollection = schema.DEFAULT_KNOWLEDGE_COLLECTION
	}
	if cfg.DocumentCollection == "" {
		cfg.DocumentCollection = schema.DEFAULT_DOCUMENT_COLLECTION
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

	if cfg.KnowledgeCollection == "" {
		return fmt.Errorf("milvus knowledge collection is required")
	}
	if cfg.DocumentCollection == "" {
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
	client              client.Client
	config              *config.VectorDBConfig
	knowledgeCollection string
	documentCollection  string
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
		client:              milvusClient,
		config:              cfg,
		knowledgeCollection: cfg.KnowledgeCollection,
		documentCollection:  cfg.DocumentCollection,
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
	document_exists, err := m.client.HasCollection(ctx, m.documentCollection)
	if err != nil {
		return fmt.Errorf("failed to check %s collection existence: %w", m.documentCollection, err)
	}

	knowledge_exists, err := m.client.HasCollection(ctx, m.knowledgeCollection)
	if err != nil {
		return fmt.Errorf("failed to check %s collection existence: %w", m.knowledgeCollection, err)
	}

	if !document_exists {
		fmt.Printf("create collection %s\n", m.documentCollection)
		// 创建schema
		schema := entity.NewSchema().
			WithName(m.documentCollection).
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

		// 文档索引字段 - DocumentIndex
		documentIndexField := entity.NewField().
			WithName("document_index").
			WithDataType(entity.FieldTypeInt32)
		schema.WithField(documentIndexField)

		// 分数字段 - Score (用于搜索结果)
		knowledgeIDField := entity.NewField().
			WithName("Knowledge_id").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(128)
		schema.WithField(knowledgeIDField)

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

		err = m.client.CreateIndex(ctx, m.documentCollection, "vector", vectorIndex, false, client.WithIndexName("vector_index"))
		if err != nil {
			return fmt.Errorf("failed to create vector index: %w", err)
		}
		// Create INVERTED index for knowledge_id
		knowledgeIDIndex := entity.NewScalarIndexWithType(entity.Inverted)
		if err := m.client.CreateIndex(ctx, m.documentCollection, "knowledge_id", knowledgeIDIndex, false, client.WithIndexName("knowledge_id_index")); err != nil {
			return fmt.Errorf("failed to create metadata knowledge_id index: %w", err)
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

	if !knowledge_exists {
		fmt.Printf("create collection %s\n", m.knowledgeCollection)
		// 创建knowledge集合的schema
		knowledgeSchema := entity.NewSchema().
			WithName(m.knowledgeCollection).
			WithDescription("Knowledge collection").
			WithAutoID(false).
			WithDynamicFieldEnabled(false)

		// 添加字段 - 根据 schema.Knowledge 结构
		// 主键字段 - ID
		knowledgePkField := entity.NewField().
			WithName("id").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(256).
			WithIsPrimaryKey(true).
			WithIsAutoID(false)
		knowledgeSchema.WithField(knowledgePkField)

		// 名称字段 - Name
		nameField := entity.NewField().
			WithName("name").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(512)
		knowledgeSchema.WithField(nameField)

		// 源URL字段 - SourceURL
		sourceURLField := entity.NewField().
			WithName("source_url").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(512)
		knowledgeSchema.WithField(sourceURLField)

		// 状态字段 - Status
		statusField := entity.NewField().
			WithName("status").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(64)
		knowledgeSchema.WithField(statusField)

		// 文件大小字段 - FileSize
		fileSizeField := entity.NewField().
			WithName("file_size").
			WithDataType(entity.FieldTypeInt64)
		knowledgeSchema.WithField(fileSizeField)

		// 块数量字段 - ChunkCount
		chunkCountField := entity.NewField().
			WithName("chunk_count").
			WithDataType(entity.FieldTypeInt32)
		knowledgeSchema.WithField(chunkCountField)

		// 多模态启用字段 - EnableMultimodel
		enableMultimodelField := entity.NewField().
			WithName("enable_multimodel").
			WithDataType(entity.FieldTypeBool)
		knowledgeSchema.WithField(enableMultimodelField)

		// 元数据字段 - Metadata
		knowledgeMetadataField := entity.NewField().
			WithName("metadata").
			WithDataType(entity.FieldTypeJSON)
		knowledgeSchema.WithField(knowledgeMetadataField)

		// 向量字段 - Vector demo
		vectorField := entity.NewField().
			WithName("vector").
			WithDataType(entity.FieldTypeFloatVector).
			WithDim(int64(MILVUS_DUMMY_DIM))
		knowledgeSchema.WithField(vectorField)

		// 创建时间字段 - CreatedAt (存储为Unix时间戳)
		createdAtField := entity.NewField().
			WithName("created_at").
			WithDataType(entity.FieldTypeInt64)
		knowledgeSchema.WithField(createdAtField)

		// 更新时间字段 - UpdatedAt (存储为Unix时间戳)
		updatedAtField := entity.NewField().
			WithName("updated_at").
			WithDataType(entity.FieldTypeInt64)
		knowledgeSchema.WithField(updatedAtField)

		// 完成时间字段 - CompletedAt (存储为Unix时间戳，可为空)
		completedAtField := entity.NewField().
			WithName("completed_at").
			WithDataType(entity.FieldTypeInt64)
		knowledgeSchema.WithField(completedAtField)

		// 创建knowledge集合
		err = m.client.CreateCollection(ctx, knowledgeSchema, entity.DefaultShardNumber)
		if err != nil {
			return fmt.Errorf("failed to create knowledge collection: %w", err)
		}

		// 创建向量索引
		vectorIndex, err := entity.NewIndexHNSW(entity.IP, 8, 64)
		if err != nil {
			return fmt.Errorf("failed to create vector index: %w", err)
		}
		// 为knowledge集合创建索引
		err = m.client.CreateIndex(ctx, m.knowledgeCollection, "vector", vectorIndex, false, client.WithIndexName("vector_index"))
		if err != nil {
			return fmt.Errorf("failed to create vector index: %w", err)
		}
		// 为name字段创建INVERTED索引
		nameIndex := entity.NewScalarIndexWithType(entity.Inverted)
		if err := m.client.CreateIndex(ctx, m.knowledgeCollection, "name", nameIndex, false, client.WithIndexName("name_index")); err != nil {
			return fmt.Errorf("failed to create name index: %w", err)
		}
	}

	// 加载集合
	err = m.client.LoadCollection(ctx, m.documentCollection, false)
	if err != nil {
		return fmt.Errorf("failed to load document collection: %w", err)
	}

	err = m.client.LoadCollection(ctx, m.knowledgeCollection, false)
	if err != nil {
		return fmt.Errorf("failed to load knowledge collection: %w", err)
	}

	return nil
}

// DropCollection 删除集合
func (m *MilvusProvider) DropCollection(ctx context.Context) error {
	// 检查集合是否存在
	exists, err := m.client.HasCollection(ctx, m.documentCollection)
	if err != nil {
		return fmt.Errorf("failed to check %s collection existence: %w", m.documentCollection, err)
	}
	if !exists {
		return fmt.Errorf("collection %s does not exist", m.documentCollection)
	}
	// 删除集合
	err = m.client.DropCollection(ctx, m.documentCollection)
	if err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}

	exists, err = m.client.HasCollection(ctx, m.knowledgeCollection)
	if err != nil {
		return fmt.Errorf("failed to check %s collection existence: %w", m.knowledgeCollection, err)
	}
	if !exists {
		return fmt.Errorf("collection %s does not exist", m.knowledgeCollection)
	}
	// 删除集合
	err = m.client.DropCollection(ctx, m.knowledgeCollection)
	if err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}
	return nil
}

// AddDoc 添加文档到向量数据库
func (m *MilvusProvider) AddDoc(ctx context.Context, filename string, docs []schema.Document) error {
	if len(docs) == 0 {
		return nil
	}

	// 准备数据
	ids := make([]string, len(docs))
	contents := make([]string, len(docs))
	vectors := make([][]float32, len(docs))
	metadatas := make([][]byte, len(docs))
	documentIndexes := make([]int32, len(docs))
	scores := make([]float32, len(docs))

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

		documentIndexes[i] = int32(doc.DocumentIndex)
		scores[i] = float32(doc.Score)
	}

	// 构建插入数据
	columns := []entity.Column{
		entity.NewColumnVarChar("id", ids),
		entity.NewColumnVarChar("content", contents),
		entity.NewColumnFloatVector("vector", len(vectors[0]), vectors),
		entity.NewColumnJSONBytes("metadata", metadatas),
		entity.NewColumnInt32("document_index", documentIndexes),
	}

	// 插入数据
	_, err := m.client.Insert(ctx, m.documentCollection, "", columns...)
	if err != nil {
		return fmt.Errorf("failed to insert documents: %w", err)
	}

	// 刷新数据
	err = m.client.Flush(ctx, m.documentCollection, false)
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
	err := m.client.Delete(ctx, m.documentCollection, "", expr)
	if err != nil {
		return fmt.Errorf("failed to delete documents for id %s: %w", id, err)
	}

	// 刷新数据
	err = m.client.Flush(ctx, m.documentCollection, false)
	if err != nil {
		return fmt.Errorf("failed to flush collection after delete: %w", err)
	}

	return nil
}

// UpdateDoc 更新文档 - 先删除再添加
func (m *MilvusProvider) UpdateDoc(ctx context.Context, filename string, docs []schema.Document) error {
	// 先删除现有文档
	if err := m.DeleteDoc(ctx, filename); err != nil {
		return fmt.Errorf("failed to delete existing documents: %w", err)
	}

	// 添加新文档
	if err := m.AddDoc(ctx, filename, docs); err != nil {
		return fmt.Errorf("failed to add new documents: %w", err)
	}

	return nil
}

// SearchDocs 搜索相似文档
func (m *MilvusProvider) SearchDocs(ctx context.Context, vector []float32, options *schema.SearchOptions) ([]schema.Document, error) {
	if options == nil {
		options = &schema.SearchOptions{TopK: 10}
	}

	// 构建搜索参数
	sp, _ := entity.NewIndexHNSWSearchParam(16)

	// 构建过滤表达式
	expr := ""
	// if options.Filters != nil {
	// 	if knowledgeID, ok := options.Filters[schema.META_KNOWLEDGE_ID].(string); ok && knowledgeID != "" {
	// 		expr = fmt.Sprintf(`json_contains(metadata, '{"knowledge_id": "%s"}')`)
	// 	}
	// }

	searchResult, err := m.client.Search(
		ctx,
		m.documentCollection,
		[]string{}, // 分区名
		expr,       // 过滤表达式
		[]string{"id", "content", "metadata", "document_index"}, // 输出字段
		[]entity.Vector{entity.FloatVector(vector)},
		"vector",  // anns_field
		entity.IP, // metric_type
		options.TopK,
		sp,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	// 解析结果
	var results []schema.Document
	for _, result := range searchResult {
		for i := 0; i < result.ResultCount; i++ {
			id, _ := result.IDs.Get(i)
			score := result.Scores[i]
			// 获取字段数据
			var content string
			var metadata map[string]interface{}
			var documentIndex int

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
				case "document_index":
					if docIndexCol, ok := field.(*entity.ColumnInt32); ok {
						if docIndexVal, err := docIndexCol.Get(i); err == nil {
							if docIndexInt, ok := docIndexVal.(int32); ok {
								documentIndex = int(docIndexInt)
							}
						}
					}
				}
			}

			results = append(results, schema.Document{
				ID:            fmt.Sprintf("%v", id),
				Content:       content,
				Metadata:      metadata,
				DocumentIndex: documentIndex,
				Score:         float64(score), // 使用搜索返回的相似度分数
			})
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
	expr := fmt.Sprintf("id in [%s]", joinStrings(ids, ","))

	// 删除数据
	err := m.client.Delete(ctx, m.documentCollection, "", expr)
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	// 刷新数据
	err = m.client.Flush(ctx, m.documentCollection, false)
	if err != nil {
		return fmt.Errorf("failed to flush collection after delete: %w", err)
	}

	return nil
}

func (m *MilvusProvider) ListDocs(ctx context.Context, knowledgeID string, limit int) ([]schema.Document, error) {
	// 构建查询表达式
	expr := fmt.Sprintf(`knowledge_id == "%s"`, knowledgeID)
	// 查询所有相关文档
	queryResult, err := m.client.Query(
		ctx,
		m.documentCollection,
		[]string{}, // 分区
		expr,       // 过滤条件
		[]string{"id", "content", "metadata", "document_index", "created_at"},
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
			id            string
			content       string
			metadata      map[string]interface{}
			documentIndex int32
			createdAt     int64
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
			case "document_index":
				if v, err := col.(*entity.ColumnInt32).Get(i); err == nil {
					documentIndex = v.(int32)
				}
			case "created_at":
				if v, err := col.(*entity.ColumnInt64).Get(i); err == nil {
					createdAt = v.(int64)
				}
			}
		}

		doc := schema.Document{
			ID:            id,
			Content:       content,
			Metadata:      metadata,
			DocumentIndex: int(documentIndex),
			CreatedAt:     time.UnixMilli(createdAt),
		}
		documents = append(documents, doc)
	}
	// 按照 document_index 升序排序
	sort.Slice(documents, func(i, j int) bool {
		return documents[i].DocumentIndex < documents[j].DocumentIndex
	})
	return documents, nil
}

func (m *MilvusProvider) CreateKnowledge(ctx context.Context, knowledge schema.Knowledge) error {
	// Prepare single-row data slices
	ids := []string{knowledge.ID}
	names := []string{knowledge.Name}
	sourceURLs := []string{knowledge.SourceURL}
	statuses := []string{knowledge.Status}
	fileSizes := []int64{knowledge.FileSize}
	chunkCounts := []int32{int32(knowledge.ChunkCount)}
	enableMultimodels := []bool{knowledge.EnableMultimodel}

	// Marshal metadata
	var metaBytes []byte
	if knowledge.Metadata != nil {
		var err error
		metaBytes, err = json.Marshal(knowledge.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metaBytes = []byte("{}")
	}
	metadatas := [][]byte{metaBytes}

	createdAts := []int64{knowledge.CreatedAt.UnixMilli()}
	updatedAts := []int64{knowledge.UpdatedAt.UnixMilli()}
	completedAts := []int64{knowledge.CompletedAt.UnixMilli()}

	// Dummy vector for compatibility with schema definition
	dummyVec := make([]float32, MILVUS_DUMMY_DIM)
	vectors := [][]float32{dummyVec}

	columns := []entity.Column{
		entity.NewColumnVarChar("id", ids),
		entity.NewColumnVarChar("name", names),
		entity.NewColumnVarChar("source_url", sourceURLs),
		entity.NewColumnVarChar("status", statuses),
		entity.NewColumnInt64("file_size", fileSizes),
		entity.NewColumnInt32("chunk_count", chunkCounts),
		entity.NewColumnBool("enable_multimodel", enableMultimodels),
		entity.NewColumnJSONBytes("metadata", metadatas),
		entity.NewColumnInt64("created_at", createdAts),
		entity.NewColumnInt64("updated_at", updatedAts),
		entity.NewColumnInt64("completed_at", completedAts),
		entity.NewColumnFloatVector("vector", MILVUS_DUMMY_DIM, vectors),
	}

	// Insert into Milvus
	if _, err := m.client.Insert(ctx, m.knowledgeCollection, "", columns...); err != nil {
		return fmt.Errorf("failed to insert knowledge: %w", err)
	}

	// Flush to make sure data is persisted
	if err := m.client.Flush(ctx, m.knowledgeCollection, false); err != nil {
		return fmt.Errorf("failed to flush knowledge collection: %w", err)
	}

	return nil
}

// ListKnowledge 列出所有知识
func (m *MilvusProvider) ListKnowledge(ctx context.Context, limit int) ([]schema.Knowledge, error) {
	// 查询所有 knowledge 记录
	queryResult, err := m.client.Query(
		ctx,
		m.knowledgeCollection,
		[]string{}, // 分区
		"",         // 无过滤条件
		[]string{"id", "name", "source_url", "status", "file_size", "chunk_count", "enable_multimodel", "metadata", "created_at", "updated_at", "completed_at"},
		client.WithLimit(int64(limit)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query knowledge list: %w", err)
	}

	if len(queryResult) == 0 {
		return []schema.Knowledge{}, nil
	}

	rowCount := queryResult[0].Len()
	knowledgeList := make([]schema.Knowledge, 0, rowCount)

	for i := 0; i < rowCount; i++ {
		var (
			id               string
			name             string
			sourceURL        string
			status           string
			fileSize         int64
			chunkCount       int32
			enableMultimodel bool
			metadata         map[string]interface{}
			createdAt        int64
			updatedAt        int64
			completedAt      int64
		)

		for _, col := range queryResult {
			switch col.Name() {
			case "id":
				if v, err := col.(*entity.ColumnVarChar).Get(i); err == nil {
					id = v.(string)
				}
			case "name":
				if v, err := col.(*entity.ColumnVarChar).Get(i); err == nil {
					name = v.(string)
				}
			case "source_url":
				if v, err := col.(*entity.ColumnVarChar).Get(i); err == nil {
					sourceURL = v.(string)
				}
			case "status":
				if v, err := col.(*entity.ColumnVarChar).Get(i); err == nil {
					status = v.(string)
				}
			case "file_size":
				if v, err := col.(*entity.ColumnInt64).Get(i); err == nil {
					fileSize = v.(int64)
				}
			case "chunk_count":
				if v, err := col.(*entity.ColumnInt32).Get(i); err == nil {
					chunkCount = v.(int32)
				}
			case "enable_multimodel":
				if v, err := col.(*entity.ColumnBool).Get(i); err == nil {
					enableMultimodel = v.(bool)
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
			case "updated_at":
				if v, err := col.(*entity.ColumnInt64).Get(i); err == nil {
					updatedAt = v.(int64)
				}
			case "completed_at":
				if v, err := col.(*entity.ColumnInt64).Get(i); err == nil {
					completedAt = v.(int64)
				}
			}
		}

		knowledge := schema.Knowledge{
			ID:               id,
			Name:             name,
			SourceURL:        sourceURL,
			Status:           status,
			FileSize:         fileSize,
			ChunkCount:       int(chunkCount),
			EnableMultimodel: enableMultimodel,
			Metadata:         metadata,
			CreatedAt:        time.UnixMilli(createdAt),
			UpdatedAt:        time.UnixMilli(updatedAt),
			CompletedAt:      time.UnixMilli(completedAt),
		}
		knowledgeList = append(knowledgeList, knowledge)
	}

	return knowledgeList, nil
}

// UpdateKnowledge 更新知识
func (m *MilvusProvider) UpdateKnowledge(ctx context.Context, knowledge schema.Knowledge) error {
	// 构建更新表达式
	if err := m.DeleteKnowledge(ctx, knowledge.ID); err != nil {
		return fmt.Errorf("delete knowledge failed, err: %w", err)
	}

	if err := m.CreateKnowledge(ctx, knowledge); err != nil {
		return fmt.Errorf("add knowledge failed, err: %w", err)
	}
	return nil
}

// GetKnowledge 获取特定知识详情
func (m *MilvusProvider) GetKnowledge(ctx context.Context, knowledgeID string) (*schema.Knowledge, error) {
	// // 先查询 knowledge 元信息
	expr := fmt.Sprintf(`id == "%s"`, knowledgeID)
	knowledgeResult, err := m.client.Query(ctx, m.knowledgeCollection, []string{}, expr, []string{"id", "name", "source_url", "status", "file_size", "chunk_count", "enable_multimodel", "metadata", "created_at", "updated_at", "completed_at"})
	if err != nil {
		return nil, fmt.Errorf("failed to query knowledge: %w", err)
	}
	if len(knowledgeResult) == 0 || knowledgeResult[0].Len() == 0 {
		return nil, fmt.Errorf("knowledge not found: %s", knowledgeID)
	}

	// 解析第一行
	var k schema.Knowledge
	{
		cols := knowledgeResult
		for _, col := range cols {
			switch col.Name() {
			case "id":
				if v, _ := col.(*entity.ColumnVarChar).Get(0); v != nil {
					k.ID = v.(string)
				}
			case "name":
				if v, _ := col.(*entity.ColumnVarChar).Get(0); v != nil {
					k.Name = v.(string)
				}
			case "source_url":
				if v, _ := col.(*entity.ColumnVarChar).Get(0); v != nil {
					k.SourceURL = v.(string)
				}
			case "status":
				if v, _ := col.(*entity.ColumnVarChar).Get(0); v != nil {
					k.Status = v.(string)
				}
			case "file_size":
				if v, _ := col.(*entity.ColumnInt64).Get(0); v != nil {
					k.FileSize = v.(int64)
				}
			case "chunk_count":
				if v, _ := col.(*entity.ColumnInt32).Get(0); v != nil {
					k.ChunkCount = int(v.(int32))
				}
			case "enable_multimodel":
				if v, _ := col.(*entity.ColumnBool).Get(0); v != nil {
					k.EnableMultimodel = v.(bool)
				}
			case "metadata":
				if v, _ := col.(*entity.ColumnJSONBytes).Get(0); v != nil {
					bytes := v.([]byte)
					_ = json.Unmarshal(bytes, &k.Metadata)
				}
			case "created_at":
				if v, _ := col.(*entity.ColumnInt64).Get(0); v != nil {
					k.CreatedAt = time.UnixMilli(v.(int64))
				}
			case "updated_at":
				if v, _ := col.(*entity.ColumnInt64).Get(0); v != nil {
					k.UpdatedAt = time.UnixMilli(v.(int64))
				}
			case "completed_at":
				if v, _ := col.(*entity.ColumnInt64).Get(0); v != nil {
					k.CompletedAt = time.UnixMilli(v.(int64))
				}
			}
		}

	}

	// // 查询关联的 documents
	// docExpr := fmt.Sprintf(`knowledge_id == "%s"`, knowledgeID)
	// docResult, err := m.client.Query(ctx, m.documentCollection, []string{}, docExpr, []string{"id", "content", "metadata", "document_index", "score"})
	// if err != nil {
	//     return nil, fmt.Errorf("failed to query knowledge documents: %w", err)
	// }

	// if len(docResult) > 0 {
	//     rowCnt := docResult[0].Len()
	//     k.Documents = make([]schema.Document, 0, rowCnt)
	//     for i := 0; i < rowCnt; i++ {
	//         var (
	//             docID         string
	//             content       string
	//             metadata      map[string]interface{}
	//             documentIndex int32
	//             score         float64
	//         )

	//         for _, col := range docResult {
	//             switch col.Name() {
	//             case "id":
	//                 if v, _ := col.(*entity.ColumnVarChar).Get(i); v != nil {
	//                     docID = v.(string)
	//                 }
	//             case "content":
	//                 if v, _ := col.(*entity.ColumnVarChar).Get(i); v != nil {
	//                     content = v.(string)
	//                 }
	//             case "metadata":
	//                 if v, _ := col.(*entity.ColumnJSONBytes).Get(i); v != nil {
	//                     bytes := v.([]byte)
	//                     _ = json.Unmarshal(bytes, &metadata)
	//                 }
	//             case "document_index":
	//                 if v, _ := col.(*entity.ColumnInt32).Get(i); v != nil {
	//                     documentIndex = v.(int32)
	//                 }
	//         }
	//         k.Documents = append(k.Documents, schema.Document{
	//              ID:            docID,
	//              KnowledgeID:   knowledgeID,
	//              Content:       content,
	//              Metadata:      metadata,
	//              DocumentIndex: int(documentIndex),
	//              Score:         score,
	//          })
	//     }
	// }

	return &k, nil
}

// DeleteKnowledge 删除特定知识
func (m *MilvusProvider) DeleteKnowledge(ctx context.Context, knowledgeID string) error {
	// 删除 knowledge 记录
	exprKnowledge := fmt.Sprintf(`id == "%s"`, knowledgeID)
	if err := m.client.Delete(ctx, m.knowledgeCollection, "", exprKnowledge); err != nil {
		return fmt.Errorf("failed to delete knowledge record: %w", err)
	}

	// 删除关联的 document 记录
	exprDocs := fmt.Sprintf(`knowledge_id == "%s"`, knowledgeID)
	if err := m.client.Delete(ctx, m.documentCollection, "", exprDocs); err != nil {
		return fmt.Errorf("failed to delete knowledge documents: %w", err)
	}

	// 刷新
	if err := m.client.Flush(ctx, m.knowledgeCollection, false); err != nil {
		return fmt.Errorf("failed to flush knowledge collection: %w", err)
	}
	if err := m.client.Flush(ctx, m.documentCollection, false); err != nil {
		return fmt.Errorf("failed to flush document collection: %w", err)
	}
	return nil
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
