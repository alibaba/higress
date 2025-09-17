package vectordb

import (
	"context"
	"testing"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/config"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/rag/schema"
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
		Provider:            PROVIDER_TYPE_MILVUS,
		Host:                "127.0.0.1",
		Port:                19530, // unlikely to be used
		Database:            "default",
		KnowledgeCollection: "knowledge_test",
		DocumentCollection:  "document_test",
	}

	provider, err := NewMilvusProvider(cfg, 128)
	return provider, err
}

func TestCreateKnowledgeNilClient(t *testing.T) {
	p, err := getMilvusProvider()
	if err != nil {
		t.Fatalf("expected error when connecting to unavailable Milvus server, got nil")
	}
	knowledge := schema.Knowledge{
		ID:               "test-id-02",
		Name:             "test-name-02",
		SourceURL:        "test-source-url",
		Status:           "Pending",
		FileSize:         100,
		ChunkCount:       1,
		EnableMultimodel: false,
		Metadata: map[string]interface{}{
			"test_key": "test_value",
		},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CompletedAt: time.Now(),
	}

	err = p.CreateKnowledge(context.Background(), knowledge)
	if err != nil {
		t.Fatalf("create knowledge failed, got nil error %v", err)
	}
}

func TestGetKnowledge(t *testing.T) {
	p, err := getMilvusProvider()
	if err != nil {
		t.Fatalf("expected error when connecting to unavailable Milvus server, got nil")
	}
	knowledge, err := p.GetKnowledge(context.Background(), "test-id-02")
	if err != nil {
		t.Fatalf("get knowledge failed, got nil error %v", err)
	}
	if knowledge.ID != "test-id-02" {
		t.Fatalf("expected knowledge ID to be test-id-02, got %s", knowledge.ID)
	}
}

func TestListKnowledge(t *testing.T) {
	p, err := getMilvusProvider()
	if err != nil {
		t.Fatalf("expected error when connecting to unavailable Milvus server, got nil")
	}
	knowledgeList, err := p.ListKnowledge(context.Background(), 10)
	if err != nil {
		t.Fatalf("list knowledge failed, got nil error %v", err)
	}
	if len(knowledgeList) != 2 {
		t.Fatalf("expected knowledge list length to be 2, got %d", len(knowledgeList))
	}
}
