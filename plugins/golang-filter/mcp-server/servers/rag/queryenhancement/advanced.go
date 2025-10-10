package queryenhancement

import (
	"context"
	"strings"
	"time"
)

// SemanticQueryEnhancer implements semantic-based query enhancement
type SemanticQueryEnhancer struct {
	embeddingProvider EmbeddingProvider
	vectorDatabase    VectorDatabaseProvider
	config           *QueryEnhancementConfig
	cache            QueryEnhancementCache
}

// NewSemanticQueryEnhancer creates a new semantic query enhancer
func NewSemanticQueryEnhancer(
	embeddingProvider EmbeddingProvider,
	vectorDatabase VectorDatabaseProvider,
	config *QueryEnhancementConfig,
	cache QueryEnhancementCache,
) *SemanticQueryEnhancer {
	if config == nil {
		config = DefaultQueryEnhancementConfig()
	}
	
	return &SemanticQueryEnhancer{
		embeddingProvider: embeddingProvider,
		vectorDatabase:    vectorDatabase,
		config:           config,
		cache:            cache,
	}
}

// EnhanceQuery enhances query using semantic similarity
func (s *SemanticQueryEnhancer) EnhanceQuery(ctx context.Context, query string, options *EnhancementOptions) (*EnhancedQuery, error) {
	if options == nil {
		options = s.config.DefaultOptions
	}
	
	startTime := time.Now()
	
	// Check cache first
	if s.cache != nil && s.config.CacheEnabled {
		if cached, err := s.cache.Get(ctx, query, options); err == nil && cached != nil {
			return cached, nil
		}
	}
	
	enhanced := &EnhancedQuery{
		OriginalQuery: query,
		ProcessedAt:   time.Now(),
	}
	
	var techniques []string
	
	// Semantic expansion
	if options.EnableExpansion {
		expanded, err := s.semanticExpansion(ctx, query)
		if err == nil {
			enhanced.ExpandedTerms = s.convertToExpandedTermStrings(expanded.ExpandedTerms)
			techniques = append(techniques, "semantic_expansion")
		}
	}
	
	// Similarity-based rewriting
	if options.EnableRewrite {
		rewrites, err := s.semanticRewriting(ctx, query)
		if err == nil {
			enhanced.RewrittenQueries = rewrites
			techniques = append(techniques, "semantic_rewrite")
		}
	}
	
	// Intent classification using embeddings
	if options.EnableIntentClassification {
		intent, err := s.semanticIntentClassification(ctx, query)
		if err == nil {
			enhanced.Intent = intent
			techniques = append(techniques, "semantic_intent")
		}
	}
	
	// Create enhancement summary
	enhanced.Enhancement = EnhancementSummary{
		TechniquesApplied:  techniques,
		RewriteCount:       len(enhanced.RewrittenQueries),
		ExpansionCount:     len(enhanced.ExpandedTerms),
		DecompositionCount: len(enhanced.SubQueries),
		QualityScore:       s.calculateQualityScore(enhanced),
		ProcessingTime:     time.Since(startTime),
	}
	
	// Cache the result
	if s.cache != nil && s.config.CacheEnabled {
		_ = s.cache.Set(ctx, query, options, enhanced, s.config.CacheTTL)
	}
	
	return enhanced, nil
}

// semanticExpansion performs semantic-based query expansion
func (s *SemanticQueryEnhancer) semanticExpansion(ctx context.Context, query string) (*ExpandedQuery, error) {
	// Generate embedding for the query
	queryEmbedding, err := s.embeddingProvider.GetEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}
	
	// Find semantically similar terms from vector database
	similarTerms, err := s.vectorDatabase.FindSimilarTerms(ctx, queryEmbedding, 20)
	if err != nil {
		return nil, err
	}
	
	// Convert to ExpandedTerms
	var expandedTerms []ExpandedTerm
	for _, term := range similarTerms {
		expandedTerms = append(expandedTerms, ExpandedTerm{
			Term:       term.Term,
			Weight:     term.Score,
			Source:     "semantic_similarity",
			Confidence: term.Score,
		})
	}
	
	// Extract related concepts
	relatedTerms := s.extractRelatedTerms(similarTerms)
	conceptTerms := s.extractConceptTerms(similarTerms)
	
	return &ExpandedQuery{
		OriginalQuery: query,
		ExpandedTerms: expandedTerms,
		Synonyms:      s.groupSynonyms(similarTerms),
		RelatedTerms:  relatedTerms,
		ConceptTerms:  conceptTerms,
		ProcessedAt:   time.Now(),
	}, nil
}

// semanticRewriting performs semantic-based query rewriting
func (s *SemanticQueryEnhancer) semanticRewriting(ctx context.Context, query string) ([]string, error) {
	// Generate query embedding
	queryEmbedding, err := s.embeddingProvider.GetEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}
	
	// Find similar queries from history or database
	similarQueries, err := s.vectorDatabase.FindSimilarQueries(ctx, queryEmbedding, 5)
	if err != nil {
		return nil, err
	}
	
	// Generate rewrites based on similar queries
	var rewrites []string
	queryWords := strings.Fields(strings.ToLower(query))
	
	for _, similar := range similarQueries {
		if similar.Score > 0.7 { // High similarity threshold
			rewrite := s.generateRewriteFromSimilar(query, similar.Query, queryWords)
			if rewrite != "" && rewrite != query {
				rewrites = append(rewrites, rewrite)
			}
		}
	}
	
	// Limit number of rewrites
	if len(rewrites) > 3 {
		rewrites = rewrites[:3]
	}
	
	return rewrites, nil
}

// semanticIntentClassification classifies intent using semantic analysis
func (s *SemanticQueryEnhancer) semanticIntentClassification(ctx context.Context, query string) (*IntentClassification, error) {
	// Generate query embedding
	queryEmbedding, err := s.embeddingProvider.GetEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}
	
	// Compare with intent pattern embeddings
	intentScores := make(map[QueryIntent]float64)
	
	// This would typically use pre-computed intent embeddings
	// For now, use simple keyword-based classification
	return s.keywordBasedIntentClassification(query), nil
}

// Helper methods

func (s *SemanticQueryEnhancer) extractRelatedTerms(similarTerms []SimilarTerm) []string {
	var related []string
	for _, term := range similarTerms {
		if term.Score > 0.6 && term.Score < 0.9 { // Medium similarity
			related = append(related, term.Term)
		}
	}
	if len(related) > 5 {
		related = related[:5]
	}
	return related
}

func (s *SemanticQueryEnhancer) extractConceptTerms(similarTerms []SimilarTerm) []string {
	var concepts []string
	for _, term := range similarTerms {
		if term.Score > 0.4 && term.Score < 0.7 { // Lower similarity, broader concepts
			concepts = append(concepts, term.Term)
		}
	}
	if len(concepts) > 3 {
		concepts = concepts[:3]
	}
	return concepts
}

func (s *SemanticQueryEnhancer) groupSynonyms(similarTerms []SimilarTerm) map[string][]string {
	synonyms := make(map[string][]string)
	
	for _, term := range similarTerms {
		if term.Score > 0.8 { // High similarity indicates synonyms
			// Group by first letter for simplicity
			key := string(term.Term[0])
			synonyms[key] = append(synonyms[key], term.Term)
		}
	}
	
	return synonyms
}

func (s *SemanticQueryEnhancer) generateRewriteFromSimilar(original, similar string, originalWords []string) string {
	// Simple rewrite generation by combining terms
	similarWords := strings.Fields(strings.ToLower(similar))
	
	// Find unique words in similar query
	uniqueWords := make(map[string]bool)
	for _, word := range originalWords {
		uniqueWords[word] = true
	}
	
	var newWords []string
	for _, word := range similarWords {
		if !uniqueWords[word] && len(word) > 2 {
			newWords = append(newWords, word)
		}
	}
	
	if len(newWords) > 0 {
		return original + " " + strings.Join(newWords[:min(len(newWords), 2)], " ")
	}
	
	return ""
}

func (s *SemanticQueryEnhancer) keywordBasedIntentClassification(query string) *IntentClassification {
	query = strings.ToLower(query)
	
	// Simple keyword-based classification
	var primaryIntent QueryIntent = InformationSeeking
	var queryType QueryType = FactualQuery
	var complexity Complexity = SimpleComplexity
	confidence := 0.6
	
	// Intent classification
	if strings.Contains(query, "how to") || strings.Contains(query, "tutorial") {
		primaryIntent = Learning
		queryType = ProcedualQuery
		confidence = 0.8
	} else if strings.Contains(query, "compare") || strings.Contains(query, "vs") || strings.Contains(query, "versus") {
		primaryIntent = Comparison
		queryType = ComparativeQuery
		confidence = 0.8
	} else if strings.Contains(query, "recommend") || strings.Contains(query, "suggest") || strings.Contains(query, "best") {
		primaryIntent = Recommendation
		queryType = FactualQuery
		confidence = 0.7
	} else if strings.Contains(query, "solve") || strings.Contains(query, "fix") || strings.Contains(query, "troubleshoot") {
		primaryIntent = ProblemSolving
		queryType = ProcedualQuery
		confidence = 0.7
	}
	
	// Query type classification
	if strings.Contains(query, "what is") || strings.Contains(query, "define") {
		queryType = DefinitionQuery
	} else if strings.Contains(query, "why") || strings.Contains(query, "because") {
		queryType = CausalQuery
	} else if strings.Contains(query, "when") || strings.Contains(query, "time") {
		queryType = TemporalQuery
	} else if strings.Contains(query, "list") || strings.Contains(query, "types of") {
		queryType = ListQuery
	}
	
	// Complexity assessment
	wordCount := len(strings.Fields(query))
	if wordCount > 12 {
		complexity = HighComplexity
	} else if wordCount > 6 {
		complexity = ModerateComplexity
	}
	
	return &IntentClassification{
		PrimaryIntent: primaryIntent,
		Confidence:    confidence,
		QueryType:     queryType,
		Complexity:    complexity,
		Language:      "en",
		ClassifiedAt:  time.Now(),
	}
}

func (s *SemanticQueryEnhancer) calculateQualityScore(enhanced *EnhancedQuery) float64 {
	score := 0.5 // Base score
	
	// Add points for each enhancement technique applied
	if len(enhanced.RewrittenQueries) > 0 {
		score += 0.15
	}
	if len(enhanced.ExpandedTerms) > 0 {
		score += 0.2 // Higher weight for semantic expansion
	}
	if len(enhanced.SubQueries) > 0 {
		score += 0.1
	}
	if enhanced.Intent != nil {
		score += 0.15
	}
	
	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

func (s *SemanticQueryEnhancer) convertToExpandedTermStrings(terms []ExpandedTerm) []string {
	var strings []string
	for _, term := range terms {
		strings = append(strings, term.Term)
	}
	return strings
}

// HybridQueryEnhancer combines multiple enhancement strategies
type HybridQueryEnhancer struct {
	llmEnhancer      QueryEnhancer
	semanticEnhancer QueryEnhancer
	simpleEnhancer   QueryEnhancer
	config          *QueryEnhancementConfig
	cache           QueryEnhancementCache
}

// NewHybridQueryEnhancer creates a new hybrid query enhancer
func NewHybridQueryEnhancer(
	llmEnhancer QueryEnhancer,
	semanticEnhancer QueryEnhancer,
	simpleEnhancer QueryEnhancer,
	config *QueryEnhancementConfig,
	cache QueryEnhancementCache,
) *HybridQueryEnhancer {
	if config == nil {
		config = DefaultQueryEnhancementConfig()
	}
	
	return &HybridQueryEnhancer{
		llmEnhancer:      llmEnhancer,
		semanticEnhancer: semanticEnhancer,
		simpleEnhancer:   simpleEnhancer,
		config:          config,
		cache:           cache,
	}
}

// EnhanceQuery enhances query using multiple strategies
func (h *HybridQueryEnhancer) EnhanceQuery(ctx context.Context, query string, options *EnhancementOptions) (*EnhancedQuery, error) {
	if options == nil {
		options = h.config.DefaultOptions
	}
	
	startTime := time.Now()
	
	// Check cache first
	if h.cache != nil && h.config.CacheEnabled {
		if cached, err := h.cache.Get(ctx, query, options); err == nil && cached != nil {
			return cached, nil
		}
	}
	
	// Try enhancers in order of preference
	enhancers := []QueryEnhancer{h.llmEnhancer, h.semanticEnhancer, h.simpleEnhancer}
	
	var bestResult *EnhancedQuery
	var bestScore float64
	
	for _, enhancer := range enhancers {
		if enhancer == nil {
			continue
		}
		
		result, err := enhancer.EnhanceQuery(ctx, query, options)
		if err != nil {
			continue // Try next enhancer
		}
		
		// Use the enhancer with the best quality score
		if result.Enhancement.QualityScore > bestScore {
			bestResult = result
			bestScore = result.Enhancement.QualityScore
		}
	}
	
	// If no enhancer succeeded, create a basic enhanced query
	if bestResult == nil {
		bestResult = &EnhancedQuery{
			OriginalQuery: query,
			Enhancement: EnhancementSummary{
				TechniquesApplied: []string{"fallback"},
				QualityScore:      0.3,
				ProcessingTime:    time.Since(startTime),
			},
			ProcessedAt: time.Now(),
		}
	}
	
	// Update processing time
	bestResult.Enhancement.ProcessingTime = time.Since(startTime)
	
	// Cache the result
	if h.cache != nil && h.config.CacheEnabled {
		_ = h.cache.Set(ctx, query, options, bestResult, h.config.CacheTTL)
	}
	
	return bestResult, nil
}

// SimilarTerm represents a semantically similar term
type SimilarTerm struct {
	Term  string  `json:"term"`
	Score float64 `json:"score"`
}

// SimilarQuery represents a semantically similar query
type SimilarQuery struct {
	Query string  `json:"query"`
	Score float64 `json:"score"`
}

// VectorDatabaseProvider interface for vector database operations
type VectorDatabaseProvider interface {
	FindSimilarTerms(ctx context.Context, embedding []float64, topK int) ([]SimilarTerm, error)
	FindSimilarQueries(ctx context.Context, embedding []float64, topK int) ([]SimilarQuery, error)
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}