package postprocessing

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
)

// StandardDeduplicator implements comprehensive duplicate removal
type StandardDeduplicator struct {
	config *PostProcessingConfig
}

// NewStandardDeduplicator creates a new standard deduplicator
func NewStandardDeduplicator(config *PostProcessingConfig) *StandardDeduplicator {
	return &StandardDeduplicator{
		config: config,
	}
}

// Deduplicate removes duplicate or very similar results
func (d *StandardDeduplicator) Deduplicate(ctx context.Context, results []SearchResult, options *DeduplicationOptions) ([]SearchResult, error) {
	if options == nil {
		options = DefaultDeduplicationOptions()
	}

	if len(results) <= 1 {
		return results, nil
	}

	// Apply deduplication based on method
	switch options.Method {
	case TextSimilarity:
		return d.textSimilarityDeduplicate(ctx, results, options)
	case HashBased:
		return d.hashBasedDeduplicate(ctx, results, options)
	case SemanticSimilarity:
		return d.semanticSimilarityDeduplicate(ctx, results, options)
	case URLBased:
		return d.urlBasedDeduplicate(ctx, results, options)
	case HybridDeduplication:
		return d.hybridDeduplicate(ctx, results, options)
	default:
		return d.textSimilarityDeduplicate(ctx, results, options)
	}
}

// CalculateSimilarity calculates similarity between two results
func (d *StandardDeduplicator) CalculateSimilarity(ctx context.Context, result1, result2 SearchResult) (float64, error) {
	// Calculate different types of similarity
	contentSim := d.calculateContentSimilarity(result1.Content, result2.Content)
	titleSim := d.calculateContentSimilarity(result1.Title, result2.Title)
	urlSim := d.calculateURLSimilarity(result1.URL, result2.URL)

	// Weight different similarity types
	totalSim := (contentSim * 0.6) + (titleSim * 0.3) + (urlSim * 0.1)

	return totalSim, nil
}

// textSimilarityDeduplicate removes duplicates based on text similarity
func (d *StandardDeduplicator) textSimilarityDeduplicate(ctx context.Context, results []SearchResult, options *DeduplicationOptions) ([]SearchResult, error) {
	var deduplicated []SearchResult
	
	for _, result := range results {
		isDuplicate := false
		
		for _, existing := range deduplicated {
			similarity, err := d.CalculateSimilarity(ctx, result, existing)
			if err != nil {
				continue
			}
			
			if similarity >= options.SimilarityThreshold {
				isDuplicate = true
				
				// Decide which result to keep based on preferences
				if d.shouldReplaceExisting(result, existing, options) {
					// Replace existing with current result
					for i, dup := range deduplicated {
						if dup.ID == existing.ID {
							deduplicated[i] = result
							break
						}
					}
				}
				break
			}
		}
		
		if !isDuplicate {
			deduplicated = append(deduplicated, result)
		}
	}
	
	return deduplicated, nil
}

// hashBasedDeduplicate removes duplicates based on content hashing
func (d *StandardDeduplicator) hashBasedDeduplicate(ctx context.Context, results []SearchResult, options *DeduplicationOptions) ([]SearchResult, error) {
	seen := make(map[string]SearchResult)
	
	for _, result := range results {
		hash := d.calculateContentHash(result.Content)
		
		if existing, exists := seen[hash]; exists {
			// Decide which result to keep
			if d.shouldReplaceExisting(result, existing, options) {
				seen[hash] = result
			}
		} else {
			seen[hash] = result
		}
	}
	
	// Convert map back to slice
	var deduplicated []SearchResult
	for _, result := range seen {
		deduplicated = append(deduplicated, result)
	}
	
	// Sort by score to maintain order
	sort.Slice(deduplicated, func(i, j int) bool {
		return deduplicated[i].Score > deduplicated[j].Score
	})
	
	return deduplicated, nil
}

// semanticSimilarityDeduplicate removes duplicates based on semantic similarity
func (d *StandardDeduplicator) semanticSimilarityDeduplicate(ctx context.Context, results []SearchResult, options *DeduplicationOptions) ([]SearchResult, error) {
	// For now, fall back to text similarity
	// In a real implementation, this would use embeddings
	return d.textSimilarityDeduplicate(ctx, results, options)
}

// urlBasedDeduplicate removes duplicates based on URL similarity
func (d *StandardDeduplicator) urlBasedDeduplicate(ctx context.Context, results []SearchResult, options *DeduplicationOptions) ([]SearchResult, error) {
	seen := make(map[string]SearchResult)
	
	for _, result := range results {
		normalizedURL := d.normalizeURL(result.URL)
		
		if existing, exists := seen[normalizedURL]; exists {
			if d.shouldReplaceExisting(result, existing, options) {
				seen[normalizedURL] = result
			}
		} else {
			seen[normalizedURL] = result
		}
	}
	
	// Convert map back to slice
	var deduplicated []SearchResult
	for _, result := range seen {
		deduplicated = append(deduplicated, result)
	}
	
	// Sort by score
	sort.Slice(deduplicated, func(i, j int) bool {
		return deduplicated[i].Score > deduplicated[j].Score
	})
	
	return deduplicated, nil
}

// hybridDeduplicate combines multiple deduplication methods
func (d *StandardDeduplicator) hybridDeduplicate(ctx context.Context, results []SearchResult, options *DeduplicationOptions) ([]SearchResult, error) {
	// First apply URL-based deduplication
	urlDeduplicated, err := d.urlBasedDeduplicate(ctx, results, options)
	if err != nil {
		return nil, err
	}
	
	// Then apply text similarity deduplication
	return d.textSimilarityDeduplicate(ctx, urlDeduplicated, options)
}

// Helper methods

func (d *StandardDeduplicator) shouldReplaceExisting(new, existing SearchResult, options *DeduplicationOptions) bool {
	// Prefer higher score if enabled
	if options.PreferHigherScore && new.Score > existing.Score {
		return true
	}
	
	// Prefer more recent if enabled
	if options.PreferMoreRecent && new.RetrievedAt.After(existing.RetrievedAt) {
		return true
	}
	
	// Default: keep existing
	return false
}

func (d *StandardDeduplicator) calculateContentSimilarity(content1, content2 string) float64 {
	if content1 == "" && content2 == "" {
		return 1.0
	}
	if content1 == "" || content2 == "" {
		return 0.0
	}
	
	// Simple Jaccard similarity
	words1 := d.extractWords(content1)
	words2 := d.extractWords(content2)
	
	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}
	
	// Create word sets
	set1 := make(map[string]bool)
	for _, word := range words1 {
		set1[word] = true
	}
	
	set2 := make(map[string]bool)
	for _, word := range words2 {
		set2[word] = true
	}
	
	// Calculate intersection
	intersection := 0
	for word := range set1 {
		if set2[word] {
			intersection++
		}
	}
	
	// Calculate union
	union := len(set1) + len(set2) - intersection
	
	if union == 0 {
		return 0.0
	}
	
	return float64(intersection) / float64(union)
}

func (d *StandardDeduplicator) calculateURLSimilarity(url1, url2 string) float64 {
	if url1 == "" && url2 == "" {
		return 1.0
	}
	if url1 == "" || url2 == "" {
		return 0.0
	}
	
	norm1 := d.normalizeURL(url1)
	norm2 := d.normalizeURL(url2)
	
	if norm1 == norm2 {
		return 1.0
	}
	
	// Calculate edit distance or other similarity metric
	// For simplicity, return 0 if not exact match
	return 0.0
}

func (d *StandardDeduplicator) calculateContentHash(content string) string {
	// Normalize content before hashing
	normalized := strings.ToLower(strings.TrimSpace(content))
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "\n", "")
	normalized = strings.ReplaceAll(normalized, "\t", "")
	
	hash := md5.Sum([]byte(normalized))
	return fmt.Sprintf("%x", hash)
}

func (d *StandardDeduplicator) normalizeURL(url string) string {
	if url == "" {
		return ""
	}
	
	// Basic URL normalization
	normalized := strings.ToLower(url)
	
	// Remove common URL parameters
	if idx := strings.Index(normalized, "?"); idx != -1 {
		normalized = normalized[:idx]
	}
	
	// Remove fragment
	if idx := strings.Index(normalized, "#"); idx != -1 {
		normalized = normalized[:idx]
	}
	
	// Remove trailing slash
	normalized = strings.TrimSuffix(normalized, "/")
	
	return normalized
}

func (d *StandardDeduplicator) extractWords(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	var filtered []string
	
	for _, word := range words {
		// Remove punctuation and filter short words
		cleaned := strings.Trim(word, ".,!?;:\"'()[]{}/-")
		if len(cleaned) > 2 {
			filtered = append(filtered, cleaned)
		}
	}
	
	return filtered
}

// ExactDeduplicator removes exact duplicates only
type ExactDeduplicator struct {
	config *PostProcessingConfig
}

// NewExactDeduplicator creates a new exact deduplicator
func NewExactDeduplicator(config *PostProcessingConfig) *ExactDeduplicator {
	return &ExactDeduplicator{
		config: config,
	}
}

// Deduplicate removes only exact duplicates
func (e *ExactDeduplicator) Deduplicate(ctx context.Context, results []SearchResult, options *DeduplicationOptions) ([]SearchResult, error) {
	seen := make(map[string]SearchResult)
	
	for _, result := range results {
		key := result.ID
		if key == "" {
			key = result.Content + "|" + result.Title + "|" + result.URL
		}
		
		if existing, exists := seen[key]; exists {
			// Keep the one with higher score
			if result.Score > existing.Score {
				seen[key] = result
			}
		} else {
			seen[key] = result
		}
	}
	
	// Convert back to slice
	var deduplicated []SearchResult
	for _, result := range seen {
		deduplicated = append(deduplicated, result)
	}
	
	// Sort by score
	sort.Slice(deduplicated, func(i, j int) bool {
		return deduplicated[i].Score > deduplicated[j].Score
	})
	
	return deduplicated, nil
}

func (e *ExactDeduplicator) CalculateSimilarity(ctx context.Context, result1, result2 SearchResult) (float64, error) {
	if result1.ID == result2.ID || 
		(result1.Content == result2.Content && result1.Title == result2.Title && result1.URL == result2.URL) {
		return 1.0, nil
	}
	return 0.0, nil
}

// AdvancedDeduplicator implements sophisticated deduplication
type AdvancedDeduplicator struct {
	config         *PostProcessingConfig
	similarityCache map[string]float64 // Cache for similarity calculations
}

// NewAdvancedDeduplicator creates a new advanced deduplicator
func NewAdvancedDeduplicator(config *PostProcessingConfig) *AdvancedDeduplicator {
	return &AdvancedDeduplicator{
		config:         config,
		similarityCache: make(map[string]float64),
	}
}

// Deduplicate implements advanced deduplication with clustering
func (a *AdvancedDeduplicator) Deduplicate(ctx context.Context, results []SearchResult, options *DeduplicationOptions) ([]SearchResult, error) {
	if len(results) <= 1 {
		return results, nil
	}
	
	// Create similarity matrix
	similarities := make([][]float64, len(results))
	for i := range similarities {
		similarities[i] = make([]float64, len(results))
	}
	
	// Calculate all pairwise similarities
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			sim, err := a.CalculateSimilarity(ctx, results[i], results[j])
			if err != nil {
				sim = 0.0
			}
			similarities[i][j] = sim
			similarities[j][i] = sim
		}
	}
	
	// Find clusters of similar results
	clusters := a.findClusters(similarities, options.SimilarityThreshold)
	
	// Select best representative from each cluster
	var deduplicated []SearchResult
	for _, cluster := range clusters {
		best := a.selectBestFromCluster(results, cluster, options)
		deduplicated = append(deduplicated, best)
	}
	
	// Sort by score
	sort.Slice(deduplicated, func(i, j int) bool {
		return deduplicated[i].Score > deduplicated[j].Score
	})
	
	return deduplicated, nil
}

func (a *AdvancedDeduplicator) CalculateSimilarity(ctx context.Context, result1, result2 SearchResult) (float64, error) {
	// Create cache key
	key := fmt.Sprintf("%s|%s", result1.ID, result2.ID)
	if cached, exists := a.similarityCache[key]; exists {
		return cached, nil
	}
	
	// Calculate comprehensive similarity
	contentSim := a.calculateJaccardSimilarity(result1.Content, result2.Content)
	titleSim := a.calculateJaccardSimilarity(result1.Title, result2.Title)
	urlSim := a.calculateURLSimilarity(result1.URL, result2.URL)
	
	// Weighted combination
	totalSim := (contentSim * 0.6) + (titleSim * 0.3) + (urlSim * 0.1)
	
	// Cache result
	a.similarityCache[key] = totalSim
	
	return totalSim, nil
}

func (a *AdvancedDeduplicator) findClusters(similarities [][]float64, threshold float64) [][]int {
	n := len(similarities)
	visited := make([]bool, n)
	var clusters [][]int
	
	for i := 0; i < n; i++ {
		if !visited[i] {
			cluster := a.dfsCluster(similarities, i, threshold, visited)
			clusters = append(clusters, cluster)
		}
	}
	
	return clusters
}

func (a *AdvancedDeduplicator) dfsCluster(similarities [][]float64, start int, threshold float64, visited []bool) []int {
	var cluster []int
	var stack []int
	
	stack = append(stack, start)
	
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		
		if visited[current] {
			continue
		}
		
		visited[current] = true
		cluster = append(cluster, current)
		
		// Add similar unvisited nodes to stack
		for j := 0; j < len(similarities[current]); j++ {
			if !visited[j] && similarities[current][j] >= threshold {
				stack = append(stack, j)
			}
		}
	}
	
	return cluster
}

func (a *AdvancedDeduplicator) selectBestFromCluster(results []SearchResult, cluster []int, options *DeduplicationOptions) SearchResult {
	if len(cluster) == 1 {
		return results[cluster[0]]
	}
	
	best := results[cluster[0]]
	
	for i := 1; i < len(cluster); i++ {
		candidate := results[cluster[i]]
		
		// Apply selection criteria
		if options.PreferHigherScore && candidate.Score > best.Score {
			best = candidate
		} else if options.PreferMoreRecent && candidate.RetrievedAt.After(best.RetrievedAt) {
			best = candidate
		}
	}
	
	return best
}

func (a *AdvancedDeduplicator) calculateJaccardSimilarity(text1, text2 string) float64 {
	if text1 == "" && text2 == "" {
		return 1.0
	}
	if text1 == "" || text2 == "" {
		return 0.0
	}
	
	words1 := a.extractNormalizedWords(text1)
	words2 := a.extractNormalizedWords(text2)
	
	set1 := make(map[string]bool)
	for _, word := range words1 {
		set1[word] = true
	}
	
	set2 := make(map[string]bool)
	for _, word := range words2 {
		set2[word] = true
	}
	
	intersection := 0
	for word := range set1 {
		if set2[word] {
			intersection++
		}
	}
	
	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}
	
	return float64(intersection) / float64(union)
}

func (a *AdvancedDeduplicator) calculateURLSimilarity(url1, url2 string) float64 {
	if url1 == "" && url2 == "" {
		return 1.0
	}
	if url1 == "" || url2 == "" {
		return 0.0
	}
	
	// Normalize URLs
	norm1 := a.normalizeURL(url1)
	norm2 := a.normalizeURL(url2)
	
	if norm1 == norm2 {
		return 1.0
	}
	
	// Calculate domain similarity
	domain1 := a.extractDomain(norm1)
	domain2 := a.extractDomain(norm2)
	
	if domain1 == domain2 {
		return 0.5 // Same domain but different paths
	}
	
	return 0.0
}

func (a *AdvancedDeduplicator) extractNormalizedWords(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	var normalized []string
	
	for _, word := range words {
		cleaned := strings.Trim(word, ".,!?;:\"'()[]{}/-")
		if len(cleaned) > 2 {
			normalized = append(normalized, cleaned)
		}
	}
	
	return normalized
}

func (a *AdvancedDeduplicator) normalizeURL(url string) string {
	normalized := strings.ToLower(strings.TrimSpace(url))
	
	// Remove query parameters and fragments
	if idx := strings.Index(normalized, "?"); idx != -1 {
		normalized = normalized[:idx]
	}
	if idx := strings.Index(normalized, "#"); idx != -1 {
		normalized = normalized[:idx]
	}
	
	// Remove trailing slash
	normalized = strings.TrimSuffix(normalized, "/")
	
	return normalized
}

func (a *AdvancedDeduplicator) extractDomain(url string) string {
	// Simple domain extraction
	if strings.HasPrefix(url, "http://") {
		url = url[7:]
	} else if strings.HasPrefix(url, "https://") {
		url = url[8:]
	}
	
	if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}
	
	return url
}