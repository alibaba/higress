package crag

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Mock implementations for testing

type MockLLMProvider struct {
	responses map[string]string
}

func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{
		responses: make(map[string]string),
	}
}

func (m *MockLLMProvider) AddResponse(prompt, response string) {
	m.responses[prompt] = response
}

func (m *MockLLMProvider) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	if response, exists := m.responses[prompt]; exists {
		return response, nil
	}
	
	// Check if this is for low confidence docs by looking for specific content
	if strings.Contains(prompt, "Somewhat relevant content") {
		return `{
			"relevance_score": 0.2,
			"quality_score": 0.2,
			"explanation": "Mock evaluation with low scores"
		}`, nil
	}
	
	// Default response for testing with high scores to ensure high confidence
	return `{
		"relevance_score": 0.9,
		"quality_score": 0.9,
		"explanation": "Mock evaluation with high scores"
	}`, nil
}

// Test functions

func TestLLMBasedEvaluator_EvaluateRetrieval(t *testing.T) {
	llmProvider := NewMockLLMProvider()
	evaluator := NewLLMBasedEvaluator(llmProvider, nil)
	
	ctx := context.Background()
	query := "What is machine learning?"
	
	documents := []Document{
		{
			ID:      "doc1",
			Content: "Machine learning is a subset of artificial intelligence that focuses on algorithms.",
			Title:   "Introduction to ML",
			Score:   0.9,
		},
		{
			ID:      "doc2", 
			Content: "Deep learning uses neural networks with multiple layers.",
			Title:   "Deep Learning Basics",
			Score:   0.7,
		},
	}
	
	result, err := evaluator.EvaluateRetrieval(ctx, query, documents)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}
	
	if result.ConfidenceLevel != HighConfidence {
		t.Errorf("Expected high confidence, got %v", result.ConfidenceLevel)
	}
	
	if len(result.DocumentScores) != 2 {
		t.Errorf("Expected 2 document scores, got %d", len(result.DocumentScores))
	}
	
	if result.OverallScore <= 0 {
		t.Errorf("Expected positive overall score, got %f", result.OverallScore)
	}
}

func TestLLMBasedEvaluator_EmptyDocuments(t *testing.T) {
	llmProvider := NewMockLLMProvider()
	evaluator := NewLLMBasedEvaluator(llmProvider, nil)
	
	ctx := context.Background()
	query := "test query"
	documents := []Document{}
	
	result, err := evaluator.EvaluateRetrieval(ctx, query, documents)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}
	
	if result.ConfidenceLevel != NoConfidence {
		t.Errorf("Expected no confidence for empty documents, got %v", result.ConfidenceLevel)
	}
	
	if result.OverallScore != 0 {
		t.Errorf("Expected zero score for empty documents, got %f", result.OverallScore)
	}
}

func TestDuckDuckGoSearcher_Search(t *testing.T) {
	searcher := NewDuckDuckGoSearcher()
	
	ctx := context.Background()
	query := "golang programming"
	maxResults := 3
	
	// Note: This test requires internet connectivity
	// In a real test environment, you might want to mock the HTTP client
	results, err := searcher.Search(ctx, query, maxResults)
	if err != nil {
		t.Skipf("Skipping web search test (requires internet): %v", err)
	}
	
	if len(results) > maxResults {
		t.Errorf("Expected at most %d results, got %d", maxResults, len(results))
	}
	
	for _, result := range results {
		if result.Title == "" {
			t.Error("Expected non-empty title")
		}
		if result.Content == "" {
			t.Error("Expected non-empty content")
		}
		if result.Source == "" {
			t.Error("Expected non-empty source")
		}
	}
}

func TestMockWebSearcher(t *testing.T) {
	searcher := NewMockWebSearcher()
	
	// Add mock results
	mockResults := []WebDocument{
		{
			Title:       "Test Result 1",
			Content:     "This is test content 1",
			URL:         "https://example.com/1",
			Score:       0.9,
			Source:      "mock",
			RetrievedAt: time.Now(),
		},
		{
			Title:       "Test Result 2",
			Content:     "This is test content 2", 
			URL:         "https://example.com/2",
			Score:       0.8,
			Source:      "mock",
			RetrievedAt: time.Now(),
		},
	}
	
	searcher.AddMockResult("test query", mockResults)
	
	ctx := context.Background()
	results, err := searcher.Search(ctx, "test query", 5)
	if err != nil {
		t.Fatalf("Mock search failed: %v", err)
	}
	
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	
	if results[0].Title != "Test Result 1" {
		t.Errorf("Expected first result title 'Test Result 1', got '%s'", results[0].Title)
	}
}

func TestStandardKnowledgeRefinement_RefineKnowledge(t *testing.T) {
	llmProvider := NewMockLLMProvider()
	evaluator := NewLLMBasedEvaluator(llmProvider, nil)
	refinement := NewStandardKnowledgeRefinement(evaluator)
	
	ctx := context.Background()
	query := "machine learning"
	
	documents := []Document{
		{
			ID:          "doc1",
			Content:     "Machine learning is a subset of AI that enables computers to learn.",
			Title:       "ML Basics",
			Score:       0.9,
			RetrievedAt: time.Now(),
		},
		{
			ID:          "doc2",
			Content:     "Random content about cooking recipes.",
			Title:       "Cooking",
			Score:       0.2,
			RetrievedAt: time.Now(),
		},
		{
			ID:          "doc3",
			Content:     "Machine learning algorithms include supervised and unsupervised learning.",
			Title:       "ML Algorithms",
			Score:       0.8,
			RetrievedAt: time.Now(),
		},
	}
	
	refined, err := refinement.RefineKnowledge(ctx, query, documents)
	if err != nil {
		t.Fatalf("Knowledge refinement failed: %v", err)
	}
	
	// Should filter out irrelevant documents
	if len(refined) >= len(documents) {
		t.Errorf("Expected refinement to filter documents, got %d from %d", len(refined), len(documents))
	}
	
	// First document should be highly relevant
	if len(refined) > 0 && refined[0].Score < 0.5 {
		t.Errorf("Expected first refined document to have high score, got %f", refined[0].Score)
	}
}

func TestStandardCRAGProcessor_ProcessQuery(t *testing.T) {
	// Setup components
	llmProvider := NewMockLLMProvider()
	evaluator := NewLLMBasedEvaluator(llmProvider, nil)
	webSearcher := NewMockWebSearcher()
	refinement := NewStandardKnowledgeRefinement(evaluator)
	
	// Configure mock web search results
	webResults := []WebDocument{
		{
			Title:       "Web Result 1",
			Content:     "Additional information from web search",
			URL:         "https://example.com/web1",
			Score:       0.8,
			Source:      "mock_web",
			RetrievedAt: time.Now(),
		},
	}
	webSearcher.AddMockResult("test query", webResults)
	
	config := DefaultCRAGConfig()
	processor := NewStandardCRAGProcessor(evaluator, webSearcher, refinement, config)
	
	ctx := context.Background()
	query := "test query"
	
	// Test with high-confidence documents
	highConfidenceDocs := []Document{
		{
			ID:      "doc1",
			Content: "Highly relevant content for test query",
			Title:   "Perfect Match",
			Score:   0.95,
		},
	}
	
	result, err := processor.ProcessQuery(ctx, query, highConfidenceDocs)
	if err != nil {
		t.Fatalf("CRAG processing failed: %v", err)
	}
	
	if result.RoutingDecision.Action != UseRetrieved {
		t.Errorf("Expected UseRetrieved action for high confidence, got %v", result.RoutingDecision.Action)
	}
	
	if result.WebSearchUsed {
		t.Error("Expected no web search for high confidence documents")
	}
	
	// Test with low-confidence documents
	lowConfidenceDocs := []Document{
		{
			ID:      "doc2",
			Content: "Somewhat relevant content",
			Title:   "Partial Match",
			Score:   0.1, // Very low score to trigger low confidence
		},
	}
	
	result, err = processor.ProcessQuery(ctx, query, lowConfidenceDocs)
	if err != nil {
		t.Fatalf("CRAG processing failed: %v", err)
	}
	
	// Should use web search for low confidence
	expectedActions := []CRAGAction{EnrichWithWeb, ReplaceWithWeb}
	actionFound := false
	for _, expectedAction := range expectedActions {
		if result.RoutingDecision.Action == expectedAction {
			actionFound = true
			break
		}
	}
	
	if !actionFound {
		t.Errorf("Expected web search action for low confidence, got %v", result.RoutingDecision.Action)
	}
}

func TestSimpleCRAGProcessor_ProcessQuery(t *testing.T) {
	webSearcher := NewMockWebSearcher()
	processor := NewSimpleCRAGProcessor(webSearcher)
	
	// Add mock web results
	webResults := []WebDocument{
		{
			Title:       "Simple Web Result",
			Content:     "Web content",
			URL:         "https://example.com/simple",
			Score:       0.7,
			Source:      "simple_web",
			RetrievedAt: time.Now(),
		},
	}
	webSearcher.AddMockResult("simple query", webResults)
	
	ctx := context.Background()
	query := "simple query"
	
	// Test with medium-confidence documents
	docs := []Document{
		{
			ID:      "doc1",
			Content: "Medium relevance content",
			Title:   "Medium Match",
			Score:   0.6,
		},
	}
	
	result, err := processor.ProcessQuery(ctx, query, docs)
	if err != nil {
		t.Fatalf("Simple CRAG processing failed: %v", err)
	}
	
	if result.Query != query {
		t.Errorf("Expected query '%s', got '%s'", query, result.Query)
	}
	
	if result.ProcessingTime <= 0 {
		t.Error("Expected positive processing time")
	}
	
	if len(result.FinalDocuments) == 0 {
		t.Error("Expected some final documents")
	}
}

func BenchmarkLLMBasedEvaluator_EvaluateRetrieval(b *testing.B) {
	llmProvider := NewMockLLMProvider()
	evaluator := NewLLMBasedEvaluator(llmProvider, nil)
	
	ctx := context.Background()
	query := "benchmark query"
	documents := []Document{
		{
			ID:      "doc1",
			Content: "This is benchmark content for testing evaluation performance.",
			Title:   "Benchmark Doc",
			Score:   0.8,
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := evaluator.EvaluateRetrieval(ctx, query, documents)
		if err != nil {
			b.Fatalf("Evaluation failed: %v", err)
		}
	}
}

func BenchmarkStandardCRAGProcessor_ProcessQuery(b *testing.B) {
	// Setup
	llmProvider := NewMockLLMProvider()
	evaluator := NewLLMBasedEvaluator(llmProvider, nil)
	webSearcher := NewMockWebSearcher()
	refinement := NewStandardKnowledgeRefinement(evaluator)
	processor := NewStandardCRAGProcessor(evaluator, webSearcher, refinement, nil)
	
	ctx := context.Background()
	query := "benchmark query"
	documents := []Document{
		{
			ID:      "doc1",
			Content: "Benchmark content for CRAG processing performance testing.",
			Title:   "Benchmark",
			Score:   0.7,
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processor.ProcessQuery(ctx, query, documents)
		if err != nil {
			b.Fatalf("CRAG processing failed: %v", err)
		}
	}
}