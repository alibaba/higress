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
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/alibaba/higress/pkg/ingress/kube/common"
	kubeclient "github.com/alibaba/higress/pkg/kube"
)

// 这个文件展示了如何在实际代码中使用增强的 WasmPlugin 控制器

// SetupMCPController 设置 MCP 控制器
func SetupMCPController(mgr manager.Manager, client kubeclient.Client, options common.Options) error {
	// 创建记录器
	recorder := mgr.GetEventRecorderFor("mcp-wasmplugin-controller")

	// 创建配置管理器
	configManager := NewConfigManager(mgr.GetClient(), options.WatchNamespace)

	// 加载配置
	ctx := context.Background()
	config, err := configManager.LoadConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load MCP config: %w", err)
	}

	// 验证配置
	if err := configManager.ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid MCP config: %w", err)
	}

	// 转换为控制器选项
	mcpOptions := config.ToMCPControllerOptions()

	// 创建 MCP 控制器
	mcpController := NewMCPController(client, mgr.GetClient(), recorder, options, mcpOptions)

	// 注册控制器到管理器
	if err := mgr.Add(mcpController); err != nil {
		return fmt.Errorf("failed to add MCP controller to manager: %w", err)
	}

	return nil
}

// MCPControllerRunner 控制器运行器
type MCPControllerRunner struct {
	controller *MCPWasmPluginController
	stopCh     chan struct{}
}

// NewMCPControllerRunner 创建控制器运行器
func NewMCPControllerRunner(controller *MCPWasmPluginController) *MCPControllerRunner {
	return &MCPControllerRunner{
		controller: controller,
		stopCh:     make(chan struct{}),
	}
}

// Start 启动控制器
func (r *MCPControllerRunner) Start(ctx context.Context) error {
	go r.run(ctx)
	return nil
}

// Stop 停止控制器
func (r *MCPControllerRunner) Stop() {
	close(r.stopCh)
}

// run 运行控制器
func (r *MCPControllerRunner) run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 定期协调
			if err := r.reconcileAll(ctx); err != nil {
				fmt.Printf("Failed to reconcile: %v\n", err)
			}
		case <-r.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// reconcileAll 协调所有资源
func (r *MCPControllerRunner) reconcileAll(ctx context.Context) error {
	// 触发协调
	req := reconcile.Request{
		NamespacedName: client.ObjectKey{
			Namespace: r.controller.options.WatchNamespace,
			Name:      MCPPluginName,
		},
	}

	_, err := r.controller.Reconcile(ctx, req)
	return err
}

// ExampleUsage 使用示例
func ExampleUsage() {
	// 这是一个完整的使用示例
	fmt.Println("=== MCP WasmPlugin Controller Example ===")

	// 1. 创建配置
	config := &MCPShardingConfig{
		Enabled:      true,
		MaxSize:      1024 * 1024, // 1MB
		MaxInstances: 100,
		Strategy:     string(GroupByHash),
		Compression: MCPCompressionConfig{
			Enabled:   true,
			Algorithm: "gzip",
			Level:     6,
		},
		ConfigRef: MCPConfigRefConfig{
			Enabled: false,
			Storage: "configmap",
		},
		Monitoring: MCPMonitoringConfig{
			Enabled:        true,
			SizeThreshold:  819200, // 800KB
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

	// 2. 打印配置
	PrintConfig(config)

	// 3. 获取配置模板
	template := GetConfigTemplate()
	fmt.Printf("\nConfig Template:\n%s\n", template)

	// 4. 验证配置
	configManager := &ConfigManager{}
	if err := configManager.ValidateConfig(config); err != nil {
		fmt.Printf("Config validation failed: %v\n", err)
		return
	}

	fmt.Println("Config validation passed!")

	// 5. 转换为控制器选项
	mcpOptions := config.ToMCPControllerOptions()
	fmt.Printf("MCP Controller Options: %+v\n", mcpOptions)

	// 6. 验证控制器选项
	if err := ValidateShardingConfig(mcpOptions); err != nil {
		fmt.Printf("Controller options validation failed: %v\n", err)
		return
	}

	fmt.Println("Controller options validation passed!")

	// 7. 示例 MCP 实例
	// 注意：生产环境应通过Kubernetes Secrets存储敏感信息
	instances := []MCPInstance{
		{
			Name:     "weather-service",
			Endpoint: "https://api.weather.com/mcp",
			Enabled:  true,
			Weight:   1,
			Config: map[string]interface{}{
				"apiKey": "your-api-key",
				"region": "us-east-1",
			},
		},
		{
			Name:     "search-service",
			Endpoint: "https://api.search.com/mcp",
			Enabled:  true,
			Weight:   2,
			Config: map[string]interface{}{
				"index":   "products",
				"timeout": "30s",
			},
		},
		{
			Name:     "notification-service",
			Endpoint: "https://api.notify.com/mcp",
			Enabled:  true,
			Weight:   1,
			Config: map[string]interface{}{
				"channels": []string{"email", "sms"},
				"template": "default",
			},
		},
	}

	// 8. 检查是否需要分片
	if NeedsSharding(instances) {
		fmt.Println("Sharding is required for these instances")
		configSize := CalculateConfigSize(instances)
		fmt.Printf("Config size: %d bytes\n", configSize)
	} else {
		fmt.Println("No sharding needed")
	}

	// 9. 创建分组器并分组
	grouper := NewMCPGrouper(50, GroupByDomain)
	groups := grouper.GroupMCPInstances(instances)
	fmt.Printf("Created %d groups:\n", len(groups))
	for i, group := range groups {
		fmt.Printf("  Group %d: %d instances\n", i, len(group))
		for _, instance := range group {
			fmt.Printf("    - %s (%s)\n", instance.Name, instance.Endpoint)
		}
	}
}

// MonitoringExample 监控示例
func MonitoringExample(controller *MCPWasmPluginController) {
	ctx := context.Background()

	// 获取分片统计信息
	stats, err := controller.GetShardStatistics(ctx)
	if err != nil {
		fmt.Printf("Failed to get shard statistics: %v\n", err)
		return
	}

	fmt.Printf("=== Shard Statistics ===\n")
	fmt.Printf("Total Shards: %d\n", stats.TotalShards)
	fmt.Printf("Total Instances: %d\n", stats.TotalInstances)
	fmt.Printf("Total Size: %d bytes\n", stats.TotalSize)
	fmt.Printf("Average Shard Size: %d bytes\n", stats.AverageShardSize)
	fmt.Printf("Average Instances per Shard: %d\n", stats.AverageInstancesPerShard)
	fmt.Printf("Strategy: %s\n", stats.Strategy)

	// 监控分片
	if err := controller.monitorWasmPlugins(ctx); err != nil {
		fmt.Printf("Failed to monitor WasmPlugins: %v\n", err)
	}
}

// ConfigurationExample 配置示例
func ConfigurationExample(mgr manager.Manager) {
	ctx := context.Background()

	// 创建配置管理器
	configManager := NewConfigManager(mgr.GetClient(), "higress-system")

	// 加载配置
	config, err := configManager.LoadConfig(ctx)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	fmt.Printf("Loaded config: %+v\n", config)

	// 修改配置
	config.MaxInstances = 200
	config.Strategy = string(GroupByDomain)
	config.Compression.Enabled = true

	// 保存配置
	if err := configManager.SaveConfig(ctx, config); err != nil {
		fmt.Printf("Failed to save config: %v\n", err)
		return
	}

	fmt.Println("Config saved successfully!")
}

// DeploymentExample 部署示例
func DeploymentExample() {
	fmt.Println("=== Deployment Example ===")

	// 1. 创建 ConfigMap
	configMapYAML := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: higress-config
  namespace: higress-system
data:
  config.yaml: |
    wasmplugin:
      mcp:
        sharding:
          enabled: true
          maxSize: 1048576  # 1MB
          maxInstances: 100
          strategy: "hash"
          compression:
            enabled: true
            algorithm: "gzip"
            level: 6
          configRef:
            enabled: false
            storage: "configmap"
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
            maxShards: 50
`

	fmt.Printf("1. Apply ConfigMap:\n%s\n", configMapYAML)

	// 2. 创建 McpBridge 资源示例
	mcpBridgeYAML := `
apiVersion: networking.higress.io/v1
kind: McpBridge
metadata:
  name: example-mcp-bridge
  namespace: higress-system
spec:
  registries:
    - type: "nacos"
      name: "nacos-registry"
      domain: "nacos.example.com"
      port: 8848
      protocol: "http"
      nacosNamespaceId: "public"
      nacosNamespace: "default"
      nacosGroups: ["DEFAULT_GROUP"]
      nacosRefreshInterval: 30000
      enableMCPServer: true
      mcpServerBaseUrl: "http://nacos.example.com:8848/mcp"
      mcpServerExportDomains: ["*.example.com"]
      enableScopeMcpServers: true
      allowMcpServers: ["weather-api", "search-api"]
      authSecretName: "nacos-auth"
      metadata:
        region:
          inner_map:
            zone: "us-east-1"
            cluster: "prod"
        
    - type: "consul"
      name: "consul-registry"
      domain: "consul.example.com"
      port: 8500
      protocol: "http"
      consulNamespace: "default"
      consulDatacenter: "dc1"
      consulServiceTag: "mcp"
      consulRefreshInterval: 60000
      enableMCPServer: true
      mcpServerBaseUrl: "http://consul.example.com:8500/mcp"
      mcpServerExportDomains: ["*.consul.local"]
      
    - type: "zookeeper"
      name: "zk-registry"
      domain: "zk.example.com"
      port: 2181
      protocol: "tcp"
      zkServicesPath: ["/services", "/dubbo"]
      enableMCPServer: true
      mcpServerBaseUrl: "http://zk.example.com:8080/mcp"
`

	fmt.Printf("2. Apply McpBridge:\n%s\n", mcpBridgeYAML)

	// 3. 创建 MCP 实例配置 ConfigMap
	mcpInstancesConfigYAML := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-instances-config
  namespace: higress-system
data:
  instances.json: |
    [
      {
        "name": "external-weather-api",
        "endpoint": "https://api.weather.com/mcp",
        "enabled": true,
        "weight": 2,
        "timeout": "30s",
        "retry": 3,
        "config": {
          "apiKey": "weather-api-key",
          "region": "global",
          "features": ["current", "forecast", "alerts"]
        },
        "metadata": {
          "provider": "external",
          "category": "weather",
          "priority": "high"
        }
      },
      {
        "name": "internal-search-api",
        "endpoint": "http://search-api.internal:8080/mcp",
        "enabled": true,
        "weight": 1,
        "timeout": "15s",
        "retry": 2,
        "config": {
          "index": "products",
          "maxResults": 100
        },
        "metadata": {
          "provider": "internal",
          "category": "search",
          "priority": "medium"
        }
      }
    ]
  
  instance-custom-ai.json: |
    {
      "name": "custom-ai-service",
      "endpoint": "https://ai.company.com/mcp",
      "enabled": true,
      "weight": 3,
      "timeout": "60s",
      "retry": 1,
      "config": {
        "model": "gpt-4",
        "temperature": 0.7,
        "maxTokens": 2000
      },
      "metadata": {
        "provider": "openai",
        "category": "ai",
        "priority": "highest"
      }
    }
`

	fmt.Printf("3. Apply MCP Instances Config:\n%s\n", mcpInstancesConfigYAML)

	// 4. 创建 MCP Service 示例
	mcpServiceYAML := `
apiVersion: v1
kind: Service
metadata:
  name: notification-mcp-service
  namespace: higress-system
  labels:
    higress.io/mcp-server: "true"
  annotations:
    higress.io/mcp-protocol: "http"
    higress.io/mcp-weight: "2"
    higress.io/mcp-timeout: "20s"
    higress.io/mcp-retry: "3"
    higress.io/mcp-channels: "email,sms,push"
    higress.io/mcp-template: "default"
spec:
  selector:
    app: notification-service
  ports:
    - name: mcp
      port: 8080
      targetPort: 8080
      protocol: TCP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notification-service
  namespace: higress-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: notification-service
  template:
    metadata:
      labels:
        app: notification-service
    spec:
      containers:
        - name: notification-service
          image: company/notification-mcp:v1.0.0
          ports:
            - containerPort: 8080
          env:
            - name: MCP_SERVER_ENABLED
              value: "true"
            - name: MCP_SERVER_PORT
              value: "8080"
`

	fmt.Printf("4. Apply MCP Service:\n%s\n", mcpServiceYAML)

	// 5. 环境变量配置
	envVars := `
# 环境变量配置示例
HIGRESS_MCP_SHARDING_ENABLED=true
HIGRESS_MCP_SHARDING_MAX_SIZE=1048576
HIGRESS_MCP_SHARDING_MAX_INSTANCES=100
HIGRESS_MCP_SHARDING_STRATEGY=hash
HIGRESS_MCP_COMPRESSION_ENABLED=true
HIGRESS_MCP_MONITORING_ENABLED=true
`

	fmt.Printf("5. Environment Variables:\n%s\n", envVars)

	// 6. 部署命令
	deployCommands := `
# 部署命令示例
kubectl apply -f higress-config.yaml
kubectl apply -f mcp-bridge.yaml
kubectl apply -f mcp-instances-config.yaml
kubectl apply -f mcp-service.yaml
kubectl rollout restart deployment/higress-controller -n higress-system
kubectl logs -f deployment/higress-controller -n higress-system
`

	fmt.Printf("6. Deployment Commands:\n%s\n", deployCommands)

	// 7. 验证命令
	verifyCommands := `
# 验证命令示例
# 查看 McpBridge 资源
kubectl get mcpbridges -n higress-system
kubectl describe mcpbridge example-mcp-bridge -n higress-system

# 查看 MCP 实例配置
kubectl get configmap mcp-instances-config -n higress-system -o yaml

# 查看 MCP 服务
kubectl get services -n higress-system -l higress.io/mcp-server=true

# 查看 WasmPlugin 分片
kubectl get wasmplugins -n higress-system -l higress.io/mcp=true
kubectl describe wasmplugin mcp-bridge-shard-0 -n higress-system

# 查看事件
kubectl get events -n higress-system --field-selector involvedObject.kind=WasmPlugin
kubectl get events -n higress-system --field-selector involvedObject.kind=McpBridge

# 查看控制器日志
kubectl logs -f deployment/higress-controller -n higress-system | grep -i mcp
`

	fmt.Printf("7. Verification Commands:\n%s\n", verifyCommands)
}
