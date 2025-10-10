package fusion

import (
	"context"
	"testing"
	"fmt"
)

// Mock implementations for testing

type MockVectorSearcher struct {
	results []*SearchResult
}

func NewMockVectorSearcher() *MockVectorSearcher {
	return &MockVectorSearcher{
		results: []*SearchResult{},
	}
}

func (m *MockVectorSearcher) SetResults(results []*SearchResult) {
	m.results = results
}

func (m *MockVectorSearcher) Search(ctx context.Context, query string, topK int) ([]*SearchResult, error) {
	if len(m.results) > topK {
		return m.results[:topK], nil
	}
	return m.results, nil
}

type MockBM25Searcher struct {
	results []*SearchResult
}

func NewMockBM25Searcher() *MockBM25Searcher {
	return &MockBM25Searcher{
		results: []*SearchResult{},
	}
}

func (m *MockBM25Searcher) SetResults(results []*SearchResult) {
	m.results = results
}

func (m *MockBM25Searcher) Search(ctx context.Context, query string, topK int) ([]*SearchResult, error) {
	if len(m.results) > topK {
		return m.results[:topK], nil
	}
	return m.results, nil
}

// Test functions

func TestStandardHybridRetriever_HybridSearch(t *testing.T) {
	// Setup mock searchers
	vectorSearcher := NewMockVectorSearcher()
	bm25Searcher := NewMockBM25Searcher()
	
	// Setup mock results
	vectorResults := []*SearchResult{
		{
			DocumentID: "doc1",
			Content:    "Vector document 1",
			Score:      0.9,
		},
		{
			DocumentID: "doc2",
			Content:    "Vector document 2",
			Score:      0.8,
		},
		{
			DocumentID: "doc3",
			Content:    "Vector document 3",
			Score:      0.7,
		},
	}
	
	bm25Results := []*SearchResult{
		{
			DocumentID: "doc2",
			Content:    "BM25 document 2",
			Score:      0.85,
		},
		{
			DocumentID: "doc4",
			Content:    "BM25 document 4",
			Score:      0.75,
		},
		{
			DocumentID: "doc1",
			Content:    "BM25 document 1",
			Score:      0.65,
		},
	}
	
	vectorSearcher.SetResults(vectorResults)
	bm25Searcher.SetResults(bm25Results)
	
	// Create hybrid retriever
	retriever := NewStandardHybridRetriever(vectorSearcher, bm25Searcher)
	
	ctx := context.Background()
	query := "test query"
	options := DefaultHybridSearchOptions()
	options.FinalTopK = 5
	
	results, err := retriever.HybridSearch(ctx, query, options)
	if err != nil {
		t.Fatalf("Hybrid search failed: %v", err)
	}
	
	if len(results) == 0 {
		t.Fatal("Expected some results from hybrid search")
	}
	
	// Should have results from both vector and BM25
	hasVectorDoc := false
	hasBM25Doc := false
	
	for _, result := range results {
		if result.Method == "rrf_fusion" {
			// Check if it's a fusion result
			if result.Metadata["has_vector"].(bool) {
				hasVectorDoc = true
			}
			if result.Metadata["has_bm25"].(bool) {
				hasBM25Doc = true
			}
		}
	}
	
	if !hasVectorDoc {
		t.Error("Expected at least one result from vector search")
	}
	if !hasBM25Doc {
		t.Error("Expected at least one result from BM25 search")
	}
	
	// Results should be sorted by score (descending)
	for i := 1; i < len(results); i++ {
		if results[i-1].Score < results[i].Score {
			t.Errorf("Results not sorted by score: %f < %f at positions %d, %d",
				results[i-1].Score, results[i].Score, i-1, i)
		}
	}
}

func TestStandardHybridRetriever_VectorSearchOnly(t *testing.T) {
	vectorSearcher := NewMockVectorSearcher()
	
	vectorResults := []*SearchResult{
		{DocumentID: "doc1", Content: "Vector doc 1", Score: 0.9},
		{DocumentID: "doc2", Content: "Vector doc 2", Score: 0.8},
	}
	vectorSearcher.SetResults(vectorResults)
	
	retriever := NewStandardHybridRetriever(vectorSearcher, nil)
	
	ctx := context.Background()
	options := DefaultHybridSearchOptions()
	options.EnableBM25 = false
	
	results, err := retriever.HybridSearch(ctx, "test", options)
	if err != nil {
		t.Fatalf("Vector search failed: %v", err)
	}
	
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	
	for _, result := range results {
		if result.Method != "vector_search" {
			t.Errorf("Expected vector_search method, got %s", result.Method)
		}
	}
}

func TestStandardHybridRetriever_BM25SearchOnly(t *testing.T) {
	bm25Searcher := NewMockBM25Searcher()
	
	bm25Results := []*SearchResult{
		{DocumentID: "doc1", Content: "BM25 doc 1", Score: 0.9},
		{DocumentID: "doc2", Content: "BM25 doc 2", Score: 0.8},
	}
	bm25Searcher.SetResults(bm25Results)
	
	retriever := NewStandardHybridRetriever(nil, bm25Searcher)
	
	ctx := context.Background()
	options := DefaultHybridSearchOptions()
	options.EnableVector = false
	
	results, err := retriever.HybridSearch(ctx, "test", options)
	if err != nil {
		t.Fatalf("BM25 search failed: %v", err)
	}
	
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	
	for _, result := range results {
		if result.Method != "bm25_search" {
			t.Errorf("Expected bm25_search method, got %s", result.Method)
		}
	}
}

func TestStandardFusionEngine_RRFFusion(t *testing.T) {
	engine := NewStandardFusionEngine()
	
	vectorResults := []*SearchResult{
		{DocumentID: "doc1", Content: "Vector doc 1", Score: 0.9},
		{DocumentID: "doc2", Content: "Vector doc 2", Score: 0.8},
		{DocumentID: "doc3", Content: "Vector doc 3", Score: 0.7},
	}
	
	bm25Results := []*SearchResult{
		{DocumentID: "doc2", Content: "BM25 doc 2", Score: 0.85},
		{DocumentID: "doc4", Content: "BM25 doc 4", Score: 0.75},
		{DocumentID: "doc1", Content: "BM25 doc 1", Score: 0.65},
	}
	
	options := DefaultFusionOptions()
	
	results, err := engine.Fuse(vectorResults, bm25Results, options)
	if err != nil {
		t.Fatalf("Fusion failed: %v", err)
	}
	
	// Should have 4 unique documents
	if len(results) != 4 {
		t.Errorf("Expected 4 unique documents, got %d", len(results))
	}
	
	// Documents appearing in both lists should have higher RRF scores
	doc1Score := 0.0
	doc2Score := 0.0
	doc3Score := 0.0
	doc4Score := 0.0
	
	for _, result := range results {
		switch result.DocumentID {
		case "doc1":
			doc1Score = result.Score
		case "doc2":
			doc2Score = result.Score
		case "doc3":
			doc3Score = result.Score
		case "doc4":
			doc4Score = result.Score
		}
	}
	
	// doc1 and doc2 appear in both lists, so should have higher scores than doc3 and doc4
	if doc1Score <= doc3Score {
		t.Errorf("doc1 (in both lists) should have higher score than doc3 (vector only)")
	}
	if doc2Score <= doc4Score {
		t.Errorf("doc2 (in both lists) should have higher score than doc4 (BM25 only)")
	}
	
	// Results should be sorted by score
	for i := 1; i < len(results); i++ {
		if results[i-1].Score < results[i].Score {
			t.Errorf("Results not sorted: %f < %f", results[i-1].Score, results[i].Score)
		}
	}
}

func TestStandardFusionEngine_EmptyResults(t *testing.T) {
	engine := NewStandardFusionEngine()
	
	// Test with empty vector results
	results, err := engine.Fuse([]*SearchResult{}, []*SearchResult{
		{DocumentID: "doc1", Content: "BM25 doc", Score: 0.8},
	}, DefaultFusionOptions())
	
	if err != nil {
		t.Fatalf("Fusion with empty vector failed: %v", err)
	}
	
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	
	// Test with empty BM25 results
	results, err = engine.Fuse([]*SearchResult{
		{DocumentID: "doc1", Content: "Vector doc", Score: 0.8},
	}, []*SearchResult{}, DefaultFusionOptions())
	
	if err != nil {
		t.Fatalf("Fusion with empty BM25 failed: %v", err)
	}
	
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	
	// Test with both empty
	results, err = engine.Fuse([]*SearchResult{}, []*SearchResult{}, DefaultFusionOptions())
	
	if err != nil {
		t.Fatalf("Fusion with both empty failed: %v", err)
	}
	
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestScoreNormalization(t *testing.T) {
	engine := NewStandardFusionEngine()
	
	results := []*SearchResult{
		{DocumentID: "doc1", Score: 10.0},
		{DocumentID: "doc2", Score: 5.0},
		{DocumentID: "doc3", Score: 0.0},
	}
	
	// Test MinMax normalization
	engine.normalizeScores(results)
	
	// Should be normalized to [0, 1]
	if results[0].Score != 1.0 {
		t.Errorf("Expected max score to be 1.0, got %f", results[0].Score)
	}
	if results[2].Score != 0.0 {
		t.Errorf("Expected min score to be 0.0, got %f", results[2].Score)
	}
	if results[1].Score != 0.5 {
		t.Errorf("Expected middle score to be 0.5, got %f", results[1].Score)
	}
}

func TestHybridSearchOptions_Validation(t *testing.T) {
	options := DefaultHybridSearchOptions()
	
	// Test default values
	if options.FusionMethod != RRFFusion {
		t.Errorf("Expected RRF fusion by default, got %v", options.FusionMethod)
	}
	
	if options.VectorTopK <= 0 {
		t.Error("Expected positive VectorTopK")
	}
	
	if options.BM25TopK <= 0 {
		t.Error("Expected positive BM25TopK")
	}
	
	if options.FinalTopK <= 0 {
		t.Error("Expected positive FinalTopK")
	}
	
	if !options.EnableVector {
		t.Error("Expected vector search to be enabled by default")
	}
	
	if !options.EnableBM25 {
		t.Error("Expected BM25 search to be enabled by default")
	}
}

func TestFusionOptions_Validation(t *testing.T) {
	options := DefaultFusionOptions()
	
	if options.RRFConstant <= 0 {
		t.Error("Expected positive RRF constant")
	}
	
	if options.ScoreNormalization != MinMaxNormalization {
		t.Errorf("Expected MinMax normalization by default, got %v", options.ScoreNormalization)
	}
	
	if options.TieBreaking != PreferVector {
		t.Errorf("Expected PreferVector tie breaking by default, got %v", options.TieBreaking)
	}
}

func TestHybridRetriever_MinScoreFiltering(t *testing.T) {
	vectorSearcher := NewMockVectorSearcher()
	bm25Searcher := NewMockBM25Searcher()
	
	vectorResults := []*SearchResult{
		{DocumentID: "doc1", Content: "Vector doc 1", Score: 0.9},
		{DocumentID: "doc2", Content: "Vector doc 2", Score: 0.2}, // Low score
	}
	
	bm25Results := []*SearchResult{
		{DocumentID: "doc1", Content: "BM25 doc 1", Score: 0.8},
		{DocumentID: "doc3", Content: "BM25 doc 3", Score: 0.1}, // Low score
	}
	
	vectorSearcher.SetResults(vectorResults)
	bm25Searcher.SetResults(bm25Results)
	
	retriever := NewStandardHybridRetriever(vectorSearcher, bm25Searcher)
	
	ctx := context.Background()
	options := DefaultHybridSearchOptions()
	options.MinScore = 0.5 // High threshold
	
	results, err := retriever.HybridSearch(ctx, "test", options)
	if err != nil {
		t.Fatalf("Hybrid search failed: %v", err)
	}
	
	// Should filter out low-scoring documents
	for _, result := range results {
		if result.Score < options.MinScore {
			t.Errorf("Result with score %f should be filtered out (min: %f)", 
				result.Score, options.MinScore)
		}
	}
}

func BenchmarkStandardFusionEngine_Fuse(b *testing.B) {
	engine := NewStandardFusionEngine()
	
	// Create test data
	var vectorResults []*SearchResult
	var bm25Results []*SearchResult
	
	for i := 0; i < 100; i++ {
		vectorResults = append(vectorResults, &SearchResult{
			DocumentID: fmt.Sprintf("vec_doc_%d", i),
			Content:    fmt.Sprintf("Vector document %d", i),
			Score:      1.0 - float64(i)*0.01,
		})
		
		bm25Results = append(bm25Results, &SearchResult{
			DocumentID: fmt.Sprintf("bm25_doc_%d", i),
			Content:    fmt.Sprintf("BM25 document %d", i),
			Score:      0.9 - float64(i)*0.009,
		})
	}
	
	options := DefaultFusionOptions()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Fuse(vectorResults, bm25Results, options)
		if err != nil {
			b.Fatalf("Fusion failed: %v", err)
		}
	}
}

func BenchmarkStandardHybridRetriever_HybridSearch(b *testing.B) {
	vectorSearcher := NewMockVectorSearcher()
	bm25Searcher := NewMockBM25Searcher()
	
	// Setup test data
	var vectorResults []*SearchResult
	var bm25Results []*SearchResult
	
	for i := 0; i < 50; i++ {
		vectorResults = append(vectorResults, &SearchResult{
			DocumentID: fmt.Sprintf("doc_%d", i),
			Content:    fmt.Sprintf("Document %d content", i),
			Score:      1.0 - float64(i)*0.02,
		})
		
		bm25Results = append(bm25Results, &SearchResult{
			DocumentID: fmt.Sprintf("doc_%d", i+25),
			Content:    fmt.Sprintf("Document %d content", i+25),
			Score:      0.9 - float64(i)*0.018,
		})
	}
	
	vectorSearcher.SetResults(vectorResults)
	bm25Searcher.SetResults(bm25Results)
	
	retriever := NewStandardHybridRetriever(vectorSearcher, bm25Searcher)
	ctx := context.Background()
	options := DefaultHybridSearchOptions()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := retriever.HybridSearch(ctx, "benchmark query", options)
		if err != nil {
			b.Fatalf("Hybrid search failed: %v", err)
		}
	}
}