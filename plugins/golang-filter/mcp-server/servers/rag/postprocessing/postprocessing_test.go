package postprocessing

import (
	"context"
	"testing"
	"time"
)

func TestStandardPostProcessor_ProcessResults(t *testing.T) {
	config := DefaultPostProcessingConfig()
	processor := NewStandardPostProcessor(config)

	// Create test results
	results := []SearchResult{
		{
			ID:          "1",
			Content:     "This is a test document about artificial intelligence and machine learning.",
			Title:       "AI and ML Guide",
			URL:         "https://example.com/ai-guide",
			Score:       0.9,
			Rank:        1,
			Source:      "vector",
			Method:      "vector_search",
			Metadata:    map[string]interface{}{"category": "technology"},
			RetrievedAt: time.Now(),
		},
		{
			ID:          "2",
			Content:     "This is a test document about artificial intelligence and machine learning.",
			Title:       "Duplicate AI Guide",
			URL:         "https://example.com/ai-guide-copy",
			Score:       0.8,
			Rank:        2,
			Source:      "bm25",
			Method:      "bm25_search",
			Metadata:    map[string]interface{}{"category": "technology"},
			RetrievedAt: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:          "3",
			Content:     "Short content",
			Title:       "Short",
			URL:         "https://example.com/short",
			Score:       0.7,
			Rank:        3,
			Source:      "vector",
			Method:      "vector_search",
			Metadata:    map[string]interface{}{"category": "other"},
			RetrievedAt: time.Now(),
		},
		{
			ID:          "4",
			Content:     "This document discusses deep learning neural networks and their applications in computer vision.",
			Title:       "Deep Learning in Computer Vision",
			URL:         "https://example.com/deep-learning",
			Score:       0.85,
			Rank:        4,
			Source:      "hybrid",
			Method:      "fusion",
			Metadata:    map[string]interface{}{"category": "technology"},
			RetrievedAt: time.Now(),
		},
	}

	query := "artificial intelligence machine learning"
	options := DefaultProcessingOptions()

	processed, err := processor.ProcessResults(context.Background(), query, results, options)
	if err != nil {
		t.Fatalf("ProcessResults failed: %v", err)
	}

	// Check that processing occurred
	if processed.OriginalCount != len(results) {
		t.Errorf("Expected original count %d, got %d", len(results), processed.OriginalCount)
	}

	// Should have fewer results due to deduplication and filtering
	if len(processed.FinalResults) >= len(results) {
		t.Errorf("Expected fewer final results, got %d from %d", len(processed.FinalResults), len(results))
	}

	// Check that results are sorted by score
	for i := 1; i < len(processed.FinalResults); i++ {
		if processed.FinalResults[i-1].Score < processed.FinalResults[i].Score {
			t.Errorf("Results not properly sorted by score")
		}
	}

	// Check processing summary
	if len(processed.ProcessingSummary.TechniquesApplied) == 0 {
		t.Errorf("No processing techniques were applied")
	}

	t.Logf("Original results: %d", processed.OriginalCount)
	t.Logf("Final results: %d", len(processed.FinalResults))
	t.Logf("Techniques applied: %v", processed.ProcessingSummary.TechniquesApplied)
	t.Logf("Processing time: %v", processed.ProcessingSummary.ProcessingTime)
}

func TestStandardReranker_Rerank(t *testing.T) {
	config := DefaultPostProcessingConfig()
	reranker := NewStandardReranker(config)

	results := []SearchResult{
		{
			ID:          "1",
			Content:     "Machine learning algorithms are essential for AI systems.",
			Title:       "ML Algorithms",
			Score:       0.7,
			RetrievedAt: time.Now(),
		},
		{
			ID:          "2",
			Content:     "Artificial intelligence is transforming technology.",
			Title:       "AI Transformation",
			Score:       0.8,
			RetrievedAt: time.Now(),
		},
		{
			ID:          "3",
			Content:     "Deep learning neural networks process complex data.",
			Title:       "Deep Learning",
			Score:       0.6,
			RetrievedAt: time.Now().Add(-1 * time.Hour),
		},
	}

	query := "artificial intelligence machine learning"
	options := DefaultRerankingOptions()

	reranked, err := reranker.Rerank(context.Background(), query, results, options)
	if err != nil {
		t.Fatalf("Rerank failed: %v", err)
	}

	if len(reranked) != len(results) {
		t.Errorf("Expected %d results, got %d", len(results), len(reranked))
	}

	// Check that metadata was added
	for _, result := range reranked {
		if result.Metadata["reranked_score"] == nil {
			t.Errorf("Reranked score not added to metadata")
		}
	}

	t.Logf("Reranked %d results", len(reranked))
}

func TestStandardFilter_Filter(t *testing.T) {
	config := DefaultPostProcessingConfig()
	filter := NewStandardFilter(config)

	results := []SearchResult{
		{
			ID:          "1",
			Content:     "This is a good quality document with sufficient content length.",
			Score:       0.8,
			RetrievedAt: time.Now(),
		},
		{
			ID:          "2",
			Content:     "Short",
			Score:       0.9,
			RetrievedAt: time.Now(),
		},
		{
			ID:          "3",
			Content:     "This document has a low relevance score.",
			Score:       0.05,
			RetrievedAt: time.Now(),
		},
		{
			ID:          "4",
			Content:     "This is an old document with good content.",
			Score:       0.7,
			RetrievedAt: time.Now().Add(-400 * 24 * time.Hour), // Very old
		},
	}

	query := "quality document"
	options := &FilteringOptions{
		MinScore:         0.1,
		MaxAge:           365 * 24 * time.Hour,
		MinContentLength: 10,
		MaxContentLength: 1000,
	}

	filtered, err := filter.Filter(context.Background(), query, results, options)
	if err != nil {
		t.Fatalf("Filter failed: %v", err)
	}

	// Should filter out low score and very old documents
	if len(filtered) >= len(results) {
		t.Errorf("Expected fewer filtered results, got %d from %d", len(filtered), len(results))
	}

	// Check that all remaining results meet criteria
	for _, result := range filtered {
		if result.Score < options.MinScore {
			t.Errorf("Result with score %f should have been filtered out", result.Score)
		}
		
		age := time.Since(result.RetrievedAt)
		if age > options.MaxAge {
			t.Errorf("Result with age %v should have been filtered out", age)
		}
		
		if len(result.Content) < options.MinContentLength {
			t.Errorf("Result with content length %d should have been filtered out", len(result.Content))
		}
	}

	t.Logf("Filtered %d results to %d", len(results), len(filtered))
}

func TestStandardDeduplicator_Deduplicate(t *testing.T) {
	config := DefaultPostProcessingConfig()
	deduplicator := NewStandardDeduplicator(config)

	results := []SearchResult{
		{
			ID:          "1",
			Content:     "This is a unique document about machine learning.",
			Title:       "ML Document",
			Score:       0.9,
			RetrievedAt: time.Now(),
		},
		{
			ID:          "2",
			Content:     "This is a unique document about machine learning.",
			Title:       "ML Document Copy",
			Score:       0.8,
			RetrievedAt: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:          "3",
			Content:     "This is a completely different document about natural language processing.",
			Title:       "NLP Document",
			Score:       0.7,
			RetrievedAt: time.Now(),
		},
		{
			ID:          "4",
			Content:     "Another document about deep learning and neural networks.",
			Title:       "Deep Learning",
			Score:       0.85,
			RetrievedAt: time.Now(),
		},
	}

	options := &DeduplicationOptions{
		Method:              TextSimilarity,
		SimilarityThreshold: 0.8,
		ContentSimilarity:   true,
		TitleSimilarity:     true,
		PreferHigherScore:   true,
		PreferMoreRecent:    true,
	}

	deduplicated, err := deduplicator.Deduplicate(context.Background(), results, options)
	if err != nil {
		t.Fatalf("Deduplicate failed: %v", err)
	}

	// Should have fewer results due to deduplication
	if len(deduplicated) >= len(results) {
		t.Errorf("Expected fewer deduplicated results, got %d from %d", len(deduplicated), len(results))
	}

	// Should prefer higher scored duplicate
	foundHighScore := false
	for _, result := range deduplicated {
		if result.ID == "1" && result.Score == 0.9 {
			foundHighScore = true
			break
		}
	}
	if !foundHighScore {
		t.Errorf("Should have kept higher scored duplicate")
	}

	t.Logf("Deduplicated %d results to %d", len(results), len(deduplicated))
}

func TestStandardCompressor_Compress(t *testing.T) {
	config := DefaultPostProcessingConfig()
	compressor := NewStandardCompressor(config)

	results := []SearchResult{
		{
			ID:      "1",
			Content: "Machine learning is a subset of artificial intelligence that enables computers to learn and improve from experience without being explicitly programmed. It focuses on developing algorithms that can analyze data, identify patterns, and make predictions or decisions.",
			Title:   "Introduction to Machine Learning",
			Score:   0.9,
		},
		{
			ID:      "2",
			Content: "Deep learning is a specialized subset of machine learning that uses neural networks with multiple layers to model and understand complex patterns in data. It has revolutionized fields like computer vision, natural language processing, and speech recognition.",
			Title:   "Deep Learning Fundamentals",
			Score:   0.8,
		},
		{
			ID:      "3",
			Content: "Natural language processing (NLP) is a field of artificial intelligence that focuses on the interaction between computers and human language. It combines computational linguistics with machine learning to enable computers to understand, interpret, and generate human language.",
			Title:   "Natural Language Processing Overview",
			Score:   0.7,
		},
	}

	query := "machine learning artificial intelligence"
	options := &CompressionOptions{
		Method:            ExtractiveSummary,
		TargetLength:      500,
		MaxSummaryLength:  200,
		PreserveKeyPoints: true,
		IncludeReferences: true,
		QualityThreshold:  0.7,
	}

	compressed, err := compressor.Compress(context.Background(), query, results, options)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Check that compression occurred
	if compressed.OriginalLength <= compressed.CompressedLength {
		t.Errorf("Expected compression, but compressed length %d >= original length %d", 
			compressed.CompressedLength, compressed.OriginalLength)
	}

	// Check compression ratio
	if compressed.CompressionRatio <= 0 {
		t.Errorf("Expected positive compression ratio, got %f", compressed.CompressionRatio)
	}

	// Check that summary exists
	if compressed.Summary == "" {
		t.Errorf("Expected non-empty summary")
	}

	// Check that key points exist
	if len(compressed.KeyPoints) == 0 {
		t.Errorf("Expected key points")
	}

	// Check quality score
	if compressed.QualityScore <= 0 {
		t.Errorf("Expected positive quality score, got %f", compressed.QualityScore)
	}

	t.Logf("Original length: %d", compressed.OriginalLength)
	t.Logf("Compressed length: %d", compressed.CompressedLength)
	t.Logf("Compression ratio: %.2f", compressed.CompressionRatio)
	t.Logf("Quality score: %.2f", compressed.QualityScore)
	t.Logf("Summary: %s", compressed.Summary[:min(len(compressed.Summary), 100)])
	t.Logf("Key points: %d", len(compressed.KeyPoints))
}

func TestPipelinePostProcessor(t *testing.T) {
	config := DefaultPostProcessingConfig()
	pipeline := CreateDefaultPipeline(config)

	results := []SearchResult{
		{
			ID:          "1",
			Content:     "High quality machine learning content with good relevance score.",
			Title:       "ML Quality",
			Score:       0.9,
			RetrievedAt: time.Now(),
		},
		{
			ID:          "2",
			Content:     "Duplicate machine learning content with good relevance score.",
			Title:       "ML Duplicate",
			Score:       0.8,
			RetrievedAt: time.Now(),
		},
		{
			ID:          "3",
			Content:     "Low quality short text.",
			Title:       "Short",
			Score:       0.1,
			RetrievedAt: time.Now(),
		},
	}

	query := "machine learning quality"
	options := DefaultProcessingOptions()
	options.MaxResults = 2

	processed, err := pipeline.ProcessResults(context.Background(), query, results, options)
	if err != nil {
		t.Fatalf("Pipeline processing failed: %v", err)
	}

	// Should apply all pipeline steps
	expectedSteps := []string{"filtering", "deduplication", "reranking", "limiting"}
	for _, step := range expectedSteps {
		found := false
		for _, applied := range processed.ProcessingSummary.TechniquesApplied {
			if applied == step {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected pipeline step %s was not applied", step)
		}
	}

	// Should respect max results limit
	if len(processed.FinalResults) > options.MaxResults {
		t.Errorf("Expected max %d results, got %d", options.MaxResults, len(processed.FinalResults))
	}

	t.Logf("Pipeline applied %d techniques: %v", 
		len(processed.ProcessingSummary.TechniquesApplied),
		processed.ProcessingSummary.TechniquesApplied)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}