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

package reconcile

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"path"
	"reflect"
	"sort"
	"sync"
	"time"

	"istio.io/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
	v1 "github.com/alibaba/higress/client/pkg/apis/networking/v1"
	higressmcpserver "github.com/alibaba/higress/pkg/ingress/kube/mcpserver"
	"github.com/alibaba/higress/pkg/kube"
	. "github.com/alibaba/higress/registry"
	"github.com/alibaba/higress/registry/config"
	"github.com/alibaba/higress/registry/consul"
	"github.com/alibaba/higress/registry/direct"
	"github.com/alibaba/higress/registry/eureka"
	"github.com/alibaba/higress/registry/memory"
	"github.com/alibaba/higress/registry/nacos"
	nacosv2 "github.com/alibaba/higress/registry/nacos/v2"
	"github.com/alibaba/higress/registry/zookeeper"
)

const (
	DefaultReadyTimeout = time.Second * 60
	
	// ConfigMap watching and caching
	ConfigMapResyncPeriod = time.Minute * 5
	MaxConfigMapRetries   = 3
	ConfigMapRetryDelay   = time.Second * 5
	
	// Load balancing defaults
	DefaultLoadBalanceMode = apiv1.LoadBalanceModeRoundRobin
	DefaultWeight          = 100
)

type Reconciler struct {
	memory.Cache
	registries       map[string]*apiv1.RegistryConfig
	watchers         map[string]Watcher
	serviceUpdate    func()
	client           kube.Client
	namespace        string
	clusterId        string
	
	// Configuration management using the new abstraction layer
	configManager    *config.Manager
	loadBalancers    map[string]*LoadBalancer
}

func NewReconciler(serviceUpdate func(), client kube.Client, namespace, clusterId string) *Reconciler {
	// Setup configuration manager with the new abstraction layer
	configManager, err := config.SetupConfigManager(client.Kube(), namespace)
	if err != nil {
		log.Errorf("Failed to setup configuration manager: %v", err)
		// Fallback to basic reconciler without config management
		configManager = nil
	}
	
	r := &Reconciler{
		Cache:          memory.NewCache(),
		registries:     make(map[string]*apiv1.RegistryConfig),
		watchers:       make(map[string]Watcher),
		serviceUpdate:  serviceUpdate,
		client:         client,
		namespace:      namespace,
		clusterId:      clusterId,
		configManager:  configManager,
		loadBalancers:  make(map[string]*LoadBalancer),
	}
	
	// Start configuration watcher if manager is available
	if configManager != nil {
		go r.startConfigWatcher()
	}
	
	return r
}

func (r *Reconciler) Reconcile(mcpbridge *v1.McpBridge) error {
	newRegistries := make(map[string]*apiv1.RegistryConfig)
	if mcpbridge != nil {
		for _, registry := range mcpbridge.Spec.Registries {
			newRegistries[path.Join(registry.Type, registry.Name)] = registry
		}
	}
	var wg sync.WaitGroup
	toBeCreated := make(map[string]*apiv1.RegistryConfig)
	toBeUpdated := make(map[string]*apiv1.RegistryConfig)
	toBeDeleted := make(map[string]*apiv1.RegistryConfig)

	for key, newRegistry := range newRegistries {
		if oldRegistry, ok := r.registries[key]; !ok {
			toBeCreated[key] = newRegistry
		} else if reflect.DeepEqual(newRegistry, oldRegistry) {
			continue
		} else {
			toBeUpdated[key] = newRegistry
		}
	}

	for key, oldRegistry := range r.registries {
		if _, ok := newRegistries[key]; !ok {
			toBeDeleted[key] = oldRegistry
		}
	}
	errHappened := false
	log.Infof("ReconcileRegistries, toBeCreated: %d, toBeUpdated: %d, toBeDeleted: %d",
		len(toBeCreated), len(toBeUpdated), len(toBeDeleted))
	for k := range toBeDeleted {
		r.watchers[k].Stop()
		delete(r.registries, k)
		delete(r.watchers, k)
	}
	for k, v := range toBeUpdated {
		r.watchers[k].Stop()
		delete(r.registries, k)
		delete(r.watchers, k)
		watcher, err := r.generateWatcherFromRegistryConfig(v, &wg)
		if err != nil {
			errHappened = true
			log.Errorf("ReconcileRegistries failed, err:%v", err)
			continue
		}

		go watcher.Run()
		r.watchers[k] = watcher
		r.registries[k] = v
	}
	for k, v := range toBeCreated {
		watcher, err := r.generateWatcherFromRegistryConfig(v, &wg)
		if err != nil {
			errHappened = true
			log.Errorf("ReconcileRegistries failed, err:%v", err)
			continue
		}

		go watcher.Run()
		r.watchers[k] = watcher
		r.registries[k] = v
	}
	if errHappened {
		return errors.New("ReconcileRegistries failed, Init Watchers failed")
	}
	var ready = make(chan struct{})
	readyTimer := time.NewTimer(DefaultReadyTimeout)
	go func() {
		wg.Wait()
		ready <- struct{}{}
	}()
	select {
	case <-ready:
	case <-readyTimer.C:
		return errors.New("ReoncileRegistries failed, waiting for ready timeout")
	}
	r.Cache.PurgeStaleService()
	log.Infof("Registries is reconciled")
	return nil
}

func (r *Reconciler) generateWatcherFromRegistryConfig(registry *apiv1.RegistryConfig, wg *sync.WaitGroup) (Watcher, error) {
	var watcher Watcher
	var err error

	authOption, err := r.getAuthOption(registry)
	if err != nil {
		return nil, err
	}

	// Get MCP configuration if specified with fallback mechanism
	selectedInstance, err := r.selectMCPInstance(registry)
	if err != nil {
		log.Warnf("Failed to get MCP config for registry %s: %v, falling back to direct configuration", 
			registry.Name, err)
		// Fallback to direct configuration from registry
		selectedInstance = &apiv1.MCPInstance{
			Domain: registry.Domain,
			Port:   int32(registry.Port),
			Weight: DefaultWeight,
		}
	}

	// Apply selected MCP instance configuration
	if selectedInstance != nil {
		log.Infof("Using MCP instance for registry %s: %s:%d (weight: %d)", 
			registry.Name, selectedInstance.Domain, selectedInstance.Port, selectedInstance.Weight)
		
		// Override domain and port from MCP configuration
		registry.Domain = selectedInstance.Domain
		registry.Port = uint32(selectedInstance.Port)
	}

	switch registry.Type {
	case string(Nacos):
		watcher, err = nacos.NewWatcher(
			r.Cache,
			nacos.WithType(registry.Type),
			nacos.WithName(registry.Name),
			nacos.WithDomain(registry.Domain),
			nacos.WithPort(registry.Port),
			nacos.WithNacosNamespaceId(registry.NacosNamespaceId),
			nacos.WithNacosNamespace(registry.NacosNamespace),
			nacos.WithNacosGroups(registry.NacosGroups),
			nacos.WithNacosRefreshInterval(registry.NacosRefreshInterval),
			nacos.WithAuthOption(authOption),
		)
	case string(Nacos2), string(Nacos3):
		watcher, err = nacosv2.NewWatcher(
			r.Cache,
			nacosv2.WithType(registry.Type),
			nacosv2.WithName(registry.Name),
			nacosv2.WithNacosAddressServer(registry.NacosAddressServer),
			nacosv2.WithDomain(registry.Domain),
			nacosv2.WithPort(registry.Port),
			nacosv2.WithNacosAccessKey(registry.NacosAccessKey),
			nacosv2.WithNacosSecretKey(registry.NacosSecretKey),
			nacosv2.WithNacosNamespaceId(registry.NacosNamespaceId),
			nacosv2.WithNacosNamespace(registry.NacosNamespace),
			nacosv2.WithNacosGroups(registry.NacosGroups),
			nacosv2.WithNacosRefreshInterval(registry.NacosRefreshInterval),
			nacosv2.WithMcpExportDomains(registry.McpServerExportDomains),
			nacosv2.WithMcpBaseUrl(registry.McpServerBaseUrl),
			nacosv2.WithEnableMcpServer(registry.EnableMCPServer),
			nacosv2.WithClusterId(r.clusterId),
			nacosv2.WithNamespace(r.namespace),
			nacosv2.WithAuthOption(authOption),
		)
	case string(Zookeeper):
		watcher, err = zookeeper.NewWatcher(
			r.Cache,
			zookeeper.WithType(registry.Type),
			zookeeper.WithName(registry.Name),
			zookeeper.WithDomain(registry.Domain),
			zookeeper.WithPort(registry.Port),
			zookeeper.WithZkServicesPath(registry.ZkServicesPath),
		)
	case string(Consul):
		watcher, err = consul.NewWatcher(
			r.Cache,
			consul.WithType(registry.Type),
			consul.WithName(registry.Name),
			consul.WithDomain(registry.Domain),
			consul.WithPort(registry.Port),
			consul.WithDatacenter(registry.ConsulDatacenter),
			consul.WithServiceTag(registry.ConsulServiceTag),
			consul.WithRefreshInterval(registry.ConsulRefreshInterval),
			consul.WithAuthOption(authOption),
		)
	case string(Static), string(DNS):
		watcher, err = direct.NewWatcher(
			r.Cache,
			direct.WithType(registry.Type),
			direct.WithName(registry.Name),
			direct.WithDomain(registry.Domain),
			direct.WithPort(registry.Port),
			direct.WithProtocol(registry.Protocol),
			direct.WithSNI(registry.Sni),
		)
	case string(Eureka):
		watcher, err = eureka.NewWatcher(
			r.Cache,
			eureka.WithName(registry.Name),
			eureka.WithDomain(registry.Domain),
			eureka.WithType(registry.Type),
			eureka.WithPort(registry.Port),
		)
	default:
		return nil, errors.New("unsupported registry type:" + registry.Type)
	}

	if err != nil {
		return nil, err
	}

	wg.Add(1)
	var once sync.Once
	watcher.ReadyHandler(func(ready bool) {
		once.Do(func() {
			wg.Done()
			if ready {
				log.Infof("Registry Watcher is ready, type:%s, name:%s", registry.Type, registry.Name)
			}
		})
	})
	watcher.AppendServiceUpdateHandler(r.serviceUpdate)

	return watcher, nil
}

func (r *Reconciler) getAuthOption(registry *apiv1.RegistryConfig) (AuthOption, error) {
	authOption := AuthOption{}
	authSecretName := registry.AuthSecretName

	if len(authSecretName) == 0 {
		return authOption, nil
	}

	authSecret, err := r.client.Kube().CoreV1().Secrets(r.namespace).Get(context.Background(), authSecretName, metav1.GetOptions{})
	if err != nil {
		return authOption, errors.New(fmt.Sprintf("get auth secret %s in namespace %s error:%v", authSecretName, r.namespace, err))
	}

	if nacosUsername, ok := authSecret.Data[AuthNacosUsernameKey]; ok {
		authOption.NacosUsername = string(nacosUsername)
	}

	if nacosPassword, ok := authSecret.Data[AuthNacosPasswordKey]; ok {
		authOption.NacosPassword = string(nacosPassword)
	}

	if consulToken, ok := authSecret.Data[AuthConsulTokenKey]; ok {
		authOption.ConsulToken = string(consulToken)
	}

	if etcdUsername, ok := authSecret.Data[AuthEtcdUsernameKey]; ok {
		authOption.EtcdUsername = string(etcdUsername)
	}

	if etcdPassword, ok := authSecret.Data[AuthEtcdPasswordKey]; ok {
		authOption.EtcdPassword = string(etcdPassword)
	}

	return authOption, nil
}

func (r *Reconciler) GetMcpServers() []*higressmcpserver.McpServer {
	mcpServersFromMcp := r.GetAllConfigs(higressmcpserver.GvkMcpServer)
	servers := make([]*higressmcpserver.McpServer, 0, len(mcpServersFromMcp))
	for _, c := range mcpServersFromMcp {
		if server, ok := c.Spec.(*higressmcpserver.McpServer); ok {
			servers = append(servers, server)
		}
	}
	return servers
}

type RegistryWatcherStatus struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Healthy bool   `json:"healthy"`
	Ready   bool   `json:"ready"`
}

func (r *Reconciler) GetRegistryWatcherStatusList() []RegistryWatcherStatus {
	var registryStatusList []RegistryWatcherStatus
	for key, watcher := range r.watchers {
		_, name := path.Split(key)
		registryStatus := RegistryWatcherStatus{
			Name:    name,
			Type:    watcher.GetRegistryType(),
			Healthy: watcher.IsHealthy(),
			Ready:   watcher.IsReady(),
		}
		registryStatusList = append(registryStatusList, registryStatus)
	}
	return registryStatusList
}

// LoadBalancer provides load balancing functionality for MCP instances
type LoadBalancer struct {
	config    *apiv1.MCPConfig
	roundRobinIndex int
	mutex     sync.Mutex
}

// startConfigWatcher starts configuration watching using the new abstraction layer
func (r *Reconciler) startConfigWatcher() {
	if r.configManager == nil {
		log.Warn("Configuration manager not available, skipping config watching")
		return
	}
	
	handler := func(configRef string, config *apiv1.MCPConfig, eventType config.ConfigEventType) error {
		return r.handleConfigUpdate(configRef, config, eventType)
	}
	
	ctx := context.Background()
	if err := r.configManager.StartWatching(ctx, handler); err != nil {
		log.Errorf("Failed to start configuration watching: %v", err)
	}
}

// handleConfigUpdate processes configuration updates
func (r *Reconciler) handleConfigUpdate(configRef string, mcpConfig *apiv1.MCPConfig, eventType config.ConfigEventType) error {
	switch eventType {
	case config.ConfigEventTypeAdded, config.ConfigEventTypeModified:
		if mcpConfig != nil {
			// Update load balancer
			r.loadBalancers[configRef] = &LoadBalancer{
				config: mcpConfig,
			}
			log.Infof("Updated MCP config for %s with %d instances", configRef, len(mcpConfig.Instances))
		}
	case config.ConfigEventTypeDeleted:
		delete(r.loadBalancers, configRef)
		log.Infof("Removed MCP config for %s", configRef)
	}
	
	// Trigger service update if needed
	if r.serviceUpdate != nil {
		r.serviceUpdate()
	}
	
	return nil
}

// selectMCPInstance selects an MCP instance using the new configuration manager
func (r *Reconciler) selectMCPInstance(registry *apiv1.RegistryConfig) (*apiv1.MCPInstance, error) {
	if registry.McpConfigRef == "" {
		return nil, nil // No MCP config reference
	}
	
	if r.configManager == nil {
		return nil, fmt.Errorf("configuration manager not available")
	}
	
	// Get configuration using the abstraction layer
	ctx := context.Background()
	config, err := r.configManager.GetMCPConfig(ctx, config.ConfigSourceConfigMap, registry.McpConfigRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP config from ConfigMap %s: %w", registry.McpConfigRef, err)
	}
	
	// Get or create load balancer
	loadBalancer, exists := r.loadBalancers[registry.McpConfigRef]
	if !exists {
		loadBalancer = &LoadBalancer{config: config}
		r.loadBalancers[registry.McpConfigRef] = loadBalancer
	}
	
	// Select instance based on load balancing mode
	return loadBalancer.selectInstance(registry.Name), nil
}

// selectInstance selects an instance based on the configured load balancing mode
func (lb *LoadBalancer) selectInstance(registryName string) *apiv1.MCPInstance {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	instances := lb.getHealthyInstances()
	if len(instances) == 0 {
		log.Warnf("No healthy instances available for registry %s", registryName)
		return nil
	}
	
	mode := lb.config.LoadBalanceMode
	if mode == "" {
		mode = apiv1.LoadBalanceModeRoundRobin
	}
	
	switch mode {
	case apiv1.LoadBalanceModeRoundRobin:
		return lb.selectRoundRobin(instances)
	case apiv1.LoadBalanceModeWeighted:
		return lb.selectWeighted(instances)
	case apiv1.LoadBalanceModeRandom:
		return lb.selectRandom(instances)
	default:
		log.Warnf("Unknown load balance mode %s, falling back to round robin", mode)
		return lb.selectRoundRobin(instances)
	}
}

// getHealthyInstances returns instances sorted by priority
func (lb *LoadBalancer) getHealthyInstances() []*apiv1.MCPInstance {
	instances := make([]*apiv1.MCPInstance, len(lb.config.Instances))
	copy(instances, lb.config.Instances)
	
	// Sort by priority (lower priority number = higher priority)
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Priority < instances[j].Priority
	})
	
	return instances
}

// selectRoundRobin implements round-robin load balancing
func (lb *LoadBalancer) selectRoundRobin(instances []*apiv1.MCPInstance) *apiv1.MCPInstance {
	if len(instances) == 0 {
		return nil
	}
	
	instance := instances[lb.roundRobinIndex%len(instances)]
	lb.roundRobinIndex++
	return instance
}

// selectWeighted implements weighted load balancing
func (lb *LoadBalancer) selectWeighted(instances []*apiv1.MCPInstance) *apiv1.MCPInstance {
	if len(instances) == 0 {
		return nil
	}
	
	totalWeight := int32(0)
	for _, instance := range instances {
		weight := instance.Weight
		if weight <= 0 {
			weight = DefaultWeight
		}
		totalWeight += weight
	}
	
	if totalWeight <= 0 {
		return instances[0]
	}
	
	target := rand.Int31n(totalWeight)
	current := int32(0)
	
	for _, instance := range instances {
		weight := instance.Weight
		if weight <= 0 {
			weight = DefaultWeight
		}
		current += weight
		if current > target {
			return instance
		}
	}
	
	return instances[len(instances)-1]
}

// selectRandom implements random load balancing
func (lb *LoadBalancer) selectRandom(instances []*apiv1.MCPInstance) *apiv1.MCPInstance {
	if len(instances) == 0 {
		return nil
	}
	
	return instances[rand.Intn(len(instances))]
}

