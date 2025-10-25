package fusion

import (
	"context"
	"math"
	"sort"
	"time"
)

// SearchResult represents a unified search result from any retrieval method
type SearchResult struct {
	DocumentID  string                 `json:"document_id"`
	Content     string                 `json:"content"`
	Title       string                 `json:"title,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Score       float64                `json:"score"`
	Source      string                 `json:"source"`      // "vector", "bm25", "hybrid"
	Method      string                 `json:"method"`      // "vector_search", "bm25_search", "fusion"
	Rank        int                    `json:"rank"`        // Original rank in source method
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RetrievedAt time.Time              `json:"retrieved_at"`
}

// FusionMethod defines different fusion algorithms
type FusionMethod int

const (
	// RRFFusion uses Reciprocal Rank Fusion
	RRFFusion FusionMethod = iota
	// WeightedFusion uses weighted score combination
	WeightedFusion
	// BordaFusion uses Borda count method
	BordaFusion
	// CombSUMFusion uses CombSUM algorithm
	CombSUMFusion
	// CombMNZFusion uses CombMNZ algorithm
	CombMNZFusion
)

func (f FusionMethod) String() string {
	switch f {
	case RRFFusion:
		return "rrf"
	case WeightedFusion:
		return "weighted"
	case BordaFusion:
		return "borda"
	case CombSUMFusion:
		return "combsum"
	case CombMNZFusion:
		return "combmnz"
	default:
		return "unknown"
	}
}

// HybridRetriever defines the interface for hybrid search
type HybridRetriever interface {
	// HybridSearch performs search using multiple retrieval methods and fuses results
	HybridSearch(ctx context.Context, query string, options *HybridSearchOptions) ([]*SearchResult, error)
	
	// VectorSearch performs vector-based semantic search
	VectorSearch(ctx context.Context, query string, topK int) ([]*SearchResult, error)
	
	// BM25Search performs BM25-based keyword search
	BM25Search(ctx context.Context, query string, topK int) ([]*SearchResult, error)
	
	// FuseResults combines results from multiple retrieval methods
	FuseResults(vectorResults, bm25Results []*SearchResult, options *FusionOptions) ([]*SearchResult, error)
}

// HybridSearchOptions defines options for hybrid search
type HybridSearchOptions struct {
	// Fusion method to use
	FusionMethod FusionMethod `json:"fusion_method"`
	
	// Number of results to retrieve from each method
	VectorTopK int `json:"vector_top_k"`
	BM25TopK   int `json:"bm25_top_k"`
	
	// Final number of results to return
	FinalTopK int `json:"final_top_k"`
	
	// Fusion-specific options
	FusionOptions *FusionOptions `json:"fusion_options,omitempty"`
	
	// Method weights (for weighted fusion)
	VectorWeight float64 `json:"vector_weight"`
	BM25Weight   float64 `json:"bm25_weight"`
	
	// Minimum score threshold
	MinScore float64 `json:"min_score"`
	
	// Enable/disable individual methods
	EnableVector bool `json:"enable_vector"`
	EnableBM25   bool `json:"enable_bm25"`
}

// FusionOptions contains options specific to fusion algorithms
type FusionOptions struct {
	// RRF parameter (typically 60)
	RRFConstant float64 `json:"rrf_constant"`
	
	// Score normalization method
	ScoreNormalization NormalizationMethod `json:"score_normalization"`
	
	// Rank normalization options
	RankNormalization bool `json:"rank_normalization"`
	
	// Tie-breaking strategy
	TieBreaking TieBreakingStrategy `json:"tie_breaking"`
	
	// Diversity options
	EnableDiversity bool    `json:"enable_diversity"`
	DiversityWeight float64 `json:"diversity_weight"`
}

// NormalizationMethod defines score normalization methods
type NormalizationMethod int

const (
	NoNormalization NormalizationMethod = iota
	MinMaxNormalization
	ZScoreNormalization
	SumNormalization
)

// TieBreakingStrategy defines how to handle tied scores
type TieBreakingStrategy int

const (
	// PreferVector prefers vector search results in ties
	PreferVector TieBreakingStrategy = iota
	// PreferBM25 prefers BM25 results in ties
	PreferBM25
	// PreferHigherOriginalScore prefers result with higher original score
	PreferHigherOriginalScore
	// PreferLowerRank prefers result with lower (better) original rank
	PreferLowerRank
)

// DefaultHybridSearchOptions returns default hybrid search options
func DefaultHybridSearchOptions() *HybridSearchOptions {
	return &HybridSearchOptions{
		FusionMethod:      RRFFusion,
		VectorTopK:        20,
		BM25TopK:          20,
		FinalTopK:         10,
		VectorWeight:      0.6,
		BM25Weight:        0.4,
		MinScore:          0.0,
		EnableVector:      true,
		EnableBM25:        true,
		FusionOptions:     DefaultFusionOptions(),
	}
}

// DefaultFusionOptions returns default fusion options
func DefaultFusionOptions() *FusionOptions {
	return &FusionOptions{
		RRFConstant:        60.0,
		ScoreNormalization: MinMaxNormalization,
		RankNormalization:  true,
		TieBreaking:       PreferVector,
		EnableDiversity:   false,
		DiversityWeight:   0.1,
	}
}

// StandardHybridRetriever implements hybrid retrieval using multiple backends
type StandardHybridRetriever struct {
	vectorSearcher VectorSearcher
	bm25Searcher   BM25Searcher
	fusionEngine   FusionEngine
	config         *FusionConfig
}

// VectorSearcher defines interface for vector search
type VectorSearcher interface {
	Search(ctx context.Context, query string, topK int) ([]*SearchResult, error)
}

// BM25Searcher defines interface for BM25 search  
type BM25Searcher interface {
	Search(ctx context.Context, query string, topK int) ([]*SearchResult, error)
}

// FusionEngine defines interface for result fusion
type FusionEngine interface {
	Fuse(vectorResults, bm25Results []*SearchResult, options *FusionOptions) ([]*SearchResult, error)
}

// NewStandardHybridRetriever creates a new standard hybrid retriever
func NewStandardHybridRetriever(vectorSearcher VectorSearcher, bm25Searcher BM25Searcher) *StandardHybridRetriever {
	return &StandardHybridRetriever{
		vectorSearcher: vectorSearcher,
		bm25Searcher:   bm25Searcher,
		fusionEngine:   NewStandardFusionEngine(),
		config:         DefaultFusionConfig(),
	}
}

// HybridSearch performs search using multiple retrieval methods and fuses results
func (h *StandardHybridRetriever) HybridSearch(ctx context.Context, query string, options *HybridSearchOptions) ([]*SearchResult, error) {
	if options == nil {
		options = DefaultHybridSearchOptions()
	}
	
	var vectorResults, bm25Results []*SearchResult
	var err error
	
	// Perform vector search if enabled
	if options.EnableVector && h.vectorSearcher != nil {
		vectorResults, err = h.VectorSearch(ctx, query, options.VectorTopK)
		if err != nil {
			// Log error but continue with BM25 only
			vectorResults = []*SearchResult{}
		}
	}
	
	// Perform BM25 search if enabled
	if options.EnableBM25 && h.bm25Searcher != nil {
		bm25Results, err = h.BM25Search(ctx, query, options.BM25TopK)
		if err != nil {
			// Log error but continue with vector only
			bm25Results = []*SearchResult{}
		}
	}
	
	// If both searches failed or are disabled, return empty results
	if len(vectorResults) == 0 && len(bm25Results) == 0 {
		return []*SearchResult{}, nil
	}
	
	// Fuse results
	fusedResults, err := h.FuseResults(vectorResults, bm25Results, options.FusionOptions)
	if err != nil {
		return nil, err
	}
	
	// Apply final filtering and limiting
	finalResults := h.applyFinalFiltering(fusedResults, options)
	
	return finalResults, nil
}

// VectorSearch performs vector-based semantic search
func (h *StandardHybridRetriever) VectorSearch(ctx context.Context, query string, topK int) ([]*SearchResult, error) {
	if h.vectorSearcher == nil {
		return []*SearchResult{}, nil
	}
	
	results, err := h.vectorSearcher.Search(ctx, query, topK)
	if err != nil {
		return nil, err
	}
	
	// Ensure metadata is populated
	for i, result := range results {
		result.Method = "vector_search"
		result.Source = "vector"
		result.Rank = i + 1
		result.RetrievedAt = time.Now()
	}
	
	return results, nil
}

// BM25Search performs BM25-based keyword search
func (h *StandardHybridRetriever) BM25Search(ctx context.Context, query string, topK int) ([]*SearchResult, error) {
	if h.bm25Searcher == nil {
		return []*SearchResult{}, nil
	}
	
	results, err := h.bm25Searcher.Search(ctx, query, topK)
	if err != nil {
		return nil, err
	}
	
	// Ensure metadata is populated
	for i, result := range results {
		result.Method = "bm25_search"
		result.Source = "bm25"
		result.Rank = i + 1
		result.RetrievedAt = time.Now()
	}
	
	return results, nil
}

// FuseResults combines results from multiple retrieval methods
func (h *StandardHybridRetriever) FuseResults(vectorResults, bm25Results []*SearchResult, options *FusionOptions) ([]*SearchResult, error) {
	if options == nil {
		options = DefaultFusionOptions()
	}
	
	return h.fusionEngine.Fuse(vectorResults, bm25Results, options)
}

// applyFinalFiltering applies final filtering and limiting to fused results
func (h *StandardHybridRetriever) applyFinalFiltering(results []*SearchResult, options *HybridSearchOptions) []*SearchResult {
	var filtered []*SearchResult
	
	// Apply minimum score filter
	for _, result := range results {
		if result.Score >= options.MinScore {
			filtered = append(filtered, result)
		}
	}
	
	// Limit to final topK
	if options.FinalTopK > 0 && len(filtered) > options.FinalTopK {
		filtered = filtered[:options.FinalTopK]
	}
	
	return filtered
}

// StandardFusionEngine implements various fusion algorithms
type StandardFusionEngine struct{
	config *FusionConfig
}

// NewStandardFusionEngine creates a new standard fusion engine
func NewStandardFusionEngine() *StandardFusionEngine {
	return &StandardFusionEngine{
		config: DefaultFusionConfig(),
	}
}

// Fuse combines results using the specified fusion method
func (f *StandardFusionEngine) Fuse(vectorResults, bm25Results []*SearchResult, options *FusionOptions) ([]*SearchResult, error) {
	if options == nil {
		options = DefaultFusionOptions()
	}
	
	// Handle edge cases
	if len(vectorResults) == 0 && len(bm25Results) == 0 {
		return []*SearchResult{}, nil
	}
	if len(vectorResults) == 0 {
		return bm25Results, nil
	}
	if len(bm25Results) == 0 {
		return vectorResults, nil
	}
	
	// Create document map for fusion
	docMap := f.createDocumentMap(vectorResults, bm25Results)
	
	// Apply fusion algorithm based on method
	switch options.ScoreNormalization {
	case MinMaxNormalization:
		f.applyMinMaxNormalization(vectorResults, bm25Results)
	case ZScoreNormalization:
		f.applyZScoreNormalization(vectorResults, bm25Results)
	case SumNormalization:
		f.applySumNormalization(vectorResults, bm25Results)
	}
	
	// Perform fusion using RRF (default method)
	fusedResults := f.performRRFFusion(docMap, options.RRFConstant)
	
	// Apply tie breaking
	f.applyTieBreaking(fusedResults, options.TieBreaking)
	
	// Sort by final score
	sort.Slice(fusedResults, func(i, j int) bool {
		return fusedResults[i].Score > fusedResults[j].Score
	})
	
	return fusedResults, nil
}

// createDocumentMap creates a map of unique documents from both result sets
func (f *StandardFusionEngine) createDocumentMap(vectorResults, bm25Results []*SearchResult) map[string]*FusionDocument {
	docMap := make(map[string]*FusionDocument)
	
	// Add vector results
	for i, result := range vectorResults {
		doc := &FusionDocument{
			DocumentID:   result.DocumentID,
			Content:      result.Content,
			Title:        result.Title,
			URL:          result.URL,
			Metadata:     result.Metadata,
			VectorScore:  result.Score,
			VectorRank:   i + 1,
			HasVector:    true,
			RetrievedAt:  result.RetrievedAt,
		}
		docMap[result.DocumentID] = doc
	}
	
	// Add or update with BM25 results
	for i, result := range bm25Results {
		if doc, exists := docMap[result.DocumentID]; exists {
			// Document exists in both results
			doc.BM25Score = result.Score
			doc.BM25Rank = i + 1
			doc.HasBM25 = true
		} else {
			// Document only in BM25 results
			doc := &FusionDocument{
				DocumentID:  result.DocumentID,
				Content:     result.Content,
				Title:       result.Title,
				URL:         result.URL,
				Metadata:    result.Metadata,
				BM25Score:   result.Score,
				BM25Rank:    i + 1,
				HasBM25:     true,
				RetrievedAt: result.RetrievedAt,
			}
			docMap[result.DocumentID] = doc
		}
	}
	
	return docMap
}

// performRRFFusion performs Reciprocal Rank Fusion
func (f *StandardFusionEngine) performRRFFusion(docMap map[string]*FusionDocument, constant float64) []*SearchResult {
	var results []*SearchResult
	
	for _, doc := range docMap {
		var rrfScore float64
		
		// Add RRF score from vector results
		if doc.HasVector {
			rrfScore += 1.0 / (constant + float64(doc.VectorRank))
		}
		
		// Add RRF score from BM25 results
		if doc.HasBM25 {
			rrfScore += 1.0 / (constant + float64(doc.BM25Rank))
		}
		
		result := &SearchResult{
			DocumentID:  doc.DocumentID,
			Content:     doc.Content,
			Title:       doc.Title,
			URL:         doc.URL,
			Score:       rrfScore,
			Source:      "hybrid",
			Method:      "rrf_fusion",
			Metadata:    doc.Metadata,
			RetrievedAt: doc.RetrievedAt,
		}
		
		// Add fusion metadata
		if result.Metadata == nil {
			result.Metadata = make(map[string]interface{})
		}
		result.Metadata["vector_score"] = doc.VectorScore
		result.Metadata["vector_rank"] = doc.VectorRank
		result.Metadata["bm25_score"] = doc.BM25Score
		result.Metadata["bm25_rank"] = doc.BM25Rank
		result.Metadata["has_vector"] = doc.HasVector
		result.Metadata["has_bm25"] = doc.HasBM25
		
		results = append(results, result)
	}
	
	return results
}

// Score normalization methods

func (f *StandardFusionEngine) applyMinMaxNormalization(vectorResults, bm25Results []*SearchResult) {
	f.normalizeScores(vectorResults)
	f.normalizeScores(bm25Results)
}

func (f *StandardFusionEngine) applyZScoreNormalization(vectorResults, bm25Results []*SearchResult) {
	f.zScoreNormalize(vectorResults)
	f.zScoreNormalize(bm25Results)
}

func (f *StandardFusionEngine) applySumNormalization(vectorResults, bm25Results []*SearchResult) {
	f.sumNormalize(vectorResults)
	f.sumNormalize(bm25Results)
}

func (f *StandardFusionEngine) normalizeScores(results []*SearchResult) {
	if len(results) == 0 {
		return
	}
	
	var minScore, maxScore float64
	minScore = results[0].Score
	maxScore = results[0].Score
	
	// Find min and max scores
	for _, result := range results {
		if result.Score < minScore {
			minScore = result.Score
		}
		if result.Score > maxScore {
			maxScore = result.Score
		}
	}
	
	// Normalize scores to [0, 1]
	scoreRange := maxScore - minScore
	if scoreRange > 0 {
		for _, result := range results {
			result.Score = (result.Score - minScore) / scoreRange
		}
	}
}

func (f *StandardFusionEngine) zScoreNormalize(results []*SearchResult) {
	if len(results) == 0 {
		return
	}
	
	// Calculate mean
	var sum float64
	for _, result := range results {
		sum += result.Score
	}
	mean := sum / float64(len(results))
	
	// Calculate standard deviation
	var variance float64
	for _, result := range results {
		variance += math.Pow(result.Score-mean, 2)
	}
	stdDev := math.Sqrt(variance / float64(len(results)))
	
	// Apply z-score normalization
	if stdDev > 0 {
		for _, result := range results {
			result.Score = (result.Score - mean) / stdDev
		}
	}
}

func (f *StandardFusionEngine) sumNormalize(results []*SearchResult) {
	if len(results) == 0 {
		return
	}
	
	// Calculate sum of scores
	var sum float64
	for _, result := range results {
		sum += result.Score
	}
	
	// Normalize by sum
	if sum > 0 {
		for _, result := range results {
			result.Score = result.Score / sum
		}
	}
}

// applyTieBreaking applies tie breaking strategy
func (f *StandardFusionEngine) applyTieBreaking(results []*SearchResult, strategy TieBreakingStrategy) {
	// Sort with tie breaking
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			// Apply tie breaking strategy
			switch strategy {
			case PreferVector:
				iHasVector := results[i].Metadata["has_vector"].(bool)
				jHasVector := results[j].Metadata["has_vector"].(bool)
				if iHasVector && !jHasVector {
					return true
				}
				if !iHasVector && jHasVector {
					return false
				}
			case PreferBM25:
				iHasBM25 := results[i].Metadata["has_bm25"].(bool)
				jHasBM25 := results[j].Metadata["has_bm25"].(bool)
				if iHasBM25 && !jHasBM25 {
					return true
				}
				if !iHasBM25 && jHasBM25 {
					return false
				}
			case PreferLowerRank:
				iMinRank := f.getMinRank(results[i])
				jMinRank := f.getMinRank(results[j])
				return iMinRank < jMinRank
			}
			
			// Default: prefer by document ID for consistency
			return results[i].DocumentID < results[j].DocumentID
		}
		return results[i].Score > results[j].Score
	})
}

// getMinRank returns the minimum rank from both vector and BM25 results
func (f *StandardFusionEngine) getMinRank(result *SearchResult) int {
	minRank := math.MaxInt32
	
	if vectorRank, ok := result.Metadata["vector_rank"].(int); ok && vectorRank > 0 {
		minRank = min(minRank, vectorRank)
	}
	
	if bm25Rank, ok := result.Metadata["bm25_rank"].(int); ok && bm25Rank > 0 {
		minRank = min(minRank, bm25Rank)
	}
	
	return minRank
}

// FusionDocument represents a document during fusion process
type FusionDocument struct {
	DocumentID  string
	Content     string
	Title       string
	URL         string
	Metadata    map[string]interface{}
	VectorScore float64
	VectorRank  int
	BM25Score   float64
	BM25Rank    int
	HasVector   bool
	HasBM25     bool
	RetrievedAt time.Time
}

// Helper function for minimum
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}