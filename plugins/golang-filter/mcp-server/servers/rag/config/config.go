package config

import "fmt"

// Config represents the main configuration structure for the MCP server
type Config struct {
	RAG       RAGConfig       `json:"rag" yaml:"rag"`
	LLM       LLMConfig       `json:"llm" yaml:"llm"`
	Embedding EmbeddingConfig `json:"embedding" yaml:"embedding"`
	VectorDB  VectorDBConfig  `json:"vectordb" yaml:"vectordb"`
	// Enhanced features configuration
	Enhancement EnhancementConfig `json:"enhancement,omitempty" yaml:"enhancement,omitempty"`
}

// RAGConfig contains basic configuration for the RAG system
type RAGConfig struct {
	Splitter  SplitterConfig `json:"splitter" yaml:"splitter"`
	Threshold float64        `json:"threshold,omitempty" yaml:"threshold,omitempty"`
	TopK      int            `json:"top_k,omitempty" yaml:"top_k,omitempty"`
}

// EnhancementConfig contains configuration for enhanced RAG features
type EnhancementConfig struct {
	// Query Enhancement
	QueryEnhancement QueryEnhancementConfig `json:"query_enhancement,omitempty" yaml:"query_enhancement,omitempty"`
	
	// Hybrid Search
	HybridSearch HybridSearchConfig `json:"hybrid_search,omitempty" yaml:"hybrid_search,omitempty"`
	
	// CRAG (Corrective RAG)
	CRAG CRAGConfig `json:"crag,omitempty" yaml:"crag,omitempty"`
	
	// Post-processing
	PostProcessing PostProcessingConfig `json:"post_processing,omitempty" yaml:"post_processing,omitempty"`
	
	// Performance settings
	Performance PerformanceConfig `json:"performance,omitempty" yaml:"performance,omitempty"`
}

// QueryEnhancementConfig defines configuration for query enhancement
type QueryEnhancementConfig struct {
	Enabled                   bool    `json:"enabled" yaml:"enabled"`
	EnableRewrite             bool    `json:"enable_rewrite" yaml:"enable_rewrite"`
	EnableExpansion           bool    `json:"enable_expansion" yaml:"enable_expansion"`
	EnableDecomposition       bool    `json:"enable_decomposition" yaml:"enable_decomposition"`
	EnableIntentClassification bool   `json:"enable_intent_classification" yaml:"enable_intent_classification"`
	MaxRewriteCount           int     `json:"max_rewrite_count" yaml:"max_rewrite_count"`
	MaxExpansionTerms         int     `json:"max_expansion_terms" yaml:"max_expansion_terms"`
	MaxSubQueries             int     `json:"max_sub_queries" yaml:"max_sub_queries"`
	CacheEnabled              bool    `json:"cache_enabled" yaml:"cache_enabled"`
	CacheSize                 int     `json:"cache_size" yaml:"cache_size"`
	CacheTTLMinutes           int     `json:"cache_ttl_minutes" yaml:"cache_ttl_minutes"`
}

// HybridSearchConfig defines configuration for hybrid search
type HybridSearchConfig struct {
	Enabled                bool              `json:"enabled" yaml:"enabled"`
	FusionMethod           string            `json:"fusion_method" yaml:"fusion_method"` // rrf, weighted, borda, combsum, combmnz
	VectorWeight           float64           `json:"vector_weight" yaml:"vector_weight"`
	BM25Weight             float64           `json:"bm25_weight" yaml:"bm25_weight"`
	RRFConstant            float64           `json:"rrf_constant" yaml:"rrf_constant"`
	EnableNormalization    bool              `json:"enable_normalization" yaml:"enable_normalization"`
	NormalizationMethod    string            `json:"normalization_method" yaml:"normalization_method"` // minmax, zscore, sum
	EnableDiversity        bool              `json:"enable_diversity" yaml:"enable_diversity"`
	DiversityWeight        float64           `json:"diversity_weight" yaml:"diversity_weight"`
	TieBreakingStrategy    string            `json:"tie_breaking_strategy" yaml:"tie_breaking_strategy"` // prefer_vector, prefer_bm25, prefer_higher_score, prefer_lower_rank
	BM25Config             BM25Config        `json:"bm25_config" yaml:"bm25_config"`
}

// BM25Config defines configuration for BM25 search
type BM25Config struct {
	K1         float64 `json:"k1" yaml:"k1"`
	B          float64 `json:"b" yaml:"b"`
	MaxResults int     `json:"max_results" yaml:"max_results"`
}

// CRAGConfig defines configuration for Corrective RAG
type CRAGConfig struct {
	Enabled                bool    `json:"enabled" yaml:"enabled"`
	ConfidenceThreshold    float64 `json:"confidence_threshold" yaml:"confidence_threshold"`
	EnableWebSearch        bool    `json:"enable_web_search" yaml:"enable_web_search"`
	EnableRefinement       bool    `json:"enable_refinement" yaml:"enable_refinement"`
	MaxWebResults          int     `json:"max_web_results" yaml:"max_web_results"`
	MaxRefinements         int     `json:"max_refinements" yaml:"max_refinements"`
	WebSearchEngine        string  `json:"web_search_engine" yaml:"web_search_engine"` // bing, google, duckduckgo
	WebSearchAPIKey        string  `json:"web_search_api_key" yaml:"web_search_api_key"`
	RefinementStrategy     string  `json:"refinement_strategy" yaml:"refinement_strategy"` // merge, replace, augment
}

// PostProcessingConfig defines configuration for result post-processing
type PostProcessingConfig struct {
	Enabled                  bool               `json:"enabled" yaml:"enabled"`
	EnableReranking          bool               `json:"enable_reranking" yaml:"enable_reranking"`
	EnableFiltering          bool               `json:"enable_filtering" yaml:"enable_filtering"`
	EnableDeduplication      bool               `json:"enable_deduplication" yaml:"enable_deduplication"`
	EnableCompression        bool               `json:"enable_compression" yaml:"enable_compression"`
	RerankingConfig          RerankingConfig    `json:"reranking_config" yaml:"reranking_config"`
	FilteringConfig          FilteringConfig    `json:"filtering_config" yaml:"filtering_config"`
	DeduplicationConfig      DeduplicationConfig `json:"deduplication_config" yaml:"deduplication_config"`
	CompressionConfig        CompressionConfig  `json:"compression_config" yaml:"compression_config"`
}

// RerankingConfig defines configuration for result reranking
type RerankingConfig struct {
	Method              string  `json:"method" yaml:"method"` // simple, semantic, hybrid, llm, ml
	RelevanceWeight     float64 `json:"relevance_weight" yaml:"relevance_weight"`
	QualityWeight       float64 `json:"quality_weight" yaml:"quality_weight"`
	FreshnessWeight     float64 `json:"freshness_weight" yaml:"freshness_weight"`
	DiversityWeight     float64 `json:"diversity_weight" yaml:"diversity_weight"`
	AuthorityWeight     float64 `json:"authority_weight" yaml:"authority_weight"`
	MaxRerankedResults  int     `json:"max_reranked_results" yaml:"max_reranked_results"`
	UseMLModel          bool    `json:"use_ml_model" yaml:"use_ml_model"`
	MLModelPath         string  `json:"ml_model_path" yaml:"ml_model_path"`
}

// FilteringConfig defines configuration for result filtering
type FilteringConfig struct {
	MinScore            float64   `json:"min_score" yaml:"min_score"`
	MaxAgeHours         int       `json:"max_age_hours" yaml:"max_age_hours"`
	RequiredKeywords    []string  `json:"required_keywords" yaml:"required_keywords"`
	ExcludedKeywords    []string  `json:"excluded_keywords" yaml:"excluded_keywords"`
	AllowedSources      []string  `json:"allowed_sources" yaml:"allowed_sources"`
	ExcludedSources     []string  `json:"excluded_sources" yaml:"excluded_sources"`
	MinContentLength    int       `json:"min_content_length" yaml:"min_content_length"`
	MaxContentLength    int       `json:"max_content_length" yaml:"max_content_length"`
	LanguageFilter      string    `json:"language_filter" yaml:"language_filter"`
	ContentTypeFilter   []string  `json:"content_type_filter" yaml:"content_type_filter"`
}

// DeduplicationConfig defines configuration for duplicate removal
type DeduplicationConfig struct {
	Method              string  `json:"method" yaml:"method"` // text_similarity, hash_based, semantic_similarity, url_based, hybrid
	SimilarityThreshold float64 `json:"similarity_threshold" yaml:"similarity_threshold"`
	ContentSimilarity   bool    `json:"content_similarity" yaml:"content_similarity"`
	TitleSimilarity     bool    `json:"title_similarity" yaml:"title_similarity"`
	URLSimilarity       bool    `json:"url_similarity" yaml:"url_similarity"`
	PreferHigherScore   bool    `json:"prefer_higher_score" yaml:"prefer_higher_score"`
	PreferMoreRecent    bool    `json:"prefer_more_recent" yaml:"prefer_more_recent"`
	MaxSimilarResults   int     `json:"max_similar_results" yaml:"max_similar_results"`
}

// CompressionConfig defines configuration for context compression
type CompressionConfig struct {
	Method              string  `json:"method" yaml:"method"` // extractive_summary, abstractive_summary, keyword_extraction, template_based, llm_compression
	TargetLength        int     `json:"target_length" yaml:"target_length"`
	MaxSummaryLength    int     `json:"max_summary_length" yaml:"max_summary_length"`
	PreserveKeyPoints   bool    `json:"preserve_key_points" yaml:"preserve_key_points"`
	IncludeReferences   bool    `json:"include_references" yaml:"include_references"`
	QualityThreshold    float64 `json:"quality_threshold" yaml:"quality_threshold"`
	SummaryModel        string  `json:"summary_model" yaml:"summary_model"`
}

// PerformanceConfig defines performance-related settings
type PerformanceConfig struct {
	MaxConcurrency    int `json:"max_concurrency" yaml:"max_concurrency"`
	RequestTimeoutMs  int `json:"request_timeout_ms" yaml:"request_timeout_ms"`
	CacheEnabled      bool `json:"cache_enabled" yaml:"cache_enabled"`
	CacheTTLMinutes   int `json:"cache_ttl_minutes" yaml:"cache_ttl_minutes"`
	EnableProfiling   bool `json:"enable_profiling" yaml:"enable_profiling"`
	EnableMetrics     bool `json:"enable_metrics" yaml:"enable_metrics"`
	EnableLogging     bool `json:"enable_logging" yaml:"enable_logging"`
	LogLevel          string `json:"log_level" yaml:"log_level"` // debug, info, warn, error
}

// DefaultEnhancementConfig returns default enhancement configuration
func DefaultEnhancementConfig() EnhancementConfig {
	return EnhancementConfig{
		QueryEnhancement: QueryEnhancementConfig{
			Enabled:                   true,
			EnableRewrite:             true,
			EnableExpansion:           true,
			EnableDecomposition:       false,
			EnableIntentClassification: true,
			MaxRewriteCount:           3,
			MaxExpansionTerms:         10,
			MaxSubQueries:             5,
			CacheEnabled:              true,
			CacheSize:                 1000,
			CacheTTLMinutes:           60,
		},
		HybridSearch: HybridSearchConfig{
			Enabled:                true,
			FusionMethod:           "rrf",
			VectorWeight:           0.6,
			BM25Weight:             0.4,
			RRFConstant:            60.0,
			EnableNormalization:    true,
			NormalizationMethod:    "minmax",
			EnableDiversity:        false,
			DiversityWeight:        0.1,
			TieBreakingStrategy:    "prefer_vector",
			BM25Config: BM25Config{
				K1:         1.2,
				B:          0.75,
				MaxResults: 100,
			},
		},
		CRAG: CRAGConfig{
			Enabled:             true,
			ConfidenceThreshold: 0.7,
			EnableWebSearch:     true,
			EnableRefinement:    true,
			MaxWebResults:       5,
			MaxRefinements:      3,
			WebSearchEngine:     "duckduckgo",
			RefinementStrategy:  "augment",
		},
		PostProcessing: PostProcessingConfig{
			Enabled:             true,
			EnableReranking:     true,
			EnableFiltering:     true,
			EnableDeduplication: true,
			EnableCompression:   false,
			RerankingConfig: RerankingConfig{
				Method:             "simple",
				RelevanceWeight:    0.4,
				QualityWeight:      0.3,
				FreshnessWeight:    0.1,
				DiversityWeight:    0.1,
				AuthorityWeight:    0.1,
				MaxRerankedResults: 20,
				UseMLModel:         false,
			},
			FilteringConfig: FilteringConfig{
				MinScore:         0.1,
				MaxAgeHours:      8760, // 1 year
				MinContentLength: 50,
				MaxContentLength: 10000,
			},
			DeduplicationConfig: DeduplicationConfig{
				Method:              "text_similarity",
				SimilarityThreshold: 0.8,
				ContentSimilarity:   true,
				TitleSimilarity:     true,
				URLSimilarity:       false,
				PreferHigherScore:   true,
				PreferMoreRecent:    true,
				MaxSimilarResults:   1,
			},
			CompressionConfig: CompressionConfig{
				Method:            "extractive_summary",
				TargetLength:      1000,
				MaxSummaryLength:  200,
				PreserveKeyPoints: true,
				IncludeReferences: true,
				QualityThreshold:  0.7,
			},
		},
		Performance: PerformanceConfig{
			MaxConcurrency:   5,
			RequestTimeoutMs: 30000,
			CacheEnabled:     true,
			CacheTTLMinutes:  60,
			EnableProfiling:  false,
			EnableMetrics:    true,
			EnableLogging:    true,
			LogLevel:         "info",
		},
	}
}

// SplitterConfig defines document splitter configuration
type SplitterConfig struct {
	Provider     string `json:"provider" yaml:"provider"` // Available options: recursive, character, token
	ChunkSize    int    `json:"chunk_size,omitempty" yaml:"chunk_size,omitempty"`
	ChunkOverlap int    `json:"chunk_overlap,omitempty" yaml:"chunk_overlap,omitempty"`
}

// LLMConfig defines configuration for Large Language Models
type LLMConfig struct {
	Provider    string  `json:"provider" yaml:"provider"` // Available options: openai, dashscope, qwen
	APIKey      string  `json:"api_key,omitempty" yaml:"api_key"`
	BaseURL     string  `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Model       string  `json:"model" yaml:"model"`
	Temperature float64 `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
}

// EmbeddingConfig defines configuration for embedding models
type EmbeddingConfig struct {
	Provider   string `json:"provider" yaml:"provider"` // Available options: openai, dashscope
	APIKey     string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	BaseURL    string `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Model      string `json:"model,omitempty" yaml:"model,omitempty"`
	Dimensions int    `json:"dimensions,omitempty" yaml:"dimension,omitempty"`
}

// VectorDBConfig defines configuration for vector databases
type VectorDBConfig struct {
	Provider   string        `json:"provider" yaml:"provider"` // Available options: milvus, qdrant, chroma
	Host       string        `json:"host,omitempty" yaml:"host,omitempty"`
	Port       int           `json:"port,omitempty" yaml:"port,omitempty"`
	Database   string        `json:"database,omitempty" yaml:"database,omitempty"`
	Collection string        `json:"collection,omitempty" yaml:"collection,omitempty"`
	Username   string        `json:"username,omitempty" yaml:"username,omitempty"`
	Password   string        `json:"password,omitempty" yaml:"password,omitempty"`
	Mapping    MappingConfig `json:"mapping,omitempty" yaml:"mapping,omitempty"`
}

// MappingConfig defines field mapping configuration for vector databases
type MappingConfig struct {
	Fields []FieldMapping `json:"fields,omitempty" yaml:"fields,omitempty"`
	Index  IndexConfig    `json:"index,omitempty" yaml:"index,omitempty"`
	Search SearchConfig   `json:"search,omitempty" yaml:"search,omitempty"`
}

// // CollectionMapping defines field mapping for collection
// type CollectionMapping struct {
// 	Fields []FieldMapping `json:"fields,omitempty" yaml:"fields,omitempty"`
// }

// FieldMapping defines mapping for a single field
type FieldMapping struct {
	StandardName string                 `json:"standard_name" yaml:"standard_name"`
	RawName      string                 `json:"raw_name" yaml:"raw_name"`
	Properties   map[string]interface{} `json:"properties,omitempty" yaml:"properties,omitempty"`
}

func (f FieldMapping) IsPrimaryKey() bool {
	return f.StandardName == "id"
}

func (f FieldMapping) IsAutoID() bool {
	if f.Properties == nil {
		return false
	}
	autoID, ok := f.Properties["auto_id"].(bool)
	if !ok {
		return false
	}
	return autoID
}

func (f FieldMapping) IsVectorField() bool {
	return f.StandardName == "vector"
}

func (f FieldMapping) MaxLength() int {
	if f.Properties == nil {
		return 0
	}
	maxLength, ok := f.Properties["max_length"].(int)
	if !ok {
		return 256
	}
	return maxLength
}

// IndexConfig defines configuration for index parameters
type IndexConfig struct {
	// Index type, e.g., IVF_FLAT, IVF_SQ8, HNSW, etc.
	IndexType string `json:"index_type" yaml:"index_type"`
	// Index parameter configuration
	Params map[string]interface{} `json:"params" yaml:"params"`
}

func (i IndexConfig) ParamsString(key string) (string, error) {
	if mVal, ok := i.Params[key].(string); ok {
		return mVal, nil
	}
	return "", fmt.Errorf("params %s not found", key)
}

func (i IndexConfig) ParamsInt64(key string) (int64, error) {
	if mVal, ok := i.Params[key].(int64); ok {
		return mVal, nil
	}
	if mVal, ok := i.Params[key].(int); ok {
		return int64(mVal), nil
	}
	return 0, fmt.Errorf("params %s not found", key)
}

func (i IndexConfig) ParamsFloat64(key string) (float64, error) {
	if mVal, ok := i.Params[key].(float64); ok {
		return mVal, nil
	}
	if mVal, ok := i.Params[key].(float32); ok {
		return float64(mVal), nil
	}
	return 0, fmt.Errorf("params %s not found", key)
}

func (i IndexConfig) ParamsBool(key string) (bool, error) {
	if mVal, ok := i.Params[key].(bool); ok {
		return mVal, nil
	}
	return false, fmt.Errorf("params %s not found", key)
}

// SearchConfig defines configuration for search parameters
type SearchConfig struct {
	// Metric type, e.g., L2, IP, etc.
	MetricType string `json:"metric_type,omitempty" yaml:"metric_type,omitempty"`
	// Search parameter configuration
	Params map[string]interface{} `json:"params" yaml:"params"`
}

func (i SearchConfig) ParamsString(key string) (string, error) {
	if mVal, ok := i.Params[key].(string); ok {
		return mVal, nil
	}
	return "", fmt.Errorf("params %s not found", key)
}

func (i SearchConfig) ParamsInt64(key string) (int64, error) {
	if mVal, ok := i.Params[key].(int64); ok {
		return mVal, nil
	}
	return 0, fmt.Errorf("params %s not found", key)
}

func (i SearchConfig) ParamsFloat64(key string) (float64, error) {
	if mVal, ok := i.Params[key].(float64); ok {
		return mVal, nil
	}
	return 0, fmt.Errorf("params %s not found", key)
}

func (i SearchConfig) ParamsBool(key string) (bool, error) {
	if mVal, ok := i.Params[key].(bool); ok {
		return mVal, nil
	}
	return false, fmt.Errorf("params %s not found", key)
}
