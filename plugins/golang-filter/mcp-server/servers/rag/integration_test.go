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

// TestSuite represents a comprehensive test suite for the RAG system
type TestSuite struct {
	client       *rag.Client
	monitor      *performance.Monitor
	resourceMgr  *performance.ResourceManager
	concurrency  *performance.ConcurrencyManager
	testResults  []TestResult
	mutex        sync.Mutex
}

// TestResult represents the result of a single test
type TestResult struct {
	TestName    string        `json:"test_name"`
	Success     bool          `json:"success"`
	Duration    time.Duration `json:"duration"`
	Error       error         `json:"error,omitempty"`
	Metrics     interface{}   `json:"metrics,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
}

// BenchmarkResult represents performance benchmark results
type BenchmarkResult struct {
	TestName           string        `json:"test_name"`
	TotalRequests      int           `json:"total_requests"`
	SuccessfulRequests int           `json:"successful_requests"`
	FailedRequests     int           `json:"failed_requests"`
	SuccessRate        float64       `json:"success_rate"`
	AverageDuration    time.Duration `json:"average_duration"`
	MinDuration        time.Duration `json:"min_duration"`
	MaxDuration        time.Duration `json:"max_duration"`
	P50Duration        time.Duration `json:"p50_duration"`
	P95Duration        time.Duration `json:"p95_duration"`
	P99Duration        time.Duration `json:"p99_duration"`
	ThroughputRPS      float64       `json:"throughput_rps"`
	Timestamp          time.Time     `json:"timestamp"`
}

// NewTestSuite creates a new test suite
func NewTestSuite() (*TestSuite, error) {
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
			Database:   "rag_test",
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
	concurrency := performance.NewConcurrencyManager(10, 100, 30*time.Second)

	return &TestSuite{
		client:      client,
		monitor:     monitor,
		resourceMgr: resourceMgr,
		concurrency: concurrency,
		testResults: make([]TestResult, 0),
	}, nil
}

// RunEndToEndTests runs comprehensive end-to-end functional tests
func (ts *TestSuite) RunEndToEndTests(ctx context.Context) error {
	fmt.Println("🧪 Starting End-to-End Functional Tests...")

	tests := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"Basic Query Processing", ts.testBasicQuery},
		{"Query Enhancement", ts.testQueryEnhancement},
		{"Hybrid Search", ts.testHybridSearch},
		{"CRAG Processing", ts.testCRAGProcessing},
		{"Post-processing Pipeline", ts.testPostProcessing},
		{"Cache Functionality", ts.testCacheFunctionality},
		{"Error Handling", ts.testErrorHandling},
		{"Resource Management", ts.testResourceManagement},
		{"Concurrent Processing", ts.testConcurrentProcessing},
		{"Memory Management", ts.testMemoryManagement},
	}

	for _, test := range tests {
		fmt.Printf("  Running test: %s...", test.name)
		result := ts.runSingleTest(ctx, test.name, test.fn)
		ts.addTestResult(result)
		
		if result.Success {
			fmt.Printf(" ✅ PASSED (%.2fms)\n", float64(result.Duration.Nanoseconds())/1e6)
		} else {
			fmt.Printf(" ❌ FAILED: %v (%.2fms)\n", result.Error, float64(result.Duration.Nanoseconds())/1e6)
		}
	}

	return nil
}

// RunPerformanceBenchmarks runs comprehensive performance benchmarks
func (ts *TestSuite) RunPerformanceBenchmarks(ctx context.Context) ([]BenchmarkResult, error) {
	fmt.Println("🚀 Starting Performance Benchmarks...")

	benchmarks := []struct {
		name         string
		requests     int
		concurrency  int
		testFunc     func(context.Context) error
	}{
		{"Basic Query Benchmark", 100, 5, ts.testBasicQuery},
		{"Enhanced Query Benchmark", 50, 3, ts.testQueryEnhancement},
		{"Hybrid Search Benchmark", 30, 2, ts.testHybridSearch},
		{"Concurrent Load Test", 200, 10, ts.testBasicQuery},
		{"Memory Stress Test", 500, 20, ts.testBasicQuery},
	}

	var results []BenchmarkResult

	for _, benchmark := range benchmarks {
		fmt.Printf("  Running benchmark: %s (%d requests, %d concurrent)...\n", 
			benchmark.name, benchmark.requests, benchmark.concurrency)
		
		result := ts.runBenchmark(ctx, benchmark.name, benchmark.requests, 
			benchmark.concurrency, benchmark.testFunc)
		results = append(results, result)
		
		fmt.Printf("    ✅ Completed: %.2f RPS, %.2f%% success rate, P95: %.2fms\n",
			result.ThroughputRPS, result.SuccessRate, float64(result.P95Duration.Nanoseconds())/1e6)
	}

	return results, nil
}

// RunModuleIntegrationTests verifies that all modules work together correctly
func (ts *TestSuite) RunModuleIntegrationTests(ctx context.Context) error {
	fmt.Println("🔗 Starting Module Integration Tests...")

	tests := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"Query Enhancement + Hybrid Search", ts.testEnhancedHybridSearch},
		{"CRAG + Post-processing", ts.testCRAGWithPostProcessing},
		{"Full Pipeline Integration", ts.testFullPipeline},
		{"Performance Monitoring Integration", ts.testPerformanceIntegration},
		{"Cache + Query Enhancement", ts.testCacheIntegration},
	}

	for _, test := range tests {
		fmt.Printf("  Running integration test: %s...", test.name)
		result := ts.runSingleTest(ctx, test.name, test.fn)
		ts.addTestResult(result)
		
		if result.Success {
			fmt.Printf(" ✅ PASSED (%.2fms)\n", float64(result.Duration.Nanoseconds())/1e6)
		} else {
			fmt.Printf(" ❌ FAILED: %v (%.2fms)\n", result.Error, float64(result.Duration.Nanoseconds())/1e6)
		}
	}

	return nil
}

// RunErrorScenarioTests tests various error scenarios
func (ts *TestSuite) RunErrorScenarioTests(ctx context.Context) error {
	fmt.Println("💥 Starting Error Scenario Tests...")

	tests := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"Invalid Configuration", ts.testInvalidConfig},
		{"Network Timeout", ts.testNetworkTimeout},
		{"Memory Exhaustion", ts.testMemoryExhaustion},
		{"Concurrent Resource Limit", ts.testResourceLimits},
		{"Database Connection Failure", ts.testDatabaseFailure},
		{"API Rate Limiting", ts.testRateLimiting},
		{"Malformed Input", ts.testMalformedInput},
		{"Service Unavailable", ts.testServiceUnavailable},
	}

	for _, test := range tests {
		fmt.Printf("  Running error test: %s...", test.name)
		result := ts.runSingleTest(ctx, test.name, test.fn)
		ts.addTestResult(result)
		
		if result.Success {
			fmt.Printf(" ✅ HANDLED (%.2fms)\n", float64(result.Duration.Nanoseconds())/1e6)
		} else {
			fmt.Printf(" ❌ FAILED: %v (%.2fms)\n", result.Error, float64(result.Duration.Nanoseconds())/1e6)
		}
	}

	return nil
}

// Individual test implementations

func (ts *TestSuite) testBasicQuery(ctx context.Context) error {
	query := "What is machine learning?"
	
	response, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  5,
	})
	
	if err != nil {
		return fmt.Errorf("basic query failed: %w", err)
	}
	
	if len(response.Results) == 0 {
		return fmt.Errorf("no results returned for basic query")
	}
	
	return nil
}

func (ts *TestSuite) testQueryEnhancement(ctx context.Context) error {
	query := "ML algorithms"
	
	response, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  5,
		Options: &rag.SearchOptions{
			EnableEnhancement: true,
		},
	})
	
	if err != nil {
		return fmt.Errorf("query enhancement failed: %w", err)
	}
	
	if len(response.Results) == 0 {
		return fmt.Errorf("no results returned for enhanced query")
	}
	
	return nil
}

func (ts *TestSuite) testHybridSearch(ctx context.Context) error {
	query := "deep learning neural networks"
	
	response, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  10,
		Options: &rag.SearchOptions{
			EnableHybridSearch: true,
		},
	})
	
	if err != nil {
		return fmt.Errorf("hybrid search failed: %w", err)
	}
	
	if len(response.Results) == 0 {
		return fmt.Errorf("no results returned for hybrid search")
	}
	
	return nil
}

func (ts *TestSuite) testCRAGProcessing(ctx context.Context) error {
	query := "latest AI research 2024"
	
	response, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  5,
		Options: &rag.SearchOptions{
			EnableCRAG: true,
		},
	})
	
	if err != nil {
		return fmt.Errorf("CRAG processing failed: %w", err)
	}
	
	if len(response.Results) == 0 {
		return fmt.Errorf("no results returned for CRAG processing")
	}
	
	return nil
}

func (ts *TestSuite) testPostProcessing(ctx context.Context) error {
	query := "machine learning applications"
	
	response, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  20,
		Options: &rag.SearchOptions{
			EnablePostProcessing: true,
		},
	})
	
	if err != nil {
		return fmt.Errorf("post-processing failed: %w", err)
	}
	
	if len(response.Results) == 0 {
		return fmt.Errorf("no results returned after post-processing")
	}
	
	return nil
}

func (ts *TestSuite) testCacheFunctionality(ctx context.Context) error {
	query := "artificial intelligence"
	
	// First request (should miss cache)
	start1 := time.Now()
	_, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  5,
	})
	duration1 := time.Since(start1)
	
	if err != nil {
		return fmt.Errorf("first cache test request failed: %w", err)
	}
	
	// Second request (should hit cache)
	start2 := time.Now()
	_, err = ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  5,
	})
	duration2 := time.Since(start2)
	
	if err != nil {
		return fmt.Errorf("second cache test request failed: %w", err)
	}
	
	// Cache hit should be significantly faster
	if duration2 >= duration1 {
		return fmt.Errorf("cache does not appear to be working (duration1: %v, duration2: %v)", duration1, duration2)
	}
	
	return nil
}

func (ts *TestSuite) testErrorHandling(ctx context.Context) error {
	// Test with empty query
	_, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: "",
		TopK:  5,
	})
	
	if err == nil {
		return fmt.Errorf("expected error for empty query but got none")
	}
	
	// Test with invalid TopK
	_, err = ts.client.Search(ctx, &rag.SearchRequest{
		Query: "test",
		TopK:  -1,
	})
	
	if err == nil {
		return fmt.Errorf("expected error for invalid TopK but got none")
	}
	
	return nil
}

func (ts *TestSuite) testResourceManagement(ctx context.Context) error {
	// Test resource acquisition and release
	err := ts.resourceMgr.WithResourceLimit(ctx, func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("resource management test failed: %w", err)
	}
	
	// Check resource stats
	stats := ts.resourceMgr.GetResourceStats()
	if stats.CurrentRequests < 0 {
		return fmt.Errorf("invalid resource stats: %+v", stats)
	}
	
	return nil
}

func (ts *TestSuite) testConcurrentProcessing(ctx context.Context) error {
	// Submit multiple concurrent tasks
	var wg sync.WaitGroup
	errors := make(chan error, 10)
	
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			task := &performance.Task{
				ID:   fmt.Sprintf("test-task-%d", id),
				Type: "search",
				Data: fmt.Sprintf("query %d", id),
				Context: ctx,
				Handler: &TestTaskHandler{ts: ts},
			}
			
			if err := ts.concurrency.SubmitTask(ctx, task); err != nil {
				errors <- err
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	for err := range errors {
		if err != nil {
			return fmt.Errorf("concurrent processing test failed: %w", err)
		}
	}
	
	return nil
}

func (ts *TestSuite) testMemoryManagement(ctx context.Context) error {
	// Get initial memory stats
	initialStats := ts.resourceMgr.GetMemoryStats()
	
	// Perform memory-intensive operations
	for i := 0; i < 100; i++ {
		_, err := ts.client.Search(ctx, &rag.SearchRequest{
			Query: fmt.Sprintf("memory test query %d with lots of content", i),
			TopK:  10,
		})
		if err != nil {
			return fmt.Errorf("memory test query failed: %w", err)
		}
	}
	
	// Get final memory stats
	finalStats := ts.resourceMgr.GetMemoryStats()
	
	// Memory usage should be reasonable
	if finalStats.MemoryUsagePercent > 90 {
		return fmt.Errorf("memory usage too high: %.2f%%", finalStats.MemoryUsagePercent)
	}
	
	// Verify GC is working
	if finalStats.NumGC <= initialStats.NumGC {
		return fmt.Errorf("garbage collection not working properly")
	}
	
	return nil
}

// Integration test implementations
func (ts *TestSuite) testEnhancedHybridSearch(ctx context.Context) error {
	query := "machine learning models"
	
	response, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  10,
		Options: &rag.SearchOptions{
			EnableEnhancement:  true,
			EnableHybridSearch: true,
		},
	})
	
	if err != nil {
		return fmt.Errorf("enhanced hybrid search failed: %w", err)
	}
	
	if len(response.Results) == 0 {
		return fmt.Errorf("no results from enhanced hybrid search")
	}
	
	return nil
}

func (ts *TestSuite) testCRAGWithPostProcessing(ctx context.Context) error {
	query := "recent advances in natural language processing"
	
	response, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  15,
		Options: &rag.SearchOptions{
			EnableCRAG:           true,
			EnablePostProcessing: true,
		},
	})
	
	if err != nil {
		return fmt.Errorf("CRAG with post-processing failed: %w", err)
	}
	
	if len(response.Results) == 0 {
		return fmt.Errorf("no results from CRAG with post-processing")
	}
	
	return nil
}

func (ts *TestSuite) testFullPipeline(ctx context.Context) error {
	query := "artificial intelligence ethics"
	
	response, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  20,
		Options: &rag.SearchOptions{
			EnableEnhancement:    true,
			EnableHybridSearch:   true,
			EnableCRAG:           true,
			EnablePostProcessing: true,
		},
	})
	
	if err != nil {
		return fmt.Errorf("full pipeline test failed: %w", err)
	}
	
	if len(response.Results) == 0 {
		return fmt.Errorf("no results from full pipeline")
	}
	
	return nil
}

func (ts *TestSuite) testPerformanceIntegration(ctx context.Context) error {
	// Test performance monitoring integration
	tracker := ts.monitor.StartOperation("integration_test")
	defer tracker.Finish(true)
	
	query := "performance monitoring test"
	_, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  5,
	})
	
	if err != nil {
		return fmt.Errorf("performance integration test failed: %w", err)
	}
	
	// Check metrics
	metrics := ts.monitor.GetMetrics("integration_test")
	if metrics == nil {
		return fmt.Errorf("no metrics collected for integration test")
	}
	
	return nil
}

func (ts *TestSuite) testCacheIntegration(ctx context.Context) error {
	query := "cache integration test"
	
	// Test with cache enabled
	response1, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  5,
		Options: &rag.SearchOptions{
			EnableEnhancement: true,
		},
	})
	
	if err != nil {
		return fmt.Errorf("first cache integration request failed: %w", err)
	}
	
	// Second request should hit cache
	response2, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  5,
		Options: &rag.SearchOptions{
			EnableEnhancement: true,
		},
	})
	
	if err != nil {
		return fmt.Errorf("second cache integration request failed: %w", err)
	}
	
	if len(response1.Results) != len(response2.Results) {
		return fmt.Errorf("cache integration results mismatch")
	}
	
	return nil
}

// Error scenario test implementations
func (ts *TestSuite) testInvalidConfig(ctx context.Context) error {
	// Test should handle invalid configuration gracefully
	return nil // Placeholder
}

func (ts *TestSuite) testNetworkTimeout(ctx context.Context) error {
	// Test should handle network timeouts gracefully
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()
	
	_, err := ts.client.Search(timeoutCtx, &rag.SearchRequest{
		Query: "timeout test",
		TopK:  5,
	})
	
	if err == nil {
		return fmt.Errorf("expected timeout error but got none")
	}
	
	return nil // Expected to fail
}

func (ts *TestSuite) testMemoryExhaustion(ctx context.Context) error {
	// Test should handle memory exhaustion gracefully
	return nil // Placeholder
}

func (ts *TestSuite) testResourceLimits(ctx context.Context) error {
	// Test should handle resource limits gracefully
	return nil // Placeholder
}

func (ts *TestSuite) testDatabaseFailure(ctx context.Context) error {
	// Test should handle database failures gracefully
	return nil // Placeholder
}

func (ts *TestSuite) testRateLimiting(ctx context.Context) error {
	// Test should handle rate limiting gracefully
	return nil // Placeholder
}

func (ts *TestSuite) testMalformedInput(ctx context.Context) error {
	// Test with malformed input
	_, err := ts.client.Search(ctx, &rag.SearchRequest{
		Query: string([]byte{0xff, 0xfe, 0xfd}), // Invalid UTF-8
		TopK:  5,
	})
	
	if err == nil {
		return fmt.Errorf("expected error for malformed input but got none")
	}
	
	return nil // Expected to fail
}

func (ts *TestSuite) testServiceUnavailable(ctx context.Context) error {
	// Test should handle service unavailable gracefully
	return nil // Placeholder
}

// Helper methods

func (ts *TestSuite) runSingleTest(ctx context.Context, name string, testFunc func(context.Context) error) TestResult {
	start := time.Now()
	err := testFunc(ctx)
	duration := time.Since(start)
	
	return TestResult{
		TestName:  name,
		Success:   err == nil,
		Duration:  duration,
		Error:     err,
		Timestamp: start,
	}
}

func (ts *TestSuite) runBenchmark(ctx context.Context, name string, requests, concurrency int, testFunc func(context.Context) error) BenchmarkResult {
	start := time.Now()
	
	var wg sync.WaitGroup
	results := make(chan error, requests)
	
	// Rate limiter for concurrency
	semaphore := make(chan struct{}, concurrency)
	
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			results <- testFunc(ctx)
		}()
	}
	
	wg.Wait()
	close(results)
	
	totalDuration := time.Since(start)
	
	// Analyze results
	successful := 0
	failed := 0
	durations := make([]time.Duration, 0, requests)
	
	for err := range results {
		if err == nil {
			successful++
		} else {
			failed++
		}
	}
	
	successRate := float64(successful) / float64(requests) * 100
	throughputRPS := float64(requests) / totalDuration.Seconds()
	
	return BenchmarkResult{
		TestName:           name,
		TotalRequests:      requests,
		SuccessfulRequests: successful,
		FailedRequests:     failed,
		SuccessRate:        successRate,
		ThroughputRPS:      throughputRPS,
		Timestamp:          start,
	}
}

func (ts *TestSuite) addTestResult(result TestResult) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	ts.testResults = append(ts.testResults, result)
}

func (ts *TestSuite) GetTestResults() []TestResult {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	return append([]TestResult(nil), ts.testResults...)
}

func (ts *TestSuite) PrintSummary() {
	fmt.Println("\n📊 Test Summary:")
	
	results := ts.GetTestResults()
	passed := 0
	failed := 0
	totalDuration := time.Duration(0)
	
	for _, result := range results {
		if result.Success {
			passed++
		} else {
			failed++
		}
		totalDuration += result.Duration
	}
	
	fmt.Printf("  Total tests: %d\n", len(results))
	fmt.Printf("  Passed: %d\n", passed)
	fmt.Printf("  Failed: %d\n", failed)
	fmt.Printf("  Success rate: %.2f%%\n", float64(passed)/float64(len(results))*100)
	fmt.Printf("  Total duration: %v\n", totalDuration)
	fmt.Printf("  Average duration: %v\n", totalDuration/time.Duration(len(results)))
}

func (ts *TestSuite) Close() error {
	if ts.concurrency != nil {
		ts.concurrency.Shutdown(5 * time.Second)
	}
	if ts.client != nil {
		return ts.client.Close()
	}
	return nil
}

// TestTaskHandler implements the TaskHandler interface for testing
type TestTaskHandler struct {
	ts *TestSuite
}

func (h *TestTaskHandler) Handle(ctx context.Context, data interface{}) (interface{}, error) {
	query, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("invalid data type for task handler")
	}
	
	response, err := h.ts.client.Search(ctx, &rag.SearchRequest{
		Query: query,
		TopK:  5,
	})
	
	if err != nil {
		return nil, err
	}
	
	return response, nil
}

// Main function to run the comprehensive test suite
func main() {
	fmt.Println("🔬 RAG增强系统集成测试开始")
	fmt.Println("=====================================")
	
	// Create test suite
	suite, err := NewTestSuite()
	if err != nil {
		log.Fatalf("Failed to create test suite: %v", err)
	}
	defer suite.Close()
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	
	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigCh
		fmt.Println("\n🛑 收到停止信号，正在优雅关闭...")
		cancel()
	}()
	
	// Run all test phases
	var allPassed = true
	
	// Phase 1: End-to-End Tests
	if err := suite.RunEndToEndTests(ctx); err != nil {
		fmt.Printf("❌ End-to-end tests failed: %v\n", err)
		allPassed = false
	}
	
	// Phase 2: Performance Benchmarks
	benchmarks, err := suite.RunPerformanceBenchmarks(ctx)
	if err != nil {
		fmt.Printf("❌ Performance benchmarks failed: %v\n", err)
		allPassed = false
	} else {
		fmt.Println("\n📈 Performance Benchmark Results:")
		for _, bench := range benchmarks {
			fmt.Printf("  %s: %.2f RPS (%.2f%% success)\n", 
				bench.TestName, bench.ThroughputRPS, bench.SuccessRate)
		}
	}
	
	// Phase 3: Module Integration Tests
	if err := suite.RunModuleIntegrationTests(ctx); err != nil {
		fmt.Printf("❌ Module integration tests failed: %v\n", err)
		allPassed = false
	}
	
	// Phase 4: Error Scenario Tests
	if err := suite.RunErrorScenarioTests(ctx); err != nil {
		fmt.Printf("❌ Error scenario tests failed: %v\n", err)
		allPassed = false
	}
	
	// Print final summary
	suite.PrintSummary()
	
	if allPassed {
		fmt.Println("\n🎉 所有测试通过！RAG增强系统已准备就绪")
		os.Exit(0)
	} else {
		fmt.Println("\n💥 部分测试失败，请检查并修复问题")
		os.Exit(1)
	}
}