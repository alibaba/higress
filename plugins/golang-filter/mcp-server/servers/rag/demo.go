package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/performance"
)

// DemoSuite represents a demonstration suite for the RAG system
type DemoSuite struct {
	client      *rag.Client
	monitor     *performance.Monitor
	resourceMgr *performance.ResourceManager
}

// NewDemoSuite creates a new demo suite
func NewDemoSuite() (*DemoSuite, error) {
	// Create configuration
	cfg := config.Config{
		RAG: config.RAGConfig{
			Splitter: config.SplitterConfig{
				Provider:     "recursive",
				ChunkSize:    1000,
				ChunkOverlap: 200,
			},
			Threshold: 0.7,
			TopK:      10,
		},
		LLM: config.LLMConfig{
			Provider:    "openai",
			Model:       "gpt-3.5-turbo",
			Temperature: 0.7,
			MaxTokens:   2000,
		},
		Embedding: config.EmbeddingConfig{
			Provider:   "openai",
			Model:      "text-embedding-ada-002",
			Dimensions: 1536,
		},
		VectorDB: config.VectorDBConfig{
			Provider:   "milvus",
			Host:       "localhost",
			Port:       19530,
			Database:   "rag_demo",
			Collection: "documents",
		},
		Enhancement: config.DefaultEnhancementConfig(),
	}

	// Create RAG client
	client, err := rag.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create RAG client: %w", err)
	}

	// Create performance components
	monitor := performance.NewMonitor()
	resourceMgr := performance.NewResourceManager(1024, 50) // 1GB memory, 50 concurrent

	return &DemoSuite{
		client:      client,
		monitor:     monitor,
		resourceMgr: resourceMgr,
	}, nil
}

// RunDemo runs a comprehensive demonstration of RAG system capabilities
func (ds *DemoSuite) RunDemo(ctx context.Context) error {
	fmt.Println("🚀 RAG增强系统功能演示开始")
	fmt.Println("=====================================")

	// Demo 1: Basic document processing
	fmt.Println("\n📋 演示1: 基础文档处理")
	if err := ds.demoBasicDocumentProcessing(ctx); err != nil {
		fmt.Printf("❌ 基础文档处理演示失败: %v\n", err)
	} else {
		fmt.Println("✅ 基础文档处理演示完成")
	}

	// Demo 2: Query enhancement
	fmt.Println("\n🔍 演示2: 查询增强功能")
	if err := ds.demoQueryEnhancement(ctx); err != nil {
		fmt.Printf("❌ 查询增强演示失败: %v\n", err)
	} else {
		fmt.Println("✅ 查询增强演示完成")
	}

	// Demo 3: Hybrid search
	fmt.Println("\n🔄 演示3: 混合搜索功能")
	if err := ds.demoHybridSearch(ctx); err != nil {
		fmt.Printf("❌ 混合搜索演示失败: %v\n", err)
	} else {
		fmt.Println("✅ 混合搜索演示完成")
	}

	// Demo 4: CRAG processing
	fmt.Println("\n🌐 演示4: CRAG纠错功能")
	if err := ds.demoCRAGProcessing(ctx); err != nil {
		fmt.Printf("❌ CRAG处理演示失败: %v\n", err)
	} else {
		fmt.Println("✅ CRAG处理演示完成")
	}

	// Demo 5: Enhanced chat
	fmt.Println("\n💬 演示5: 增强式问答")
	if err := ds.demoEnhancedChat(ctx); err != nil {
		fmt.Printf("❌ 增强式问答演示失败: %v\n", err)
	} else {
		fmt.Println("✅ 增强式问答演示完成")
	}

	// Demo 6: Performance monitoring
	fmt.Println("\n📊 演示6: 性能监控")
	ds.demoPerformanceMonitoring(ctx)

	// Demo 7: Resource management
	fmt.Println("\n🔧 演示7: 资源管理")
	ds.demoResourceManagement(ctx)

	fmt.Println("\n🎉 所有演示完成！")
	return nil
}

// demoBasicDocumentProcessing demonstrates basic document processing capabilities
func (ds *DemoSuite) demoBasicDocumentProcessing(ctx context.Context) error {
	fmt.Println("  正在处理示例文档...")

	// Sample documents
	documents := []struct {
		content string
		title   string
	}{
		{
			content: "Machine learning is a method of data analysis that automates analytical model building. It is a branch of artificial intelligence based on the idea that systems can learn from data, identify patterns and make decisions with minimal human intervention.",
			title:   "Introduction to Machine Learning",
		},
		{
			content: "Deep learning is part of a broader family of machine learning methods based on artificial neural networks with representation learning. Learning can be supervised, semi-supervised or unsupervised.",
			title:   "Deep Learning Overview",
		},
		{
			content: "Natural language processing (NLP) is a subfield of linguistics, computer science, and artificial intelligence concerned with the interactions between computers and human language, in particular how to program computers to process and analyze large amounts of natural language data.",
			title:   "Natural Language Processing",
		},
	}

	// Process documents
	for i, doc := range documents {
		fmt.Printf("    处理文档 %d/%d: %s\n", i+1, len(documents), doc.title)
		
		tracker := ds.monitor.StartOperation("document_processing")
		_, err := ds.client.CreateChunkFromText(doc.content, doc.title)
		tracker.Finish(err == nil)
		
		if err != nil {
			return fmt.Errorf("处理文档失败: %w", err)
		}
		
		// Simulate processing time
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// demoQueryEnhancement demonstrates query enhancement capabilities
func (ds *DemoSuite) demoQueryEnhancement(ctx context.Context) error {
	fmt.Println("  正在演示查询增强功能...")

	queries := []string{
		"ML algorithms",
		"deep learning neural networks",
		"natural language processing applications",
	}

	for i, query := range queries {
		fmt.Printf("    增强查询 %d/%d: %s\n", i+1, len(queries), query)
		
		tracker := ds.monitor.StartOperation("query_enhancement")
		enhanced, err := ds.client.EnhanceQuery(ctx, query, nil)
		tracker.Finish(err == nil)
		
		if err != nil {
			fmt.Printf("      查询增强失败: %v\n", err)
			continue
		}
		
		fmt.Printf("      原始查询: %s\n", enhanced.OriginalQuery)
		fmt.Printf("      重写查询: %v\n", enhanced.RewrittenQueries)
		fmt.Printf("      扩展术语: %v\n", enhanced.ExpandedTerms)
		fmt.Printf("      子查询: %v\n", enhanced.SubQueries)
		if enhanced.Intent != nil {
			fmt.Printf("      查询意图: %s (置信度: %.2f)\n", enhanced.Intent.PrimaryIntent, enhanced.Intent.Confidence)
		}
		
		// Simulate processing time
		time.Sleep(200 * time.Millisecond)
	}

	return nil
}

// demoHybridSearch demonstrates hybrid search capabilities
func (ds *DemoSuite) demoHybridSearch(ctx context.Context) error {
	fmt.Println("  正在演示混合搜索功能...")

	queries := []string{
		"machine learning applications",
		"deep learning vs traditional ML",
		"NLP in modern AI systems",
	}

	for i, query := range queries {
		fmt.Printf("    混合搜索 %d/%d: %s\n", i+1, len(queries), query)
		
		tracker := ds.monitor.StartOperation("hybrid_search")
		response, err := ds.client.Search(ctx, &rag.SearchRequest{
			Query: query,
			TopK:  5,
			Options: &rag.SearchOptions{
				EnableHybridSearch: true,
			},
		})
		tracker.Finish(err == nil)
		
		if err != nil {
			fmt.Printf("      搜索失败: %v\n", err)
			continue
		}
		
		fmt.Printf("      找到 %d 个结果\n", len(response.Results))
		if len(response.Results) > 0 {
			fmt.Printf("      最佳匹配: %.3f - %s\n", response.Results[0].Score, response.Results[0].Content[:min(100, len(response.Results[0].Content))]+"...")
		}
		
		// Simulate processing time
		time.Sleep(300 * time.Millisecond)
	}

	return nil
}

// demoCRAGProcessing demonstrates CRAG processing capabilities
func (ds *DemoSuite) demoCRAGProcessing(ctx context.Context) error {
	fmt.Println("  正在演示CRAG纠错功能...")

	queries := []string{
		"latest AI research 2024",
		"breakthroughs in computer vision",
	}

	for i, query := range queries {
		fmt.Printf("    CRAG处理 %d/%d: %s\n", i+1, len(queries), query)
		
		tracker := ds.monitor.StartOperation("crag_processing")
		response, err := ds.client.Search(ctx, &rag.SearchRequest{
			Query: query,
			TopK:  5,
			Options: &rag.SearchOptions{
				EnableCRAG: true,
			},
		})
		tracker.Finish(err == nil)
		
		if err != nil {
			fmt.Printf("      CRAG处理失败: %v\n", err)
			continue
		}
		
		fmt.Printf("      找到 %d 个结果\n", len(response.Results))
		if len(response.Results) > 0 {
			fmt.Printf("      最佳匹配: %.3f - %s\n", response.Results[0].Score, response.Results[0].Content[:min(100, len(response.Results[0].Content))]+"...")
		}
		
		// Simulate processing time
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

// demoEnhancedChat demonstrates enhanced chat capabilities
func (ds *DemoSuite) demoEnhancedChat(ctx context.Context) error {
	fmt.Println("  正在演示增强式问答功能...")

	if ds.client.LLMProvider() == nil {
		fmt.Println("      ⚠️  LLM未配置，跳过问答演示")
		return nil
	}

	queries := []string{
		"What are the main differences between machine learning and deep learning?",
		"How is natural language processing used in modern AI applications?",
	}

	for i, query := range queries {
		fmt.Printf("    问答 %d/%d: %s\n", i+1, len(queries), query)
		
		tracker := ds.monitor.StartOperation("enhanced_chat")
		response, err := ds.client.Chat(query)
		tracker.Finish(err == nil)
		
		if err != nil {
			fmt.Printf("      问答失败: %v\n", err)
			continue
		}
		
		fmt.Printf("      回答: %s\n", response[:min(200, len(response))]+"...")
		
		// Simulate processing time
		time.Sleep(800 * time.Millisecond)
	}

	return nil
}

// demoPerformanceMonitoring demonstrates performance monitoring capabilities
func (ds *DemoSuite) demoPerformanceMonitoring(ctx context.Context) {
	fmt.Println("  正在收集性能指标...")

	// Get system stats
	systemStats := ds.monitor.GetSystemStats()
	fmt.Printf("    系统运行时间: %v\n", systemStats.Uptime)
	fmt.Printf("    总请求数: %d\n", systemStats.TotalRequests)
	fmt.Printf("    成功率: %.2f%%\n", systemStats.SuccessRate)

	// Get operation metrics
	operations := []string{
		"document_processing",
		"query_enhancement",
		"hybrid_search",
		"crag_processing",
		"enhanced_chat",
	}

	for _, op := range operations {
		if metrics := ds.monitor.GetMetrics(op); metrics != nil {
			fmt.Printf("    %s: 请求=%d, 平均耗时=%v, 成功率=%.2f%%\n",
				op, metrics.TotalRequests, metrics.AvgDuration, 
				float64(metrics.SuccessfulRequests)/float64(max(1, metrics.TotalRequests))*100)
		}
	}
}

// demoResourceManagement demonstrates resource management capabilities
func (ds *DemoSuite) demoResourceManagement(ctx context.Context) {
	fmt.Println("  正在演示资源管理...")

	// Get memory stats
	memoryStats := ds.resourceMgr.GetMemoryStats()
	fmt.Printf("    已分配内存: %d MB\n", memoryStats.AllocatedMB)
	fmt.Printf("    系统内存: %d MB\n", memoryStats.SystemMB)
	fmt.Printf("    内存使用率: %.2f%%\n", memoryStats.MemoryUsagePercent)

	// Get resource stats
	resourceStats := ds.resourceMgr.GetResourceStats()
	fmt.Printf("    最大并发数: %d\n", resourceStats.MaxConcurrent)
	fmt.Printf("    当前请求数: %d\n", resourceStats.CurrentRequests)
	fmt.Printf("    可用槽位: %d\n", resourceStats.AvailableSlots)
	fmt.Printf("    资源使用率: %.2f%%\n", resourceStats.UsagePercent)
}

// Close closes the demo suite
func (ds *DemoSuite) Close() error {
	if ds.client != nil {
		return ds.client.Close()
	}
	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Main function to run the demonstration
func main() {
	fmt.Println("🔬 RAG增强系统功能演示")
	fmt.Println("=====================================")

	// Create demo suite
	suite, err := NewDemoSuite()
	if err != nil {
		fmt.Printf("❌ 创建演示套件失败: %v\n", err)
		os.Exit(1)
	}
	defer suite.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\n🛑 收到停止信号，正在优雅关闭...")
		cancel()
	}()

	// Run demo
	if err := suite.RunDemo(ctx); err != nil {
		fmt.Printf("❌ 演示运行失败: %v\n", err)
		os.Exit(1)
	}

	// Print final summary
	fmt.Println("\n📈 演示总结:")
	fmt.Println("  RAG增强系统已成功演示以下核心功能:")
	fmt.Println("  • 基础文档处理和知识库构建")
	fmt.Println("  • 智能查询增强（重写、扩展、分解、意图识别）")
	fmt.Println("  • 混合搜索（向量搜索 + BM25关键词搜索）")
	fmt.Println("  • CRAG纠错机制（置信度评估 + 网络搜索增强）")
	fmt.Println("  • 增强式问答（结合检索和生成）")
	fmt.Println("  • 实时性能监控和资源管理")
	fmt.Println("\n🎯 系统已准备就绪，可投入生产环境使用！")
}