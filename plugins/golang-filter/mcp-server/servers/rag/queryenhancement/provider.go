package queryenhancement

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// QueryEnhancementProvider provides comprehensive query enhancement capabilities
type QueryEnhancementProvider struct {
	enhancer QueryEnhancer
	config   *QueryEnhancementConfig
}

// NewQueryEnhancementProvider creates a new query enhancement provider
func NewQueryEnhancementProvider(
	llmProvider LLMProvider,
	embeddingProvider EmbeddingProvider,
	vectorDB VectorDatabaseProvider,
	config *QueryEnhancementConfig,
) *QueryEnhancementProvider {
	if config == nil {
		config = DefaultQueryEnhancementConfig()
	}
	
	// Create cache if enabled
	var cache QueryEnhancementCache
	if config.CacheEnabled {
		cache = NewInMemoryCache(config.CacheSize)
	}
	
	// Create enhancers
	var llmEnhancer QueryEnhancer
	if llmProvider != nil {
		llmEnhancer = NewLLMBasedQueryEnhancer(llmProvider, embeddingProvider, config, cache)
	}
	
	var semanticEnhancer QueryEnhancer
	if embeddingProvider != nil && vectorDB != nil {
		semanticEnhancer = NewSemanticQueryEnhancer(embeddingProvider, vectorDB, config, cache)
	}
	
	simpleEnhancer := NewSimpleQueryEnhancer(config, cache)
	
	// Create hybrid enhancer
	enhancer := NewHybridQueryEnhancer(llmEnhancer, semanticEnhancer, simpleEnhancer, config, cache)
	
	return &QueryEnhancementProvider{
		enhancer: enhancer,
		config:   config,
	}
}

// EnhanceQuery enhances a query using the configured enhancement strategies
func (p *QueryEnhancementProvider) EnhanceQuery(ctx context.Context, query string, options *EnhancementOptions) (*EnhancedQuery, error) {
	if options == nil {
		options = p.config.DefaultOptions
	}
	
	return p.enhancer.EnhanceQuery(ctx, query, options)
}

// EnhanceQueries enhances multiple queries in batch
func (p *QueryEnhancementProvider) EnhanceQueries(ctx context.Context, queries []string, options *EnhancementOptions) ([]*EnhancedQuery, error) {
	var enhanced []*EnhancedQuery
	
	for _, query := range queries {
		result, err := p.EnhanceQuery(ctx, query, options)
		if err != nil {
			// Continue with other queries on error
			result = &EnhancedQuery{
				OriginalQuery: query,
				Enhancement: EnhancementSummary{
					TechniquesApplied: []string{"failed"},
					QualityScore:      0.0,
					ProcessingTime:    0,
				},
				ProcessedAt: time.Now(),
			}
		}
		enhanced = append(enhanced, result)
	}
	
	return enhanced, nil
}

// AnalyzeQueryCharacteristics analyzes query characteristics for optimization
func (p *QueryEnhancementProvider) AnalyzeQueryCharacteristics(ctx context.Context, query string) (*QueryAnalysis, error) {
	analysis := &QueryAnalysis{
		Query:       query,
		AnalyzedAt:  time.Now(),
	}
	
	// Basic analysis
	analysis.WordCount = len(strings.Fields(query))
	analysis.CharCount = len(query)
	analysis.Language = p.detectLanguage(query)
	analysis.Complexity = p.analyzeComplexity(query)
	analysis.QueryType = p.classifyQueryType(query)
	analysis.Keywords = p.extractKeywords(query)
	analysis.Entities = p.extractEntities(query)
	analysis.Sentiment = p.analyzeSentiment(query)
	
	return analysis, nil
}

// OptimizeForRetrieval optimizes query specifically for retrieval tasks
func (p *QueryEnhancementProvider) OptimizeForRetrieval(ctx context.Context, query string, retrievalType RetrievalType) (*OptimizedQuery, error) {
	// Analyze original query
	analysis, err := p.AnalyzeQueryCharacteristics(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze query: %w", err)
	}
	
	// Create optimization options based on retrieval type
	options := p.createOptimizationOptions(retrievalType, analysis)
	
	// Enhance query
	enhanced, err := p.EnhanceQuery(ctx, query, options)
	if err != nil {
		return nil, fmt.Errorf("failed to enhance query: %w", err)
	}
	
	// Create optimized query
	optimized := &OptimizedQuery{
		OriginalQuery:    query,
		OptimizedQuery:   p.selectBestQuery(enhanced),
		RetrievalType:    retrievalType,
		Analysis:         analysis,
		Enhancement:      enhanced,
		OptimizationHints: p.generateOptimizationHints(enhanced, retrievalType),
		ProcessedAt:      time.Now(),
	}
	
	return optimized, nil
}

// Helper methods

func (p *QueryEnhancementProvider) detectLanguage(query string) string {
	// Simple language detection (could be enhanced with proper library)
	if containsNonASCII(query) {
		return "unknown"
	}
	return "en"
}

func (p *QueryEnhancementProvider) analyzeComplexity(query string) Complexity {
	wordCount := len(strings.Fields(query))
	hasQuestionWords := strings.ContainsAny(strings.ToLower(query), "what when where why how")
	hasMultipleClauses := strings.Contains(query, " and ") || strings.Contains(query, " or ")
	
	if wordCount > 15 || hasMultipleClauses {
		return HighComplexity
	} else if wordCount > 8 || hasQuestionWords {
		return ModerateComplexity
	} else {
		return SimpleComplexity
	}
}

func (p *QueryEnhancementProvider) classifyQueryType(query string) QueryType {
	query = strings.ToLower(query)
	
	if strings.Contains(query, "what is") || strings.Contains(query, "define") {
		return DefinitionQuery
	} else if strings.Contains(query, "how to") || strings.Contains(query, "tutorial") {
		return ProcedualQuery
	} else if strings.Contains(query, "compare") || strings.Contains(query, "vs") {
		return ComparativeQuery
	} else if strings.Contains(query, "why") || strings.Contains(query, "because") {
		return CausalQuery
	} else if strings.Contains(query, "when") || strings.Contains(query, "time") {
		return TemporalQuery
	} else if strings.Contains(query, "list") || strings.Contains(query, "types of") {
		return ListQuery
	} else if strings.Contains(query, "concept") || strings.Contains(query, "theory") {
		return ConceptualQuery
	} else {
		return FactualQuery
	}
}

func (p *QueryEnhancementProvider) extractKeywords(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	var keywords []string
	
	// Filter out common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"what": true, "how": true, "when": true, "where": true, "why": true,
	}
	
	for _, word := range words {
		cleaned := strings.Trim(word, ".,!?;:\"'()[]{}/-")
		if len(cleaned) > 2 && !stopWords[cleaned] {
			keywords = append(keywords, cleaned)
		}
	}
	
	return keywords
}

func (p *QueryEnhancementProvider) extractEntities(query string) []Entity {
	// Simple entity extraction (could be enhanced with NER models)
	var entities []Entity
	
	words := strings.Fields(query)
	for _, word := range words {
		// Detect potential entities (capitalized words, numbers, etc.)
		if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
			entities = append(entities, Entity{
				Text:  word,
				Type:  "PERSON_OR_ORG", // Generic type
				Start: 0, // Would need proper position tracking
				End:   len(word),
			})
		}
	}
	
	return entities
}

func (p *QueryEnhancementProvider) analyzeSentiment(query string) float64 {
	// Simple sentiment analysis based on keywords
	query = strings.ToLower(query)
	
	positiveWords := []string{"good", "great", "excellent", "best", "amazing", "wonderful"}
	negativeWords := []string{"bad", "terrible", "worst", "awful", "horrible", "problem"}
	
	positiveCount := 0
	negativeCount := 0
	
	for _, word := range positiveWords {
		if strings.Contains(query, word) {
			positiveCount++
		}
	}
	
	for _, word := range negativeWords {
		if strings.Contains(query, word) {
			negativeCount++
		}
	}
	
	// Return sentiment score between -1 (negative) and 1 (positive)
	if positiveCount > 0 || negativeCount > 0 {
		return float64(positiveCount-negativeCount) / float64(positiveCount+negativeCount)
	}
	
	return 0.0 // Neutral
}

func (p *QueryEnhancementProvider) createOptimizationOptions(retrievalType RetrievalType, analysis *QueryAnalysis) *EnhancementOptions {
	options := &EnhancementOptions{
		EnableRewrite:             true,
		EnableExpansion:           true,
		EnableDecomposition:       analysis.Complexity >= ModerateComplexity,
		EnableIntentClassification: true,
		MaxRewriteCount:           3,
		MaxExpansionTerms:         10,
		MaxSubQueries:             5,
	}
	
	// Adjust based on retrieval type
	switch retrievalType {
	case VectorRetrieval:
		options.EnableExpansion = true
		options.MaxExpansionTerms = 15
	case KeywordRetrieval:
		options.EnableRewrite = true
		options.MaxRewriteCount = 5
	case HybridRetrieval:
		options.EnableRewrite = true
		options.EnableExpansion = true
	case SemanticRetrieval:
		options.EnableExpansion = true
		options.EnableIntentClassification = true
	}
	
	return options
}

func (p *QueryEnhancementProvider) selectBestQuery(enhanced *EnhancedQuery) string {
	// Select the best query from enhanced results
	if len(enhanced.RewrittenQueries) > 0 {
		return enhanced.RewrittenQueries[0] // Return first rewrite
	}
	
	// If no rewrites, enhance original with expanded terms
	if len(enhanced.ExpandedTerms) > 0 {
		expandedQuery := enhanced.OriginalQuery
		for i, term := range enhanced.ExpandedTerms {
			if i >= 3 { // Limit expansion
				break
			}
			expandedQuery += " " + term
		}
		return expandedQuery
	}
	
	return enhanced.OriginalQuery
}

func (p *QueryEnhancementProvider) generateOptimizationHints(enhanced *EnhancedQuery, retrievalType RetrievalType) []OptimizationHint {
	var hints []OptimizationHint
	
	// Generate hints based on enhancement results
	if len(enhanced.RewrittenQueries) > 0 {
		hints = append(hints, OptimizationHint{
			Type:        "rewrite",
			Description: fmt.Sprintf("Query rewritten %d times for better matching", len(enhanced.RewrittenQueries)),
			Impact:      "medium",
			Suggestion:  "Consider using rewritten queries for better results",
		})
	}
	
	if len(enhanced.ExpandedTerms) > 0 {
		hints = append(hints, OptimizationHint{
			Type:        "expansion",
			Description: fmt.Sprintf("Query expanded with %d related terms", len(enhanced.ExpandedTerms)),
			Impact:      "high",
			Suggestion:  "Expanded terms may improve recall",
		})
	}
	
	if len(enhanced.SubQueries) > 0 {
		hints = append(hints, OptimizationHint{
			Type:        "decomposition",
			Description: fmt.Sprintf("Complex query decomposed into %d sub-queries", len(enhanced.SubQueries)),
			Impact:      "high",
			Suggestion:  "Process sub-queries separately for comprehensive results",
		})
	}
	
	if enhanced.Intent != nil {
		hints = append(hints, OptimizationHint{
			Type:        "intent",
			Description: fmt.Sprintf("Intent classified as %s with %.2f confidence", enhanced.Intent.PrimaryIntent, enhanced.Intent.Confidence),
			Impact:      "medium",
			Suggestion:  "Adjust retrieval strategy based on detected intent",
		})
	}
	
	return hints
}

// containsNonASCII checks if string contains non-ASCII characters
func containsNonASCII(s string) bool {
	for _, c := range s {
		if c > 127 {
			return true
		}
	}
	return false
}

// Additional types for the provider

// QueryAnalysis contains detailed query analysis
type QueryAnalysis struct {
	Query       string      `json:"query"`
	WordCount   int         `json:"word_count"`
	CharCount   int         `json:"char_count"`
	Language    string      `json:"language"`
	Complexity  Complexity  `json:"complexity"`
	QueryType   QueryType   `json:"query_type"`
	Keywords    []string    `json:"keywords"`
	Entities    []Entity    `json:"entities"`
	Sentiment   float64     `json:"sentiment"`
	AnalyzedAt  time.Time   `json:"analyzed_at"`
}

// OptimizedQuery contains optimization results
type OptimizedQuery struct {
	OriginalQuery     string              `json:"original_query"`
	OptimizedQuery    string              `json:"optimized_query"`
	RetrievalType     RetrievalType       `json:"retrieval_type"`
	Analysis          *QueryAnalysis      `json:"analysis"`
	Enhancement       *EnhancedQuery      `json:"enhancement"`
	OptimizationHints []OptimizationHint  `json:"optimization_hints"`
	ProcessedAt       time.Time           `json:"processed_at"`
}

// Entity represents a named entity in the query
type Entity struct {
	Text  string `json:"text"`
	Type  string `json:"type"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// OptimizationHint provides suggestions for query optimization
type OptimizationHint struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Suggestion  string `json:"suggestion"`
}

// RetrievalType defines different retrieval strategies
type RetrievalType string

const (
	VectorRetrieval   RetrievalType = "vector"
	KeywordRetrieval  RetrievalType = "keyword"
	HybridRetrieval   RetrievalType = "hybrid"
	SemanticRetrieval RetrievalType = "semantic"
)