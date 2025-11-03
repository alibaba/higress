package provider

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
)

const (
	DefaultMemoryKeyPrefix = "higress-ai-memory:"
	DefaultMemoryTTL       = 3600 // 1 hour default TTL for memory entries
)

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

// MemoryService 内存管理服务接口
type MemoryService interface {
	// SaveContext 保存上下文并返回唯一ID
	SaveContext(ctx wrapper.HttpContext, content string) (string, error)
	// ReadContext 根据ID读取上下文
	ReadContext(ctx wrapper.HttpContext, contextId string) (string, error)
	// IsEnabled 检查服务是否启用
	IsEnabled() bool
}

// redisMemoryService Redis实现的内存管理服务
type redisMemoryService struct {
	config      *ContextCompressionConfig
	redisClient wrapper.RedisClient
	keyPrefix   string
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

// SaveContext 保存上下文到Redis
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

	// 保存到Redis
	err = s.redisClient.Set(key, content, nil)
	if err != nil {
		log.Errorf("failed to save context to redis: %v", err)
		return "", fmt.Errorf("failed to save context: %v", err)
	}

	// 设置过期时间
	if s.config.MemoryTTL > 0 {
		err = s.redisClient.Expire(key, s.config.MemoryTTL, nil)
		if err != nil {
			log.Warnf("failed to set expiration for context %s: %v", contextId, err)
		}
	}

	log.Infof("saved context %s to redis, content length: %d", contextId, len(content))
	return contextId, nil
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

	return config
}
