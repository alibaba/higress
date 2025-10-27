package postprocessing

import (
	"context"
	"time"
)

// PostProcessor defines the interface for retrieval result post-processing
type PostProcessor interface {
	// ProcessResults applies post-processing to search results
	ProcessResults(ctx context.Context, query string, results []SearchResult, options *ProcessingOptions) (*ProcessedResults, error)
	
	// RerankResults reorders results based on relevance and quality
	RerankResults(ctx context.Context, query string, results []SearchResult, options *RerankingOptions) ([]SearchResult, error)
	
	// FilterResults filters results based on criteria
	FilterResults(ctx context.Context, query string, results []SearchResult, options *FilteringOptions) ([]SearchResult, error)
	
	// CompressContext reduces the amount of context while preserving relevance
	CompressContext(ctx context.Context, query string, results []SearchResult, options *CompressionOptions) (*CompressedContext, error)
	
	// DeduplicateResults removes duplicate or very similar results
	DeduplicateResults(ctx context.Context, results []SearchResult, options *DeduplicationOptions) ([]SearchResult, error)
}

// SearchResult represents a search result to be post-processed
type SearchResult struct {
	ID          string                 `json:"id"`
	Content     string                 `json:"content"`
	Title       string                 `json:"title,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Score       float64                `json:"score"`
	Rank        int                    `json:"rank"`
	Source      string                 `json:"source"`
	Method      string                 `json:"method"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RetrievedAt time.Time              `json:"retrieved_at"`
}

// ProcessedResults contains the final processed results
type ProcessedResults struct {
	Query             string              `json:"query"`
	OriginalCount     int                 `json:"original_count"`
	FinalResults      []SearchResult      `json:"final_results"`
	CompressedContext *CompressedContext  `json:"compressed_context,omitempty"`
	ProcessingSummary ProcessingSummary   `json:"processing_summary"`
	ProcessedAt       time.Time           `json:"processed_at"`
}

// CompressedContext represents compressed context information
type CompressedContext struct {
	Summary           string            `json:"summary"`
	KeyPoints         []string          `json:"key_points"`
	RelevantSections  []ContextSection  `json:"relevant_sections"`
	OriginalLength    int               `json:"original_length"`
	CompressedLength  int               `json:"compressed_length"`
	CompressionRatio  float64           `json:"compression_ratio"`
	QualityScore      float64           `json:"quality_score"`
}

// ContextSection represents a section of relevant context
type ContextSection struct {
	Content     string  `json:"content"`
	Source      string  `json:"source"`
	Relevance   float64 `json:"relevance"`
	Position    int     `json:"position"`
	Length      int     `json:"length"`
}

// ProcessingSummary provides summary of applied processing
type ProcessingSummary struct {
	TechniquesApplied  []string      `json:"techniques_applied"`
	OriginalCount      int           `json:"original_count"`
	FilteredCount      int           `json:"filtered_count"`
	DeduplicatedCount  int           `json:"deduplicated_count"`
	RerankedCount      int           `json:"reranked_count"`
	FinalCount         int           `json:"final_count"`
	ProcessingTime     time.Duration `json:"processing_time"`
	QualityImprovement float64       `json:"quality_improvement"`
}

// ProcessingOptions configures post-processing behavior
type ProcessingOptions struct {
	// Enable/disable processing techniques
	EnableReranking      bool `json:"enable_reranking"`
	EnableFiltering      bool `json:"enable_filtering"`
	EnableDeduplication  bool `json:"enable_deduplication"`
	EnableCompression    bool `json:"enable_compression"`
	
	// Processing parameters
	MaxResults           int     `json:"max_results"`
	MinRelevanceScore    float64 `json:"min_relevance_score"`
	DiversityWeight      float64 `json:"diversity_weight"`
	
	// Specific processing options
	RerankingOptions     *RerankingOptions     `json:"reranking_options,omitempty"`
	FilteringOptions     *FilteringOptions     `json:"filtering_options,omitempty"`
	DeduplicationOptions *DeduplicationOptions `json:"deduplication_options,omitempty"`
	CompressionOptions   *CompressionOptions   `json:"compression_options,omitempty"`
}

// RerankingOptions configures result reranking
type RerankingOptions struct {
	Method              RerankingMethod `json:"method"`
	RelevanceWeight     float64         `json:"relevance_weight"`
	QualityWeight       float64         `json:"quality_weight"`
	FreshnessWeight     float64         `json:"freshness_weight"`
	DiversityWeight     float64         `json:"diversity_weight"`
	AuthorityWeight     float64         `json:"authority_weight"`
	MaxRerankedResults  int             `json:"max_reranked_results"`
	UseMLModel          bool            `json:"use_ml_model"`
	ModelConfig         ModelConfig     `json:"model_config,omitempty"`
}

// FilteringOptions configures result filtering
type FilteringOptions struct {
	MinScore            float64           `json:"min_score"`
	MaxAge              time.Duration     `json:"max_age"`
	RequiredKeywords    []string          `json:"required_keywords,omitempty"`
	ExcludedKeywords    []string          `json:"excluded_keywords,omitempty"`
	AllowedSources      []string          `json:"allowed_sources,omitempty"`
	ExcludedSources     []string          `json:"excluded_sources,omitempty"`
	MinContentLength    int               `json:"min_content_length"`
	MaxContentLength    int               `json:"max_content_length"`
	CustomFilters       []CustomFilter    `json:"custom_filters,omitempty"`
	LanguageFilter      string            `json:"language_filter,omitempty"`
	ContentTypeFilter   []string          `json:"content_type_filter,omitempty"`
}

// DeduplicationOptions configures duplicate removal
type DeduplicationOptions struct {
	Method              DeduplicationMethod `json:"method"`
	SimilarityThreshold float64             `json:"similarity_threshold"`
	ContentSimilarity   bool                `json:"content_similarity"`
	TitleSimilarity     bool                `json:"title_similarity"`
	URLSimilarity       bool                `json:"url_similarity"`
	PreferHigherScore   bool                `json:"prefer_higher_score"`
	PreferMoreRecent    bool                `json:"prefer_more_recent"`
	MaxSimilarResults   int                 `json:"max_similar_results"`
}

// CompressionOptions configures context compression
type CompressionOptions struct {
	Method              CompressionMethod `json:"method"`
	TargetLength        int               `json:"target_length"`
	MaxSummaryLength    int               `json:"max_summary_length"`
	PreserveKeyPoints   bool              `json:"preserve_key_points"`
	IncludeReferences   bool              `json:"include_references"`
	QualityThreshold    float64           `json:"quality_threshold"`
	SummaryModel        string            `json:"summary_model,omitempty"`
}

// RerankingMethod defines different reranking algorithms
type RerankingMethod int

const (
	// SimpleReranking uses basic scoring combination
	SimpleReranking RerankingMethod = iota
	// MLReranking uses machine learning models
	MLReranking
	// SemanticReranking uses semantic similarity
	SemanticReranking
	// HybridReranking combines multiple methods
	HybridReranking
	// LLMReranking uses LLM-based reranking
	LLMReranking
)

func (r RerankingMethod) String() string {
	switch r {
	case SimpleReranking:
		return "simple"
	case MLReranking:
		return "ml"
	case SemanticReranking:
		return "semantic"
	case HybridReranking:
		return "hybrid"
	case LLMReranking:
		return "llm"
	default:
		return "unknown"
	}
}

// DeduplicationMethod defines different deduplication algorithms
type DeduplicationMethod int

const (
	// TextSimilarity uses text similarity comparison
	TextSimilarity DeduplicationMethod = iota
	// HashBased uses content hashing
	HashBased
	// SemanticSimilarity uses semantic embeddings
	SemanticSimilarity
	// URLBased uses URL comparison
	URLBased
	// HybridDeduplication combines multiple methods
	HybridDeduplication
)

func (d DeduplicationMethod) String() string {
	switch d {
	case TextSimilarity:
		return "text_similarity"
	case HashBased:
		return "hash_based"
	case SemanticSimilarity:
		return "semantic_similarity"
	case URLBased:
		return "url_based"
	case HybridDeduplication:
		return "hybrid"
	default:
		return "unknown"
	}
}

// CompressionMethod defines different compression algorithms
type CompressionMethod int

const (
	// ExtractiveSummary extracts key sentences
	ExtractiveSummary CompressionMethod = iota
	// AbstractiveSummary generates new summary
	AbstractiveSummary
	// KeywordExtraction extracts important keywords
	KeywordExtraction
	// TemplateBased uses templates for compression
	TemplateBased
	// LLMCompression uses LLM for compression
	LLMCompression
)

func (c CompressionMethod) String() string {
	switch c {
	case ExtractiveSummary:
		return "extractive_summary"
	case AbstractiveSummary:
		return "abstractive_summary"
	case KeywordExtraction:
		return "keyword_extraction"
	case TemplateBased:
		return "template_based"
	case LLMCompression:
		return "llm_compression"
	default:
		return "unknown"
	}
}

// CustomFilter defines a custom filtering function
type CustomFilter struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Predicate   func(SearchResult) bool `json:"-"` // Function not serialized
	Config      map[string]interface{} `json:"config,omitempty"`
}

// ModelConfig configures ML models for reranking
type ModelConfig struct {
	ModelName     string                 `json:"model_name"`
	ModelVersion  string                 `json:"model_version"`
	ModelPath     string                 `json:"model_path,omitempty"`
	Features      []string               `json:"features"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	Endpoint      string                 `json:"endpoint,omitempty"`
	APIKey        string                 `json:"api_key,omitempty"`
}

// DefaultProcessingOptions returns default processing options
func DefaultProcessingOptions() *ProcessingOptions {
	return &ProcessingOptions{
		EnableReranking:     true,
		EnableFiltering:     true,
		EnableDeduplication: true,
		EnableCompression:   false,
		MaxResults:          10,
		MinRelevanceScore:   0.3,
		DiversityWeight:     0.1,
		RerankingOptions:    DefaultRerankingOptions(),
		FilteringOptions:    DefaultFilteringOptions(),
		DeduplicationOptions: DefaultDeduplicationOptions(),
		CompressionOptions:  DefaultCompressionOptions(),
	}
}

// DefaultRerankingOptions returns default reranking options
func DefaultRerankingOptions() *RerankingOptions {
	return &RerankingOptions{
		Method:             SimpleReranking,
		RelevanceWeight:    0.4,
		QualityWeight:      0.3,
		FreshnessWeight:    0.1,
		DiversityWeight:    0.1,
		AuthorityWeight:    0.1,
		MaxRerankedResults: 20,
		UseMLModel:         false,
	}
}

// DefaultFilteringOptions returns default filtering options
func DefaultFilteringOptions() *FilteringOptions {
	return &FilteringOptions{
		MinScore:         0.1,
		MaxAge:           365 * 24 * time.Hour, // 1 year
		MinContentLength: 50,
		MaxContentLength: 10000,
	}
}

// DefaultDeduplicationOptions returns default deduplication options
func DefaultDeduplicationOptions() *DeduplicationOptions {
	return &DeduplicationOptions{
		Method:              TextSimilarity,
		SimilarityThreshold: 0.8,
		ContentSimilarity:   true,
		TitleSimilarity:     true,
		URLSimilarity:       false,
		PreferHigherScore:   true,
		PreferMoreRecent:    true,
		MaxSimilarResults:   1,
	}
}

// DefaultCompressionOptions returns default compression options
func DefaultCompressionOptions() *CompressionOptions {
	return &CompressionOptions{
		Method:            ExtractiveSummary,
		TargetLength:      1000,
		MaxSummaryLength:  200,
		PreserveKeyPoints: true,
		IncludeReferences: true,
		QualityThreshold:  0.7,
	}
}

// Reranker defines interface for result reranking
type Reranker interface {
	// Rerank reorders results based on the specified criteria
	Rerank(ctx context.Context, query string, results []SearchResult, options *RerankingOptions) ([]SearchResult, error)
	
	// CalculateRelevanceScore calculates relevance score for a result
	CalculateRelevanceScore(ctx context.Context, query string, result SearchResult) (float64, error)
}

// Filter defines interface for result filtering
type Filter interface {
	// Filter applies filtering criteria to results
	Filter(ctx context.Context, query string, results []SearchResult, options *FilteringOptions) ([]SearchResult, error)
	
	// ShouldInclude determines if a result should be included
	ShouldInclude(ctx context.Context, result SearchResult, criteria FilteringOptions) bool
}

// Deduplicator defines interface for duplicate removal
type Deduplicator interface {
	// Deduplicate removes duplicate or very similar results
	Deduplicate(ctx context.Context, results []SearchResult, options *DeduplicationOptions) ([]SearchResult, error)
	
	// CalculateSimilarity calculates similarity between two results
	CalculateSimilarity(ctx context.Context, result1, result2 SearchResult) (float64, error)
}

// ContextCompressor defines interface for context compression
type ContextCompressor interface {
	// Compress reduces context while preserving important information
	Compress(ctx context.Context, query string, results []SearchResult, options *CompressionOptions) (*CompressedContext, error)
	
	// Summarize creates a summary of the content
	Summarize(ctx context.Context, content string, maxLength int) (string, error)
	
	// ExtractKeyPoints extracts key points from content
	ExtractKeyPoints(ctx context.Context, content string, maxPoints int) ([]string, error)
}

// PostProcessingConfig contains configuration for post-processing
type PostProcessingConfig struct {
	// Provider configurations
	LLMConfig       LLMConfig       `json:"llm_config,omitempty"`
	EmbeddingConfig EmbeddingConfig `json:"embedding_config,omitempty"`
	
	// Default processing options
	DefaultOptions *ProcessingOptions `json:"default_options"`
	
	// Performance settings
	MaxConcurrency  int           `json:"max_concurrency"`
	RequestTimeout  time.Duration `json:"request_timeout"`
	CacheEnabled    bool          `json:"cache_enabled"`
	CacheTTL        time.Duration `json:"cache_ttl"`
	
	// Quality thresholds
	MinQualityScore     float64 `json:"min_quality_score"`
	MaxProcessingTime   time.Duration `json:"max_processing_time"`
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

// DefaultPostProcessingConfig returns default configuration
func DefaultPostProcessingConfig() *PostProcessingConfig {
	return &PostProcessingConfig{
		DefaultOptions:    DefaultProcessingOptions(),
		MaxConcurrency:    5,
		RequestTimeout:    30 * time.Second,
		CacheEnabled:      true,
		CacheTTL:         1 * time.Hour,
		MinQualityScore:  0.6,
		MaxProcessingTime: 10 * time.Second,
	}
}

// PostProcessingMetrics contains metrics for post-processing performance
type PostProcessingMetrics struct {
	ProcessingTime      time.Duration `json:"processing_time"`
	InputResultCount    int           `json:"input_result_count"`
	OutputResultCount   int           `json:"output_result_count"`
	FilteredCount       int           `json:"filtered_count"`
	DeduplicatedCount   int           `json:"deduplicated_count"`
	CompressionRatio    float64       `json:"compression_ratio"`
	QualityImprovement  float64       `json:"quality_improvement"`
	CacheHitRate        float64       `json:"cache_hit_rate"`
	LLMCallCount        int           `json:"llm_call_count"`
	EmbeddingCallCount  int           `json:"embedding_call_count"`
}