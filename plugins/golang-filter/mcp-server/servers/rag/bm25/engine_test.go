package bm25

import (
	"testing"
	"context"
	"fmt"
)

func TestSimpleTokenizer(t *testing.T) {
	config := DefaultTokenizerConfig()
	tokenizer := NewSimpleTokenizer(config)
	
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "Hello world! This is a test.",
			expected: []string{"hello", "world", "test"},
		},
		{
			input:    "The quick brown fox jumps over the lazy dog.",
			expected: []string{"quick", "brown", "fox", "jumps", "over", "lazy", "dog"},
		},
		{
			input:    "BM25 is a ranking algorithm.",
			expected: []string{"bm25", "ranking", "algorithm"},
		},
	}
	
	for _, test := range tests {
		result := tokenizer.Tokenize(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("Expected %d tokens, got %d for input: %s", len(test.expected), len(result), test.input)
			continue
		}
		
		for i, token := range result {
			if token != test.expected[i] {
				t.Errorf("Expected token %s, got %s at position %d", test.expected[i], token, i)
			}
		}
	}
}

func TestMemoryBM25Engine_AddAndSearch(t *testing.T) {
	engine, err := NewMemoryBM25Engine(nil)
	if err != nil {
		t.Fatalf("Failed to create BM25 engine: %v", err)
	}
	
	ctx := context.Background()
	
	// Add test documents
	docs := []*BM25Document{
		{
			ID:      "doc1",
			Content: "Information retrieval is the activity of obtaining information system resources.",
			Metadata: map[string]interface{}{"title": "Information Retrieval"},
		},
		{
			ID:      "doc2", 
			Content: "Machine learning algorithms can improve information retrieval effectiveness.",
			Metadata: map[string]interface{}{"title": "ML and IR"},
		},
		{
			ID:      "doc3",
			Content: "Search engines use various ranking algorithms including BM25.",
			Metadata: map[string]interface{}{"title": "Search Engines"},
		},
	}
	
	err = engine.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}
	
	// Test search
	options := &BM25SearchOptions{
		TopK:     10,
		MinScore: 0.0,
		Highlight: true,
	}
	
	results, err := engine.Search(ctx, "information retrieval", options)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	
	if len(results) == 0 {
		t.Fatal("Expected search results, got none")
	}
	
	// First result should be doc1 (highest relevance)
	if results[0].DocumentID != "doc1" {
		t.Errorf("Expected first result to be doc1, got %s", results[0].DocumentID)
	}
	
	if results[0].Score <= 0 {
		t.Errorf("Expected positive score, got %f", results[0].Score)
	}
	
	// Test document count
	if engine.GetDocumentCount() != 3 {
		t.Errorf("Expected 3 documents, got %d", engine.GetDocumentCount())
	}
	
	// Test term count
	if engine.GetTermCount() == 0 {
		t.Error("Expected non-zero term count")
	}
}

func TestMemoryBM25Engine_DeleteDocument(t *testing.T) {
	engine, err := NewMemoryBM25Engine(nil)
	if err != nil {
		t.Fatalf("Failed to create BM25 engine: %v", err)
	}
	
	ctx := context.Background()
	
	// Add a document
	doc := &BM25Document{
		ID:      "test-doc",
		Content: "This is a test document for deletion.",
		Metadata: map[string]interface{}{"title": "Test Doc"},
	}
	
	err = engine.AddDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}
	
	// Verify document exists
	if engine.GetDocumentCount() != 1 {
		t.Fatalf("Expected 1 document, got %d", engine.GetDocumentCount())
	}
	
	// Delete document
	err = engine.DeleteDocument(ctx, "test-doc")
	if err != nil {
		t.Fatalf("Failed to delete document: %v", err)
	}
	
	// Verify document is deleted
	if engine.GetDocumentCount() != 0 {
		t.Errorf("Expected 0 documents after deletion, got %d", engine.GetDocumentCount())
	}
}

func TestMemoryBM25Engine_UpdateDocument(t *testing.T) {
	engine, err := NewMemoryBM25Engine(nil)
	if err != nil {
		t.Fatalf("Failed to create BM25 engine: %v", err)
	}
	
	ctx := context.Background()
	
	// Add original document
	originalDoc := &BM25Document{
		ID:      "update-doc",
		Content: "Original content about cats.",
		Metadata: map[string]interface{}{"title": "Original"},
	}
	
	err = engine.AddDocument(ctx, originalDoc)
	if err != nil {
		t.Fatalf("Failed to add original document: %v", err)
	}
	
	// Update document
	updatedDoc := &BM25Document{
		ID:      "update-doc",
		Content: "Updated content about dogs and cats.",
		Metadata: map[string]interface{}{"title": "Updated"},
	}
	
	err = engine.UpdateDocument(ctx, updatedDoc)
	if err != nil {
		t.Fatalf("Failed to update document: %v", err)
	}
	
	// Search for new content
	results, err := engine.Search(ctx, "dogs", &BM25SearchOptions{TopK: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	
	if len(results) == 0 {
		t.Fatal("Expected to find updated document with 'dogs'")
	}
	
	if results[0].DocumentID != "update-doc" {
		t.Errorf("Expected to find update-doc, got %s", results[0].DocumentID)
	}
	
	// Verify only one document exists
	if engine.GetDocumentCount() != 1 {
		t.Errorf("Expected 1 document after update, got %d", engine.GetDocumentCount())
	}
}

func TestMemoryBM25Engine_BM25Scoring(t *testing.T) {
	engine, err := NewMemoryBM25Engine(nil)
	if err != nil {
		t.Fatalf("Failed to create BM25 engine: %v", err)
	}
	
	ctx := context.Background()
	
	// Add documents with different term frequencies
	docs := []*BM25Document{
		{
			ID:      "doc1",
			Content: "apple apple apple fruit",  // High TF for "apple"
			Metadata: map[string]interface{}{},
		},
		{
			ID:      "doc2",
			Content: "apple fruit banana orange",  // Low TF for "apple"
			Metadata: map[string]interface{}{},
		},
		{
			ID:      "doc3",
			Content: "banana orange grape fruit",  // No "apple"
			Metadata: map[string]interface{}{},
		},
	}
	
	err = engine.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}
	
	// Search for "apple"
	results, err := engine.Search(ctx, "apple", &BM25SearchOptions{TopK: 10})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	
	// Should find 2 documents (doc1 and doc2)
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
	
	// doc1 should have higher score than doc2 (higher TF)
	if results[0].DocumentID != "doc1" {
		t.Errorf("Expected doc1 to have highest score, got %s", results[0].DocumentID)
	}
	
	if results[0].Score <= results[1].Score {
		t.Errorf("Expected doc1 score (%f) to be higher than doc2 score (%f)", 
			results[0].Score, results[1].Score)
	}
}

func TestMemoryBM25Engine_Clear(t *testing.T) {
	engine, err := NewMemoryBM25Engine(nil)
	if err != nil {
		t.Fatalf("Failed to create BM25 engine: %v", err)
	}
	
	ctx := context.Background()
	
	// Add documents
	docs := []*BM25Document{
		{ID: "doc1", Content: "test content 1", Metadata: map[string]interface{}{}},
		{ID: "doc2", Content: "test content 2", Metadata: map[string]interface{}{}},
	}
	
	err = engine.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}
	
	// Verify documents exist
	if engine.GetDocumentCount() != 2 {
		t.Fatalf("Expected 2 documents, got %d", engine.GetDocumentCount())
	}
	
	// Clear engine
	err = engine.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear engine: %v", err)
	}
	
	// Verify everything is cleared
	if engine.GetDocumentCount() != 0 {
		t.Errorf("Expected 0 documents after clear, got %d", engine.GetDocumentCount())
	}
	
	if engine.GetTermCount() != 0 {
		t.Errorf("Expected 0 terms after clear, got %d", engine.GetTermCount())
	}
}

func TestMemoryBM25Engine_Stats(t *testing.T) {
	engine, err := NewMemoryBM25Engine(nil)
	if err != nil {
		t.Fatalf("Failed to create BM25 engine: %v", err)
	}
	
	ctx := context.Background()
	
	// Initial stats
	stats := engine.GetStats()
	if stats.TotalDocuments != 0 {
		t.Errorf("Expected 0 initial documents, got %d", stats.TotalDocuments)
	}
	
	// Add a document
	doc := &BM25Document{
		ID:      "stats-doc",
		Content: "This is a test document for statistics validation.",
		Metadata: map[string]interface{}{},
	}
	
	err = engine.AddDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}
	
	// Updated stats
	stats = engine.GetStats()
	if stats.TotalDocuments != 1 {
		t.Errorf("Expected 1 document in stats, got %d", stats.TotalDocuments)
	}
	
	if stats.TotalTerms == 0 {
		t.Error("Expected non-zero terms in stats")
	}
	
	if stats.AverageDocLength <= 0 {
		t.Errorf("Expected positive average doc length, got %f", stats.AverageDocLength)
	}
	
	if stats.LastUpdated.IsZero() {
		t.Error("Expected LastUpdated to be set")
	}
}

func BenchmarkBM25Engine_AddDocument(b *testing.B) {
	engine, err := NewMemoryBM25Engine(nil)
	if err != nil {
		b.Fatalf("Failed to create BM25 engine: %v", err)
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := &BM25Document{
			ID:      fmt.Sprintf("doc-%d", i),
			Content: "This is a test document for benchmarking the BM25 engine performance.",
			Metadata: map[string]interface{}{"index": i},
		}
		
		err := engine.AddDocument(ctx, doc)
		if err != nil {
			b.Fatalf("Failed to add document: %v", err)
		}
	}
}

func BenchmarkBM25Engine_Search(b *testing.B) {
	engine, err := NewMemoryBM25Engine(nil)
	if err != nil {
		b.Fatalf("Failed to create BM25 engine: %v", err)
	}
	
	ctx := context.Background()
	
	// Pre-populate with documents
	for i := 0; i < 1000; i++ {
		doc := &BM25Document{
			ID:      fmt.Sprintf("doc-%d", i),
			Content: fmt.Sprintf("Document %d contains information about search engines and ranking algorithms.", i),
			Metadata: map[string]interface{}{"index": i},
		}
		
		err := engine.AddDocument(ctx, doc)
		if err != nil {
			b.Fatalf("Failed to add document: %v", err)
		}
	}
	
	options := &BM25SearchOptions{TopK: 10}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Search(ctx, "search engines ranking", options)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}