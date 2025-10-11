package fusion

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// AdvancedFusionEngine implements advanced fusion algorithms
type AdvancedFusionEngine struct {
	config *FusionConfig
}

// NewAdvancedFusionEngine creates a new advanced fusion engine
func NewAdvancedFusionEngine(config *FusionConfig) *AdvancedFusionEngine {
	if config == nil {
		config = DefaultFusionConfig()
	}
	
	return &AdvancedFusionEngine{
		config: config,
	}
}

// Fuse combines results using advanced fusion algorithms
func (f *AdvancedFusionEngine) Fuse(vectorResults, bm25Results []*SearchResult, options *FusionOptions) ([]*SearchResult, error) {
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
	
	// Apply score normalization
	f.normalizeScores(vectorResults, bm25Results, options)
	
	// Create document map
	docMap := f.createDocumentMap(vectorResults, bm25Results)
	
	// Perform fusion based on method
	var fusedResults []*SearchResult
	var err error
	
	switch options.ScoreNormalization {
	case MinMaxNormalization:
		fusedResults = f.performRRFFusion(docMap, options.RRFConstant)
	default:
		fusedResults = f.performRRFFusion(docMap, options.RRFConstant)
	}
	
	if err != nil {
		return nil, fmt.Errorf("fusion failed: %w", err)
	}
	
	// Apply post-processing
	fusedResults = f.applyPostProcessing(fusedResults, options)
	
	return fusedResults, nil
}

// normalizeScores applies score normalization to both result sets
func (f *AdvancedFusionEngine) normalizeScores(vectorResults, bm25Results []*SearchResult, options *FusionOptions) {
	switch options.ScoreNormalization {
	case MinMaxNormalization:
		f.minMaxNormalize(vectorResults)
		f.minMaxNormalize(bm25Results)
	case ZScoreNormalization:
		f.zScoreNormalize(vectorResults)
		f.zScoreNormalize(bm25Results)
	case SumNormalization:
		f.sumNormalize(vectorResults)
		f.sumNormalize(bm25Results)
	}
}

// createDocumentMap creates a unified document map from both result sets
func (f *AdvancedFusionEngine) createDocumentMap(vectorResults, bm25Results []*SearchResult) map[string]*FusionDocument {
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
			doc.BM25Score = result.Score
			doc.BM25Rank = i + 1
			doc.HasBM25 = true
		} else {
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

// performRRFFusion implements Reciprocal Rank Fusion
func (f *AdvancedFusionEngine) performRRFFusion(docMap map[string]*FusionDocument, constant float64) []*SearchResult {
	var results []*SearchResult
	
	for _, doc := range docMap {
		var rrfScore float64
		
		// Calculate RRF score
		if doc.HasVector {
			rrfScore += 1.0 / (constant + float64(doc.VectorRank))
		}
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
			Metadata:    f.createFusionMetadata(doc),
			RetrievedAt: doc.RetrievedAt,
		}
		
		results = append(results, result)
	}
	
	return results
}

// performWeightedFusion implements weighted score combination
func (f *AdvancedFusionEngine) performWeightedFusion(docMap map[string]*FusionDocument, vectorWeight, bm25Weight float64) []*SearchResult {
	var results []*SearchResult
	
	for _, doc := range docMap {
		var weightedScore float64
		
		if doc.HasVector {
			weightedScore += doc.VectorScore * vectorWeight
		}
		if doc.HasBM25 {
			weightedScore += doc.BM25Score * bm25Weight
		}
		
		result := &SearchResult{
			DocumentID:  doc.DocumentID,
			Content:     doc.Content,
			Title:       doc.Title,
			URL:         doc.URL,
			Score:       weightedScore,
			Source:      "hybrid",
			Method:      "weighted_fusion",
			Metadata:    f.createFusionMetadata(doc),
			RetrievedAt: doc.RetrievedAt,
		}
		
		results = append(results, result)
	}
	
	return results
}

// performBordaFusion implements Borda count method
func (f *AdvancedFusionEngine) performBordaFusion(docMap map[string]*FusionDocument) []*SearchResult {
	var results []*SearchResult
	totalDocs := len(docMap)
	
	for _, doc := range docMap {
		var bordaScore float64
		
		if doc.HasVector {
			bordaScore += float64(totalDocs - doc.VectorRank + 1)
		}
		if doc.HasBM25 {
			bordaScore += float64(totalDocs - doc.BM25Rank + 1)
		}
		
		result := &SearchResult{
			DocumentID:  doc.DocumentID,
			Content:     doc.Content,
			Title:       doc.Title,
			URL:         doc.URL,
			Score:       bordaScore,
			Source:      "hybrid",
			Method:      "borda_fusion",
			Metadata:    f.createFusionMetadata(doc),
			RetrievedAt: doc.RetrievedAt,
		}
		
		results = append(results, result)
	}
	
	return results
}

// performCombSUMFusion implements CombSUM algorithm
func (f *AdvancedFusionEngine) performCombSUMFusion(docMap map[string]*FusionDocument) []*SearchResult {
	var results []*SearchResult
	
	for _, doc := range docMap {
		var combSumScore float64
		
		if doc.HasVector {
			combSumScore += doc.VectorScore
		}
		if doc.HasBM25 {
			combSumScore += doc.BM25Score
		}
		
		result := &SearchResult{
			DocumentID:  doc.DocumentID,
			Content:     doc.Content,
			Title:       doc.Title,
			URL:         doc.URL,
			Score:       combSumScore,
			Source:      "hybrid",
			Method:      "combsum_fusion",
			Metadata:    f.createFusionMetadata(doc),
			RetrievedAt: doc.RetrievedAt,
		}
		
		results = append(results, result)
	}
	
	return results
}

// performCombMNZFusion implements CombMNZ algorithm
func (f *AdvancedFusionEngine) performCombMNZFusion(docMap map[string]*FusionDocument) []*SearchResult {
	var results []*SearchResult
	
	for _, doc := range docMap {
		var combMnzScore float64
		var methodCount float64
		
		if doc.HasVector {
			combMnzScore += doc.VectorScore
			methodCount++
		}
		if doc.HasBM25 {
			combMnzScore += doc.BM25Score
			methodCount++
		}
		
		// Multiply by number of non-zero methods
		combMnzScore *= methodCount
		
		result := &SearchResult{
			DocumentID:  doc.DocumentID,
			Content:     doc.Content,
			Title:       doc.Title,
			URL:         doc.URL,
			Score:       combMnzScore,
			Source:      "hybrid",
			Method:      "combmnz_fusion",
			Metadata:    f.createFusionMetadata(doc),
			RetrievedAt: doc.RetrievedAt,
		}
		
		results = append(results, result)
	}
	
	return results
}

// Score normalization methods
func (f *AdvancedFusionEngine) minMaxNormalize(results []*SearchResult) {
	if len(results) == 0 {
		return
	}
	
	minScore := results[0].Score
	maxScore := results[0].Score
	
	// Find min and max
	for _, result := range results {
		if result.Score < minScore {
			minScore = result.Score
		}
		if result.Score > maxScore {
			maxScore = result.Score
		}
	}
	
	// Normalize to [0, 1]
	scoreRange := maxScore - minScore
	if scoreRange > 0 {
		for _, result := range results {
			result.Score = (result.Score - minScore) / scoreRange
		}
	}
}

func (f *AdvancedFusionEngine) zScoreNormalize(results []*SearchResult) {
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

func (f *AdvancedFusionEngine) sumNormalize(results []*SearchResult) {
	if len(results) == 0 {
		return
	}
	
	// Calculate sum
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

// applyPostProcessing applies post-processing steps
func (f *AdvancedFusionEngine) applyPostProcessing(results []*SearchResult, options *FusionOptions) []*SearchResult {
	// Apply tie breaking
	f.applyTieBreaking(results, options.TieBreaking)
	
	// Apply diversity if enabled
	if options.EnableDiversity {
		results = f.applyDiversityFiltering(results, options.DiversityWeight)
	}
	
	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	return results
}

// applyTieBreaking handles tied scores
func (f *AdvancedFusionEngine) applyTieBreaking(results []*SearchResult, strategy TieBreakingStrategy) {
	sort.Slice(results, func(i, j int) bool {
		// If scores are equal, apply tie breaking
		if math.Abs(results[i].Score-results[j].Score) < 1e-9 {
			switch strategy {
			case PreferVector:
				iHasVector := f.hasVector(results[i])
				jHasVector := f.hasVector(results[j])
				if iHasVector && !jHasVector {
					return true
				}
				if !iHasVector && jHasVector {
					return false
				}
			case PreferBM25:
				iHasBM25 := f.hasBM25(results[i])
				jHasBM25 := f.hasBM25(results[j])
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

// applyDiversityFiltering promotes diverse results
func (f *AdvancedFusionEngine) applyDiversityFiltering(results []*SearchResult, diversityWeight float64) []*SearchResult {
	if diversityWeight <= 0 {
		return results
	}
	
	// Simple content-based diversity using title/content similarity
	diverse := make([]*SearchResult, 0, len(results))
	
	for _, result := range results {
		isDiverse := true
		
		// Check similarity with already selected results
		for _, selected := range diverse {
			similarity := f.calculateContentSimilarity(result, selected)
			if similarity > 0.8 { // High similarity threshold
				isDiverse = false
				break
			}
		}
		
		if isDiverse {
			// Boost score for diversity
			result.Score = result.Score * (1.0 + diversityWeight)
			diverse = append(diverse, result)
		} else {
			diverse = append(diverse, result)
		}
	}
	
	return diverse
}

// Helper methods
func (f *AdvancedFusionEngine) createFusionMetadata(doc *FusionDocument) map[string]interface{} {
	metadata := make(map[string]interface{})
	
	// Copy original metadata
	for k, v := range doc.Metadata {
		metadata[k] = v
	}
	
	// Add fusion-specific metadata
	metadata["vector_score"] = doc.VectorScore
	metadata["vector_rank"] = doc.VectorRank
	metadata["bm25_score"] = doc.BM25Score
	metadata["bm25_rank"] = doc.BM25Rank
	metadata["has_vector"] = doc.HasVector
	metadata["has_bm25"] = doc.HasBM25
	metadata["fusion_timestamp"] = time.Now().Unix()
	
	return metadata
}

func (f *AdvancedFusionEngine) hasVector(result *SearchResult) bool {
	if result.Metadata != nil {
		if hasVector, ok := result.Metadata["has_vector"].(bool); ok {
			return hasVector
		}
	}
	return false
}

func (f *AdvancedFusionEngine) hasBM25(result *SearchResult) bool {
	if result.Metadata != nil {
		if hasBM25, ok := result.Metadata["has_bm25"].(bool); ok {
			return hasBM25
		}
	}
	return false
}

func (f *AdvancedFusionEngine) getMinRank(result *SearchResult) int {
	minRank := math.MaxInt32
	
	if result.Metadata != nil {
		if vectorRank, ok := result.Metadata["vector_rank"].(int); ok && vectorRank > 0 {
			if vectorRank < minRank {
				minRank = vectorRank
			}
		}
		
		if bm25Rank, ok := result.Metadata["bm25_rank"].(int); ok && bm25Rank > 0 {
			if bm25Rank < minRank {
				minRank = bm25Rank
			}
		}
	}
	
	if minRank == math.MaxInt32 {
		return result.Rank
	}
	
	return minRank
}

func (f *AdvancedFusionEngine) calculateContentSimilarity(result1, result2 *SearchResult) float64 {
	// Simple Jaccard similarity based on words
	words1 := f.extractWords(result1.Content + " " + result1.Title)
	words2 := f.extractWords(result2.Content + " " + result2.Title)
	
	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}
	
	// Calculate Jaccard similarity
	intersection := 0
	wordSet2 := make(map[string]bool)
	for _, word := range words2 {
		wordSet2[word] = true
	}
	
	for _, word := range words1 {
		if wordSet2[word] {
			intersection++
		}
	}
	
	union := len(words1) + len(words2) - intersection
	if union == 0 {
		return 0.0
	}
	
	return float64(intersection) / float64(union)
}

func (f *AdvancedFusionEngine) extractWords(text string) []string {
	// Simple word extraction
	words := strings.Fields(strings.ToLower(text))
	var filtered []string
	
	for _, word := range words {
		// Remove punctuation and filter short words
		word = strings.Trim(word, ".,!?;:\"'()[]{}/-")
		if len(word) > 2 {
			filtered = append(filtered, word)
		}
	}
	
	return filtered
}

// FusionPerformanceMetrics tracks fusion performance
type FusionPerformanceMetrics struct {
	FusionMethod     string        `json:"fusion_method"`
	ProcessingTime   time.Duration `json:"processing_time"`
	InputVectorCount int           `json:"input_vector_count"`
	InputBM25Count   int           `json:"input_bm25_count"`
	OutputCount      int           `json:"output_count"`
	DuplicatesFound  int           `json:"duplicates_found"`
	QualityScore     float64       `json:"quality_score"`
}

// CalculateQualityScore computes a quality score for fusion results
func (f *AdvancedFusionEngine) CalculateQualityScore(results []*SearchResult, options *FusionOptions) float64 {
	if len(results) == 0 {
		return 0.0
	}
	
	score := 0.0
	
	// Score based on score distribution
	var scores []float64
	for _, result := range results {
		scores = append(scores, result.Score)
	}
	
	// Check if scores are well-distributed
	if len(scores) > 1 {
		variance := f.calculateVariance(scores)
		score += math.Min(variance, 1.0) * 0.3
	}
	
	// Score based on diversity
	diversityScore := f.calculateDiversityScore(results)
	score += diversityScore * 0.3
	
	// Score based on method coverage
	coverageScore := f.calculateCoverageScore(results)
	score += coverageScore * 0.4
	
	return math.Min(score, 1.0)
}

func (f *AdvancedFusionEngine) calculateVariance(scores []float64) float64 {
	if len(scores) == 0 {
		return 0.0
	}
	
	// Calculate mean
	var sum float64
	for _, score := range scores {
		sum += score
	}
	mean := sum / float64(len(scores))
	
	// Calculate variance
	var variance float64
	for _, score := range scores {
		variance += math.Pow(score-mean, 2)
	}
	
	return variance / float64(len(scores))
}

func (f *AdvancedFusionEngine) calculateDiversityScore(results []*SearchResult) float64 {
	if len(results) <= 1 {
		return 1.0
	}
	
	uniqueSources := make(map[string]bool)
	for _, result := range results {
		uniqueSources[result.Source] = true
	}
	
	return float64(len(uniqueSources)) / 2.0 // Assuming max 2 sources (vector + bm25)
}

func (f *AdvancedFusionEngine) calculateCoverageScore(results []*SearchResult) float64 {
	hasVector := false
	hasBM25 := false
	
	for _, result := range results {
		if f.hasVector(result) {
			hasVector = true
		}
		if f.hasBM25(result) {
			hasBM25 = true
		}
	}
	
	coverage := 0.0
	if hasVector {
		coverage += 0.5
	}
	if hasBM25 {
		coverage += 0.5
	}
	
	return coverage
}