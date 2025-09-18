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
	MaxResults int            `json:"max_results" yaml:"max_results"`
}

// SplitterConfig 文档分块器配置
type SplitterConfig struct {
	Provider     string `json:"provider" yaml:"provider"` // recursive, character, token
	ChunkSize    int    `json:"chunk_size" yaml:"chunk_size"`
	ChunkOverlap int    `json:"chunk_overlap" yaml:"chunk_overlap"`
}

// LLMConfig LLM配置
type LLMConfig struct {
	Provider    string  `json:"provider" yaml:"provider"` // openai, dashscope, qwen
	APIKey      string  `json:"api_key" yaml:"api_key"`
	BaseURL     string  `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Model       string  `json:"model" yaml:"model"`
	Temperature float64 `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
}

// EmbeddingConfig 嵌入模型配置
type EmbeddingConfig struct {
	Provider  string `json:"provider" yaml:"provider"` // openai, dashscope
	APIKey    string `json:"api_key" yaml:"api_key"`
	BaseURL   string `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Model     string `json:"model" yaml:"model"`
	Dimension int    `json:"dimension,omitempty" yaml:"dimension,omitempty"`
}

// VectorDBConfig 向量数据库配置
type VectorDBConfig struct {
	Provider   string `json:"provider" yaml:"provider"` // milvus, qdrant, chroma
	Host       string `json:"host" yaml:"host"`
	Port       int    `json:"port" yaml:"port"`
	Database   string `json:"database" yaml:"database"`
	Collection string `json:"collection" yaml:"collection"`
	Username   string `json:"username,omitempty" yaml:"username,omitempty"`
	Password   string `json:"password,omitempty" yaml:"password,omitempty"`
}
