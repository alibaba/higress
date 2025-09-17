package config

import (
	"fmt"
)

// Config 是整个MCP服务器的配置结构
type Config struct {
	RAG       RAGConfig       `json:"rag" yaml:"rag"`
	LLM       LLMConfig       `json:"llm" yaml:"llm"`
	Embedding EmbeddingConfig `json:"embedding" yaml:"embedding"`
	VectorDB  VectorDBConfig  `json:"vectordb" yaml:"vectordb"`
	Rerank    RerankConfig    `json:"rerank" yaml:"rerank"`
}

// RAGConfig RAG系统基础配置
type RAGConfig struct {
	KnowledgeBase string         `json:"knowledge_base" yaml:"knowledge_base"`
	Splitter      SplitterConfig `json:"splitter" yaml:"splitter"`
	MaxResults    int            `json:"max_results" yaml:"max_results"`
}

// SplitterConfig 文档分块器配置
type SplitterConfig struct {
	Type         string `json:"type" yaml:"type"` // recursive, character, token
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
	Provider            string `json:"provider" yaml:"provider"` // milvus, qdrant, chroma
	Host                string `json:"host" yaml:"host"`
	Port                int    `json:"port" yaml:"port"`
	Database            string `json:"database" yaml:"database"`
	KnowledgeCollection string `json:"knowledge_collection" yaml:"knowledge_collection"`
	DocumentCollection  string `json:"document_collection" yaml:"document_collection"`
	Username            string `json:"username,omitempty" yaml:"username,omitempty"`
	Password            string `json:"password,omitempty" yaml:"password,omitempty"`
}

// RerankConfig 重排序配置
type RerankConfig struct {
	Provider string `json:"provider" yaml:"provider"` // cohere, bge, jina
	APIKey   string `json:"api_key" yaml:"api_key"`
	Model    string `json:"model" yaml:"model"`
	TopK     int    `json:"top_k,omitempty" yaml:"top_k,omitempty"`
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证LLM配置
	if c.LLM.Provider == "" {
		return fmt.Errorf("llm provider is required")
	}
	if c.LLM.APIKey == "" {
		return fmt.Errorf("llm api_key is required")
	}

	// 验证Embedding配置
	if c.Embedding.Provider == "" {
		return fmt.Errorf("embedding provider is required")
	}
	if c.Embedding.APIKey == "" {
		return fmt.Errorf("embedding api_key is required")
	}

	// 验证VectorDB配置
	if c.VectorDB.Provider == "" {
		return fmt.Errorf("vectordb provider is required")
	}

	// 验证Rerank配置
	if c.Rerank.Provider == "" {
		return fmt.Errorf("rerank provider is required")
	}
	if c.Rerank.APIKey == "" {
		return fmt.Errorf("rerank api_key is required")
	}

	return nil
}
