package schema

import "time"

const (
	META_KNOWLEDGE_ID     = "knowledge_id"
	META_KNOWLEDGE_SOURCE = "knowledge_source"
	META_KNOWLEDGE_TITLE  = "knowledge_title"

	DEFAULT_KNOWLEDGE_COLLECTION = "knowledge"
	DEFAULT_DOCUMENT_COLLECTION  = "document"
)

// Document represents a document with its vector embedding and metadata
type Document struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Vector    []float32              `json:"-"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
}

type SearchResult struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"`
}

// Knowledge represents a knowledge entity with associated documents
type Knowledge struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	SourceURL        string                 `json:"source_url"`
	Status           string                 `json:"status"`
	FileSize         int64                  `json:"file_size"`
	ChunkCount       int                    `json:"chunk_count"`
	EnableMultimodel bool                   `json:"enable_multimodel"`
	Metadata         map[string]interface{} `json:"metadata"`
	Documents        []Document             `json:"-"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	CompletedAt      time.Time              `json:"completed_at,omitempty"`
}

// SearchOptions contains options for vector search
type SearchOptions struct {
	TopK      int                    `json:"top_k"`
	Threshold float64                `json:"threshold"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
}
