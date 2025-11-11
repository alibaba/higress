package provider

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

const (
	DefaultMemoryKeyPrefix     = "higress-ai-memory:"
	DefaultMemorySummaryPrefix = "higress-ai-memory-summary:"
	DefaultMemoryTTL           = 3600 // 1 hour default TTL for memory entries
	DefaultSummaryMaxLength    = 500  // 默认摘要最大长度（字符数）
	DefaultSummaryLines        = 10   // 默认摘要行数
)

// ContextWithSummary 带摘要的上下文结构
type ContextWithSummary struct {
	Content string `json:"content"` // 完整内容
	Summary string `json:"summary"` // 摘要
}

// ContextCompressionConfig 上下文压缩配置
type ContextCompressionConfig struct {
	// @Title zh-CN 启用上下文压缩
	// @Description zh-CN 是否启用对Agent透明的上下文压缩功能
	Enabled bool `yaml:"enabled" json:"enabled"`
	// @Title zh-CN Redis配置
	// @Description zh-CN Redis服务配置，用于存储压缩的上下文
	Redis *RedisConfig `yaml:"redis" json:"redis"`
	// @Title zh-CN 压缩字节阈值
	// @Description zh-CN 只有当节省的字节数超过此阈值时才进行压缩，默认1000字节
	CompressionBytesThreshold int `yaml:"compressionBytesThreshold" json:"compressionBytesThreshold"`
	// @Title zh-CN 内存条目TTL
	// @Description zh-CN 内存条目的过期时间（秒），默认3600秒（1小时）
	MemoryTTL int `yaml:"memoryTTL" json:"memoryTTL"`
	// @Title zh-CN 摘要生成配置
	// @Description zh-CN 摘要生成方式配置，支持simple（简单提取）和llm（LLM智能摘要）
	SummaryConfig *SummaryConfig `yaml:"summaryConfig" json:"summaryConfig"`
}

// SummaryConfig 摘要生成配置
type SummaryConfig struct {
	// @Title zh-CN 摘要生成方式
	// @Description zh-CN 摘要生成方式：simple（简单文本提取，默认）或llm（使用LLM模型生成智能摘要）
	Method string `yaml:"method" json:"method"` // "simple" or "llm"
	// @Title zh-CN LLM摘要模型
	// @Description zh-CN 用于生成摘要的LLM模型名称（当method为llm时使用）
	LLMModel string `yaml:"llmModel" json:"llmModel"`
	// @Title zh-CN LLM摘要提供商标识
	// @Description zh-CN 用于生成摘要的LLM提供商标识（当method为llm时使用，可选，默认使用当前provider）
	LLMProviderId string `yaml:"llmProviderId" json:"llmProviderId"`
	// @Title zh-CN LLM服务URL
	// @Description zh-CN LLM服务的完整URL（当method为llm时使用，例如：https://api.openai.com/v1/chat/completions）
	LLMServiceUrl string `yaml:"llmServiceUrl" json:"llmServiceUrl"`
	// @Title zh-CN LLM服务Cluster
	// @Description zh-CN LLM服务的Cluster名称（当method为llm时使用，可选，如果配置了llmServiceUrl则不需要）
	LLMServiceCluster string `yaml:"llmServiceCluster" json:"llmServiceCluster"`
	// @Title zh-CN LLM调用超时
	// @Description zh-CN LLM API调用的超时时间（毫秒），默认5000
	LLMTimeout int `yaml:"llmTimeout" json:"llmTimeout"`
	// @Title zh-CN 摘要最大长度
	// @Description zh-CN 摘要的最大长度（字符数），默认500
	MaxLength int `yaml:"maxLength" json:"maxLength"`
	// @Title zh-CN 摘要提示词
	// @Description zh-CN 用于LLM生成摘要的提示词模板（当method为llm时使用）
	PromptTemplate string `yaml:"promptTemplate" json:"promptTemplate"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	// @Title zh-CN 服务名称
	// @Description zh-CN Redis服务的完整FQDN名称
	ServiceName string `yaml:"serviceName" json:"serviceName"`
	// @Title zh-CN 服务端口
	// @Description zh-CN Redis服务端口，默认6379
	ServicePort int `yaml:"servicePort" json:"servicePort"`
	// @Title zh-CN 用户名
	// @Description zh-CN Redis用户名（可选）
	Username string `yaml:"username" json:"username"`
	// @Title zh-CN 密码
	// @Description zh-CN Redis密码（可选）
	Password string `yaml:"password" json:"password"`
	// @Title zh-CN 超时时间
	// @Description zh-CN Redis请求超时时间（毫秒），默认1000
	Timeout int `yaml:"timeout" json:"timeout"`
	// @Title zh-CN 数据库编号
	// @Description zh-CN Redis数据库编号，默认0
	Database int `yaml:"database" json:"database"`
}

// SummaryGenerator 摘要生成器接口
// 用于支持不同的摘要生成方式（简单提取或LLM生成）
type SummaryGenerator interface {
	// GenerateSummary 生成内容摘要
	// ctx: HTTP上下文
	// content: 需要生成摘要的内容
	// 返回: 生成的摘要
	GenerateSummary(ctx wrapper.HttpContext, content string) (string, error)
}

// MemoryService 内存管理服务接口
type MemoryService interface {
	// SaveContext 保存上下文并返回唯一ID
	SaveContext(ctx wrapper.HttpContext, content string) (string, error)
	// ReadContext 根据ID读取上下文
	ReadContext(ctx wrapper.HttpContext, contextId string) (string, error)
	// ReadContextSummary 根据ID读取上下文摘要
	ReadContextSummary(ctx wrapper.HttpContext, contextId string) (string, error)
	// IsEnabled 检查服务是否启用
	IsEnabled() bool
	// SetSummaryGenerator 设置摘要生成器（可选，用于LLM摘要）
	SetSummaryGenerator(generator SummaryGenerator)
}

// redisMemoryService Redis实现的内存管理服务
type redisMemoryService struct {
	config           *ContextCompressionConfig
	redisClient      wrapper.RedisClient
	keyPrefix        string
	summaryGenerator SummaryGenerator // 摘要生成器（可选，用于LLM摘要）
}

// NewMemoryService 创建内存管理服务
func NewMemoryService(config *ContextCompressionConfig) (MemoryService, error) {
	if config == nil || !config.Enabled {
		return &disabledMemoryService{}, nil
	}

	if config.Redis == nil {
		return nil, errors.New("redis configuration is required when context compression is enabled")
	}

	// 设置默认值
	if config.CompressionBytesThreshold == 0 {
		config.CompressionBytesThreshold = 1000
	}
	if config.MemoryTTL == 0 {
		config.MemoryTTL = DefaultMemoryTTL
	}
	if config.Redis.ServicePort == 0 {
		config.Redis.ServicePort = 6379
	}
	if config.Redis.Timeout == 0 {
		config.Redis.Timeout = 1000
	}

	// 创建Redis客户端
	redisClient := wrapper.NewRedisClusterClient(wrapper.FQDNCluster{
		FQDN: config.Redis.ServiceName,
		Port: int64(config.Redis.ServicePort),
	})

	err := redisClient.Init(
		config.Redis.Username,
		config.Redis.Password,
		int64(config.Redis.Timeout),
		wrapper.WithDataBase(config.Redis.Database),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize redis client: %v", err)
	}

	return &redisMemoryService{
		config:      config,
		redisClient: redisClient,
		keyPrefix:   DefaultMemoryKeyPrefix,
	}, nil
}

// IsEnabled 检查服务是否启用
func (s *redisMemoryService) IsEnabled() bool {
	return s.config != nil && s.config.Enabled
}

// SaveContext 保存上下文到Redis，同时生成并保存摘要
func (s *redisMemoryService) SaveContext(ctx wrapper.HttpContext, content string) (string, error) {
	if !s.IsEnabled() {
		return "", errors.New("memory service is not enabled")
	}

	// 生成唯一的context ID
	contextId, err := generateContextId()
	if err != nil {
		return "", fmt.Errorf("failed to generate context id: %v", err)
	}

	key := s.keyPrefix + contextId
	summaryKey := DefaultMemorySummaryPrefix + contextId

	// 生成摘要：优先使用LLM摘要生成器，否则使用简单摘要
	var summary string
	if s.summaryGenerator != nil {
		// 使用LLM生成智能摘要
		llmSummary, err := s.summaryGenerator.GenerateSummary(ctx, content)
		if err != nil {
			log.Warnf("failed to generate LLM summary, falling back to simple summary: %v", err)
			// 降级到简单摘要
			summary = generateSummary(content)
		} else {
			summary = llmSummary
			log.Infof("generated LLM summary for context %s, length: %d", contextId, len(summary))
		}
	} else {
		// 使用简单摘要
		summary = generateSummary(content)
	}

	// 保存完整内容到Redis
	err = s.redisClient.Set(key, content, nil)
	if err != nil {
		log.Errorf("failed to save context to redis: %v", err)
		return "", fmt.Errorf("failed to save context: %v", err)
	}

	// 保存摘要到Redis
	err = s.redisClient.Set(summaryKey, summary, nil)
	if err != nil {
		log.Warnf("failed to save summary to redis: %v, continuing without summary", err)
		// 摘要保存失败不影响主流程
	}

	// 设置过期时间
	if s.config.MemoryTTL > 0 {
		err = s.redisClient.Expire(key, s.config.MemoryTTL, nil)
		if err != nil {
			log.Warnf("failed to set expiration for context %s: %v", contextId, err)
		}
		// 同时设置摘要的过期时间
		err = s.redisClient.Expire(summaryKey, s.config.MemoryTTL, nil)
		if err != nil {
			log.Warnf("failed to set expiration for summary %s: %v", contextId, err)
		}
	}

	log.Infof("saved context %s to redis, content length: %d, summary length: %d", contextId, len(content), len(summary))
	return contextId, nil
}

// SetSummaryGenerator 设置摘要生成器
func (s *redisMemoryService) SetSummaryGenerator(generator SummaryGenerator) {
	s.summaryGenerator = generator
}

// ReadContext 从Redis读取上下文
func (s *redisMemoryService) ReadContext(ctx wrapper.HttpContext, contextId string) (string, error) {
	if !s.IsEnabled() {
		return "", errors.New("memory service is not enabled")
	}

	key := s.keyPrefix + contextId

	var content string
	var readErr error

	// 同步读取Redis
	err := s.redisClient.Get(key, func(response resp.Value) {
		if err := response.Error(); err != nil {
			readErr = fmt.Errorf("redis get failed: %v", err)
			return
		}
		if response.IsNull() {
			readErr = fmt.Errorf("context not found: %s", contextId)
			return
		}
		content = response.String()
	})

	if err != nil {
		return "", fmt.Errorf("failed to read context: %v", err)
	}

	if readErr != nil {
		return "", readErr
	}

	log.Infof("read context %s from redis, content length: %d", contextId, len(content))
	return content, nil
}

// ReadContextSummary 从Redis读取上下文摘要
func (s *redisMemoryService) ReadContextSummary(ctx wrapper.HttpContext, contextId string) (string, error) {
	if !s.IsEnabled() {
		return "", errors.New("memory service is not enabled")
	}

	summaryKey := DefaultMemorySummaryPrefix + contextId

	var summary string
	var readErr error

	// 同步读取Redis摘要
	err := s.redisClient.Get(summaryKey, func(response resp.Value) {
		if err := response.Error(); err != nil {
			readErr = fmt.Errorf("redis get failed: %v", err)
			return
		}
		if response.IsNull() {
			// 如果摘要不存在，尝试读取完整内容并生成摘要
			readErr = fmt.Errorf("summary not found: %s", contextId)
			return
		}
		summary = response.String()
	})

	if err != nil {
		return "", fmt.Errorf("failed to read summary: %v", err)
	}

	if readErr != nil {
		// 如果摘要不存在，尝试读取完整内容并生成摘要（降级策略）
		log.Warnf("summary not found for context %s, generating from full content", contextId)
		fullContent, err := s.ReadContext(ctx, contextId)
		if err != nil {
			return "", fmt.Errorf("failed to read context for summary generation: %v", err)
		}
		summary = generateSummary(fullContent)
		log.Infof("generated summary for context %s, length: %d", contextId, len(summary))
		return summary, nil
	}

	log.Infof("read summary %s from redis, summary length: %d", contextId, len(summary))
	return summary, nil
}

// generateContextId 生成唯一的上下文ID
func generateContextId() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// disabledMemoryService 禁用状态的内存服务
type disabledMemoryService struct{}

func (s *disabledMemoryService) IsEnabled() bool {
	return false
}

func (s *disabledMemoryService) SaveContext(ctx wrapper.HttpContext, content string) (string, error) {
	return "", errors.New("memory service is not enabled")
}

func (s *disabledMemoryService) ReadContext(ctx wrapper.HttpContext, contextId string) (string, error) {
	return "", errors.New("memory service is not enabled")
}

func (s *disabledMemoryService) ReadContextSummary(ctx wrapper.HttpContext, contextId string) (string, error) {
	return "", errors.New("memory service is not enabled")
}

func (s *disabledMemoryService) SetSummaryGenerator(generator SummaryGenerator) {
	// 禁用状态，不设置生成器
}

// ToolContext 工具上下文信息
type ToolContext struct {
	ContextId string     `json:"context_id"`
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
}

// ShouldCompress 判断是否应该压缩
func (s *redisMemoryService) ShouldCompress(contentSize int) bool {
	return contentSize > s.config.CompressionBytesThreshold
}

// ParseContextCompressionConfig 解析上下文压缩配置
func ParseContextCompressionConfig(json gjson.Result) *ContextCompressionConfig {
	if !json.Exists() {
		return nil
	}

	config := &ContextCompressionConfig{
		Enabled: json.Get("enabled").Bool(),
	}

	if !config.Enabled {
		return config
	}

	// 解析Redis配置
	redisJson := json.Get("redis")
	if redisJson.Exists() {
		config.Redis = &RedisConfig{
			ServiceName: redisJson.Get("serviceName").String(),
			ServicePort: int(redisJson.Get("servicePort").Int()),
			Username:    redisJson.Get("username").String(),
			Password:    redisJson.Get("password").String(),
			Timeout:     int(redisJson.Get("timeout").Int()),
			Database:    int(redisJson.Get("database").Int()),
		}
	}

	config.CompressionBytesThreshold = int(json.Get("compressionBytesThreshold").Int())
	config.MemoryTTL = int(json.Get("memoryTTL").Int())

	// 解析摘要配置
	summaryJson := json.Get("summaryConfig")
	if summaryJson.Exists() {
		config.SummaryConfig = &SummaryConfig{
			Method:            summaryJson.Get("method").String(),
			LLMModel:          summaryJson.Get("llmModel").String(),
			LLMProviderId:     summaryJson.Get("llmProviderId").String(),
			LLMServiceUrl:     summaryJson.Get("llmServiceUrl").String(),
			LLMServiceCluster: summaryJson.Get("llmServiceCluster").String(),
			LLMTimeout:        int(summaryJson.Get("llmTimeout").Int()),
			MaxLength:         int(summaryJson.Get("maxLength").Int()),
			PromptTemplate:    summaryJson.Get("promptTemplate").String(),
		}

		// 设置默认值
		if config.SummaryConfig.Method == "" {
			config.SummaryConfig.Method = "simple"
		}
		if config.SummaryConfig.MaxLength == 0 {
			config.SummaryConfig.MaxLength = DefaultSummaryMaxLength
		}
		// PromptTemplate如果为空，会在NewLLMSummaryGenerator中设置默认值
	}

	return config
}

// generateSummary 生成工具输出的摘要
// 摘要策略：
// 1. 如果内容较短（<500字符），直接返回
// 2. 提取前N行关键信息
// 3. 如果包含结构化数据（JSON），提取关键字段
// 4. 保留开头和结尾的重要信息
func generateSummary(content string) string {
	if len(content) == 0 {
		return ""
	}

	// 如果内容较短，直接返回
	if len(content) <= DefaultSummaryMaxLength {
		return content
	}

	// 尝试解析JSON，如果是结构化数据，提取关键信息
	var jsonData interface{}
	if err := json.Unmarshal([]byte(content), &jsonData); err == nil {
		// 是JSON格式，提取关键信息
		return generateJSONSummary(jsonData, DefaultSummaryMaxLength)
	}

	// 非JSON格式，使用行提取策略
	return generateTextSummary(content, DefaultSummaryMaxLength, DefaultSummaryLines)
}

// generateJSONSummary 从JSON数据生成摘要
func generateJSONSummary(data interface{}, maxLength int) string {
	var summary strings.Builder

	switch v := data.(type) {
	case map[string]interface{}:
		// 提取关键字段
		keyFields := []string{"result", "data", "output", "content", "message", "summary", "title", "name"}
		for _, key := range keyFields {
			if val, ok := v[key]; ok {
				valStr := fmt.Sprintf("%v", val)
				if len(valStr) > 0 {
					if summary.Len() > 0 {
						summary.WriteString("; ")
					}
					if len(valStr) > maxLength/2 {
						summary.WriteString(valStr[:maxLength/2])
						summary.WriteString("...")
					} else {
						summary.WriteString(valStr)
					}
					if summary.Len() >= maxLength {
						break
					}
				}
			}
		}
		// 如果还没有足够的内容，添加其他字段
		if summary.Len() < maxLength/2 {
			count := 0
			for k, val := range v {
				if count >= 3 {
					break
				}
				skip := false
				for _, key := range keyFields {
					if k == key {
						skip = true
						break
					}
				}
				if skip {
					continue
				}
				valStr := fmt.Sprintf("%v", val)
				if len(valStr) > 0 {
					if summary.Len() > 0 {
						summary.WriteString("; ")
					}
					summary.WriteString(fmt.Sprintf("%s: %s", k, valStr))
					count++
				}
			}
		}
	case []interface{}:
		// 数组类型，提取前几个元素
		maxItems := 3
		if len(v) < maxItems {
			maxItems = len(v)
		}
		for i := 0; i < maxItems; i++ {
			if i > 0 {
				summary.WriteString("; ")
			}
			itemStr := fmt.Sprintf("%v", v[i])
			if len(itemStr) > maxLength/3 {
				summary.WriteString(itemStr[:maxLength/3])
				summary.WriteString("...")
			} else {
				summary.WriteString(itemStr)
			}
		}
		if len(v) > maxItems {
			summary.WriteString(fmt.Sprintf(" ... (共%d项)", len(v)))
		}
	default:
		// 其他类型，转换为字符串
		contentStr := fmt.Sprintf("%v", v)
		if len(contentStr) > maxLength {
			summary.WriteString(contentStr[:maxLength])
			summary.WriteString("...")
		} else {
			summary.WriteString(contentStr)
		}
	}

	result := summary.String()
	if len(result) == 0 {
		// 如果摘要生成失败，使用文本摘要策略
		return generateTextSummary(fmt.Sprintf("%v", data), maxLength, DefaultSummaryLines)
	}

	if len(result) > maxLength {
		return result[:maxLength] + "..."
	}
	return result
}

// generateTextSummary 从文本内容生成摘要
func generateTextSummary(content string, maxLength int, maxLines int) string {
	lines := strings.Split(content, "\n")

	// 如果行数较少，直接返回前几行
	if len(lines) <= maxLines {
		result := strings.Join(lines, "\n")
		if len(result) > maxLength {
			return result[:maxLength] + "..."
		}
		return result
	}

	// 提取前N行和后M行（保留开头和结尾信息）
	headLines := maxLines / 2
	tailLines := maxLines - headLines

	var summary strings.Builder

	// 添加前几行
	for i := 0; i < headLines && i < len(lines); i++ {
		if summary.Len() > 0 {
			summary.WriteString("\n")
		}
		summary.WriteString(lines[i])
		if summary.Len() >= maxLength/2 {
			break
		}
	}

	// 添加省略标记
	if len(lines) > maxLines {
		summary.WriteString("\n...")
		summary.WriteString(fmt.Sprintf(" (省略 %d 行) ", len(lines)-maxLines))
	}

	// 添加后几行
	startIdx := len(lines) - tailLines
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < len(lines); i++ {
		if summary.Len() >= maxLength {
			break
		}
		summary.WriteString("\n")
		summary.WriteString(lines[i])
	}

	result := summary.String()
	if len(result) > maxLength {
		return result[:maxLength] + "..."
	}
	return result
}
