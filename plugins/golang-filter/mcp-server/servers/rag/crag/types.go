package crag

import (
	"context"
	"time"
)

// ConfidenceLevel represents the three-tier confidence classification
type ConfidenceLevel int

const (
	// HighConfidence indicates retrieved context is highly relevant and accurate
	HighConfidence ConfidenceLevel = iota
	// LowConfidence indicates retrieved context is somewhat relevant but needs enrichment
	LowConfidence
	// NoConfidence indicates retrieved context is irrelevant and requires web search
	NoConfidence
)

func (c ConfidenceLevel) String() string {
	switch c {
	case HighConfidence:
		return "high"
	case LowConfidence:
		return "low"
	case NoConfidence:
		return "none"
	default:
		return "unknown"
	}
}

// RetrievalEvaluator defines the interface for evaluating retrieval quality
type RetrievalEvaluator interface {
	// EvaluateRetrieval assesses the quality of retrieved documents for a given query
	EvaluateRetrieval(ctx context.Context, query string, documents []Document) (*EvaluationResult, error)
	
	// SetThresholds configures confidence thresholds
	SetThresholds(high, low float64)
}

// WebSearcher defines the interface for web search integration
type WebSearcher interface {
	// Search performs web search and returns relevant documents
	Search(ctx context.Context, query string, maxResults int) ([]WebDocument, error)
	
	// SearchWithFilters performs web search with domain/content filters
	SearchWithFilters(ctx context.Context, query string, filters *SearchFilters) ([]WebDocument, error)
}

// KnowledgeRefinement defines the interface for refining retrieved knowledge
type KnowledgeRefinement interface {
	// RefineKnowledge processes and improves the quality of retrieved information
	RefineKnowledge(ctx context.Context, query string, documents []Document) ([]Document, error)
	
	// FilterRelevant filters documents based on relevance to the query
	FilterRelevant(ctx context.Context, query string, documents []Document, threshold float64) ([]Document, error)
	
	// RerankDocuments reorders documents based on relevance and quality
	RerankDocuments(ctx context.Context, query string, documents []Document) ([]Document, error)
}

// CRAGProcessor implements the core CRAG mechanism
type CRAGProcessor interface {
	// ProcessQuery implements the full CRAG workflow
	ProcessQuery(ctx context.Context, query string, initialDocs []Document) (*CRAGResult, error)
	
	// EvaluateAndRoute evaluates retrieval quality and routes accordingly
	EvaluateAndRoute(ctx context.Context, query string, docs []Document) (*RoutingDecision, error)
}

// Document represents a knowledge document
type Document struct {
	ID          string                 `json:"id"`
	Content     string                 `json:"content"`
	Title       string                 `json:"title,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Score       float64                `json:"score"`
	Source      string                 `json:"source"` // "vector_db", "web_search", etc.
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RetrievedAt time.Time              `json:"retrieved_at"`
}

// WebDocument represents a document from web search
type WebDocument struct {
	Title       string                 `json:"title"`
	Content     string                 `json:"content"`
	URL         string                 `json:"url"`
	Score       float64                `json:"score"`
	Source      string                 `json:"source"`
	Snippet     string                 `json:"snippet,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RetrievedAt time.Time              `json:"retrieved_at"`
}

// EvaluationResult contains the result of retrieval evaluation
type EvaluationResult struct {
	ConfidenceLevel ConfidenceLevel `json:"confidence_level"`
	OverallScore    float64         `json:"overall_score"`
	DocumentScores  []DocumentScore `json:"document_scores"`
	Reasoning       string          `json:"reasoning,omitempty"`
	EvaluatedAt     time.Time       `json:"evaluated_at"`
}

// DocumentScore represents individual document evaluation
type DocumentScore struct {
	DocumentID   string  `json:"document_id"`
	RelevanceScore float64 `json:"relevance_score"`
	QualityScore   float64 `json:"quality_score"`
	OverallScore   float64 `json:"overall_score"`
}

// RoutingDecision represents the routing decision based on confidence
type RoutingDecision struct {
	Action          CRAGAction      `json:"action"`
	ConfidenceLevel ConfidenceLevel `json:"confidence_level"`
	Reasoning       string          `json:"reasoning"`
	Documents       []Document      `json:"documents"`
	DecidedAt       time.Time       `json:"decided_at"`
}

// CRAGAction defines possible actions in CRAG workflow
type CRAGAction int

const (
	// UseRetrieved uses the retrieved documents directly
	UseRetrieved CRAGAction = iota
	// EnrichWithWeb enriches retrieved documents with web search
	EnrichWithWeb
	// ReplaceWithWeb replaces retrieved documents with web search results
	ReplaceWithWeb
)

func (a CRAGAction) String() string {
	switch a {
	case UseRetrieved:
		return "use_retrieved"
	case EnrichWithWeb:
		return "enrich_with_web"
	case ReplaceWithWeb:
		return "replace_with_web"
	default:
		return "unknown"
	}
}

// CRAGResult contains the final result of CRAG processing
type CRAGResult struct {
	Query           string          `json:"query"`
	FinalDocuments  []Document      `json:"final_documents"`
	RoutingDecision RoutingDecision `json:"routing_decision"`
	WebSearchUsed   bool            `json:"web_search_used"`
	ProcessingTime  time.Duration   `json:"processing_time"`
	ProcessedAt     time.Time       `json:"processed_at"`
}

// SearchFilters defines filters for web search
type SearchFilters struct {
	Domains      []string  `json:"domains,omitempty"`       // Allowed domains
	ExcludeDomains []string `json:"exclude_domains,omitempty"` // Excluded domains
	Language     string    `json:"language,omitempty"`      // Language filter
	TimeRange    string    `json:"time_range,omitempty"`    // Time range filter
	ContentType  string    `json:"content_type,omitempty"`  // Content type filter
	MaxResults   int       `json:"max_results"`             // Maximum results
}

// CRAGConfig contains configuration for CRAG mechanism
type CRAGConfig struct {
	// Confidence thresholds
	HighConfidenceThreshold float64 `json:"high_confidence_threshold"` // Default: 0.8
	LowConfidenceThreshold  float64 `json:"low_confidence_threshold"`  // Default: 0.5
	
	// Web search settings
	WebSearchEnabled    bool   `json:"web_search_enabled"`     // Default: true
	MaxWebResults       int    `json:"max_web_results"`        // Default: 5
	WebSearchTimeout    time.Duration `json:"web_search_timeout"` // Default: 10s
	
	// Knowledge refinement settings
	RefinementEnabled   bool    `json:"refinement_enabled"`     // Default: true
	RelevanceThreshold  float64 `json:"relevance_threshold"`    // Default: 0.3
	MaxDocuments        int     `json:"max_documents"`          // Default: 10
	
	// Evaluation settings
	EvaluationModel     string  `json:"evaluation_model"`       // LLM model for evaluation
	EvaluationTimeout   time.Duration `json:"evaluation_timeout"` // Default: 5s
	
	// Web search provider settings
	SearchProvider      string                 `json:"search_provider"`      // "duckduckgo", "bing", etc.
	SearchProviderConfig map[string]interface{} `json:"search_provider_config"`
}

// DefaultCRAGConfig returns default CRAG configuration
func DefaultCRAGConfig() *CRAGConfig {
	return &CRAGConfig{
		HighConfidenceThreshold: 0.8,
		LowConfidenceThreshold:  0.5,
		WebSearchEnabled:        true,
		MaxWebResults:          5,
		WebSearchTimeout:       10 * time.Second,
		RefinementEnabled:      true,
		RelevanceThreshold:     0.3,
		MaxDocuments:          10,
		EvaluationModel:       "gpt-4",
		EvaluationTimeout:     5 * time.Second,
		SearchProvider:        "duckduckgo",
		SearchProviderConfig:  make(map[string]interface{}),
	}
}

// EvaluationCriteria defines criteria for document evaluation
type EvaluationCriteria struct {
	RelevanceWeight  float64 `json:"relevance_weight"`  // Weight for relevance scoring
	QualityWeight    float64 `json:"quality_weight"`    // Weight for quality scoring
	FreshnessWeight  float64 `json:"freshness_weight"`  // Weight for freshness scoring
	AuthorityWeight  float64 `json:"authority_weight"`  // Weight for authority scoring
}

// DefaultEvaluationCriteria returns default evaluation criteria
func DefaultEvaluationCriteria() *EvaluationCriteria {
	return &EvaluationCriteria{
		RelevanceWeight: 0.4,
		QualityWeight:   0.3,
		FreshnessWeight: 0.2,
		AuthorityWeight: 0.1,
	}
}