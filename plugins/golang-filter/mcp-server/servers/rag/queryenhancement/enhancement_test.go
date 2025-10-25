package queryenhancement

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Mock LLM Provider for testing
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
	
	// Default responses for different prompt types
	if strings.Contains(prompt, "rewrite") {
		return `{
			"rewrites": [
				"What is machine learning and how does it work?",
				"Explain the concept of machine learning",
				"Machine learning definition and applications"
			]
		}`, nil
	}
	
	if strings.Contains(prompt, "expand") {
		return `{
			"synonyms": {
				"machine": ["computer", "system", "device"],
				"learning": ["training", "education", "adaptation"]
			},
			"related_terms": ["artificial intelligence", "deep learning", "neural networks"],
			"concept_terms": ["supervised learning", "unsupervised learning", "reinforcement learning"]
		}`, nil
	}
	
	if strings.Contains(prompt, "break down") {
		return `{
			"sub_queries": [
				{
					"id": "sq1",
					"query": "What is machine learning?",
					"type": "definition",
					"priority": 1,
					"keywords": ["machine", "learning", "definition"]
				},
				{
					"id": "sq2", 
					"query": "How does machine learning work?",
					"type": "procedural",
					"priority": 2,
					"keywords": ["machine", "learning", "process"]
				}
			]
		}`, nil
	}
	
	if strings.Contains(prompt, "classify") {
		return `{
			"primary_intent": "learning",
			"secondary_intent": "information_seeking",
			"confidence": 0.9,
			"query_type": "conceptual",
			"domain": "technology",
			"complexity": "moderate",
			"language": "en"
		}`, nil
	}
	
	return "Mock response", nil
}

func (m *MockLLMProvider) GenerateStructuredResponse(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	return nil, nil
}

// Mock Embedding Provider for testing
type MockEmbeddingProvider struct{}

func (m *MockEmbeddingProvider) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Return mock embedding
	return []float64{0.1, 0.2, 0.3, 0.4, 0.5}, nil
}

func (m *MockEmbeddingProvider) GetSimilarity(ctx context.Context, text1, text2 string) (float64, error) {
	return 0.8, nil
}

// Test functions

func TestLLMBasedQueryEnhancer_EnhanceQuery(t *testing.T) {
	llmProvider := NewMockLLMProvider()
	embeddingProvider := &MockEmbeddingProvider{}
	config := DefaultQueryEnhancementConfig()
	cache := NewMemoryCache()
	
	enhancer := NewLLMBasedQueryEnhancer(llmProvider, embeddingProvider, config, cache)
	
	ctx := context.Background()
	query := "What is machine learning?"
	options := DefaultEnhancementOptions()
	
	enhanced, err := enhancer.EnhanceQuery(ctx, query, options)
	if err != nil {
		t.Fatalf("Query enhancement failed: %v", err)
	}
	
	if enhanced.OriginalQuery != query {
		t.Errorf("Expected original query '%s', got '%s'", query, enhanced.OriginalQuery)
	}
	
	if len(enhanced.RewrittenQueries) == 0 {
		t.Error("Expected some rewritten queries")
	}
	
	if len(enhanced.ExpandedTerms) == 0 {
		t.Error("Expected some expanded terms")
	}
	
	if enhanced.Intent == nil {
		t.Error("Expected intent classification")
	}
	
	if len(enhanced.Enhancement.TechniquesApplied) == 0 {
		t.Error("Expected some enhancement techniques to be applied")
	}
	
	if enhanced.Enhancement.QualityScore <= 0 {
		t.Error("Expected positive quality score")
	}
}

func TestLLMBasedQueryEnhancer_RewriteQuery(t *testing.T) {
	llmProvider := NewMockLLMProvider()
	embeddingProvider := &MockEmbeddingProvider{}
	
	enhancer := NewLLMBasedQueryEnhancer(llmProvider, embeddingProvider, nil, nil)
	
	ctx := context.Background()
	query := "machine learning"
	
	rewrites, err := enhancer.RewriteQuery(ctx, query)
	if err != nil {
		t.Fatalf("Query rewrite failed: %v", err)
	}
	
	if len(rewrites) == 0 {
		t.Error("Expected some query rewrites")
	}
	
	// Check that rewrites are different from original
	for _, rewrite := range rewrites {
		if rewrite == query {
			t.Error("Rewrite should be different from original query")
		}
	}
}

func TestLLMBasedQueryEnhancer_ExpandQuery(t *testing.T) {
	llmProvider := NewMockLLMProvider()
	embeddingProvider := &MockEmbeddingProvider{}
	
	enhancer := NewLLMBasedQueryEnhancer(llmProvider, embeddingProvider, nil, nil)
	
	ctx := context.Background()
	query := "machine learning"
	
	expanded, err := enhancer.ExpandQuery(ctx, query)
	if err != nil {
		t.Fatalf("Query expansion failed: %v", err)
	}
	
	if expanded.OriginalQuery != query {
		t.Errorf("Expected original query '%s', got '%s'", query, expanded.OriginalQuery)
	}
	
	if len(expanded.ExpandedTerms) == 0 {
		t.Error("Expected some expanded terms")
	}
	
	// Check expanded terms have proper structure
	for _, term := range expanded.ExpandedTerms {
		if term.Term == "" {
			t.Error("Expanded term should not be empty")
		}
		if term.Weight <= 0 || term.Weight > 1 {
			t.Errorf("Expanded term weight should be in (0,1], got %f", term.Weight)
		}
		if term.Source == "" {
			t.Error("Expanded term should have source")
		}
	}
}

func TestLLMBasedQueryEnhancer_DecomposeQuery(t *testing.T) {
	llmProvider := NewMockLLMProvider()
	embeddingProvider := &MockEmbeddingProvider{}
	
	enhancer := NewLLMBasedQueryEnhancer(llmProvider, embeddingProvider, nil, nil)
	
	ctx := context.Background()
	query := "What is machine learning and how does it work in practice?"
	
	subQueries, err := enhancer.DecomposeQuery(ctx, query)
	if err != nil {
		t.Fatalf("Query decomposition failed: %v", err)
	}
	
	if len(subQueries) == 0 {
		t.Error("Expected some sub-queries for complex query")
	}
	
	// Check sub-queries have proper structure
	for _, sq := range subQueries {
		if sq.ID == "" {
			t.Error("Sub-query should have ID")
		}
		if sq.Query == "" {
			t.Error("Sub-query should have query text")
		}
		if sq.Priority <= 0 {
			t.Error("Sub-query should have positive priority")
		}
	}
}

func TestLLMBasedQueryEnhancer_ClassifyIntent(t *testing.T) {
	llmProvider := NewMockLLMProvider()
	embeddingProvider := &MockEmbeddingProvider{}
	
	enhancer := NewLLMBasedQueryEnhancer(llmProvider, embeddingProvider, nil, nil)
	
	ctx := context.Background()
	query := "How to implement machine learning?"
	
	classification, err := enhancer.ClassifyIntent(ctx, query)
	if err != nil {
		t.Fatalf("Intent classification failed: %v", err)
	}
	
	if classification == nil {
		t.Fatal("Expected intent classification result")
	}
	
	if classification.Confidence <= 0 || classification.Confidence > 1 {
		t.Errorf("Confidence should be in (0,1], got %f", classification.Confidence)
	}
	
	if classification.Language == "" {
		t.Error("Expected language classification")
	}
	
	if classification.ClassifiedAt.IsZero() {
		t.Error("Expected classification timestamp")
	}
}

func TestSimpleQueryEnhancer_EnhanceQuery(t *testing.T) {
	llmProvider := NewMockLLMProvider()
	enhancer := NewSimpleQueryEnhancer(llmProvider)
	
	ctx := context.Background()
	query := "machine learning programming"
	options := DefaultEnhancementOptions()
	
	enhanced, err := enhancer.EnhanceQuery(ctx, query, options)
	if err != nil {
		t.Fatalf("Simple enhancement failed: %v", err)
	}
	
	if enhanced.OriginalQuery != query {
		t.Errorf("Expected original query '%s', got '%s'", query, enhanced.OriginalQuery)
	}
	
	if enhanced.Intent == nil {
		t.Error("Expected intent classification from simple enhancer")
	}
	
	// Simple enhancer should provide some expansion for common terms
	if len(enhanced.ExpandedTerms) == 0 {
		t.Error("Expected some expanded terms from simple enhancer")
	}
}

func TestSimpleQueryEnhancer_RewriteQuery(t *testing.T) {
	llmProvider := NewMockLLMProvider()
	enhancer := NewSimpleQueryEnhancer(llmProvider)
	
	ctx := context.Background()
	
	// Test with "how to" query
	query := "how to learn programming"
	rewrites, err := enhancer.RewriteQuery(ctx, query)
	if err != nil {
		t.Fatalf("Simple rewrite failed: %v", err)
	}
	
	if len(rewrites) == 0 {
		t.Error("Expected some rewrites for 'how to' query")
	}
	
	// Check for expected rewrite patterns
	hasAlternative := false
	for _, rewrite := range rewrites {
		if strings.Contains(rewrite, "ways to") || strings.Contains(rewrite, "methods for") {
			hasAlternative = true
			break
		}
	}
	if !hasAlternative {
		t.Error("Expected alternative phrasing for 'how to' query")
	}
}

func TestMemoryCache_Operations(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	query := "test query"
	options := DefaultEnhancementOptions()
	result := &EnhancedQuery{
		OriginalQuery: query,
		ProcessedAt:   time.Now(),
	}
	
	// Test cache miss
	cached, err := cache.Get(ctx, query, options)
	if err != nil {
		t.Fatalf("Cache get failed: %v", err)
	}
	if cached != nil {
		t.Error("Expected cache miss for new query")
	}
	
	// Test cache set
	err = cache.Set(ctx, query, options, result, 1*time.Hour)
	if err != nil {
		t.Fatalf("Cache set failed: %v", err)
	}
	
	// Test cache hit
	cached, err = cache.Get(ctx, query, options)
	if err != nil {
		t.Fatalf("Cache get failed: %v", err)
	}
	if cached == nil {
		t.Error("Expected cache hit after set")
	}
	if cached.OriginalQuery != query {
		t.Errorf("Expected cached query '%s', got '%s'", query, cached.OriginalQuery)
	}
	
	// Test cache delete
	err = cache.Delete(ctx, query, options)
	if err != nil {
		t.Fatalf("Cache delete failed: %v", err)
	}
	
	// Test cache miss after delete
	cached, err = cache.Get(ctx, query, options)
	if err != nil {
		t.Fatalf("Cache get failed: %v", err)
	}
	if cached != nil {
		t.Error("Expected cache miss after delete")
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	query := "test query"
	options := DefaultEnhancementOptions()
	result := &EnhancedQuery{
		OriginalQuery: query,
		ProcessedAt:   time.Now(),
	}
	
	// Set with very short TTL
	err := cache.Set(ctx, query, options, result, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("Cache set failed: %v", err)
	}
	
	// Wait for expiration
	time.Sleep(20 * time.Millisecond)
	
	// Should be cache miss due to expiration
	cached, err := cache.Get(ctx, query, options)
	if err != nil {
		t.Fatalf("Cache get failed: %v", err)
	}
	if cached != nil {
		t.Error("Expected cache miss due to expiration")
	}
}

func TestQueryEnhancementOptions_Validation(t *testing.T) {
	options := DefaultEnhancementOptions()
	
	if !options.EnableRewrite {
		t.Error("Expected rewrite to be enabled by default")
	}
	if !options.EnableExpansion {
		t.Error("Expected expansion to be enabled by default")
	}
	if !options.EnableDecomposition {
		t.Error("Expected decomposition to be enabled by default")
	}
	if !options.EnableIntentClassification {
		t.Error("Expected intent classification to be enabled by default")
	}
	
	if options.MaxRewrites <= 0 {
		t.Error("Expected positive max rewrites")
	}
	if options.MaxExpansions <= 0 {
		t.Error("Expected positive max expansions")
	}
	if options.Language == "" {
		t.Error("Expected default language to be set")
	}
}

func TestQueryTypes_StringConversion(t *testing.T) {
	testCases := []struct {
		queryType QueryType
		expected  string
	}{
		{FactualQuery, "factual"},
		{ConceptualQuery, "conceptual"},
		{ComparativeQuery, "comparative"},
		{ProcedualQuery, "procedural"},
		{ListQuery, "list"},
		{DefinitionQuery, "definition"},
		{CausalQuery, "causal"},
		{TemporalQuery, "temporal"},
	}
	
	for _, tc := range testCases {
		if tc.queryType.String() != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, tc.queryType.String())
		}
	}
}

func TestQueryIntents_StringConversion(t *testing.T) {
	testCases := []struct {
		intent   QueryIntent
		expected string
	}{
		{InformationSeeking, "information_seeking"},
		{ProblemSolving, "problem_solving"},
		{Learning, "learning"},
		{Comparison, "comparison"},
		{Recommendation, "recommendation"},
		{Navigation, "navigation"},
		{Verification, "verification"},
		{Analysis, "analysis"},
	}
	
	for _, tc := range testCases {
		if tc.intent.String() != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, tc.intent.String())
		}
	}
}

func BenchmarkLLMBasedQueryEnhancer_EnhanceQuery(b *testing.B) {
	llmProvider := NewMockLLMProvider()
	embeddingProvider := &MockEmbeddingProvider{}
	config := DefaultQueryEnhancementConfig()
	cache := NewMemoryCache()
	
	enhancer := NewLLMBasedQueryEnhancer(llmProvider, embeddingProvider, config, cache)
	
	ctx := context.Background()
	query := "What is machine learning and how does it work?"
	options := DefaultEnhancementOptions()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := enhancer.EnhanceQuery(ctx, query, options)
		if err != nil {
			b.Fatalf("Enhancement failed: %v", err)
		}
	}
}

func BenchmarkSimpleQueryEnhancer_EnhanceQuery(b *testing.B) {
	llmProvider := NewMockLLMProvider()
	enhancer := NewSimpleQueryEnhancer(llmProvider)
	
	ctx := context.Background()
	query := "machine learning programming tutorial"
	options := DefaultEnhancementOptions()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := enhancer.EnhanceQuery(ctx, query, options)
		if err != nil {
			b.Fatalf("Enhancement failed: %v", err)
		}
	}
}

func BenchmarkMemoryCache_Operations(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	query := "benchmark query"
	options := DefaultEnhancementOptions()
	result := &EnhancedQuery{
		OriginalQuery: query,
		ProcessedAt:   time.Now(),
	}
	
	// Set initial value
	cache.Set(ctx, query, options, result, 1*time.Hour)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.Get(ctx, query, options)
		if err != nil {
			b.Fatalf("Cache get failed: %v", err)
		}
	}
}