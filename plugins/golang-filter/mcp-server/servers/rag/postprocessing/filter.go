package postprocessing

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// StandardFilter implements comprehensive result filtering
type StandardFilter struct {
	config *PostProcessingConfig
}

// NewStandardFilter creates a new standard filter
func NewStandardFilter(config *PostProcessingConfig) *StandardFilter {
	return &StandardFilter{
		config: config,
	}
}

// Filter applies filtering criteria to results
func (f *StandardFilter) Filter(ctx context.Context, query string, results []SearchResult, options *FilteringOptions) ([]SearchResult, error) {
	if options == nil {
		options = DefaultFilteringOptions()
	}

	var filtered []SearchResult

	for _, result := range results {
		if f.ShouldInclude(ctx, result, *options) {
			filtered = append(filtered, result)
		}
	}

	return filtered, nil
}

// ShouldInclude determines if a result should be included based on criteria
func (f *StandardFilter) ShouldInclude(ctx context.Context, result SearchResult, criteria FilteringOptions) bool {
	// Score filter
	if result.Score < criteria.MinScore {
		return false
	}

	// Age filter
	if criteria.MaxAge > 0 {
		age := time.Since(result.RetrievedAt)
		if age > criteria.MaxAge {
			return false
		}
	}

	// Content length filter
	contentLength := len(result.Content)
	if criteria.MinContentLength > 0 && contentLength < criteria.MinContentLength {
		return false
	}
	if criteria.MaxContentLength > 0 && contentLength > criteria.MaxContentLength {
		return false
	}

	// Required keywords filter
	if len(criteria.RequiredKeywords) > 0 {
		if !f.containsRequiredKeywords(result, criteria.RequiredKeywords) {
			return false
		}
	}

	// Excluded keywords filter
	if len(criteria.ExcludedKeywords) > 0 {
		if f.containsExcludedKeywords(result, criteria.ExcludedKeywords) {
			return false
		}
	}

	// Source filters
	if len(criteria.AllowedSources) > 0 {
		if !f.isSourceAllowed(result.Source, criteria.AllowedSources) {
			return false
		}
	}
	if len(criteria.ExcludedSources) > 0 {
		if f.isSourceExcluded(result.Source, criteria.ExcludedSources) {
			return false
		}
	}

	// Language filter
	if criteria.LanguageFilter != "" {
		if !f.matchesLanguage(result, criteria.LanguageFilter) {
			return false
		}
	}

	// Content type filter
	if len(criteria.ContentTypeFilter) > 0 {
		if !f.matchesContentType(result, criteria.ContentTypeFilter) {
			return false
		}
	}

	// Custom filters
	for _, customFilter := range criteria.CustomFilters {
		if customFilter.Predicate != nil && !customFilter.Predicate(result) {
			return false
		}
	}

	return true
}

// Helper methods for filtering

func (f *StandardFilter) containsRequiredKeywords(result SearchResult, keywords []string) bool {
	content := strings.ToLower(result.Content + " " + result.Title)
	
	for _, keyword := range keywords {
		if !strings.Contains(content, strings.ToLower(keyword)) {
			return false
		}
	}
	
	return true
}

func (f *StandardFilter) containsExcludedKeywords(result SearchResult, keywords []string) bool {
	content := strings.ToLower(result.Content + " " + result.Title)
	
	for _, keyword := range keywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			return true
		}
	}
	
	return false
}

func (f *StandardFilter) isSourceAllowed(source string, allowedSources []string) bool {
	for _, allowed := range allowedSources {
		if source == allowed {
			return true
		}
	}
	return false
}

func (f *StandardFilter) isSourceExcluded(source string, excludedSources []string) bool {
	for _, excluded := range excludedSources {
		if source == excluded {
			return true
		}
	}
	return false
}

func (f *StandardFilter) matchesLanguage(result SearchResult, language string) bool {
	// Simple language detection based on metadata or content analysis
	if result.Metadata != nil {
		if lang, ok := result.Metadata["language"].(string); ok {
			return strings.EqualFold(lang, language)
		}
	}
	
	// Fallback: could implement basic language detection
	// For now, assume content matches
	return true
}

func (f *StandardFilter) matchesContentType(result SearchResult, contentTypes []string) bool {
	// Check content type from metadata or URL
	if result.Metadata != nil {
		if contentType, ok := result.Metadata["content_type"].(string); ok {
			for _, ct := range contentTypes {
				if strings.EqualFold(contentType, ct) {
					return true
				}
			}
			return false
		}
	}
	
	// Fallback: infer from URL or content
	if result.URL != "" {
		for _, ct := range contentTypes {
			if f.inferContentTypeFromURL(result.URL, ct) {
				return true
			}
		}
	}
	
	return true // Default to include if can't determine
}

func (f *StandardFilter) inferContentTypeFromURL(url, contentType string) bool {
	url = strings.ToLower(url)
	contentType = strings.ToLower(contentType)
	
	switch contentType {
	case "pdf":
		return strings.Contains(url, ".pdf")
	case "html", "web":
		return strings.Contains(url, "http") && !strings.Contains(url, ".pdf")
	case "doc", "docx":
		return strings.Contains(url, ".doc")
	case "txt":
		return strings.Contains(url, ".txt")
	default:
		return true
	}
}

// QualityFilter implements quality-based filtering
type QualityFilter struct {
	config *PostProcessingConfig
}

// NewQualityFilter creates a new quality filter
func NewQualityFilter(config *PostProcessingConfig) *QualityFilter {
	return &QualityFilter{
		config: config,
	}
}

// Filter applies quality-based filtering
func (q *QualityFilter) Filter(ctx context.Context, query string, results []SearchResult, options *FilteringOptions) ([]SearchResult, error) {
	var filtered []SearchResult

	for _, result := range results {
		qualityScore := q.calculateQualityScore(result)
		
		// Add quality score to metadata
		if result.Metadata == nil {
			result.Metadata = make(map[string]interface{})
		}
		result.Metadata["quality_score"] = qualityScore
		
		// Apply quality threshold
		if qualityScore >= q.config.MinQualityScore {
			filtered = append(filtered, result)
		}
	}

	return filtered, nil
}

func (q *QualityFilter) ShouldInclude(ctx context.Context, result SearchResult, criteria FilteringOptions) bool {
	qualityScore := q.calculateQualityScore(result)
	return qualityScore >= q.config.MinQualityScore
}

func (q *QualityFilter) calculateQualityScore(result SearchResult) float64 {
	score := 0.0
	
	// Content length quality
	contentLength := len(result.Content)
	if contentLength >= 100 && contentLength <= 2000 {
		score += 0.3
	} else if contentLength > 50 {
		score += 0.1
	}
	
	// Title presence and quality
	if result.Title != "" {
		score += 0.2
		if len(result.Title) >= 10 && len(result.Title) <= 100 {
			score += 0.1
		}
	}
	
	// URL presence (indicates structured content)
	if result.URL != "" {
		score += 0.1
	}
	
	// Metadata richness
	if result.Metadata != nil {
		metadataCount := len(result.Metadata)
		if metadataCount >= 3 {
			score += 0.2
		} else if metadataCount >= 1 {
			score += 0.1
		}
	}
	
	// Score confidence (higher scores indicate better matches)
	if result.Score >= 0.8 {
		score += 0.2
	} else if result.Score >= 0.5 {
		score += 0.1
	}
	
	return score
}

// RelevanceFilter implements relevance-based filtering
type RelevanceFilter struct {
	config *PostProcessingConfig
}

// NewRelevanceFilter creates a new relevance filter
func NewRelevanceFilter(config *PostProcessingConfig) *RelevanceFilter {
	return &RelevanceFilter{
		config: config,
	}
}

// Filter applies relevance-based filtering
func (r *RelevanceFilter) Filter(ctx context.Context, query string, results []SearchResult, options *FilteringOptions) ([]SearchResult, error) {
	var filtered []SearchResult

	queryWords := r.extractWords(query)
	
	for _, result := range results {
		relevanceScore := r.calculateRelevanceScore(queryWords, result)
		
		// Add relevance score to metadata
		if result.Metadata == nil {
			result.Metadata = make(map[string]interface{})
		}
		result.Metadata["relevance_score"] = relevanceScore
		
		// Apply relevance threshold
		if relevanceScore >= options.MinScore {
			filtered = append(filtered, result)
		}
	}

	return filtered, nil
}

func (r *RelevanceFilter) ShouldInclude(ctx context.Context, result SearchResult, criteria FilteringOptions) bool {
	// This would require the query, so we'll use the score threshold
	return result.Score >= criteria.MinScore
}

func (r *RelevanceFilter) calculateRelevanceScore(queryWords []string, result SearchResult) float64 {
	if len(queryWords) == 0 {
		return 0.0
	}
	
	content := strings.ToLower(result.Content + " " + result.Title)
	
	matchCount := 0
	for _, word := range queryWords {
		if strings.Contains(content, strings.ToLower(word)) {
			matchCount++
		}
	}
	
	return float64(matchCount) / float64(len(queryWords))
}

func (r *RelevanceFilter) extractWords(text string) []string {
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

// CompositeFilter combines multiple filters
type CompositeFilter struct {
	filters []Filter
	config  *PostProcessingConfig
}

// NewCompositeFilter creates a composite filter
func NewCompositeFilter(config *PostProcessingConfig, filters ...Filter) *CompositeFilter {
	return &CompositeFilter{
		filters: filters,
		config:  config,
	}
}

// Filter applies all constituent filters in sequence
func (c *CompositeFilter) Filter(ctx context.Context, query string, results []SearchResult, options *FilteringOptions) ([]SearchResult, error) {
	filtered := results
	
	for i, filter := range c.filters {
		var err error
		filtered, err = filter.Filter(ctx, query, filtered, options)
		if err != nil {
			return nil, fmt.Errorf("filter %d failed: %w", i, err)
		}
	}
	
	return filtered, nil
}

func (c *CompositeFilter) ShouldInclude(ctx context.Context, result SearchResult, criteria FilteringOptions) bool {
	for _, filter := range c.filters {
		if !filter.ShouldInclude(ctx, result, criteria) {
			return false
		}
	}
	return true
}

// CreateDefaultFilter creates a default composite filter
func CreateDefaultFilter(config *PostProcessingConfig) Filter {
	return NewCompositeFilter(
		config,
		NewStandardFilter(config),
		NewQualityFilter(config),
		NewRelevanceFilter(config),
	)
}