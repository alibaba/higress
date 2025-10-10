package queryenhancement

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LLMBasedQueryEnhancer implements query enhancement using LLM
type LLMBasedQueryEnhancer struct {
	llmProvider       LLMProvider
	embeddingProvider EmbeddingProvider
	config           *QueryEnhancementConfig
	cache            QueryEnhancementCache
}

// NewLLMBasedQueryEnhancer creates a new LLM-based query enhancer
func NewLLMBasedQueryEnhancer(
	llmProvider LLMProvider,
	embeddingProvider EmbeddingProvider,
	config *QueryEnhancementConfig,
	cache QueryEnhancementCache,
) *LLMBasedQueryEnhancer {
	if config == nil {
		config = DefaultQueryEnhancementConfig()
	}
	
	return &LLMBasedQueryEnhancer{
		llmProvider:       llmProvider,
		embeddingProvider: embeddingProvider,
		config:           config,
		cache:            cache,
	}
}

// EnhanceQuery improves the query for better retrieval
func (e *LLMBasedQueryEnhancer) EnhanceQuery(ctx context.Context, query string, options *EnhancementOptions) (*EnhancedQuery, error) {
	if options == nil {
		options = e.config.DefaultOptions
	}
	
	startTime := time.Now()
	
	// Check cache first
	if e.cache != nil && e.config.CacheEnabled {
		if cached, err := e.cache.Get(ctx, query, options); err == nil && cached != nil {
			return cached, nil
		}
	}
	
	enhanced := &EnhancedQuery{
		OriginalQuery: query,
		ProcessedAt:   time.Now(),
	}
	
	var techniques []string
	
	// Intent classification
	if options.EnableIntentClassification {
		intent, err := e.ClassifyIntent(ctx, query)
		if err == nil {
			enhanced.Intent = intent
			techniques = append(techniques, "intent_classification")
		}
	}
	
	// Query rewriting
	if options.EnableRewrite {
		rewrites, err := e.RewriteQuery(ctx, query)
		if err == nil {
			enhanced.RewrittenQueries = rewrites
			techniques = append(techniques, "query_rewrite")
		}
	}
	
	// Query expansion
	if options.EnableExpansion {
		expanded, err := e.ExpandQuery(ctx, query)
		if err == nil {
			enhanced.ExpandedTerms = e.extractExpandedTermStrings(expanded.ExpandedTerms)
			techniques = append(techniques, "query_expansion")
		}
	}
	
	// Query decomposition
	if options.EnableDecomposition {
		subQueries, err := e.DecomposeQuery(ctx, query)
		if err == nil {
			enhanced.SubQueries = subQueries
			techniques = append(techniques, "query_decomposition")
		}
	}
	
	// Create enhancement summary
	enhanced.Enhancement = EnhancementSummary{
		TechniquesApplied:  techniques,
		RewriteCount:       len(enhanced.RewrittenQueries),
		ExpansionCount:     len(enhanced.ExpandedTerms),
		DecompositionCount: len(enhanced.SubQueries),
		QualityScore:       e.calculateQualityScore(enhanced),
		ProcessingTime:     time.Since(startTime),
	}
	
	// Cache the result
	if e.cache != nil && e.config.CacheEnabled {
		_ = e.cache.Set(ctx, query, options, enhanced, e.config.CacheTTL)
	}
	
	return enhanced, nil
}

// RewriteQuery rewrites the query for better semantic matching
func (e *LLMBasedQueryEnhancer) RewriteQuery(ctx context.Context, query string) ([]string, error) {
	prompt := e.buildRewritePrompt(query)
	
	response, err := e.llmProvider.GenerateCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query rewrites: %w", err)
	}
	
	rewrites, err := e.parseRewriteResponse(response)
	if err != nil {
		return []string{}, nil // Return empty slice instead of error
	}
	
	return rewrites, nil
}

// ExpandQuery expands the query with related terms and synonyms
func (e *LLMBasedQueryEnhancer) ExpandQuery(ctx context.Context, query string) (*ExpandedQuery, error) {
	prompt := e.buildExpansionPrompt(query)
	
	response, err := e.llmProvider.GenerateCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query expansion: %w", err)
	}
	
	expanded, err := e.parseExpansionResponse(response, query)
	if err != nil {
		// Return basic expansion on parse error
		return &ExpandedQuery{
			OriginalQuery: query,
			ExpandedTerms: []ExpandedTerm{},
			Synonyms:      make(map[string][]string),
			RelatedTerms:  []string{},
			ConceptTerms:  []string{},
			ProcessedAt:   time.Now(),
		}, nil
	}
	
	return expanded, nil
}

// DecomposeQuery breaks down complex queries into sub-queries
func (e *LLMBasedQueryEnhancer) DecomposeQuery(ctx context.Context, query string) ([]SubQuery, error) {
	// First analyze complexity
	complexity, err := e.analyzeComplexity(ctx, query)
	if err != nil {
		return []SubQuery{}, nil
	}
	
	// Only decompose if complexity is moderate or high
	if complexity == SimpleComplexity {
		return []SubQuery{}, nil
	}
	
	prompt := e.buildDecompositionPrompt(query)
	
	response, err := e.llmProvider.GenerateCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query decomposition: %w", err)
	}
	
	subQueries, err := e.parseDecompositionResponse(response, query)
	if err != nil {
		return []SubQuery{}, nil
	}
	
	return subQueries, nil
}

// ClassifyIntent identifies the user's intent and query type
func (e *LLMBasedQueryEnhancer) ClassifyIntent(ctx context.Context, query string) (*IntentClassification, error) {
	prompt := e.buildIntentClassificationPrompt(query)
	
	response, err := e.llmProvider.GenerateCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to classify intent: %w", err)
	}
	
	classification, err := e.parseIntentResponse(response)
	if err != nil {
		// Return default classification on parse error
		return &IntentClassification{
			PrimaryIntent: InformationSeeking,
			Confidence:    0.5,
			QueryType:     FactualQuery,
			Complexity:    SimpleComplexity,
			Language:      "en",
			ClassifiedAt:  time.Now(),
		}, nil
	}
	
	return classification, nil
}

// buildRewritePrompt constructs prompt for query rewriting
func (e *LLMBasedQueryEnhancer) buildRewritePrompt(query string) string {
	return fmt.Sprintf(`Please rewrite the following query in 3 different ways to improve search results. 
Each rewrite should maintain the same meaning but use different phrasing, vocabulary, or structure.

Original Query: %s

Provide the rewrites in JSON format:
{
  "rewrites": ["rewrite1", "rewrite2", "rewrite3"]
}`, query)
}

// buildExpansionPrompt constructs prompt for query expansion
func (e *LLMBasedQueryEnhancer) buildExpansionPrompt(query string) string {
	return fmt.Sprintf(`Please expand the following query by providing synonyms, related terms, and conceptual terms that could help improve search results.

Query: %s

Provide the expansion in JSON format:
{
  "synonyms": {"term1": ["syn1", "syn2"], "term2": ["syn3", "syn4"]},
  "related_terms": ["related1", "related2", "related3"],
  "concept_terms": ["concept1", "concept2"]
}`, query)
}

// buildDecompositionPrompt constructs prompt for query decomposition
func (e *LLMBasedQueryEnhancer) buildDecompositionPrompt(query string) string {
	return fmt.Sprintf(`Please break down the following complex query into smaller, more focused sub-queries.
Each sub-query should address a specific aspect of the original question.

Original Query: %s

Provide the decomposition in JSON format:
{
  "sub_queries": [
    {
      "id": "sq1",
      "query": "sub-query 1",
      "type": "factual",
      "priority": 1,
      "keywords": ["keyword1", "keyword2"]
    }
  ]
}`, query)
}

// buildIntentClassificationPrompt constructs prompt for intent classification
func (e *LLMBasedQueryEnhancer) buildIntentClassificationPrompt(query string) string {
	return fmt.Sprintf(`Please analyze the following query and classify the user's intent and query characteristics.

Query: %s

Provide the classification in JSON format:
{
  "primary_intent": "information_seeking",
  "secondary_intent": "learning",
  "confidence": 0.9,
  "query_type": "factual",
  "domain": "technology",
  "complexity": "simple",
  "language": "en"
}

Intent options: information_seeking, problem_solving, learning, comparison, recommendation, navigation, verification, analysis
Query type options: factual, conceptual, comparative, procedural, list, definition, causal, temporal
Complexity options: simple, moderate, high`, query)
}

// parseRewriteResponse parses LLM response for query rewrites
func (e *LLMBasedQueryEnhancer) parseRewriteResponse(response string) ([]string, error) {
	// Extract JSON from response
	jsonStr := e.extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}
	
	var result struct {
		Rewrites []string `json:"rewrites"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	return result.Rewrites, nil
}

// parseExpansionResponse parses LLM response for query expansion
func (e *LLMBasedQueryEnhancer) parseExpansionResponse(response, originalQuery string) (*ExpandedQuery, error) {
	jsonStr := e.extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}
	
	var result struct {
		Synonyms     map[string][]string `json:"synonyms"`
		RelatedTerms []string           `json:"related_terms"`
		ConceptTerms []string           `json:"concept_terms"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	// Convert to ExpandedTerms
	var expandedTerms []ExpandedTerm
	
	// Add synonyms
	for term, synonyms := range result.Synonyms {
		for _, synonym := range synonyms {
			expandedTerms = append(expandedTerms, ExpandedTerm{
				Term:       synonym,
				Weight:     0.8,
				Source:     "synonym",
				Confidence: 0.8,
			})
		}
	}
	
	// Add related terms
	for _, term := range result.RelatedTerms {
		expandedTerms = append(expandedTerms, ExpandedTerm{
			Term:       term,
			Weight:     0.6,
			Source:     "related",
			Confidence: 0.7,
		})
	}
	
	// Add concept terms
	for _, conceptTerm := range result.ConceptTerms {
		expandedTerms = append(expandedTerms, ExpandedTerm{
			Term:       conceptTerm,
			Weight:     0.5,
			Source:     "concept",
			Confidence: 0.6,
		})
	}
	
	return &ExpandedQuery{
		OriginalQuery: originalQuery,
		ExpandedTerms: expandedTerms,
		Synonyms:      result.Synonyms,
		RelatedTerms:  result.RelatedTerms,
		ConceptTerms:  result.ConceptTerms,
		ProcessedAt:   time.Now(),
	}, nil
}

// parseDecompositionResponse parses LLM response for query decomposition
func (e *LLMBasedQueryEnhancer) parseDecompositionResponse(response, originalQuery string) ([]SubQuery, error) {
	jsonStr := e.extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}
	
	var result struct {
		SubQueries []struct {
			ID       string   `json:"id"`
			Query    string   `json:"query"`
			Type     string   `json:"type"`
			Priority int      `json:"priority"`
			Keywords []string `json:"keywords"`
		} `json:"sub_queries"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	var subQueries []SubQuery
	for _, sq := range result.SubQueries {
		subQuery := SubQuery{
			ID:       sq.ID,
			Query:    sq.Query,
			Type:     e.parseQueryType(sq.Type),
			Priority: sq.Priority,
			Keywords: sq.Keywords,
			Metadata: make(map[string]interface{}),
		}
		subQuery.Metadata["original_query"] = originalQuery
		subQueries = append(subQueries, subQuery)
	}
	
	return subQueries, nil
}

// parseIntentResponse parses LLM response for intent classification
func (e *LLMBasedQueryEnhancer) parseIntentResponse(response string) (*IntentClassification, error) {
	jsonStr := e.extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}
	
	var result struct {
		PrimaryIntent   string  `json:"primary_intent"`
		SecondaryIntent string  `json:"secondary_intent"`
		Confidence      float64 `json:"confidence"`
		QueryType       string  `json:"query_type"`
		Domain          string  `json:"domain"`
		Complexity      string  `json:"complexity"`
		Language        string  `json:"language"`
	}
	
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	return &IntentClassification{
		PrimaryIntent:   e.parseQueryIntent(result.PrimaryIntent),
		SecondaryIntent: e.parseQueryIntent(result.SecondaryIntent),
		Confidence:      result.Confidence,
		QueryType:       e.parseQueryType(result.QueryType),
		Domain:          result.Domain,
		Complexity:      e.parseComplexity(result.Complexity),
		Language:        result.Language,
		ClassifiedAt:    time.Now(),
	}, nil
}

// extractJSON extracts JSON from text response
func (e *LLMBasedQueryEnhancer) extractJSON(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	
	if start == -1 || end == -1 {
		return ""
	}
	
	return text[start : end+1]
}

// parseQueryIntent converts string to QueryIntent
func (e *LLMBasedQueryEnhancer) parseQueryIntent(intent string) QueryIntent {
	switch strings.ToLower(intent) {
	case "information_seeking":
		return InformationSeeking
	case "problem_solving":
		return ProblemSolving
	case "learning":
		return Learning
	case "comparison":
		return Comparison
	case "recommendation":
		return Recommendation
	case "navigation":
		return Navigation
	case "verification":
		return Verification
	case "analysis":
		return Analysis
	default:
		return InformationSeeking
	}
}

// parseQueryType converts string to QueryType
func (e *LLMBasedQueryEnhancer) parseQueryType(queryType string) QueryType {
	switch strings.ToLower(queryType) {
	case "factual":
		return FactualQuery
	case "conceptual":
		return ConceptualQuery
	case "comparative":
		return ComparativeQuery
	case "procedural":
		return ProcedualQuery
	case "list":
		return ListQuery
	case "definition":
		return DefinitionQuery
	case "causal":
		return CausalQuery
	case "temporal":
		return TemporalQuery
	default:
		return FactualQuery
	}
}

// parseComplexity converts string to Complexity
func (e *LLMBasedQueryEnhancer) parseComplexity(complexity string) Complexity {
	switch strings.ToLower(complexity) {
	case "simple":
		return SimpleComplexity
	case "moderate":
		return ModerateComplexity
	case "high":
		return HighComplexity
	default:
		return SimpleComplexity
	}
}

// analyzeComplexity determines query complexity
func (e *LLMBasedQueryEnhancer) analyzeComplexity(ctx context.Context, query string) (Complexity, error) {
	// Simple heuristic-based complexity analysis
	wordCount := len(strings.Fields(query))
	hasQuestionWords := strings.ContainsAny(strings.ToLower(query), "what when where why how")
	hasMultipleClauses := strings.Contains(query, " and ") || strings.Contains(query, " or ")
	
	if wordCount > 15 || hasMultipleClauses {
		return HighComplexity, nil
	} else if wordCount > 8 || hasQuestionWords {
		return ModerateComplexity, nil
	} else {
		return SimpleComplexity, nil
	}
}

// calculateQualityScore calculates quality score for enhanced query
func (e *LLMBasedQueryEnhancer) calculateQualityScore(enhanced *EnhancedQuery) float64 {
	score := 0.5 // Base score
	
	// Add points for each enhancement technique applied
	if len(enhanced.RewrittenQueries) > 0 {
		score += 0.15
	}
	if len(enhanced.ExpandedTerms) > 0 {
		score += 0.15
	}
	if len(enhanced.SubQueries) > 0 {
		score += 0.1
	}
	if enhanced.Intent != nil {
		score += 0.1
	}
	
	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

// extractExpandedTermStrings extracts term strings from ExpandedTerm slice
func (e *LLMBasedQueryEnhancer) extractExpandedTermStrings(terms []ExpandedTerm) []string {
	var strings []string
	for _, term := range terms {
		strings = append(strings, term.Term)
	}
	return strings
}