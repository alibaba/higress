package queryenhancement

import (
	"context"
	"time"
)

// QueryEnhancer defines the interface for query enhancement
type QueryEnhancer interface {
	// EnhanceQuery improves the query for better retrieval
	EnhanceQuery(ctx context.Context, query string, options *EnhancementOptions) (*EnhancedQuery, error)
	
	// RewriteQuery rewrites the query for better semantic matching
	RewriteQuery(ctx context.Context, query string) ([]string, error)
	
	// ExpandQuery expands the query with related terms and synonyms
	ExpandQuery(ctx context.Context, query string) (*ExpandedQuery, error)
	
	// DecomposeQuery breaks down complex queries into sub-queries
	DecomposeQuery(ctx context.Context, query string) ([]SubQuery, error)
	
	// ClassifyIntent identifies the user's intent and query type
	ClassifyIntent(ctx context.Context, query string) (*IntentClassification, error)
}

// EnhancedQuery represents an enhanced version of the original query
type EnhancedQuery struct {
	OriginalQuery     string                 `json:"original_query"`
	RewrittenQueries  []string               `json:"rewritten_queries"`
	ExpandedTerms     []string               `json:"expanded_terms"`
	SubQueries        []SubQuery             `json:"sub_queries"`
	Intent            *IntentClassification  `json:"intent"`
	Enhancement       EnhancementSummary     `json:"enhancement"`
	ProcessedAt       time.Time              `json:"processed_at"`
}

// ExpandedQuery represents a query with expanded terms
type ExpandedQuery struct {
	OriginalQuery string            `json:"original_query"`
	ExpandedTerms []ExpandedTerm    `json:"expanded_terms"`
	Synonyms      map[string][]string `json:"synonyms"`
	RelatedTerms  []string          `json:"related_terms"`
	ConceptTerms  []string          `json:"concept_terms"`
	ProcessedAt   time.Time         `json:"processed_at"`
}

// ExpandedTerm represents an expanded term with metadata
type ExpandedTerm struct {
	Term       string  `json:"term"`
	Weight     float64 `json:"weight"`
	Source     string  `json:"source"`     // "synonym", "related", "concept"
	Confidence float64 `json:"confidence"`
}

// SubQuery represents a decomposed sub-query
type SubQuery struct {
	Query       string                 `json:"query"`
	Type        QueryType              `json:"type"`
	Priority    int                    `json:"priority"`
	Dependency  []string               `json:"dependency,omitempty"` // IDs of dependent sub-queries
	Keywords    []string               `json:"keywords"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ID          string                 `json:"id"`
}

// QueryType defines different types of queries
type QueryType int

const (
	// FactualQuery asks for specific facts
	FactualQuery QueryType = iota
	// ConceptualQuery asks for explanations or concepts
	ConceptualQuery
	// ComparativeQuery compares multiple items
	ComparativeQuery
	// ProcedualQuery asks for how-to information
	ProcedualQuery
	// ListQuery asks for lists or enumerations
	ListQuery
	// DefinitionQuery asks for definitions
	DefinitionQuery
	// CausalQuery asks about causes and effects
	CausalQuery
	// TemporalQuery involves time-based information
	TemporalQuery
)

func (q QueryType) String() string {
	switch q {
	case FactualQuery:
		return "factual"
	case ConceptualQuery:
		return "conceptual"
	case ComparativeQuery:
		return "comparative"
	case ProcedualQuery:
		return "procedural"
	case ListQuery:
		return "list"
	case DefinitionQuery:
		return "definition"
	case CausalQuery:
		return "causal"
	case TemporalQuery:
		return "temporal"
	default:
		return "unknown"
	}
}

// IntentClassification represents the classified intent of a query
type IntentClassification struct {
	PrimaryIntent   QueryIntent `json:"primary_intent"`
	SecondaryIntent QueryIntent `json:"secondary_intent,omitempty"`
	Confidence      float64     `json:"confidence"`
	QueryType       QueryType   `json:"query_type"`
	Domain          string      `json:"domain,omitempty"`
	Complexity      Complexity  `json:"complexity"`
	Language        string      `json:"language"`
	ClassifiedAt    time.Time   `json:"classified_at"`
}

// QueryIntent defines the user's intent
type QueryIntent int

const (
	// InformationSeeking seeks factual information
	InformationSeeking QueryIntent = iota
	// ProblemSolving seeks solutions to problems
	ProblemSolving
	// Learning seeks educational content
	Learning
	// Comparison seeks to compare options
	Comparison
	// Recommendation seeks suggestions
	Recommendation
	// Navigation seeks to find specific content
	Navigation
	// Verification seeks to verify information
	Verification
	// Analysis seeks analytical insights
	Analysis
)

func (i QueryIntent) String() string {
	switch i {
	case InformationSeeking:
		return "information_seeking"
	case ProblemSolving:
		return "problem_solving"
	case Learning:
		return "learning"
	case Comparison:
		return "comparison"
	case Recommendation:
		return "recommendation"
	case Navigation:
		return "navigation"
	case Verification:
		return "verification"
	case Analysis:
		return "analysis"
	default:
		return "unknown"
	}
}

// Complexity defines query complexity levels
type Complexity int

const (
	SimpleComplexity Complexity = iota
	ModerateComplexity
	HighComplexity
)

func (c Complexity) String() string {
	switch c {
	case SimpleComplexity:
		return "simple"
	case ModerateComplexity:
		return "moderate"
	case HighComplexity:
		return "high"
	default:
		return "unknown"
	}
}

// EnhancementSummary provides a summary of applied enhancements
type EnhancementSummary struct {
	TechniquesApplied []string `json:"techniques_applied"`
	RewriteCount      int      `json:"rewrite_count"`
	ExpansionCount    int      `json:"expansion_count"`
	DecompositionCount int     `json:"decomposition_count"`
	QualityScore      float64  `json:"quality_score"`
	ProcessingTime    time.Duration `json:"processing_time"`
}

// EnhancementOptions configures query enhancement behavior
type EnhancementOptions struct {
	// Enable specific enhancement techniques
	EnableRewrite      bool `json:"enable_rewrite"`
	EnableExpansion    bool `json:"enable_expansion"`
	EnableDecomposition bool `json:"enable_decomposition"`
	EnableIntentClassification bool `json:"enable_intent_classification"`
	
	// Rewrite options
	MaxRewrites int     `json:"max_rewrites"`
	RewriteScore float64 `json:"rewrite_score_threshold"`
	
	// Expansion options
	MaxExpansions     int     `json:"max_expansions"`
	ExpansionWeight   float64 `json:"expansion_weight"`
	IncludeSynonyms   bool    `json:"include_synonyms"`
	IncludeRelated    bool    `json:"include_related"`
	IncludeConcepts   bool    `json:"include_concepts"`
	
	// Decomposition options
	MaxSubQueries     int     `json:"max_sub_queries"`
	ComplexityThreshold Complexity `json:"complexity_threshold"`
	
	// Language settings
	Language string `json:"language"`
	
	// Provider settings
	LLMProvider    string `json:"llm_provider,omitempty"`
	EmbeddingProvider string `json:"embedding_provider,omitempty"`
}

// DefaultEnhancementOptions returns default enhancement options
func DefaultEnhancementOptions() *EnhancementOptions {
	return &EnhancementOptions{
		EnableRewrite:      true,
		EnableExpansion:    true,
		EnableDecomposition: true,
		EnableIntentClassification: true,
		MaxRewrites:        3,
		RewriteScore:       0.7,
		MaxExpansions:      5,
		ExpansionWeight:    0.5,
		IncludeSynonyms:    true,
		IncludeRelated:     true,
		IncludeConcepts:    false,
		MaxSubQueries:      3,
		ComplexityThreshold: ModerateComplexity,
		Language:           "en",
	}
}

// QueryRewriter defines interface for query rewriting
type QueryRewriter interface {
	// RewriteQuery generates alternative phrasings of the query
	RewriteQuery(ctx context.Context, query string, maxRewrites int) ([]string, error)
	
	// ParaphraseQuery creates paraphrased versions
	ParaphraseQuery(ctx context.Context, query string) ([]string, error)
}

// QueryExpander defines interface for query expansion
type QueryExpander interface {
	// ExpandWithSynonyms adds synonyms to the query
	ExpandWithSynonyms(ctx context.Context, query string) ([]ExpandedTerm, error)
	
	// ExpandWithRelatedTerms adds related terms
	ExpandWithRelatedTerms(ctx context.Context, query string) ([]ExpandedTerm, error)
	
	// ExpandWithConcepts adds conceptual terms
	ExpandWithConcepts(ctx context.Context, query string) ([]ExpandedTerm, error)
}

// QueryDecomposer defines interface for query decomposition
type QueryDecomposer interface {
	// DecomposeQuery breaks down complex queries
	DecomposeQuery(ctx context.Context, query string, maxSubQueries int) ([]SubQuery, error)
	
	// AnalyzeComplexity determines query complexity
	AnalyzeComplexity(ctx context.Context, query string) (Complexity, error)
}

// IntentClassifier defines interface for intent classification
type IntentClassifier interface {
	// ClassifyIntent determines the user's intent
	ClassifyIntent(ctx context.Context, query string) (*IntentClassification, error)
	
	// ClassifyQueryType determines the type of query
	ClassifyQueryType(ctx context.Context, query string) (QueryType, error)
}

// LLMProvider defines interface for LLM integration
type LLMProvider interface {
	// GenerateCompletion generates text completion
	GenerateCompletion(ctx context.Context, prompt string) (string, error)
	
	// GenerateStructuredResponse generates structured response
	GenerateStructuredResponse(ctx context.Context, prompt string, schema interface{}) (interface{}, error)
}

// EmbeddingProvider defines interface for embedding generation
type EmbeddingProvider interface {
	// GetEmbedding generates embedding for text
	GetEmbedding(ctx context.Context, text string) ([]float64, error)
	
	// GetSimilarity calculates similarity between texts
	GetSimilarity(ctx context.Context, text1, text2 string) (float64, error)
}

// QueryEnhancementConfig contains configuration for query enhancement
type QueryEnhancementConfig struct {
	// Provider configurations
	LLMConfig       LLMConfig       `json:"llm_config"`
	EmbeddingConfig EmbeddingConfig `json:"embedding_config"`
	
	// Enhancement settings
	DefaultOptions *EnhancementOptions `json:"default_options"`
	
	// Performance settings
	CacheEnabled    bool          `json:"cache_enabled"`
	CacheTTL        time.Duration `json:"cache_ttl"`
	MaxConcurrency  int           `json:"max_concurrency"`
	RequestTimeout  time.Duration `json:"request_timeout"`
	
	// Quality thresholds
	MinConfidenceScore   float64 `json:"min_confidence_score"`
	MaxProcessingTime    time.Duration `json:"max_processing_time"`
}

// LLMConfig contains LLM provider configuration
type LLMConfig struct {
	Provider    string                 `json:"provider"`
	APIKey      string                 `json:"api_key"`
	BaseURL     string                 `json:"base_url,omitempty"`
	Model       string                 `json:"model"`
	Temperature float64                `json:"temperature"`
	MaxTokens   int                    `json:"max_tokens"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// EmbeddingConfig contains embedding provider configuration
type EmbeddingConfig struct {
	Provider   string                 `json:"provider"`
	APIKey     string                 `json:"api_key"`
	BaseURL    string                 `json:"base_url,omitempty"`
	Model      string                 `json:"model"`
	Dimensions int                    `json:"dimensions"`
	Settings   map[string]interface{} `json:"settings,omitempty"`
}

// DefaultQueryEnhancementConfig returns default configuration
func DefaultQueryEnhancementConfig() *QueryEnhancementConfig {
	return &QueryEnhancementConfig{
		DefaultOptions:       DefaultEnhancementOptions(),
		CacheEnabled:         true,
		CacheTTL:            1 * time.Hour,
		MaxConcurrency:      5,
		RequestTimeout:      30 * time.Second,
		MinConfidenceScore:  0.6,
		MaxProcessingTime:   10 * time.Second,
	}
}

// QueryEnhancementResult contains the result of query enhancement
type QueryEnhancementResult struct {
	Success     bool           `json:"success"`
	Enhanced    *EnhancedQuery `json:"enhanced_query,omitempty"`
	Error       string         `json:"error,omitempty"`
	Performance PerformanceMetrics `json:"performance"`
}

// PerformanceMetrics contains performance information
type PerformanceMetrics struct {
	ProcessingTime time.Duration `json:"processing_time"`
	CacheHit       bool          `json:"cache_hit"`
	LLMCalls       int           `json:"llm_calls"`
	EmbeddingCalls int           `json:"embedding_calls"`
	TokensUsed     int           `json:"tokens_used"`
}

// QueryEnhancementCache defines interface for caching enhancement results
type QueryEnhancementCache interface {
	// Get retrieves cached enhancement result
	Get(ctx context.Context, query string, options *EnhancementOptions) (*EnhancedQuery, error)
	
	// Set stores enhancement result in cache
	Set(ctx context.Context, query string, options *EnhancementOptions, result *EnhancedQuery, ttl time.Duration) error
	
	// Delete removes cached result
	Delete(ctx context.Context, query string, options *EnhancementOptions) error
	
	// Clear clears all cached results
	Clear(ctx context.Context) error
}