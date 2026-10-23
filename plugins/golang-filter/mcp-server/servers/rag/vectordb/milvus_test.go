package vectordb

import (
	"testing"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
)

func TestNewMilvusProvider(t *testing.T) {
	_, err := getMilvusProvider()
	if err != nil {
		t.Fatalf("expected error when connecting to unavailable Milvus server, got nil")
	}
}

func getMilvusProvider() (VectorStoreProvider, error) {
	cfg := &config.VectorDBConfig{
		Provider:   PROVIDER_TYPE_MILVUS,
		Host:       "127.0.0.1",
		Port:       19530, // unlikely to be used
		Database:   "default",
		Collection: "knowledge_test",
	}

	provider, err := NewMilvusProvider(cfg, 128)
	return provider, err
}
