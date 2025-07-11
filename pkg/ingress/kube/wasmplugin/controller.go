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
	"strconv"
	"strings"
	"time"

	_struct "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/protobuf/types/known/structpb"
	"istio.io/istio/pkg/kube/controllers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	extensionsv1alpha1 "github.com/alibaba/higress/api/extensions/v1alpha1"
	v1 "github.com/alibaba/higress/client/pkg/apis/extensions/v1alpha1"
	networkingv1 "github.com/alibaba/higress/client/pkg/apis/networking/v1"

	"github.com/alibaba/higress/client/pkg/clientset/versioned"
	informersv1 "github.com/alibaba/higress/client/pkg/informers/externalversions/extensions/v1alpha1"
	listersv1 "github.com/alibaba/higress/client/pkg/listers/extensions/v1alpha1"
	"github.com/alibaba/higress/pkg/ingress/kube/common"
	"github.com/alibaba/higress/pkg/ingress/kube/controller"
	kubeclient "github.com/alibaba/higress/pkg/kube"
)

type WasmPluginController controller.Controller[listersv1.WasmPluginLister]

// MCPWasmPluginController 增强版 WasmPlugin 控制器，支持 MCP 分片
type MCPWasmPluginController struct {
	client.Client
	recorder     record.EventRecorder
	shardManager *MCPShardManager
	lister       listersv1.WasmPluginLister
	informer     cache.SharedIndexInformer
	options      common.Options

	// 配置选项
	shardingEnabled      bool
	maxInstancesPerShard int
	groupingStrategy     GroupingStrategy
	monitoringEnabled    bool
}

// MCPControllerOptions MCP 控制器选项
type MCPControllerOptions struct {
	ShardingEnabled      bool
	MaxInstancesPerShard int
	GroupingStrategy     GroupingStrategy
	MonitoringEnabled    bool
}

// NewController 创建标准 WasmPlugin 控制器（保持向后兼容）
func NewController(client kubeclient.Client, options common.Options) WasmPluginController {
	var informer cache.SharedIndexInformer
	if options.WatchNamespace == "" {
		informer = client.HigressInformer().Extensions().V1alpha1().WasmPlugins().Informer()
	} else {
		informer = client.HigressInformer().InformerFor(&v1.WasmPlugin{}, func(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
			return informersv1.NewWasmPluginInformer(client, options.WatchNamespace, resyncPeriod, nil)
		})
	}
	return controller.NewCommonController("wasmplugin", listersv1.NewWasmPluginLister(informer.GetIndexer()), informer, GetWasmPlugin, options.ClusterId)
}

// NewMCPController 创建支持 MCP 分片的增强版控制器
func NewMCPController(client kubeclient.Client, runtimeClient client.Client, recorder record.EventRecorder, options common.Options, mcpOptions MCPControllerOptions) *MCPWasmPluginController {
	var informer cache.SharedIndexInformer
	if options.WatchNamespace == "" {
		informer = client.HigressInformer().Extensions().V1alpha1().WasmPlugins().Informer()
	} else {
		informer = client.HigressInformer().InformerFor(&v1.WasmPlugin{}, func(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
			return informersv1.NewWasmPluginInformer(client, options.WatchNamespace, resyncPeriod, nil)
		})
	}

	// 创建分片管理器
	shardManager := NewMCPShardManager(
		runtimeClient,
		recorder,
		options.WatchNamespace,
		MCPPluginName,
		mcpOptions.MaxInstancesPerShard,
		mcpOptions.GroupingStrategy,
	)

	return &MCPWasmPluginController{
		Client:               runtimeClient,
		recorder:             recorder,
		shardManager:         shardManager,
		lister:               listersv1.NewWasmPluginLister(informer.GetIndexer()),
		informer:             informer,
		options:              options,
		shardingEnabled:      mcpOptions.ShardingEnabled,
		maxInstancesPerShard: mcpOptions.MaxInstancesPerShard,
		groupingStrategy:     mcpOptions.GroupingStrategy,
		monitoringEnabled:    mcpOptions.MonitoringEnabled,
	}
}

// Reconcile 处理 MCP WasmPlugin 的协调逻辑
func (r *MCPWasmPluginController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	// 如果分片功能未启用，则跳过
	if !r.shardingEnabled {
		return reconcile.Result{}, nil
	}

	// 获取 MCP 实例列表（这里需要根据实际情况实现）
	mcpInstances, err := r.getMCPInstances(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get MCP instances: %w", err)
	}

	// 如果没有 MCP 实例，清理所有分片
	if len(mcpInstances) == 0 {
		if err := r.shardManager.cleanupExistingShards(ctx); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to cleanup shards: %w", err)
		}
		return reconcile.Result{RequeueAfter: time.Minute * 5}, nil
	}

	// 检查是否需要分片
	if r.needsSharding(mcpInstances) {
		return r.handleSharding(ctx, mcpInstances)
	}

	// 检查是否需要监控
	if r.monitoringEnabled {
		if err := r.monitorWasmPlugins(ctx); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to monitor WasmPlugins: %w", err)
		}
	}

	// 标准处理
	return r.handleStandard(ctx, mcpInstances)
}

// getMCPInstances 获取 MCP 实例列表
func (r *MCPWasmPluginController) getMCPInstances(ctx context.Context) ([]MCPInstance, error) {
	var allInstances []MCPInstance
	var errors []string

	// 1. 从 McpBridge 资源中获取 MCP 实例
	mcpBridges, err := r.listMcpBridges(ctx)
	if err != nil {
		errors = append(errors, fmt.Sprintf("failed to list McpBridge resources: %v", err))
	} else {
		for i := range mcpBridges {
			bridge := mcpBridges[i]
			bridgeInstances, err := r.extractMCPInstancesFromBridge(bridge)
			if err != nil {
				// 记录错误但继续处理其他 bridge
				r.recorder.Eventf(
					bridge,
					"Warning",
					"MCPInstanceExtractionFailed",
					"Failed to extract MCP instances from McpBridge %s: %v",
					bridge.Name,
					err,
				)
				errors = append(errors, fmt.Sprintf("failed to extract from McpBridge %s: %v", bridge.Name, err))
				continue
			}
			allInstances = append(allInstances, bridgeInstances...)
		}
	}

	// 2. 从 ConfigMap 中获取额外的 MCP 实例配置
	configMapInstances, err := r.getMCPInstancesFromConfigMap(ctx)
	if err != nil {
		// ConfigMap 是可选的，记录警告但不中断
		r.recorder.Event(
			&v1.WasmPlugin{},
			"Warning",
			"ConfigMapMCPInstancesFailed",
			fmt.Sprintf("Failed to get MCP instances from ConfigMap: %v", err),
		)
		errors = append(errors, fmt.Sprintf("failed to get from ConfigMap: %v", err))
	} else {
		allInstances = append(allInstances, configMapInstances...)
	}

	// 3. 从 Kubernetes Service 中发现 MCP 实例
	serviceInstances, err := r.getMCPInstancesFromServices(ctx)
	if err != nil {
		// Service 发现是可选的，记录警告但不中断
		r.recorder.Event(
			&v1.WasmPlugin{},
			"Warning",
			"ServiceMCPInstancesFailed",
			fmt.Sprintf("Failed to get MCP instances from Services: %v", err),
		)
		errors = append(errors, fmt.Sprintf("failed to get from Services: %v", err))
	} else {
		allInstances = append(allInstances, serviceInstances...)
	}

	// 去重：基于名称和端点去重
	instanceMap := make(map[string]MCPInstance)
	for _, instance := range allInstances {
		key := fmt.Sprintf("%s-%s", instance.Name, instance.Endpoint)
		if existing, exists := instanceMap[key]; exists {
			// 如果已存在，保留权重更高的实例
			if instance.Weight > existing.Weight {
				instanceMap[key] = instance
			}
		} else {
			instanceMap[key] = instance
		}
	}

	// 转换为切片并过滤启用的实例
	enabledInstances := make([]MCPInstance, 0, len(instanceMap))
	for _, instance := range instanceMap {
		if instance.Enabled {
			enabledInstances = append(enabledInstances, instance)
		}
	}

	// 记录统计信息
	r.recorder.Eventf(
		&v1.WasmPlugin{},
		"Normal",
		"MCPInstancesLoaded",
		"Successfully loaded %d MCP instances (%d enabled) from %d McpBridges",
		len(allInstances),
		len(enabledInstances),
		len(mcpBridges),
	)

	// 如果有错误但仍有实例，记录警告
	if len(errors) > 0 && len(enabledInstances) > 0 {
		r.recorder.Eventf(
			&v1.WasmPlugin{},
			"Warning",
			"MCPInstancesPartialFailure",
			"Some MCP instance sources failed but %d instances were loaded successfully. Errors: %s",
			len(enabledInstances),
			strings.Join(errors, "; "),
		)
	}

	// 如果没有任何实例且有错误，返回错误
	if len(enabledInstances) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to load any MCP instances: %s", strings.Join(errors, "; "))
	}

	return enabledInstances, nil
}

// needsSharding 判断是否需要分片
func (r *MCPWasmPluginController) needsSharding(instances []MCPInstance) bool {
	return NeedsSharding(instances)
}

// handleSharding 处理分片逻辑
func (r *MCPWasmPluginController) handleSharding(ctx context.Context, instances []MCPInstance) (reconcile.Result, error) {
	// 创建分片
	if err := r.shardManager.CreateShardedWasmPlugins(ctx, instances); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create shards: %w", err)
	}

	// 记录分片事件
	r.recorder.Eventf(
		&v1.WasmPlugin{},
		"Normal",
		"ShardingCompleted",
		"Successfully created %d shards for %d MCP instances",
		len(r.shardManager.grouper.GroupMCPInstances(instances)),
		len(instances),
	)

	return reconcile.Result{RequeueAfter: time.Minute * 5}, nil
}

// handleStandard 处理标准逻辑
func (r *MCPWasmPluginController) handleStandard(ctx context.Context, instances []MCPInstance) (reconcile.Result, error) {
	// 创建单个 WasmPlugin
	if err := r.createSingleWasmPlugin(ctx, instances); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create single WasmPlugin: %w", err)
	}

	return reconcile.Result{RequeueAfter: time.Minute * 5}, nil
}

// createSingleWasmPlugin 创建单个 WasmPlugin
func (r *MCPWasmPluginController) createSingleWasmPlugin(ctx context.Context, instances []MCPInstance) error {
	pluginName := MCPPluginName

	// 创建 MCP 桥接配置
	config := MCPBridgeConfig{
		Instances: instances,
		Global: struct {
			Timeout     time.Duration `json:"timeout,omitempty"`
			Retry       int           `json:"retry,omitempty"`
			LoadBalance string        `json:"loadBalance,omitempty"`
		}{
			Timeout:     time.Second * 30,
			Retry:       3,
			LoadBalance: "round_robin",
		},
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	// 检查配置大小
	if len(configBytes) > MaxWasmPluginSize {
		return fmt.Errorf("MCP config size (%d bytes) exceeds maximum allowed size (%d bytes)", len(configBytes), MaxWasmPluginSize)
	}

	// 将配置转换为struct.Struct
	var configMap map[string]interface{}
	if err := json.Unmarshal(configBytes, &configMap); err != nil {
		return fmt.Errorf("failed to unmarshal config to map: %w", err)
	}

	configStruct, err := convertMapToStruct(configMap)
	if err != nil {
		return fmt.Errorf("failed to convert config to struct: %w", err)
	}

	// 创建或更新 WasmPlugin
	plugin := &v1.WasmPlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginName,
			Namespace: r.options.WatchNamespace,
			Labels: map[string]string{
				"app":            "higress",
				"plugin":         MCPPluginName,
				"higress.io/mcp": "true",
			},
			Annotations: map[string]string{
				"higress.io/instance-count": fmt.Sprintf("%d", len(instances)),
				"higress.io/config-size":    fmt.Sprintf("%d", len(configBytes)),
				"higress.io/created-by":     "mcp-controller",
				"higress.io/created-at":     time.Now().Format(time.RFC3339),
			},
		},
		Spec: extensionsv1alpha1.WasmPlugin{
			Url:          "oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/mcp-bridge:1.0.0",
			PluginConfig: configStruct,
			Phase:        extensionsv1alpha1.PluginPhase_STATS,
		},
	}

	// 检查是否已存在
	existingPlugin := &v1.WasmPlugin{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: r.options.WatchNamespace,
		Name:      pluginName,
	}, existingPlugin); err != nil {
		// 不存在，创建新的
		if err := r.Client.Create(ctx, plugin); err != nil {
			return fmt.Errorf("failed to create WasmPlugin: %w", err)
		}
		r.recorder.Event(plugin, corev1.EventTypeNormal, "PluginCreated",
			fmt.Sprintf("Created MCP WasmPlugin with %d instances", len(instances)))
	} else {
		// 已存在，更新
		existingPlugin.Spec.Url = plugin.Spec.Url
		existingPlugin.Spec.PluginConfig = plugin.Spec.PluginConfig
		existingPlugin.Spec.Phase = plugin.Spec.Phase
		existingPlugin.Annotations = plugin.Annotations
		if err := r.Client.Update(ctx, existingPlugin); err != nil {
			return fmt.Errorf("failed to update WasmPlugin: %w", err)
		}
		r.recorder.Event(existingPlugin, corev1.EventTypeNormal, "PluginUpdated",
			fmt.Sprintf("Updated MCP WasmPlugin with %d instances", len(instances)))
	}

	return nil
}

// monitorWasmPlugins 监控 WasmPlugin 资源
func (r *MCPWasmPluginController) monitorWasmPlugins(ctx context.Context) error {
	// 获取所有 MCP 相关的 WasmPlugin
	shards, err := GetShardInfo(ctx, r.Client, r.options.WatchNamespace, MCPPluginName)
	if err != nil {
		return fmt.Errorf("failed to get shard info: %w", err)
	}

	// 监控每个分片的状态
	for i := range shards {
		if err := r.monitorShard(ctx, shards[i]); err != nil {
			return fmt.Errorf("failed to monitor shard %s: %w", shards[i].Name, err)
		}
	}

	return nil
}

// monitorShard 监控单个分片
func (r *MCPWasmPluginController) monitorShard(ctx context.Context, shard *v1.WasmPlugin) error {
	// 检查分片大小
	configSize := r.calculateShardSize(shard)
	if configSize > MaxWasmPluginSize*80/100 {
		r.recorder.Eventf(
			shard,
			"Warning",
			"ShardSizeWarning",
			"Shard %s size (%d bytes) is approaching the limit (%d bytes)",
			shard.Name,
			configSize,
			MaxWasmPluginSize,
		)
	}

	// 检查分片配置完整性
	if shard.Spec.PluginConfig == nil {
		r.recorder.Eventf(
			shard,
			"Warning",
			"ShardConfigMissing",
			"Shard %s has no configuration",
			shard.Name,
		)
		return fmt.Errorf("shard %s has no configuration", shard.Name)
	}

	// 验证配置格式
	var config MCPBridgeConfig
	configBytes, err := json.Marshal(shard.Spec.PluginConfig)
	if err != nil {
		r.recorder.Eventf(
			shard,
			"Warning",
			"ShardConfigMarshalFailed",
			"Shard %s failed to marshal configuration: %v",
			shard.Name,
			err,
		)
		return fmt.Errorf("shard %s failed to marshal configuration: %w", shard.Name, err)
	}

	if err := json.Unmarshal(configBytes, &config); err != nil {
		r.recorder.Eventf(
			shard,
			"Warning",
			"ShardConfigInvalid",
			"Shard %s has invalid configuration: %v",
			shard.Name,
			err,
		)
		return fmt.Errorf("shard %s has invalid configuration: %w", shard.Name, err)
	}

	// 检查实例数量
	if len(config.Instances) == 0 {
		r.recorder.Eventf(
			shard,
			"Warning",
			"ShardNoInstances",
			"Shard %s has no MCP instances",
			shard.Name,
		)
	}

	// 检查实例健康状态
	unhealthyInstances := 0
	for _, instance := range config.Instances {
		if !instance.Enabled {
			unhealthyInstances++
		}

		// 检查实例端点格式
		if instance.Endpoint == "" {
			r.recorder.Eventf(
				shard,
				"Warning",
				"ShardInstanceInvalidEndpoint",
				"Shard %s has instance %s with empty endpoint",
				shard.Name,
				instance.Name,
			)
		}
	}

	// 如果超过50%的实例不健康，发出警告
	if len(config.Instances) > 0 && unhealthyInstances > len(config.Instances)/2 {
		r.recorder.Eventf(
			shard,
			"Warning",
			"ShardUnhealthyInstances",
			"Shard %s has %d/%d unhealthy instances",
			shard.Name,
			unhealthyInstances,
			len(config.Instances),
		)
	}

	// 检查插件URL
	if shard.Spec.Url == "" {
		r.recorder.Eventf(
			shard,
			"Warning",
			"ShardNoURL",
			"Shard %s has no plugin URL",
			shard.Name,
		)
	}

	// 检查注解完整性
	if shard.Annotations == nil {
		shard.Annotations = make(map[string]string)
	}

	// 更新监控时间戳
	shard.Annotations["higress.io/last-monitored"] = time.Now().Format(time.RFC3339)

	// 更新分片状态
	if err := r.Client.Update(ctx, shard); err != nil {
		return fmt.Errorf("failed to update shard monitoring timestamp: %w", err)
	}

	// 记录监控完成事件
	r.recorder.Eventf(
		shard,
		"Normal",
		"ShardMonitored",
		"Shard %s monitoring completed: %d instances, %d bytes",
		shard.Name,
		len(config.Instances),
		configSize,
	)

	return nil
}

// calculateShardSize 计算分片大小
func (r *MCPWasmPluginController) calculateShardSize(shard *v1.WasmPlugin) int {
	if shard.Spec.PluginConfig != nil {
		configBytes, _ := json.Marshal(shard.Spec.PluginConfig)
		return len(configBytes)
	}
	return 0
}

// GetShardStatistics 获取分片统计信息
func (r *MCPWasmPluginController) GetShardStatistics(ctx context.Context) (*ShardStatistics, error) {
	shards, err := GetShardInfo(ctx, r.Client, r.options.WatchNamespace, MCPPluginName)
	if err != nil {
		return nil, fmt.Errorf("failed to get shard info: %w", err)
	}

	stats := &ShardStatistics{
		TotalShards: len(shards),
		Strategy:    string(r.groupingStrategy),
	}

	for i := range shards {
		stats.TotalSize += r.calculateShardSize(shards[i])
		if shardSize, ok := shards[i].Annotations["higress.io/shard-size"]; ok {
			var instances int
			fmt.Sscanf(shardSize, "%d", &instances)
			stats.TotalInstances += instances
		}
	}

	if stats.TotalShards > 0 {
		stats.AverageShardSize = stats.TotalSize / stats.TotalShards
		stats.AverageInstancesPerShard = stats.TotalInstances / stats.TotalShards
	}

	return stats, nil
}

// ShardStatistics 分片统计信息
type ShardStatistics struct {
	TotalShards              int    `json:"totalShards"`
	TotalInstances           int    `json:"totalInstances"`
	TotalSize                int    `json:"totalSize"`
	AverageShardSize         int    `json:"averageShardSize"`
	AverageInstancesPerShard int    `json:"averageInstancesPerShard"`
	Strategy                 string `json:"strategy"`
}

// GetWasmPlugin 获取 WasmPlugin（保持向后兼容）
func GetWasmPlugin(lister listersv1.WasmPluginLister, namespacedName types.NamespacedName) (controllers.Object, error) {
	return lister.WasmPlugins(namespacedName.Namespace).Get(namespacedName.Name)
}

// ValidateShardingConfig 验证分片配置
func ValidateShardingConfig(options MCPControllerOptions) error {
	if options.MaxInstancesPerShard <= 0 {
		return fmt.Errorf("maxInstancesPerShard must be greater than 0")
	}

	if options.MaxInstancesPerShard > 500 {
		return fmt.Errorf("maxInstancesPerShard should not exceed 500")
	}

	validStrategies := []GroupingStrategy{GroupByDomain, GroupByService, GroupByHash, GroupBySize}
	valid := false
	for _, strategy := range validStrategies {
		if options.GroupingStrategy == strategy {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid grouping strategy: %s", options.GroupingStrategy)
	}

	return nil
}

// DefaultMCPControllerOptions 默认的 MCP 控制器选项
func DefaultMCPControllerOptions() MCPControllerOptions {
	return MCPControllerOptions{
		ShardingEnabled:      true,
		MaxInstancesPerShard: MaxMCPInstancesPerShard,
		GroupingStrategy:     GroupByHash,
		MonitoringEnabled:    true,
	}
}

// listMcpBridges 列出所有 McpBridge 资源
func (r *MCPWasmPluginController) listMcpBridges(ctx context.Context) ([]*networkingv1.McpBridge, error) {
	var mcpBridgeList networkingv1.McpBridgeList

	listOptions := []client.ListOption{}
	if r.options.WatchNamespace != "" {
		listOptions = append(listOptions, client.InNamespace(r.options.WatchNamespace))
	}

	if err := r.Client.List(ctx, &mcpBridgeList, listOptions...); err != nil {
		r.recorder.Eventf(&v1.WasmPlugin{}, "Warning", "ListMcpBridgeFailed", "Failed to list McpBridge resources in namespace %s: %v", r.options.WatchNamespace, err)
		return nil, fmt.Errorf("failed to list McpBridge resources: %w", err)
	}

	// 直接返回指针切片，避免拷贝锁
	bridges := make([]*networkingv1.McpBridge, 0, len(mcpBridgeList.Items))
	for i := range mcpBridgeList.Items {
		if mcpBridgeList.Items[i] != nil {
			bridges = append(bridges, mcpBridgeList.Items[i])
		}
	}

	return bridges, nil
}

// extractMCPInstancesFromBridge 从 McpBridge 中提取 MCP 实例
func (r *MCPWasmPluginController) extractMCPInstancesFromBridge(bridge *networkingv1.McpBridge) ([]MCPInstance, error) {
	var instances []MCPInstance

	for i, registry := range bridge.Spec.Registries {
		// 检查是否启用了 MCP Server
		if registry.EnableMCPServer != nil && registry.EnableMCPServer.Value {
			instance := MCPInstance{
				Name:    fmt.Sprintf("%s-%s-%d", bridge.Name, registry.Name, i),
				Enabled: true,
				Weight:  1,
				Timeout: time.Second * 30,
				Retry:   3,
			}

			// 构建 MCP Server 端点
			if registry.McpServerBaseUrl != "" {
				instance.Endpoint = registry.McpServerBaseUrl
			} else {
				// 基于域名和端口构建端点
				protocol := "http"
				if registry.Protocol != "" {
					protocol = registry.Protocol
				}
				instance.Endpoint = fmt.Sprintf("%s://%s:%d", protocol, registry.Domain, registry.Port)
			}

			// 设置配置
			instance.Config = make(map[string]interface{})

			// 添加注册中心特定配置
			if registry.Type != "" {
				instance.Config["registryType"] = registry.Type
			}

			if registry.NacosNamespaceId != "" {
				instance.Config["nacosNamespaceId"] = registry.NacosNamespaceId
			}

			if registry.NacosNamespace != "" {
				instance.Config["nacosNamespace"] = registry.NacosNamespace
			}

			if len(registry.NacosGroups) > 0 {
				instance.Config["nacosGroups"] = registry.NacosGroups
			}

			if registry.ConsulNamespace != "" {
				instance.Config["consulNamespace"] = registry.ConsulNamespace
			}

			if registry.ConsulDatacenter != "" {
				instance.Config["consulDatacenter"] = registry.ConsulDatacenter
			}

			if registry.ConsulServiceTag != "" {
				instance.Config["consulServiceTag"] = registry.ConsulServiceTag
			}

			if len(registry.ZkServicesPath) > 0 {
				instance.Config["zkServicesPath"] = registry.ZkServicesPath
			}

			if len(registry.McpServerExportDomains) > 0 {
				instance.Config["exportDomains"] = registry.McpServerExportDomains
			}

			if registry.EnableScopeMcpServers != nil {
				instance.Config["enableScopeMcpServers"] = registry.EnableScopeMcpServers.Value
			}

			if len(registry.AllowMcpServers) > 0 {
				instance.Config["allowMcpServers"] = registry.AllowMcpServers
			}

			// 添加认证信息
			if registry.AuthSecretName != "" {
				instance.Config["authSecretName"] = registry.AuthSecretName
			}

			// 添加 SNI 配置
			if registry.Sni != "" {
				instance.Config["sni"] = registry.Sni
			}

			// 添加元数据
			if len(registry.Metadata) > 0 {
				metadata := make(map[string]interface{})
				for key, innerMap := range registry.Metadata {
					if innerMap != nil && innerMap.InnerMap != nil {
						metadata[key] = innerMap.InnerMap
					}
				}
				instance.Config["metadata"] = metadata
			}

			// 设置元数据
			instance.Metadata = map[string]string{
				"bridge":           bridge.Name,
				"bridge.namespace": bridge.Namespace,
				"registry.name":    registry.Name,
				"registry.type":    registry.Type,
				"registry.domain":  registry.Domain,
				"registry.port":    strconv.Itoa(int(registry.Port)),
			}

			// 添加刷新间隔配置
			if registry.NacosRefreshInterval > 0 {
				instance.Config["nacosRefreshInterval"] = registry.NacosRefreshInterval
				instance.Metadata["nacos.refreshInterval"] = strconv.FormatInt(registry.NacosRefreshInterval, 10)
			}

			if registry.ConsulRefreshInterval > 0 {
				instance.Config["consulRefreshInterval"] = registry.ConsulRefreshInterval
				instance.Metadata["consul.refreshInterval"] = strconv.FormatInt(registry.ConsulRefreshInterval, 10)
			}

			instances = append(instances, instance)
		}
	}

	return instances, nil
}

// getMCPInstancesFromConfigMap 从 ConfigMap 中获取 MCP 实例配置
func (r *MCPWasmPluginController) getMCPInstancesFromConfigMap(ctx context.Context) ([]MCPInstance, error) {
	// 尝试获取 MCP 实例配置的 ConfigMap
	configMap := &corev1.ConfigMap{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: r.options.WatchNamespace,
		Name:      "mcp-instances-config",
	}, configMap); err != nil {
		return nil, fmt.Errorf("failed to get mcp-instances-config ConfigMap: %w", err)
	}

	var instances []MCPInstance

	// 从 ConfigMap 的 data 中解析 MCP 实例
	if instancesData, ok := configMap.Data["instances.json"]; ok {
		var configMapInstances []MCPInstance
		if err := json.Unmarshal([]byte(instancesData), &configMapInstances); err != nil {
			return nil, fmt.Errorf("failed to unmarshal MCP instances from ConfigMap: %w", err)
		}
		instances = append(instances, configMapInstances...)
	}

	// 也支持 YAML 格式
	if instancesData, ok := configMap.Data["instances.yaml"]; ok {
		// 这里可以添加 YAML 解析逻辑
		// 为了简化，我们假设使用 JSON 格式
		_ = instancesData
	}

	// 从 ConfigMap 的每个键值对中解析单个实例
	for key, value := range configMap.Data {
		if strings.HasPrefix(key, "instance-") && strings.HasSuffix(key, ".json") {
			var instance MCPInstance
			if err := json.Unmarshal([]byte(value), &instance); err != nil {
				// 记录警告但继续处理其他实例
				r.recorder.Eventf(
					configMap,
					"Warning",
					"MCPInstanceParseFailed",
					"Failed to parse MCP instance from key %s: %v",
					key,
					err,
				)
				continue
			}
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

// getMCPInstancesFromServices 从 Kubernetes Service 中发现 MCP 实例
func (r *MCPWasmPluginController) getMCPInstancesFromServices(ctx context.Context) ([]MCPInstance, error) {
	var serviceList corev1.ServiceList

	// 查找带有 MCP 标签的服务
	labelSelector := client.MatchingLabels{
		"higress.io/mcp-server": "true",
	}

	listOptions := []client.ListOption{labelSelector}
	if r.options.WatchNamespace != "" {
		listOptions = append(listOptions, client.InNamespace(r.options.WatchNamespace))
	}

	if err := r.Client.List(ctx, &serviceList, listOptions...); err != nil {
		return nil, fmt.Errorf("failed to list MCP services: %w", err)
	}

	var instances []MCPInstance

	for i := range serviceList.Items {
		service := &serviceList.Items[i]

		// 从服务中提取 MCP 实例信息
		instance := MCPInstance{
			Name:    fmt.Sprintf("service-%s", service.Name),
			Enabled: true,
			Weight:  1,
			Timeout: time.Second * 30,
			Retry:   3,
		}

		// 构建端点
		port := int32(80)
		if len(service.Spec.Ports) > 0 {
			port = service.Spec.Ports[0].Port
		}

		protocol := "http"
		if service.Annotations != nil {
			if p, ok := service.Annotations["higress.io/mcp-protocol"]; ok {
				protocol = p
			}
		}

		instance.Endpoint = fmt.Sprintf("%s://%s.%s.svc.cluster.local:%d",
			protocol, service.Name, service.Namespace, port)

		// 从注解中获取配置
		instance.Config = make(map[string]interface{})
		instance.Metadata = make(map[string]string)

		if service.Annotations != nil {
			for key, value := range service.Annotations {
				if strings.HasPrefix(key, "higress.io/mcp-") {
					configKey := strings.TrimPrefix(key, "higress.io/mcp-")
					instance.Config[configKey] = value
					instance.Metadata[configKey] = value
				}
			}
		}

		// 添加服务元数据
		instance.Metadata["service.name"] = service.Name
		instance.Metadata["service.namespace"] = service.Namespace

		instances = append(instances, instance)
	}

	return instances, nil
}

// convertMapToStruct 将map转换为protobuf struct
func convertMapToStruct(m map[string]interface{}) (*_struct.Struct, error) {
	pb, err := structpb.NewStruct(m)
	if err != nil {
		return nil, err
	}

	// 转换为github.com/golang/protobuf/ptypes/struct.Struct
	return &_struct.Struct{
		Fields: convertMapStringValue(pb.Fields),
	}, nil
}

// convertMapStringValue 转换protobuf Value map
func convertMapStringValue(fields map[string]*structpb.Value) map[string]*_struct.Value {
	result := make(map[string]*_struct.Value)
	for k, v := range fields {
		result[k] = convertValue(v)
	}
	return result
}

// convertValue 转换protobuf Value
func convertValue(v *structpb.Value) *_struct.Value {
	if v == nil {
		return nil
	}

	result := &_struct.Value{}
	switch kind := v.Kind.(type) {
	case *structpb.Value_NullValue:
		result.Kind = &_struct.Value_NullValue{NullValue: _struct.NullValue(kind.NullValue)}
	case *structpb.Value_NumberValue:
		result.Kind = &_struct.Value_NumberValue{NumberValue: kind.NumberValue}
	case *structpb.Value_StringValue:
		result.Kind = &_struct.Value_StringValue{StringValue: kind.StringValue}
	case *structpb.Value_BoolValue:
		result.Kind = &_struct.Value_BoolValue{BoolValue: kind.BoolValue}
	case *structpb.Value_StructValue:
		result.Kind = &_struct.Value_StructValue{
			StructValue: &_struct.Struct{
				Fields: convertMapStringValue(kind.StructValue.Fields),
			},
		}
	case *structpb.Value_ListValue:
		values := make([]*_struct.Value, len(kind.ListValue.Values))
		for i, val := range kind.ListValue.Values {
			values[i] = convertValue(val)
		}
		result.Kind = &_struct.Value_ListValue{
			ListValue: &_struct.ListValue{Values: values},
		}
	}
	return result
}

// Start 实现 manager.Runnable 接口
func (r *MCPWasmPluginController) Start(ctx context.Context) error {
	// 启动 informer
	go r.informer.Run(ctx.Done())

	// 等待缓存同步
	if !cache.WaitForCacheSync(ctx.Done(), r.informer.HasSynced) {
		return fmt.Errorf("failed to wait for cache sync")
	}

	// 启动控制器循环 - 这里可以根据需要添加控制逻辑
	<-ctx.Done()
	return nil
}
