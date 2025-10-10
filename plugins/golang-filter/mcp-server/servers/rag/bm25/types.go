package bm25

import (
	"context"
	"time"
)

// BM25Result represents a single search result from BM25 algorithm
type BM25Result struct {
	DocumentID string                 `json:"document_id"`
	Score      float64                `json:"score"`
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata"`
	Highlight  []string               `json:"highlight,omitempty"`
}

// BM25SearchOptions defines search options for BM25 algorithm
type BM25SearchOptions struct {
	TopK       int     `json:"top_k"`
	MinScore   float64 `json:"min_score"`
	Highlight  bool    `json:"highlight"`
	BoostTerms map[string]float64 `json:"boost_terms,omitempty"`
}

// BM25Document represents a document in the BM25 index
type BM25Document struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Terms     []string               `json:"terms"`
	TermFreqs map[string]int         `json:"term_freqs"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
}

// InvertedIndex represents the inverted index structure for BM25
type InvertedIndex struct {
	// term -> document_id -> term_frequency
	TermDocFreq map[string]map[string]int `json:"term_doc_freq"`
	// document_id -> document_length
	DocLengths map[string]int `json:"doc_lengths"`
	// term -> document_count (how many documents contain this term)
	TermDocCount map[string]int `json:"term_doc_count"`
	// total number of documents
	TotalDocs int `json:"total_docs"`
	// average document length
	AvgDocLength float64 `json:"avg_doc_length"`
}

// BM25Parameters contains the tuning parameters for BM25 algorithm
type BM25Parameters struct {
	K1 float64 `json:"k1"` // Controls term frequency saturation (default: 1.2)
	B  float64 `json:"b"`  // Controls field length normalization (default: 0.75)
}

// TokenizerConfig defines configuration for text tokenization
type TokenizerConfig struct {
	Language     string   `json:"language"`      // Language for stopwords (default: "english")
	RemoveStops  bool     `json:"remove_stops"`  // Remove stopwords (default: true)
	MinTermLen   int      `json:"min_term_len"`  // Minimum term length (default: 2)
	MaxTermLen   int      `json:"max_term_len"`  // Maximum term length (default: 50)
	Stemming     bool     `json:"stemming"`      // Enable stemming (default: false)
	CustomStops  []string `json:"custom_stops"`  // Custom stopwords
	CaseSensitive bool    `json:"case_sensitive"` // Case sensitive tokenization (default: false)
}

// BM25Engine defines the interface for BM25 search engine
type BM25Engine interface {
	// AddDocument adds a document to the BM25 index
	AddDocument(ctx context.Context, doc *BM25Document) error
	
	// AddDocuments adds multiple documents to the BM25 index
	AddDocuments(ctx context.Context, docs []*BM25Document) error
	
	// DeleteDocument removes a document from the BM25 index
	DeleteDocument(ctx context.Context, docID string) error
	
	// Search performs BM25 search with the given query
	Search(ctx context.Context, query string, options *BM25SearchOptions) ([]*BM25Result, error)
	
	// UpdateDocument updates an existing document in the index
	UpdateDocument(ctx context.Context, doc *BM25Document) error
	
	// GetDocumentCount returns the total number of documents in the index
	GetDocumentCount() int
	
	// GetTermCount returns the number of unique terms in the index
	GetTermCount() int
	
	// BuildIndex rebuilds the entire inverted index
	BuildIndex(ctx context.Context) error
	
	// GetIndex returns the current inverted index (for debugging/inspection)
	GetIndex() *InvertedIndex
	
	// Clear removes all documents from the index
	Clear(ctx context.Context) error
	
	// GetStats returns engine statistics
	GetStats() *BM25Stats
}

// Tokenizer defines the interface for text tokenization
type Tokenizer interface {
	// Tokenize splits text into terms/tokens
	Tokenize(text string) []string
	
	// TokenizeWithPositions returns tokens with their positions
	TokenizeWithPositions(text string) []TokenPosition
	
	// NormalizeQuery normalizes a search query
	NormalizeQuery(query string) string
}

// TokenPosition represents a token with its position information
type TokenPosition struct {
	Term     string `json:"term"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Position int    `json:"position"`
}

// BM25Config contains configuration for the BM25 engine
type BM25Config struct {
	Parameters BM25Parameters `json:"parameters"`
	Tokenizer  TokenizerConfig `json:"tokenizer"`
	// Index storage backend (memory, redis, etc.)
	Storage BM25StorageConfig `json:"storage"`
}

// BM25StorageConfig defines storage backend configuration
type BM25StorageConfig struct {
	Type     string                 `json:"type"`     // "memory", "redis", "file"
	Settings map[string]interface{} `json:"settings"` // Storage-specific settings
}

// BM25ScoreExplanation provides detailed scoring information
type BM25ScoreExplanation struct {
	DocumentID string               `json:"document_id"`
	Query      string               `json:"query"`
	QueryTerms []string             `json:"query_terms"`
	TermScores map[string]*TermScore `json:"term_scores"`
	TotalScore float64              `json:"total_score"`
}

// TermScore provides detailed scoring for a single term
type TermScore struct {
	Term          string  `json:"term"`
	TermFrequency int     `json:"term_frequency"`
	DocumentFreq  int     `json:"document_frequency"`
	IDF           float64 `json:"idf"`
	Normalization float64 `json:"normalization"`
	Score         float64 `json:"score"`
}

// BM25Stats provides statistics about the BM25 index
type BM25Stats struct {
	TotalDocuments   int       `json:"total_documents"`
	TotalTerms       int       `json:"total_terms"`
	AverageDocLength float64   `json:"average_doc_length"`
	IndexSize        int64     `json:"index_size_bytes"`
	LastUpdated      time.Time `json:"last_updated"`
}