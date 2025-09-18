package vectordb

import (
	"testing"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
)

// TestNewMilvusProvider verifies that NewMilvusProvider returns an error when the
// Milvus server is not reachable. This keeps the unit test self-contained and
// avoids needing a real Milvus instance running during CI.
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
