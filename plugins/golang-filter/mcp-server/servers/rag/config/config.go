package config

// Config 是整个MCP服务器的配置结构
type Config struct {
	RAG       RAGConfig       `json:"rag" yaml:"rag"`
	Embedding EmbeddingConfig `json:"embedding" yaml:"embedding"`
	VectorDB  VectorDBConfig  `json:"vectordb" yaml:"vectordb"`
}

// RAGConfig RAG系统基础配置
type RAGConfig struct {
	Splitter   SplitterConfig `json:"splitter" yaml:"splitter"`
	MaxResults int            `json:"max_results,omitempty" yaml:"max_results,omitempty"`
}

// SplitterConfig 文档分块器配置
type SplitterConfig struct {
	Provider     string `json:"provider" yaml:"provider"` // recursive, character, token
	ChunkSize    int    `json:"chunk_size,omitempty" yaml:"chunk_size,omitempty"`
	ChunkOverlap int    `json:"chunk_overlap,omitempty" yaml:"chunk_overlap,omitempty"`
}

// LLMConfig LLM配置
type LLMConfig struct {
	Provider    string  `json:"provider" yaml:"provider"` // openai, dashscope, qwen
	APIKey      string  `json:"api_key,omitempty" yaml:"api_key"`
	BaseURL     string  `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Model       string  `json:"model" yaml:"model"`
	Temperature float64 `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
}

// EmbeddingConfig 嵌入模型配置
type EmbeddingConfig struct {
	Provider  string `json:"provider" yaml:"provider"` // openai, dashscope
	APIKey    string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	BaseURL   string `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Model     string `json:"model,omitempty" yaml:"model,omitempty"`
	Dimension int    `json:"dimension,omitempty" yaml:"dimension,omitempty"`
}

// VectorDBConfig 向量数据库配置
type VectorDBConfig struct {
	Provider   string `json:"provider" yaml:"provider"` // milvus, qdrant, chroma
	Host       string `json:"host,omitempty" yaml:"host,omitempty"`
	Port       int    `json:"port,omitempty" yaml:"port,omitempty"`
	Database   string `json:"database,omitempty" yaml:"database,omitempty"`
	Collection string `json:"collection,omitempty" yaml:"collection,omitempty"`
	Username   string `json:"username,omitempty" yaml:"username,omitempty"`
	Password   string `json:"password,omitempty" yaml:"password,omitempty"`
}
