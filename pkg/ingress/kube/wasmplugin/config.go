// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wasmplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// MCPShardingConfig MCP 分片配置
type MCPShardingConfig struct {
	// 是否启用分片
	Enabled bool `json:"enabled" yaml:"enabled"`

	// 每个分片的最大大小（字节）
	MaxSize int `json:"maxSize" yaml:"maxSize"`

	// 每个分片的最大实例数量
	MaxInstances int `json:"maxInstances" yaml:"maxInstances"`

	// 分组策略
	Strategy string `json:"strategy" yaml:"strategy"`

	// 压缩配置
	Compression MCPCompressionConfig `json:"compression" yaml:"compression"`

	// 配置引用
	ConfigRef MCPConfigRefConfig `json:"configRef" yaml:"configRef"`

	// 监控配置
	Monitoring MCPMonitoringConfig `json:"monitoring" yaml:"monitoring"`

	// 自动重新平衡
	AutoRebalance MCPAutoRebalanceConfig `json:"autoRebalance" yaml:"autoRebalance"`
}

// MCPCompressionConfig 压缩配置
type MCPCompressionConfig struct {
	Enabled   bool   `json:"enabled" yaml:"enabled"`
	Algorithm string `json:"algorithm" yaml:"algorithm"`
	Level     int    `json:"level" yaml:"level"`
}

// MCPConfigRefConfig 配置引用配置
type MCPConfigRefConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Storage string `json:"storage" yaml:"storage"` // configmap, secret
}

// MCPMonitoringConfig 监控配置
type MCPMonitoringConfig struct {
	Enabled        bool          `json:"enabled" yaml:"enabled"`
	SizeThreshold  int           `json:"sizeThreshold" yaml:"sizeThreshold"`
	CheckInterval  time.Duration `json:"checkInterval" yaml:"checkInterval"`
	AlertEnabled   bool          `json:"alertEnabled" yaml:"alertEnabled"`
	MetricsEnabled bool          `json:"metricsEnabled" yaml:"metricsEnabled"`
}

// MCPAutoRebalanceConfig 自动重新平衡配置
type MCPAutoRebalanceConfig struct {
	Enabled          bool          `json:"enabled" yaml:"enabled"`
	Interval         time.Duration `json:"interval" yaml:"interval"`
	ThresholdPercent int           `json:"thresholdPercent" yaml:"thresholdPercent"`
	MinShards        int           `json:"minShards" yaml:"minShards"`
	MaxShards        int           `json:"maxShards" yaml:"maxShards"`
}

// MCPConfig 完整的 MCP 配置
type MCPConfig struct {
	WasmPlugin struct {
		MCP struct {
			Sharding MCPShardingConfig `json:"sharding" yaml:"sharding"`
		} `json:"mcp" yaml:"mcp"`
	} `json:"wasmplugin" yaml:"wasmplugin"`
}

// ConfigManager 配置管理器
type ConfigManager struct {
	client    client.Client
	namespace string
}

// NewConfigManager 创建配置管理器
func NewConfigManager(client client.Client, namespace string) *ConfigManager {
	return &ConfigManager{
		client:    client,
		namespace: namespace,
	}
}

// LoadConfig 加载配置
func (cm *ConfigManager) LoadConfig(ctx context.Context) (*MCPShardingConfig, error) {
	// 尝试从 ConfigMap 加载
	config, err := cm.loadFromConfigMap(ctx)
	if err == nil {
		return config, nil
	}

	// 尝试从环境变量加载
	config, err = cm.loadFromEnv()
	if err == nil {
		return config, nil
	}

	// 使用默认配置
	return cm.defaultConfig(), nil
}

// loadFromConfigMap 从 ConfigMap 加载配置
func (cm *ConfigManager) loadFromConfigMap(ctx context.Context) (*MCPShardingConfig, error) {
	configMap := &corev1.ConfigMap{}
	if err := cm.client.Get(ctx, client.ObjectKey{
		Namespace: cm.namespace,
		Name:      "higress-config",
	}, configMap); err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	configData, ok := configMap.Data["config.yaml"]
	if !ok {
		return nil, fmt.Errorf("config.yaml not found in ConfigMap")
	}

	var config MCPConfig
	if err := yaml.Unmarshal([]byte(configData), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config.WasmPlugin.MCP.Sharding, nil
}

// loadFromEnv 从环境变量加载配置
func (cm *ConfigManager) loadFromEnv() (*MCPShardingConfig, error) {
	config := cm.defaultConfig()

	// 分片启用状态
	if val := os.Getenv("HIGRESS_MCP_SHARDING_ENABLED"); val != "" {
		enabled, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("invalid HIGRESS_MCP_SHARDING_ENABLED: %w", err)
		}
		config.Enabled = enabled
	}

	// 最大大小
	if val := os.Getenv("HIGRESS_MCP_SHARDING_MAX_SIZE"); val != "" {
		maxSize, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid HIGRESS_MCP_SHARDING_MAX_SIZE: %w", err)
		}
		config.MaxSize = maxSize
	}

	// 最大实例数
	if val := os.Getenv("HIGRESS_MCP_SHARDING_MAX_INSTANCES"); val != "" {
		maxInstances, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid HIGRESS_MCP_SHARDING_MAX_INSTANCES: %w", err)
		}
		config.MaxInstances = maxInstances
	}

	// 分组策略
	if val := os.Getenv("HIGRESS_MCP_SHARDING_STRATEGY"); val != "" {
		config.Strategy = val
	}

	// 压缩启用状态
	if val := os.Getenv("HIGRESS_MCP_COMPRESSION_ENABLED"); val != "" {
		enabled, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("invalid HIGRESS_MCP_COMPRESSION_ENABLED: %w", err)
		}
		config.Compression.Enabled = enabled
	}
	
	// 压缩算法
	if val := os.Getenv("HIGRESS_MCP_COMPRESSION_ALGORITHM"); val != "" {
		config.Compression.Algorithm = val
	}
	
	// 压缩级别
	if val := os.Getenv("HIGRESS_MCP_COMPRESSION_LEVEL"); val != "" {
		level, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid HIGRESS_MCP_COMPRESSION_LEVEL: %w", err)
		}
		config.Compression.Level = level
	}

	// 监控启用状态
	if val := os.Getenv("HIGRESS_MCP_MONITORING_ENABLED"); val != "" {
		enabled, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("invalid HIGRESS_MCP_MONITORING_ENABLED: %w", err)
		}
		config.Monitoring.Enabled = enabled
	}

	return config, nil
}

// defaultConfig 默认配置
func (cm *ConfigManager) defaultConfig() *MCPShardingConfig {
	return &MCPShardingConfig{
		Enabled:      true,
		MaxSize:      MaxWasmPluginSize,
		MaxInstances: MaxMCPInstancesPerShard,
		Strategy:     string(GroupByHash),
		Compression: MCPCompressionConfig{
			Enabled:   false,
			Algorithm: "gzip",
			Level:     6,
		},
		ConfigRef: MCPConfigRefConfig{
			Enabled: false,
			Storage: "configmap",
		},
		Monitoring: MCPMonitoringConfig{
			Enabled:        true,
			SizeThreshold:  MaxWasmPluginSize * 80 / 100, // 80% 阈值
			CheckInterval:  time.Minute * 5,
			AlertEnabled:   true,
			MetricsEnabled: true,
		},
		AutoRebalance: MCPAutoRebalanceConfig{
			Enabled:          false,
			Interval:         time.Hour,
			ThresholdPercent: 20,
			MinShards:        1,
			MaxShards:        50,
		},
	}
}

// SaveConfig 保存配置到 ConfigMap
func (cm *ConfigManager) SaveConfig(ctx context.Context, config *MCPShardingConfig) error {
	mcpConfig := MCPConfig{}
	if config == nil {
		// 使用默认配置
		mcpConfig.WasmPlugin.MCP.Sharding = *cm.defaultConfig()
	} else {
		mcpConfig.WasmPlugin.MCP.Sharding = *config
	}

	configData, err := yaml.Marshal(mcpConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "higress-config",
			Namespace: cm.namespace,
		},
		Data: map[string]string{
			"config.yaml": string(configData),
		},
	}

	if err := cm.client.Create(ctx, configMap); err != nil {
		// 如果已存在，则更新
		existingConfigMap := &corev1.ConfigMap{}
		if getErr := cm.client.Get(ctx, client.ObjectKey{
			Namespace: cm.namespace,
			Name:      "higress-config",
		}, existingConfigMap); getErr != nil {
			return fmt.Errorf("failed to get existing ConfigMap: %w", getErr)
		}

		existingConfigMap.Data["config.yaml"] = string(configData)
		if err := cm.client.Update(ctx, existingConfigMap); err != nil {
			return fmt.Errorf("failed to update ConfigMap: %w", err)
		}
	}

	return nil
}

// ValidateConfig 验证配置
func (cm *ConfigManager) ValidateConfig(config *MCPShardingConfig) error {
	if config == nil {
		// nil 配置是合法的，将使用默认配置
		return nil
	}

	if config.MaxSize <= 0 {
		return fmt.Errorf("maxSize must be greater than 0")
	}

	if config.MaxSize > 10*1024*1024 { // 10MB
		return fmt.Errorf("maxSize should not exceed 10MB")
	}

	if config.MaxInstances <= 0 {
		return fmt.Errorf("maxInstances must be greater than 0")
	}

	if config.MaxInstances > 1000 {
		return fmt.Errorf("maxInstances should not exceed 1000")
	}

	validStrategies := []string{
		string(GroupByDomain),
		string(GroupByService),
		string(GroupByHash),
		string(GroupBySize),
	}

	valid := false
	for _, strategy := range validStrategies {
		if config.Strategy == strategy {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid strategy: %s", config.Strategy)
	}

	if config.Compression.Enabled {
		validAlgorithms := []string{"gzip", "zlib"}
		valid := false
		for _, algo := range validAlgorithms {
			if strings.EqualFold(config.Compression.Algorithm, algo) {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid compression algorithm: %s", config.Compression.Algorithm)
		}

		if config.Compression.Level < 1 || config.Compression.Level > 9 {
			return fmt.Errorf("compression level must be between 1 and 9")
		}
	}

	if config.ConfigRef.Enabled {
		if config.ConfigRef.Storage != "configmap" && config.ConfigRef.Storage != "secret" {
			return fmt.Errorf("invalid configRef storage: %s", config.ConfigRef.Storage)
		}
	}

	if config.Monitoring.Enabled {
		if config.Monitoring.SizeThreshold <= 0 {
			return fmt.Errorf("monitoring sizeThreshold must be greater than 0")
		}

		if config.Monitoring.CheckInterval <= 0 {
			return fmt.Errorf("monitoring checkInterval must be greater than 0")
		}
	}

	if config.AutoRebalance.Enabled {
		if config.AutoRebalance.Interval <= 0 {
			return fmt.Errorf("autoRebalance interval must be greater than 0")
		}

		if config.AutoRebalance.ThresholdPercent <= 0 || config.AutoRebalance.ThresholdPercent > 100 {
			return fmt.Errorf("autoRebalance thresholdPercent must be between 1 and 100")
		}

		if config.AutoRebalance.MinShards <= 0 {
			return fmt.Errorf("autoRebalance minShards must be greater than 0")
		}

		if config.AutoRebalance.MaxShards <= config.AutoRebalance.MinShards {
			return fmt.Errorf("autoRebalance maxShards must be greater than minShards")
		}
	}

	return nil
}

// ToMCPControllerOptions 转换为 MCP 控制器选项
func (config *MCPShardingConfig) ToMCPControllerOptions() MCPControllerOptions {
	if config == nil {
		// 如果配置为 nil，返回默认配置对应的控制器选项
		defaultConfig := &MCPShardingConfig{
			Enabled:      true,
			MaxSize:      MaxWasmPluginSize,
			MaxInstances: MaxMCPInstancesPerShard,
			Strategy:     string(GroupByHash),
			Monitoring: MCPMonitoringConfig{
				Enabled: true,
			},
		}
		return defaultConfig.ToMCPControllerOptions()
	}

	return MCPControllerOptions{
		ShardingEnabled:      config.Enabled,
		MaxInstancesPerShard: config.MaxInstances,
		GroupingStrategy:     GroupingStrategy(config.Strategy),
		MonitoringEnabled:    config.Monitoring.Enabled,
	}
}

// GetConfigTemplate 获取配置模板
func GetConfigTemplate() string {
	return `wasmplugin:
  mcp:
    sharding:
      enabled: true
      maxSize: 1048576  # 1MB
      maxInstances: 100
      strategy: "hash"  # hash, domain, service, size
      compression:
        enabled: false
        algorithm: "gzip"
        level: 6
      configRef:
        enabled: false
        storage: "configmap"  # configmap, secret
      monitoring:
        enabled: true
        sizeThreshold: 838860  # 80% of 1MB
        checkInterval: "5m"
        alertEnabled: true
        metricsEnabled: true
      autoRebalance:
        enabled: false
        interval: "1h"
        thresholdPercent: 20
        minShards: 1
        maxShards: 50`
}

// PrintConfig 打印配置（用于调试）
func PrintConfig(config *MCPShardingConfig) {
	configBytes, _ := json.MarshalIndent(config, "", "  ")
	fmt.Printf("MCP Sharding Config:\n%s\n", string(configBytes))
}
