package config

// Config represents the main configuration structure for the MCP server
type Config struct {
	RAG       RAGConfig       `json:"rag" yaml:"rag"`
	LLM       LLMConfig       `json:"llm" yaml:"llm"`
	Embedding EmbeddingConfig `json:"embedding" yaml:"embedding"`
	VectorDB  VectorDBConfig  `json:"vectordb" yaml:"vectordb"`
}

// RAGConfig contains basic configuration for the RAG system
type RAGConfig struct {
	Splitter  SplitterConfig `json:"splitter" yaml:"splitter"`
	Threshold float64        `json:"threshold,omitempty" yaml:"threshold,omitempty"`
	TopK      int            `json:"top_k,omitempty" yaml:"top_k,omitempty"`
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
	Provider  string `json:"provider" yaml:"provider"` // Available options: openai, dashscope
	APIKey    string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	BaseURL   string `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Model     string `json:"model,omitempty" yaml:"model,omitempty"`
	Dimension int    `json:"dimension,omitempty" yaml:"dimension,omitempty"`
}

// VectorDBConfig defines configuration for vector databases
type VectorDBConfig struct {
	Provider   string `json:"provider" yaml:"provider"` // Available options: milvus, qdrant, chroma
	Host       string `json:"host,omitempty" yaml:"host,omitempty"`
	Port       int    `json:"port,omitempty" yaml:"port,omitempty"`
	Database   string `json:"database,omitempty" yaml:"database,omitempty"`
	Collection string `json:"collection,omitempty" yaml:"collection,omitempty"`
	Username   string `json:"username,omitempty" yaml:"username,omitempty"`
	Password   string `json:"password,omitempty" yaml:"password,omitempty"`
}
