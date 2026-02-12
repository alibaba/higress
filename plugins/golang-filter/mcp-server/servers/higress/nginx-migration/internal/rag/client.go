// Package rag 提供基于阿里云官方 SDK 的 RAG 客户端实现
package rag

import (
	"fmt"
	"log"
	"sync"
	"time"

	bailian "github.com/alibabacloud-go/bailian-20231229/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

// RAGQuery RAG 查询请求
type RAGQuery struct {
	Query       string                 `json:"query"`        // 查询文本
	Scenario    string                 `json:"scenario"`     // 场景标识
	TopK        int                    `json:"top_k"`        // 返回文档数量
	ContextMode string                 `json:"context_mode"` // 上下文模式
	Filters     map[string]interface{} `json:"filters"`      // 过滤条件
}

// RAGResponse RAG 查询响应
type RAGResponse struct {
	Documents []RAGDocument `json:"documents"` // 检索到的文档
	Latency   int64         `json:"latency"`   // 查询延迟（毫秒）
}

// RAGDocument 表示一个检索到的文档
type RAGDocument struct {
	Title      string   `json:"title"`      // 文档标题
	Content    string   `json:"content"`    // 文档内容
	Source     string   `json:"source"`     // 来源路径
	URL        string   `json:"url"`        // 在线链接
	Score      float64  `json:"score"`      // 相关度分数
	Highlights []string `json:"highlights"` // 高亮片段
}

// RAGClient 使用阿里云官方 SDK 的 RAG 客户端
type RAGClient struct {
	config *RAGConfig
	client *bailian.Client
	cache  *QueryCache
}

// NewRAGClient 创建基于 SDK 的 RAG 客户端
func NewRAGClient(config *RAGConfig) (*RAGClient, error) {
	// 创建 SDK 配置
	sdkConfig := &openapi.Config{
		AccessKeyId:     tea.String(config.AccessKeyID),
		AccessKeySecret: tea.String(config.AccessKeySecret),
	}

	// 设置端点（默认为北京区域）
	if config.Endpoint != "" {
		sdkConfig.Endpoint = tea.String(config.Endpoint)
	} else {
		sdkConfig.Endpoint = tea.String("bailian.cn-beijing.aliyuncs.com")
	}

	// 创建客户端
	client, err := bailian.NewClient(sdkConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Bailian SDK client: %w", err)
	}

	c := &RAGClient{
		config: config,
		client: client,
	}

	// 初始化缓存
	if config.EnableCache {
		c.cache = NewQueryCache(config.CacheMaxSize, time.Duration(config.CacheTTL)*time.Second)
	}

	return c, nil
}

// SearchWithCache 查询知识库（带缓存）
func (c *RAGClient) SearchWithCache(query *RAGQuery) (*RAGResponse, error) {
	// 检查缓存
	if c.cache != nil {
		cacheKey := c.buildCacheKey(query)
		if cached := c.cache.Get(cacheKey); cached != nil {
			if c.config.Debug {
				log.Printf("🎯 RAG cache hit: %s", query.Query)
			}
			return cached, nil
		}
	}

	// 执行查询
	startTime := time.Now()
	resp, err := c.search(query)
	if err != nil {
		return nil, err
	}

	// 记录延迟
	resp.Latency = time.Since(startTime).Milliseconds()

	// 缓存结果
	if c.cache != nil {
		cacheKey := c.buildCacheKey(query)
		c.cache.Set(cacheKey, resp)
	}

	if c.config.Debug {
		log.Printf("✅ RAG query completed: %s (latency: %dms, docs: %d)",
			query.Query, resp.Latency, len(resp.Documents))
	}

	return resp, nil
}

// search 执行实际的查询（带重试）
func (c *RAGClient) search(query *RAGQuery) (*RAGResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// 重试前等待
			time.Sleep(time.Duration(c.config.RetryDelay) * time.Second)
			log.Printf("🔄 Retrying RAG query (attempt %d/%d)", attempt, c.config.MaxRetries)
		}

		resp, err := c.doSearchSDK(query)
		if err == nil {
			return resp, nil
		}

		lastErr = err
	}

	return nil, fmt.Errorf("RAG query failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// doSearchSDK 执行单次查询（使用 SDK）
func (c *RAGClient) doSearchSDK(query *RAGQuery) (*RAGResponse, error) {
	// 构建检索请求
	request := &bailian.RetrieveRequest{
		IndexId: tea.String(c.config.KnowledgeBaseID),
		Query:   tea.String(query.Query),
	}

	// 设置可选参数
	if query.TopK > 0 {
		request.DenseSimilarityTopK = tea.Int32(int32(query.TopK))
	} else {
		request.DenseSimilarityTopK = tea.Int32(int32(c.config.DefaultTopK))
	}

	// 启用重排序
	request.EnableReranking = tea.Bool(true)

	// 准备请求头和运行时选项
	headers := make(map[string]*string)
	runtime := &util.RuntimeOptions{}

	// 调用 SDK 检索接口
	response, err := c.client.RetrieveWithOptions(
		tea.String(c.config.WorkspaceID),
		request,
		headers,
		runtime,
	)

	if err != nil {
		return nil, fmt.Errorf("SDK retrieve failed: %w", err)
	}

	// 检查响应
	if response == nil || response.Body == nil {
		return nil, fmt.Errorf("empty response from SDK")
	}

	if !tea.BoolValue(response.Body.Success) {
		return nil, fmt.Errorf("SDK returned Success=false, Code=%s, Message=%s",
			tea.StringValue(response.Body.Code),
			tea.StringValue(response.Body.Message))
	}

	// 转换为 RAGResponse
	ragResp := &RAGResponse{
		Documents: make([]RAGDocument, 0),
	}

	if response.Body.Data != nil && response.Body.Data.Nodes != nil {
		for _, node := range response.Body.Data.Nodes {
			if node == nil {
				continue
			}

			// 过滤低相关度文档
			score := tea.Float64Value(node.Score)
			if score < c.config.SimilarityThreshold {
				continue
			}

			// 从 Metadata 中提取信息
			title := ""
			source := ""
			url := ""

			if node.Metadata != nil {
				// Metadata 是 interface{} 类型，需要先转换为 map
				if meta, ok := node.Metadata.(map[string]interface{}); ok {
					if t, ok := meta["title"].(string); ok {
						title = t
					}
					if s, ok := meta["doc_name"].(string); ok {
						source = s
					}
					if u, ok := meta["file_path"].(string); ok {
						url = u
					}
				}
			}

			ragResp.Documents = append(ragResp.Documents, RAGDocument{
				Title:      title,
				Content:    tea.StringValue(node.Text),
				Source:     source,
				URL:        url,
				Score:      score,
				Highlights: []string{}, // SDK 不返回 highlights
			})
		}
	}

	return ragResp, nil
}

// buildCacheKey 构建缓存键
func (c *RAGClient) buildCacheKey(query *RAGQuery) string {
	return fmt.Sprintf("%s:%s:top%d:%s", query.Scenario, query.Query, query.TopK, query.ContextMode)
}

// QueryCache 查询缓存
type QueryCache struct {
	entries map[string]*CacheEntry
	mu      sync.RWMutex
	maxSize int
	ttl     time.Duration
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Response  *RAGResponse
	ExpiresAt time.Time
}

// NewQueryCache 创建查询缓存
func NewQueryCache(maxSize int, ttl time.Duration) *QueryCache {
	cache := &QueryCache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}

	// 启动清理协程
	go cache.cleanupLoop()

	return cache
}

// Get 获取缓存
func (c *QueryCache) Get(key string) *RAGResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil
	}

	// 检查是否过期
	if time.Now().After(entry.ExpiresAt) {
		return nil
	}

	return entry.Response
}

// Set 设置缓存
func (c *QueryCache) Set(key string, resp *RAGResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查缓存大小
	if len(c.entries) >= c.maxSize {
		// 简单的 LRU：删除第一个条目
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}

	c.entries[key] = &CacheEntry{
		Response:  resp,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// cleanupLoop 清理过期缓存
func (c *QueryCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup 执行清理
func (c *QueryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}
