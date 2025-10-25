package fusion

import (
	"context"
	"fmt"
	
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/bm25"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/embedding"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/vectordb"
)

// VectorSearchAdapter adapts vector database to VectorSearcher interface
type VectorSearchAdapter struct {
	vectorDB  vectordb.VectorStoreProvider
	embedding embedding.Provider
}

// NewVectorSearchAdapter creates a new vector search adapter
func NewVectorSearchAdapter(vectorDB vectordb.VectorStoreProvider, embeddingProvider embedding.Provider) *VectorSearchAdapter {
	return &VectorSearchAdapter{
		vectorDB:  vectorDB,
		embedding: embeddingProvider,
	}
}

// Search performs vector search and returns results in unified format
func (v *VectorSearchAdapter) Search(ctx context.Context, query string, topK int) ([]*SearchResult, error) {
	// Generate embedding for query
	vector, err := v.embedding.GetEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}
	
	// Search in vector database
	searchOptions := &schema.SearchOptions{
		TopK:      topK,
		Threshold: 0.0, // Accept all results, let fusion decide
	}
	
	vectorResults, err := v.vectorDB.SearchDocs(ctx, vector, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	
	// Convert to unified SearchResult format
	var results []*SearchResult
	for i, result := range vectorResults {
		searchResult := &SearchResult{
			DocumentID:  result.Document.ID,
			Content:     result.Document.Content,
			Title:       getStringFromMetadata(result.Document.Metadata, "chunk_title"),
			URL:         getStringFromMetadata(result.Document.Metadata, "url"),
			Score:       result.Score,
			Source:      "vector",
			Method:      "vector_search",
			Rank:        i + 1,
			Metadata:    result.Document.Metadata,
			RetrievedAt: result.Document.CreatedAt,
		}
		results = append(results, searchResult)
	}
	
	return results, nil
}

// BM25SearchAdapter adapts BM25 engine to BM25Searcher interface
type BM25SearchAdapter struct {
	bm25Engine bm25.BM25Engine
}

// NewBM25SearchAdapter creates a new BM25 search adapter
func NewBM25SearchAdapter(bm25Engine bm25.BM25Engine) *BM25SearchAdapter {
	return &BM25SearchAdapter{
		bm25Engine: bm25Engine,
	}
}

// Search performs BM25 search and returns results in unified format
func (b *BM25SearchAdapter) Search(ctx context.Context, query string, topK int) ([]*SearchResult, error) {
	// Configure BM25 search options
	searchOptions := &bm25.BM25SearchOptions{
		TopK:     topK,
		MinScore: 0.0, // Accept all results, let fusion decide
		Highlight: false,
	}
	
	// Perform BM25 search
	bm25Results, err := b.bm25Engine.Search(ctx, query, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("BM25 search failed: %w", err)
	}
	
	// Convert to unified SearchResult format
	var results []*SearchResult
	for i, result := range bm25Results {
		searchResult := &SearchResult{
			DocumentID:  result.DocumentID,
			Content:     result.Content,
			Title:       getStringFromMetadata(result.Metadata, "chunk_title"),
			URL:         getStringFromMetadata(result.Metadata, "url"),
			Score:       result.Score,
			Source:      "bm25",
			Method:      "bm25_search",
			Rank:        i + 1,
			Metadata:    result.Metadata,
			RetrievedAt: result.RetrievedAt,
		}
		results = append(results, searchResult)
	}
	
	return results, nil
}

// HybridSearchProvider provides a complete hybrid search implementation
type HybridSearchProvider struct {
	hybridRetriever HybridRetriever
	config          *FusionConfig
}

// NewHybridSearchProvider creates a new hybrid search provider
func NewHybridSearchProvider(
	vectorDB vectordb.VectorStoreProvider,
	embeddingProvider embedding.Provider,
	bm25Engine bm25.BM25Engine,
	config *FusionConfig,
) *HybridSearchProvider {
	// Create adapters
	vectorSearcher := NewVectorSearchAdapter(vectorDB, embeddingProvider)
	bm25Searcher := NewBM25SearchAdapter(bm25Engine)
	
	// Create hybrid retriever
	hybridRetriever := NewStandardHybridRetriever(vectorSearcher, bm25Searcher)
	
	if config == nil {
		config = DefaultFusionConfig()
	}
	
	return &HybridSearchProvider{
		hybridRetriever: hybridRetriever,
		config:          config,
	}
}

// Search performs hybrid search with customizable options
func (h *HybridSearchProvider) Search(ctx context.Context, query string, options *HybridSearchOptions) ([]*SearchResult, error) {
	if options == nil {
		options = DefaultHybridSearchOptions()
	}
	
	return h.hybridRetriever.HybridSearch(ctx, query, options)
}

// SearchWithDefaults performs hybrid search with default settings
func (h *HybridSearchProvider) SearchWithDefaults(ctx context.Context, query string, topK int) ([]*SearchResult, error) {
	options := &HybridSearchOptions{
		FusionMethod:   RRFFusion,
		VectorTopK:     topK * 2, // Retrieve more candidates for fusion
		BM25TopK:       topK * 2,
		FinalTopK:      topK,
		VectorWeight:   0.6,
		BM25Weight:     0.4,
		MinScore:       0.0,
		EnableVector:   true,
		EnableBM25:     true,
		FusionOptions:  DefaultFusionOptions(),
	}
	
	return h.Search(ctx, query, options)
}

// CompareSearchMethods performs comparative search to analyze method effectiveness
func (h *HybridSearchProvider) CompareSearchMethods(ctx context.Context, query string, topK int) (*SearchComparison, error) {
	// Perform individual searches
	vectorResults, err := h.hybridRetriever.VectorSearch(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	
	bm25Results, err := h.hybridRetriever.BM25Search(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("BM25 search failed: %w", err)
	}
	
	// Perform hybrid search
	hybridResults, err := h.SearchWithDefaults(ctx, query, topK)
	if err != nil {
		return nil, fmt.Errorf("hybrid search failed: %w", err)
	}
	
	// Calculate overlap metrics
	overlap := calculateResultOverlap(vectorResults, bm25Results)
	
	return &SearchComparison{
		Query:          query,
		VectorResults:  vectorResults,
		BM25Results:    bm25Results,
		HybridResults:  hybridResults,
		Overlap:        overlap,
	}, nil
}

// Helper functions

func getStringFromMetadata(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	if val, ok := metadata[key].(string); ok {
		return val
	}
	return ""
}

func calculateResultOverlap(vectorResults, bm25Results []*SearchResult) *OverlapMetrics {
	vectorIDs := make(map[string]bool)
	for _, result := range vectorResults {
		vectorIDs[result.DocumentID] = true
	}
	
	bm25IDs := make(map[string]bool)
	var commonIDs []string
	for _, result := range bm25Results {
		bm25IDs[result.DocumentID] = true
		if vectorIDs[result.DocumentID] {
			commonIDs = append(commonIDs, result.DocumentID)
		}
	}
	
	totalUnique := len(vectorIDs) + len(bm25IDs) - len(commonIDs)
	
	var overlapRatio float64
	if totalUnique > 0 {
		overlapRatio = float64(len(commonIDs)) / float64(totalUnique)
	}
	
	return &OverlapMetrics{
		VectorCount:   len(vectorResults),
		BM25Count:     len(bm25Results),
		CommonCount:   len(commonIDs),
		OverlapRatio:  overlapRatio,
		CommonIDs:     commonIDs,
	}
}

// SearchComparison contains comparative search results
type SearchComparison struct {
	Query         string           `json:"query"`
	VectorResults []*SearchResult  `json:"vector_results"`
	BM25Results   []*SearchResult  `json:"bm25_results"`
	HybridResults []*SearchResult  `json:"hybrid_results"`
	Overlap       *OverlapMetrics  `json:"overlap"`
}

// OverlapMetrics provides overlap analysis between search methods
type OverlapMetrics struct {
	VectorCount  int      `json:"vector_count"`
	BM25Count    int      `json:"bm25_count"`
	CommonCount  int      `json:"common_count"`
	OverlapRatio float64  `json:"overlap_ratio"`
	CommonIDs    []string `json:"common_ids"`
}

// FusionConfig contains configuration for fusion behavior
type FusionConfig struct {
	DefaultMethod       FusionMethod `json:"default_method"`
	RRFConstant        float64      `json:"rrf_constant"`
	VectorWeight       float64      `json:"vector_weight"`
	BM25Weight         float64      `json:"bm25_weight"`
	EnableNormalization bool        `json:"enable_normalization"`
	EnableDiversity    bool         `json:"enable_diversity"`
	DiversityWeight    float64      `json:"diversity_weight"`
}

// DefaultFusionConfig returns default fusion configuration
func DefaultFusionConfig() *FusionConfig {
	return &FusionConfig{
		DefaultMethod:       RRFFusion,
		RRFConstant:        60.0,
		VectorWeight:       0.6,
		BM25Weight:         0.4,
		EnableNormalization: true,
		EnableDiversity:    false,
		DiversityWeight:    0.1,
	}
}