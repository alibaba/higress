package rag

import (
	"fmt"
	"testing"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"gopkg.in/yaml.v3"
)

func TestRAGConfig_ParseConfig(t *testing.T) {
	config := &config.Config{
		RAG: config.RAGConfig{
			Splitter: config.SplitterConfig{
				Provider:     "nosplitter",
				ChunkSize:    500,
				ChunkOverlap: 50,
			},
			Threshold: 0.5,
			TopK:      5,
		},
		LLM: config.LLMConfig{
			Provider:    "openai",
			APIKey:      "sk-XXX",
			BaseURL:     "https://openrouter.ai/api/v1",
			Model:       "openai/gpt-4o",
			Temperature: 0.5,
			MaxTokens:   2048,
		},
		Embedding: config.EmbeddingConfig{
			Provider:  "dashscope",
			APIKey:    "sk-XXX",
			BaseURL:   "",
			Model:     "text-embedding-v4",
			Dimension: 1024,
		},
		VectorDB: config.VectorDBConfig{
			Provider:   "milvus",
			Host:       "localhost",
			Port:       19530,
			Database:   "default",
			Collection: "test_rag",
			Username:   "",
			Password:   "",
		},
	}
	// 把 config 输出 yaml 格式
	yaml, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("marshal config failed, err: %v", err)
	}
	t.Logf("config yaml: %s", string(yaml))
	fmt.Printf("\n%s", string(yaml))
}
