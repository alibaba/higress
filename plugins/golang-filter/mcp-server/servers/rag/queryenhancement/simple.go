package queryenhancement

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
	"time"
)

// SimpleQueryEnhancer provides basic query enhancement without LLM
type SimpleQueryEnhancer struct {
	config     *QueryEnhancementConfig
	cache      QueryEnhancementCache
}

// NewSimpleQueryEnhancer creates a new simple query enhancer
func NewSimpleQueryEnhancer(config *QueryEnhancementConfig) *SimpleQueryEnhancer {
	if config == nil {
		config = DefaultQueryEnhancementConfig()
	}
	
	var cache QueryEnhancementCache
	if config.CacheEnabled {
		cache = NewMemoryQueryCache(config.CacheTTL, 1000) // Default cache size
	}
	
	return &SimpleQueryEnhancer{
		config: config,
		cache:  cache,
	}
}

// EnhanceQuery performs basic query enhancement
func (e *SimpleQueryEnhancer) EnhanceQuery(ctx context.Context, query string, options *EnhancementOptions) (*EnhancedQuery, error) {
	if options == nil {
		options = e.config.DefaultOptions
	}
	
	startTime := time.Now()
	
	// Check cache first
	if e.cache != nil {
		if cached, err := e.cache.Get(ctx, query, options); err == nil && cached != nil {
			return cached, nil
		}
	}
	
	enhanced := &EnhancedQuery{
		OriginalQuery: query,
		ProcessedAt:   time.Now(),
	}
	
	var techniques []string
	
	// Basic intent classification
	if options.EnableIntentClassification {
		intent := e.simpleIntentClassification(query)
		enhanced.Intent = intent
		techniques = append(techniques, "simple_intent_classification")
	}
	
	// Simple query rewriting
	if options.EnableRewrite {
		rewrites := e.simpleQueryRewrite(query, options.MaxRewrites)
		enhanced.RewrittenQueries = rewrites
		if len(rewrites) > 0 {
			techniques = append(techniques, "simple_rewrite")
		}
	}
	
	// Basic query expansion
	if options.EnableExpansion {
		expanded := e.simpleQueryExpansion(query, options)
		enhanced.ExpandedTerms = expanded
		if len(expanded) > 0 {
			techniques = append(techniques, "simple_expansion")
		}
	}
	
	// Simple decomposition for obviously complex queries
	if options.EnableDecomposition {
		subQueries := e.simpleQueryDecomposition(query, options.MaxSubQueries)
		enhanced.SubQueries = subQueries
		if len(subQueries) > 0 {
			techniques = append(techniques, "simple_decomposition")
		}
	}
	
	// Create enhancement summary
	enhanced.Enhancement = EnhancementSummary{
		TechniquesApplied:  techniques,
		RewriteCount:       len(enhanced.RewrittenQueries),
		ExpansionCount:     len(enhanced.ExpandedTerms),
		DecompositionCount: len(enhanced.SubQueries),
		QualityScore:       e.calculateSimpleQualityScore(enhanced),
		ProcessingTime:     time.Since(startTime),
	}
	
	// Cache the result
	if e.cache != nil {
		_ = e.cache.Set(ctx, query, options, enhanced, e.config.CacheTTL)
	}
	
	return enhanced, nil
}

// RewriteQuery performs simple query rewriting
func (e *SimpleQueryEnhancer) RewriteQuery(ctx context.Context, query string) ([]string, error) {
	return e.simpleQueryRewrite(query, 3), nil
}

// ExpandQuery performs simple query expansion
func (e *SimpleQueryEnhancer) ExpandQuery(ctx context.Context, query string) (*ExpandedQuery, error) {
	options := e.config.DefaultOptions
	expandedTerms := e.simpleQueryExpansion(query, options)
	
	// Convert to ExpandedQuery format
	var expandedTermsList []ExpandedTerm
	for _, term := range expandedTerms {
		expandedTermsList = append(expandedTermsList, ExpandedTerm{
			Term:       term,
			Weight:     0.5,
			Source:     "simple",
			Confidence: 0.6,
		})
	}
	
	return &ExpandedQuery{
		OriginalQuery: query,
		ExpandedTerms: expandedTermsList,
		ProcessedAt:   time.Now(),
	}, nil
}

// DecomposeQuery performs simple query decomposition
func (e *SimpleQueryEnhancer) DecomposeQuery(ctx context.Context, query string) ([]SubQuery, error) {
	subQueries := e.simpleQueryDecomposition(query, 3)
	return subQueries, nil
}

// ClassifyIntent performs simple intent classification
func (e *SimpleQueryEnhancer) ClassifyIntent(ctx context.Context, query string) (*IntentClassification, error) {
	return e.simpleIntentClassification(query), nil
}

// Simple enhancement methods

func (e *SimpleQueryEnhancer) simpleIntentClassification(query string) *IntentClassification {
	lowerQuery := strings.ToLower(query)
	
	// Basic pattern matching for intent classification
	var intent QueryIntent
	var queryType QueryType
	var complexity Complexity
	
	// Determine intent based on keywords
	if strings.Contains(lowerQuery, "how to") || strings.Contains(lowerQuery, "how do") {
		intent = ProblemSolving
		queryType = ProcedualQuery
	} else if strings.Contains(lowerQuery, "what is") || strings.Contains(lowerQuery, "define") {
		intent = InformationSeeking
		queryType = DefinitionQuery
	} else if strings.Contains(lowerQuery, "compare") || strings.Contains(lowerQuery, "vs") || strings.Contains(lowerQuery, "versus") {
		intent = Comparison
		queryType = ComparativeQuery
	} else if strings.Contains(lowerQuery, "list") || strings.Contains(lowerQuery, "examples") {
		intent = InformationSeeking
		queryType = ListQuery
	} else if strings.Contains(lowerQuery, "why") || strings.Contains(lowerQuery, "because") {
		intent = Learning
		queryType = CausalQuery
	} else if strings.Contains(lowerQuery, "when") || strings.Contains(lowerQuery, "time") {
		intent = InformationSeeking
		queryType = TemporalQuery
	} else {
		intent = InformationSeeking
		queryType = FactualQuery
	}
	
	// Determine complexity based on query length and structure
	wordCount := len(strings.Fields(query))
	hasMultipleClauses := strings.Contains(query, " and ") || strings.Contains(query, " or ") || strings.Contains(query, ",")
	
	if wordCount > 15 || hasMultipleClauses {
		complexity = HighComplexity
	} else if wordCount > 8 {
		complexity = ModerateComplexity
	} else {
		complexity = SimpleComplexity
	}
	
	return &IntentClassification{
		PrimaryIntent: intent,
		Confidence:    0.6, // Lower confidence for simple classification
		QueryType:     queryType,
		Complexity:    complexity,
		Language:      "en",
		ClassifiedAt:  time.Now(),
	}
}

func (e *SimpleQueryEnhancer) simpleQueryRewrite(query string, maxRewrites int) []string {
	var rewrites []string
	lowerQuery := strings.ToLower(query)
	
	// Simple rewriting patterns
	patterns := map[string]string{
		"how to": "how do I",
		"what is": "define",
		"can you": "please",
		"i want to": "how to",
		"show me": "find",
		"tell me about": "explain",
	}
	
	for pattern, replacement := range patterns {
		if strings.Contains(lowerQuery, pattern) && len(rewrites) < maxRewrites {
			rewritten := strings.ReplaceAll(query, pattern, replacement)
			if rewritten != query {
				rewrites = append(rewrites, rewritten)
			}
		}
	}
	
	// Add question variations
	if !strings.HasSuffix(query, "?") && len(rewrites) < maxRewrites {
		rewrites = append(rewrites, query+"?")
	}
	
	// Add more detailed version
	if len(strings.Fields(query)) < 5 && len(rewrites) < maxRewrites {
		rewrites = append(rewrites, "Please provide detailed information about "+query)
	}
	
	return rewrites
}

func (e *SimpleQueryEnhancer) simpleQueryExpansion(query string, options *EnhancementOptions) []string {
	var expanded []string
	words := strings.Fields(strings.ToLower(query))
	
	// Simple synonym mapping
	synonyms := map[string][]string{
		"fast":    {"quick", "rapid", "speedy"},
		"big":     {"large", "huge", "massive"},
		"small":   {"tiny", "little", "mini"},
		"good":    {"excellent", "great", "fine"},
		"bad":     {"poor", "terrible", "awful"},
		"help":    {"assist", "support", "aid"},
		"find":    {"locate", "discover", "search"},
		"make":    {"create", "build", "construct"},
		"use":     {"utilize", "employ", "apply"},
		"show":    {"display", "demonstrate", "present"},
		"get":     {"obtain", "acquire", "retrieve"},
		"do":      {"perform", "execute", "accomplish"},
		"say":     {"tell", "express", "state"},
		"go":      {"move", "travel", "proceed"},
		"come":    {"arrive", "approach", "reach"},
		"know":    {"understand", "realize", "comprehend"},
		"think":   {"believe", "consider", "suppose"},
		"see":     {"view", "observe", "notice"},
		"look":    {"examine", "inspect", "check"},
		"work":    {"function", "operate", "perform"},
		"way":     {"method", "approach", "technique"},
		"time":    {"period", "duration", "moment"},
		"place":   {"location", "position", "spot"},
		"thing":   {"item", "object", "element"},
		"person":  {"individual", "human", "people"},
		"part":    {"component", "section", "piece"},
		"problem": {"issue", "difficulty", "challenge"},
		"question": {"inquiry", "query", "ask"},
		"answer":  {"response", "reply", "solution"},
		"example": {"instance", "sample", "case"},
		"reason":  {"cause", "explanation", "purpose"},
		"result":  {"outcome", "consequence", "effect"},
	}
	
	// Add synonyms for words in the query
	for _, word := range words {
		if syns, exists := synonyms[word]; exists && len(expanded) < options.MaxExpansions {
			for _, syn := range syns {
				if len(expanded) < options.MaxExpansions {
					expanded = append(expanded, syn)
				}
			}
		}
	}
	
	// Add related terms based on context
	if strings.Contains(query, "technology") || strings.Contains(query, "computer") {
		contextTerms := []string{"software", "hardware", "digital", "programming"}
		for _, term := range contextTerms {
			if len(expanded) < options.MaxExpansions {
				expanded = append(expanded, term)
			}
		}
	}
	
	return expanded
}

func (e *SimpleQueryEnhancer) simpleQueryDecomposition(query string, maxSubQueries int) []SubQuery {
	var subQueries []SubQuery
	
	// Only decompose if query contains conjunctions or is long
	wordCount := len(strings.Fields(query))
	hasConjunctions := strings.Contains(query, " and ") || strings.Contains(query, " or ")
	
	if wordCount < 8 && !hasConjunctions {
		return subQueries // No decomposition needed
	}
	
	// Split on "and" for simple decomposition
	if strings.Contains(query, " and ") {
		parts := strings.Split(query, " and ")
		for i, part := range parts {
			if len(subQueries) >= maxSubQueries {
				break
			}
			
			cleanPart := strings.TrimSpace(part)
			if len(cleanPart) > 3 {
				subQuery := SubQuery{
					ID:       fmt.Sprintf("sq%d", i+1),
					Query:    cleanPart,
					Type:     FactualQuery,
					Priority: i + 1,
					Keywords: strings.Fields(strings.ToLower(cleanPart)),
					Metadata: make(map[string]interface{}),
				}
				subQuery.Metadata["source"] = "simple_decomposition"
				subQueries = append(subQueries, subQuery)
			}
		}
	}
	
	// If still no sub-queries and query is very long, try to split into logical parts
	if len(subQueries) == 0 && wordCount > 12 {
		// Try to split at question words
		questionWords := []string{"what", "how", "why", "when", "where", "who"}
		for _, qw := range questionWords {
			if strings.Contains(strings.ToLower(query), qw) && len(subQueries) < maxSubQueries {
				// Create a focused sub-query
				subQuery := SubQuery{
					ID:       fmt.Sprintf("sq%d", len(subQueries)+1),
					Query:    fmt.Sprintf("%s about %s", qw, extractKeyTerms(query)),
					Type:     FactualQuery,
					Priority: len(subQueries) + 1,
					Keywords: []string{qw, extractKeyTerms(query)},
					Metadata: make(map[string]interface{}),
				}
				subQuery.Metadata["source"] = "simple_decomposition"
				subQueries = append(subQueries, subQuery)
				break
			}
		}
	}
	
	return subQueries
}

func (e *SimpleQueryEnhancer) calculateSimpleQualityScore(enhanced *EnhancedQuery) float64 {
	score := 0.3 // Base score for simple enhancement
	
	// Add points for each enhancement applied
	if len(enhanced.RewrittenQueries) > 0 {
		score += 0.2
	}
	if len(enhanced.ExpandedTerms) > 0 {
		score += 0.2
	}
	if len(enhanced.SubQueries) > 0 {
		score += 0.15
	}
	if enhanced.Intent != nil {
		score += 0.15
	}
	
	return score
}

// Helper functions

func extractKeyTerms(query string) string {
	words := strings.Fields(strings.ToLower(query))
	
	// Filter out common stop words
	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
		"be": true, "by": true, "for": true, "from": true, "has": true, "he": true,
		"in": true, "is": true, "it": true, "its": true, "of": true, "on": true,
		"that": true, "the": true, "to": true, "was": true, "will": true, "with": true,
		"how": true, "what": true, "when": true, "where": true, "why": true, "who": true,
	}
	
	var keyTerms []string
	for _, word := range words {
		if !stopWords[word] && len(word) > 2 {
			keyTerms = append(keyTerms, word)
		}
	}
	
	if len(keyTerms) > 0 {
		return keyTerms[0] // Return the first key term
	}
	
	return "information"
}

// MemoryQueryCache implements an in-memory cache for query enhancement results
type MemoryQueryCache struct {
	cache   map[string]*CacheEntry
	ttl     time.Duration
	maxSize int
	stats   *CacheStats
}

// CacheEntry represents a cached query enhancement result
type CacheEntry struct {
	Result    *EnhancedQuery
	ExpiresAt time.Time
}

// NewMemoryQueryCache creates a new in-memory query cache
func NewMemoryQueryCache(ttl time.Duration, maxSize int) *MemoryQueryCache {
	return &MemoryQueryCache{
		cache:   make(map[string]*CacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
		stats: &CacheStats{
			LastUpdate: time.Now(),
		},
	}
}

// Get retrieves a cached enhancement result
func (c *MemoryQueryCache) Get(ctx context.Context, query string, options *EnhancementOptions) (*EnhancedQuery, error) {
	key := c.generateKey(query, options)
	
	entry, exists := c.cache[key]
	if !exists {
		c.stats.TotalMisses++
		c.updateMissRate()
		return nil, fmt.Errorf("cache miss")
	}
	
	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		delete(c.cache, key)
		c.stats.TotalMisses++
		c.updateMissRate()
		return nil, fmt.Errorf("cache expired")
	}
	
	c.stats.TotalHits++
	c.updateHitRate()
	return entry.Result, nil
}

// Set stores an enhancement result in cache
func (c *MemoryQueryCache) Set(ctx context.Context, query string, options *EnhancementOptions, result *EnhancedQuery, ttl time.Duration) error {
	key := c.generateKey(query, options)
	
	// Check if cache is full
	if len(c.cache) >= c.maxSize {
		c.evictOldest()
	}
	
	c.cache[key] = &CacheEntry{
		Result:    result,
		ExpiresAt: time.Now().Add(ttl),
	}
	
	c.stats.CacheSize = len(c.cache)
	c.stats.LastUpdate = time.Now()
	
	return nil
}

// Delete removes a cached result
func (c *MemoryQueryCache) Delete(ctx context.Context, query string, options *EnhancementOptions) error {
	key := c.generateKey(query, options)
	delete(c.cache, key)
	c.stats.CacheSize = len(c.cache)
	return nil
}

// Clear clears all cached results
func (c *MemoryQueryCache) Clear(ctx context.Context) error {
	c.cache = make(map[string]*CacheEntry)
	c.stats.CacheSize = 0
	return nil
}

// GetStats returns cache statistics
func (c *MemoryQueryCache) GetStats() *CacheStats {
	return c.stats
}

// generateKey generates a cache key from query and options
func (c *MemoryQueryCache) generateKey(query string, options *EnhancementOptions) string {
	h := fnv.New64a()
	h.Write([]byte(query))
	if options != nil {
		h.Write([]byte(fmt.Sprintf("%+v", options)))
	}
	return fmt.Sprintf("%x", h.Sum64())
}

// evictOldest removes the oldest cache entry
func (c *MemoryQueryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range c.cache {
		if oldestKey == "" || entry.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.ExpiresAt
		}
	}
	
	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}

// updateHitRate updates the cache hit rate
func (c *MemoryQueryCache) updateHitRate() {
	total := c.stats.TotalHits + c.stats.TotalMisses
	if total > 0 {
		c.stats.HitRate = float64(c.stats.TotalHits) / float64(total)
	}
}

// updateMissRate updates the cache miss rate
func (c *MemoryQueryCache) updateMissRate() {
	total := c.stats.TotalHits + c.stats.TotalMisses
	if total > 0 {
		c.stats.MissRate = float64(c.stats.TotalMisses) / float64(total)
	}
}