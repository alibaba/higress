// Package rag 提供 RAG（检索增强生成）配置管理
package rag

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// RAGConfig RAG 完整配置
type RAGConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"` // RAG 功能总开关

	// API 配置
	Provider        string `json:"provider" yaml:"provider"`                   // 服务提供商: "bailian"
	Endpoint        string `json:"endpoint" yaml:"endpoint"`                   // API 端点（如 bailian.cn-beijing.aliyuncs.com）
	WorkspaceID     string `json:"workspace_id" yaml:"workspace_id"`           // 业务空间 ID（百炼必需）
	AccessKeyID     string `json:"access_key_id" yaml:"access_key_id"`         // 阿里云 AccessKey ID
	AccessKeySecret string `json:"access_key_secret" yaml:"access_key_secret"` // 阿里云 AccessKey Secret
	KnowledgeBaseID string `json:"knowledge_base_id" yaml:"knowledge_base_id"` // 知识库 ID（IndexId）

	// 上下文配置
	ContextMode      string `json:"context_mode" yaml:"context_mode"`             // full | summary | highlights
	MaxContextLength int    `json:"max_context_length" yaml:"max_context_length"` // 最大上下文长度（字符数）

	// 检索配置
	DefaultTopK         int     `json:"default_top_k" yaml:"default_top_k"`               // 默认返回文档数量
	SimilarityThreshold float64 `json:"similarity_threshold" yaml:"similarity_threshold"` // 相似度阈值

	// 缓存配置
	EnableCache  bool `json:"enable_cache" yaml:"enable_cache"`     // 是否启用缓存
	CacheTTL     int  `json:"cache_ttl" yaml:"cache_ttl"`           // 缓存过期时间（秒）
	CacheMaxSize int  `json:"cache_max_size" yaml:"cache_max_size"` // 最大缓存条目数

	// 性能配置
	Timeout    int `json:"timeout" yaml:"timeout"`         // 请求超时时间（秒）
	MaxRetries int `json:"max_retries" yaml:"max_retries"` // 最大重试次数
	RetryDelay int `json:"retry_delay" yaml:"retry_delay"` // 重试间隔（秒）

	// 降级策略
	FallbackOnError bool `json:"fallback_on_error" yaml:"fallback_on_error"` // RAG 失败时是否降级

	// 工具级别配置（核心功能）
	Tools map[string]*ToolConfig `json:"tools" yaml:"tools"`

	// 调试模式
	Debug      bool `json:"debug" yaml:"debug"`             // 是否启用调试日志
	LogQueries bool `json:"log_queries" yaml:"log_queries"` // 是否记录所有查询
}

// ToolConfig 工具级别的 RAG 配置
type ToolConfig struct {
	UseRAG      bool   `json:"use_rag" yaml:"use_rag"`           // 是否使用 RAG
	ContextMode string `json:"context_mode" yaml:"context_mode"` // 上下文模式（覆盖全局配置）
	TopK        int    `json:"top_k" yaml:"top_k"`               // 返回文档数量（覆盖全局配置）
}

// LoadRAGConfig 从配置文件加载 RAG 配置
// 注意：需要安装 YAML 库支持
// 运行：go get gopkg.in/yaml.v3
//
// 临时实现：使用 JSON 格式配置文件
func LoadRAGConfig(configPath string) (*RAGConfig, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 简单的 YAML 到 JSON 转换（仅支持基本格式）
	// 在生产环境中应使用真正的 YAML 解析器
	jsonData := simpleYAMLToJSON(string(data))

	// 解析 JSON
	var wrapper struct {
		RAG *RAGConfig `json:"rag"`
	}

	if err := json.Unmarshal([]byte(jsonData), &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if wrapper.RAG == nil {
		return nil, fmt.Errorf("missing 'rag' section in config")
	}

	config := wrapper.RAG

	// 展开环境变量
	config.AccessKeyID = expandEnvVar(config.AccessKeyID)
	config.AccessKeySecret = expandEnvVar(config.AccessKeySecret)
	config.KnowledgeBaseID = expandEnvVar(config.KnowledgeBaseID)
	config.WorkspaceID = expandEnvVar(config.WorkspaceID)

	// 设置默认值
	setDefaults(config)

	return config, nil
}

// simpleYAMLToJSON 简单的 YAML 到 JSON 转换
// 注意：这是一个临时实现，仅支持基本的 YAML 格式
// 生产环境请使用 gopkg.in/yaml.v3
func simpleYAMLToJSON(yamlContent string) string {
	trimmed := strings.TrimSpace(yamlContent)

	// 如果内容看起来像 JSON，直接返回
	if strings.HasPrefix(trimmed, "{") {
		return yamlContent
	}

	// 否则返回默认禁用配置
	return `{"rag": {"enabled": false}}`
}

// expandEnvVar 展开环境变量 ${VAR_NAME}
func expandEnvVar(value string) string {
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		varName := value[2 : len(value)-1]
		return os.Getenv(varName)
	}
	return value
}

// setDefaults 设置默认值
func setDefaults(config *RAGConfig) {
	if config.ContextMode == "" {
		config.ContextMode = "full"
	}
	if config.MaxContextLength == 0 {
		config.MaxContextLength = 4000
	}
	if config.DefaultTopK == 0 {
		config.DefaultTopK = 3
	}
	if config.SimilarityThreshold == 0 {
		config.SimilarityThreshold = 0.7
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 3600
	}
	if config.CacheMaxSize == 0 {
		config.CacheMaxSize = 1000
	}
	if config.Timeout == 0 {
		config.Timeout = 10
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1
	}

	// 为每个工具配置设置默认值
	for _, toolConfig := range config.Tools {
		if toolConfig.ContextMode == "" {
			toolConfig.ContextMode = config.ContextMode
		}
		if toolConfig.TopK == 0 {
			toolConfig.TopK = config.DefaultTopK
		}
	}
}

// GetToolConfig 获取指定工具的配置
func (c *RAGConfig) GetToolConfig(toolName string) *ToolConfig {
	if toolConfig, ok := c.Tools[toolName]; ok {
		return toolConfig
	}
	return nil
}

// IsToolRAGEnabled 检查指定工具是否启用 RAG
func (c *RAGConfig) IsToolRAGEnabled(toolName string) bool {
	if !c.Enabled {
		return false
	}

	toolConfig := c.GetToolConfig(toolName)
	if toolConfig == nil {
		// 没有工具级配置，使用全局配置
		return c.Enabled
	}

	return toolConfig.UseRAG
}
