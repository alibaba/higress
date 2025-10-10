package bm25

import (
	"context"
	"fmt"
	"strings"
	"time"
	
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
)

// BM25Provider provides a high-level interface for BM25 operations
type BM25Provider struct {
	engine BM25Engine
	config *BM25Config
}

// NewBM25Provider creates a new BM25 provider
func NewBM25Provider(config *BM25Config) (*BM25Provider, error) {
	if config == nil {
		config = DefaultBM25Config()
	}
	
	// Currently only memory engine is implemented
	engine, err := NewMemoryBM25Engine(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create BM25 engine: %w", err)
	}
	
	return &BM25Provider{
		engine: engine,
		config: config,
	}, nil
}

// AddDocuments adds multiple documents to the BM25 index
func (p *BM25Provider) AddDocuments(ctx context.Context, docs []schema.Document) error {
	bm25Docs := make([]*BM25Document, 0, len(docs))
	
	for _, doc := range docs {
		bm25Doc := &BM25Document{
			ID:       doc.ID,
			Content:  doc.Content,
			Metadata: doc.Metadata,
			CreatedAt: doc.CreatedAt,
		}
		
		// Copy metadata if nil
		if bm25Doc.Metadata == nil {
			bm25Doc.Metadata = make(map[string]interface{})
		}
		
		bm25Docs = append(bm25Docs, bm25Doc)
	}
	
	return p.engine.AddDocuments(ctx, bm25Docs)
}

// Search performs BM25 search and returns results
func (p *BM25Provider) Search(ctx context.Context, query string, topK int) ([]*BM25Result, error) {
	options := &BM25SearchOptions{
		TopK:     topK,
		MinScore: 0.0,
		Highlight: false,
	}
	
	return p.engine.Search(ctx, query, options)
}

// SearchWithOptions performs BM25 search with custom options
func (p *BM25Provider) SearchWithOptions(ctx context.Context, query string, options *BM25SearchOptions) ([]*BM25Result, error) {
	return p.engine.Search(ctx, query, options)
}

// DeleteDocument removes a document from the index
func (p *BM25Provider) DeleteDocument(ctx context.Context, docID string) error {
	return p.engine.DeleteDocument(ctx, docID)
}

// GetStats returns BM25 engine statistics
func (p *BM25Provider) GetStats() *BM25Stats {
	return p.engine.GetStats()
}

// Clear removes all documents from the index
func (p *BM25Provider) Clear(ctx context.Context) error {
	return p.engine.Clear(ctx)
}

// GetDocumentCount returns the total number of documents
func (p *BM25Provider) GetDocumentCount() int {
	return p.engine.GetDocumentCount()
}

// BuildIndex rebuilds the BM25 index
func (p *BM25Provider) BuildIndex(ctx context.Context) error {
	return p.engine.BuildIndex(ctx)
}

// UpdateParameters updates BM25 algorithm parameters
func (p *BM25Provider) UpdateParameters(params BM25Parameters) {
	if memEngine, ok := p.engine.(*MemoryBM25Engine); ok {
		memEngine.SetParameters(params)
	}
}

// GetParameters returns current BM25 parameters
func (p *BM25Provider) GetParameters() BM25Parameters {
	if memEngine, ok := p.engine.(*MemoryBM25Engine); ok {
		return memEngine.GetParameters()
	}
	return p.config.Parameters
}

// BulkAdd adds documents in batches for better performance
func (p *BM25Provider) BulkAdd(ctx context.Context, docs []schema.Document, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}
	
	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		
		batch := docs[i:end]
		if err := p.AddDocuments(ctx, batch); err != nil {
			return fmt.Errorf("failed to add batch %d-%d: %w", i, end, err)
		}
	}
	
	return nil
}

// MultiSearch performs multiple searches in parallel
func (p *BM25Provider) MultiSearch(ctx context.Context, queries []string, topK int) (map[string][]*BM25Result, error) {
	if memEngine, ok := p.engine.(*MemoryBM25Engine); ok {
		options := &BM25SearchOptions{
			TopK:     topK,
			MinScore: 0.0,
			Highlight: false,
		}
		
		return memEngine.MultiSearch(ctx, queries, options)
	}
	
	// Fallback to sequential search
	results := make(map[string][]*BM25Result)
	for _, query := range queries {
		searchResults, err := p.Search(ctx, query, topK)
		if err != nil {
			return nil, fmt.Errorf("search failed for query '%s': %w", query, err)
		}
		results[query] = searchResults
	}
	
	return results, nil
}

// ExplainSearch provides detailed scoring explanation for a query
func (p *BM25Provider) ExplainSearch(ctx context.Context, query string, docID string) (*BM25ScoreExplanation, error) {
	if memEngine, ok := p.engine.(*MemoryBM25Engine); ok {
		return memEngine.ExplainScore(query, docID)
	}
	
	return nil, fmt.Errorf("score explanation not supported by current engine")
}

// SuggestTerms provides term suggestions for query completion
func (p *BM25Provider) SuggestTerms(prefix string, limit int) []string {
	if memEngine, ok := p.engine.(*MemoryBM25Engine); ok {
		return memEngine.SuggestTerms(prefix, limit)
	}
	
	return []string{}
}

// GetDocument retrieves a document by ID
func (p *BM25Provider) GetDocument(docID string) (*BM25Document, bool) {
	if memEngine, ok := p.engine.(*MemoryBM25Engine); ok {
		return memEngine.GetDocument(docID)
	}
	
	return nil, false
}

// ListDocuments returns all document IDs
func (p *BM25Provider) ListDocuments() []string {
	if memEngine, ok := p.engine.(*MemoryBM25Engine); ok {
		return memEngine.ListDocuments()
	}
	
	return []string{}
}

// HealthCheck performs a health check on the BM25 engine
func (p *BM25Provider) HealthCheck(ctx context.Context) error {
	// Perform a simple search to test engine health
	_, err := p.Search(ctx, "test", 1)
	if err != nil {
		return fmt.Errorf("BM25 engine health check failed: %w", err)
	}
	
	return nil
}

// GetConfig returns the current configuration
func (p *BM25Provider) GetConfig() *BM25Config {
	return p.config
}

// UpdateConfig updates the provider configuration
func (p *BM25Provider) UpdateConfig(config *BM25Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	
	// Update parameters if they differ
	if p.config.Parameters != config.Parameters {
		p.UpdateParameters(config.Parameters)
	}
	
	p.config = config
	return nil
}

// BatchDelete removes multiple documents
func (p *BM25Provider) BatchDelete(ctx context.Context, docIDs []string) error {
	for _, docID := range docIDs {
		if err := p.DeleteDocument(ctx, docID); err != nil {
			return fmt.Errorf("failed to delete document %s: %w", docID, err)
		}
	}
	
	return nil
}

// SearchAndHighlight performs search with highlighting
func (p *BM25Provider) SearchAndHighlight(ctx context.Context, query string, topK int) ([]*BM25Result, error) {
	options := &BM25SearchOptions{
		TopK:     topK,
		MinScore: 0.0,
		Highlight: true,
	}
	
	return p.engine.Search(ctx, query, options)
}

// SearchWithBoost performs search with term boosting
func (p *BM25Provider) SearchWithBoost(ctx context.Context, query string, topK int, boostTerms map[string]float64) ([]*BM25Result, error) {
	options := &BM25SearchOptions{
		TopK:       topK,
		MinScore:   0.0,
		Highlight:  false,
		BoostTerms: boostTerms,
	}
	
	return p.engine.Search(ctx, query, options)
}

// GetTermStatistics returns statistics for specific terms
func (p *BM25Provider) GetTermStatistics(terms []string) map[string]*TermStatistics {
	index := p.engine.GetIndex()
	stats := make(map[string]*TermStatistics)
	
	for _, term := range terms {
		termStat := &TermStatistics{
			Term:          term,
			DocumentCount: index.TermDocCount[term],
			TotalFreq:     0,
		}
		
		if docFreqs, exists := index.TermDocFreq[term]; exists {
			for _, freq := range docFreqs {
				termStat.TotalFreq += freq
			}
		}
		
		if index.TotalDocs > 0 {
			termStat.IDF = calculateIDF(termStat.DocumentCount, index.TotalDocs)
		}
		
		stats[term] = termStat
	}
	
	return stats
}

// TermStatistics provides statistics for a term
type TermStatistics struct {
	Term          string  `json:"term"`
	DocumentCount int     `json:"document_count"`
	TotalFreq     int     `json:"total_frequency"`
	IDF           float64 `json:"idf"`
}

// calculateIDF calculates inverse document frequency
func calculateIDF(docCount, totalDocs int) float64 {
	if docCount == 0 || totalDocs == 0 {
		return 0
	}
	
	df := float64(docCount)
	n := float64(totalDocs)
	
	// Standard IDF formula: log((N - df + 0.5) / (df + 0.5))
	return math.Log((n - df + 0.5) / (df + 0.5))
}

// Performance monitoring

// PerformanceMetrics tracks BM25 performance
type PerformanceMetrics struct {
	SearchCount    int64         `json:"search_count"`
	TotalSearchTime time.Duration `json:"total_search_time"`
	AvgSearchTime  time.Duration `json:"avg_search_time"`
	IndexSize      int64         `json:"index_size"`
	DocumentCount  int           `json:"document_count"`
	TermCount      int           `json:"term_count"`
	LastUpdate     time.Time     `json:"last_update"`
}

// GetPerformanceMetrics returns performance metrics
func (p *BM25Provider) GetPerformanceMetrics() *PerformanceMetrics {
	stats := p.GetStats()
	
	return &PerformanceMetrics{
		IndexSize:     stats.IndexSize,
		DocumentCount: stats.TotalDocuments,
		TermCount:     stats.TotalTerms,
		LastUpdate:    stats.LastUpdated,
	}
}

// Utility functions

// ValidateQuery checks if a query is valid for BM25 search
func ValidateQuery(query string) error {
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("query cannot be empty")
	}
	
	if len(query) > 1000 {
		return fmt.Errorf("query too long (max 1000 characters)")
	}
	
	return nil
}

// NormalizeQuery normalizes a query for consistent processing
func NormalizeQuery(query string) string {
	// Basic normalization
	query = strings.TrimSpace(query)
	query = strings.ToLower(query)
	
	// Remove multiple spaces
	words := strings.Fields(query)
	return strings.Join(words, " ")
}

// CalculateTermFrequency calculates term frequency from tokens
func CalculateTermFrequency(tokens []string) map[string]int {
	termFreqs := make(map[string]int)
	for _, token := range tokens {
		termFreqs[token]++
	}
	return termFreqs
}

// Helper function to safely import math package
import "math"