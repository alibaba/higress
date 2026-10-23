//go:build higress_integration
// +build higress_integration

package mcptools

import (
	"log"
	"nginx-migration-mcp/internal/rag"
)

// MigrationContext holds the configuration context for migration operations
type MigrationContext struct {
	GatewayName      string
	GatewayNamespace string
	DefaultNamespace string
	DefaultHostname  string
	RoutePrefix      string
	ServicePort      int
	TargetPort       int
	RAGManager       *rag.RAGManager // RAG 管理器
}

// NewDefaultMigrationContext creates a MigrationContext with default values
func NewDefaultMigrationContext() *MigrationContext {
	return &MigrationContext{
		GatewayName:      "higress-gateway",
		GatewayNamespace: "higress-system",
		DefaultNamespace: "default",
		DefaultHostname:  "example.com",
		RoutePrefix:      "nginx-migrated",
		ServicePort:      80,
		TargetPort:       8080,
	}
}

// NewMigrationContextWithRAG creates a MigrationContext with RAG support
func NewMigrationContextWithRAG(ragConfigPath string) *MigrationContext {
	ctx := NewDefaultMigrationContext()

	// 加载 RAG 配置
	config, err := rag.LoadRAGConfig(ragConfigPath)
	if err != nil {
		log.Printf("⚠️  Failed to load RAG config: %v, RAG will be disabled", err)
		config = &rag.RAGConfig{Enabled: false}
	}

	// 创建 RAG 管理器
	ctx.RAGManager = rag.NewRAGManager(config)

	if ctx.RAGManager.IsEnabled() {
		log.Println("✅ MigrationContext: RAG enabled")
	} else {
		log.Println("📖 MigrationContext: RAG disabled, using rule-based approach")
	}

	return ctx
}
