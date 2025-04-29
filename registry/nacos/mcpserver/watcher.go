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

package mcpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
	"github.com/alibaba/higress/pkg/common"
	common2 "github.com/alibaba/higress/pkg/ingress/kube/common"
	provider "github.com/alibaba/higress/registry"
	"github.com/alibaba/higress/registry/memory"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"go.uber.org/atomic"
	"istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/log"
	"istio.io/istio/pkg/util/sets"
)

const (
	DefaultInitTimeout          = time.Second * 10
	DefaultNacosTimeout         = 5000
	DefaultNacosLogLevel        = "info"
	DefaultNacosLogDir          = "/var/log/nacos/log/mcp/log"
	DefaultNacosCacheDir        = "/var/log/nacos/log/mcp/cache"
	DefaultNacosNotLoadCache    = true
	DefaultNacosLogMaxAge       = 3
	DefaultRefreshInterval      = time.Second * 30
	DefaultRefreshIntervalLimit = time.Second * 10
	DefaultFetchPageSize        = 50
	DefaultJoiner               = "@@"
	NacosV3LabelKey             = "isV3"
)

var mcpServerLog = log.RegisterScope("McpServer", "Nacos Mcp Server Watcher process.")

type watcher struct {
	provider.BaseWatcher
	apiv1.RegistryConfig
	watchingConfig         map[string]bool
	watchingConfigRefs     map[string]sets.Set[string]
	configToConfigListener map[string]*MultiConfigListener
	serviceCache           map[string]*ServiceCache
	configToService        map[string]string
	credentialKeyToName    map[string]map[string]string
	RegistryType           provider.ServiceRegistryType
	Status                 provider.WatcherStatus
	configClient           config_client.IConfigClient
	serverConfig           []constant.ServerConfig
	cache                  memory.Cache
	mutex                  *sync.Mutex
	subMutex               *sync.Mutex
	callbackMutex          *sync.Mutex
	stop                   chan struct{}
	isStop                 bool
	updateCacheWhenEmpty   bool
	nacosClientConfig      *constant.ClientConfig
	namespace              string
	clusterId              string
}

type WatcherOption func(w *watcher)

func NewWatcher(cache memory.Cache, opts ...WatcherOption) (provider.Watcher, error) {
	w := &watcher{
		watchingConfig:         make(map[string]bool),
		configToService:        make(map[string]string),
		watchingConfigRefs:     make(map[string]sets.Set[string]),
		configToConfigListener: make(map[string]*MultiConfigListener),
		credentialKeyToName:    make(map[string]map[string]string),
		serviceCache:           map[string]*ServiceCache{},
		RegistryType:           "nacos3",
		Status:                 provider.UnHealthy,
		cache:                  cache,
		mutex:                  &sync.Mutex{},
		subMutex:               &sync.Mutex{},
		callbackMutex:          &sync.Mutex{},
		stop:                   make(chan struct{}),
	}

	w.NacosRefreshInterval = int64(DefaultRefreshInterval)

	for _, opt := range opts {
		opt(w)
	}

	if w.NacosNamespace == "" {
		w.NacosNamespace = w.NacosNamespaceId
	}

	mcpServerLog.Infof("new nacos mcp server watcher with config Name:%s", w.Name)

	w.nacosClientConfig = constant.NewClientConfig(
		constant.WithTimeoutMs(DefaultNacosTimeout),
		constant.WithLogLevel(DefaultNacosLogLevel),
		constant.WithLogDir(DefaultNacosLogDir),
		constant.WithCacheDir(DefaultNacosCacheDir),
		constant.WithNotLoadCacheAtStart(DefaultNacosNotLoadCache),
		constant.WithLogRollingConfig(&constant.ClientLogRollingConfig{
			MaxAge: DefaultNacosLogMaxAge,
		}),
		constant.WithUpdateCacheWhenEmpty(w.updateCacheWhenEmpty),
		constant.WithNamespaceId(w.NacosNamespaceId),
		constant.WithAccessKey(w.NacosAccessKey),
		constant.WithSecretKey(w.NacosSecretKey),
	)

	initTimer := time.NewTimer(DefaultInitTimeout)
	w.serverConfig = []constant.ServerConfig{
		*constant.NewServerConfig(w.Domain, uint64(w.Port)),
	}

	success := make(chan struct{})
	go func() {
		configClient, err := clients.NewConfigClient(vo.NacosClientParam{
			ClientConfig:  w.nacosClientConfig,
			ServerConfigs: w.serverConfig,
		})
		if err == nil {
			w.configClient = configClient
			close(success)
		} else {
			mcpServerLog.Errorf("can not create naming client, err:%v", err)
		}
	}()

	select {
	case <-initTimer.C:
		return nil, errors.New("new nacos mcp server watcher timeout")
	case <-success:
		return w, nil
	}
}

func WithNacosAddressServer(nacosAddressServer string) WatcherOption {
	return func(w *watcher) {
		w.NacosAddressServer = nacosAddressServer
	}
}

func WithNacosAccessKey(nacosAccessKey string) WatcherOption {
	return func(w *watcher) {
		w.NacosAccessKey = nacosAccessKey
	}
}

func WithNacosSecretKey(nacosSecretKey string) WatcherOption {
	return func(w *watcher) {
		w.NacosSecretKey = nacosSecretKey
	}
}

func WithNacosNamespaceId(nacosNamespaceId string) WatcherOption {
	return func(w *watcher) {
		if nacosNamespaceId == "" {
			w.NacosNamespaceId = "nacos-default-mcp"
		} else {
			w.NacosNamespaceId = nacosNamespaceId
		}
	}
}

func WithNacosNamespace(nacosNamespace string) WatcherOption {
	return func(w *watcher) {
		w.NacosNamespace = nacosNamespace
	}
}

func WithNacosGroups(nacosGroups []string) WatcherOption {
	return func(w *watcher) {
		if len(nacosGroups) == 0 {
			w.NacosGroups = []string{"mcp-server"}
		} else {
			w.NacosGroups = nacosGroups
		}
	}
}

func WithNacosRefreshInterval(refreshInterval int64) WatcherOption {
	return func(w *watcher) {
		if refreshInterval < int64(DefaultRefreshIntervalLimit) {
			refreshInterval = int64(DefaultRefreshIntervalLimit)
		}
		w.NacosRefreshInterval = refreshInterval
	}
}

func WithType(t string) WatcherOption {
	return func(w *watcher) {
		w.Type = t
	}
}

func WithName(name string) WatcherOption {
	return func(w *watcher) {
		w.Name = name
	}
}

func WithDomain(domain string) WatcherOption {
	return func(w *watcher) {
		w.Domain = domain
	}
}

func WithPort(port uint32) WatcherOption {
	return func(w *watcher) {
		w.Port = port
	}
}

func WithMcpExportDomains(exportDomains []string) WatcherOption {
	return func(w *watcher) {
		w.McpServerExportDomains = exportDomains
	}
}

func WithMcpBaseUrl(url string) WatcherOption {
	return func(w *watcher) {
		w.McpServerBaseUrl = url
	}
}

func WithEnableMcpServer(enable *wrappers.BoolValue) WatcherOption {
	return func(w *watcher) {
		w.EnableMCPServer = enable
	}
}

func WithNamespace(ns string) WatcherOption {
	return func(w *watcher) {
		w.namespace = ns
	}
}

func WithClusterId(id string) WatcherOption {
	return func(w *watcher) {
		w.clusterId = id
	}
}

func (w *watcher) Run() {
	ticker := time.NewTicker(time.Duration(w.NacosRefreshInterval))
	defer ticker.Stop()
	w.Status = provider.ProbeWatcherStatus(w.Domain, strconv.FormatUint(uint64(w.Port), 10))
	err := w.fetchAllMcpConfig()
	if err != nil {
		mcpServerLog.Errorf("first fetch mcp server config failed,  err:%v", err)
	} else {
		w.Ready(true)
	}
	for {
		select {
		case <-ticker.C:
			err := w.fetchAllMcpConfig()
			if err != nil {
				mcpServerLog.Errorf("fetch mcp server config failed, err:%v", err)
			} else {
				w.Ready(true)
			}
		case <-w.stop:
			return
		}
	}
}

func (w *watcher) fetchAllMcpConfig() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.isStop {
		return nil
	}
	fetchedConfigs := make(map[string]bool)
	var tries int
	isV3 := true
	if w.EnableMCPServer != nil {
		isV3 = w.EnableMCPServer.GetValue()
	}
	for _, groupName := range w.NacosGroups {
		for page := 1; ; page++ {
			ss, err := w.configClient.SearchConfig(vo.SearchConfigParam{
				Group:    groupName,
				Search:   "blur",
				PageNo:   page,
				PageSize: DefaultFetchPageSize,
				IsV3:     isV3,
			})
			if err != nil {
				if tries > 10 {
					return err
				}
				mcpServerLog.Errorf("fetch nacos config list failed, err:%v, pageNo:%d", err, page)
				page--
				tries++
				continue
			}
			for _, item := range ss.PageItems {
				fetchedConfigs[groupName+DefaultJoiner+item.DataId] = true
			}
			if len(ss.PageItems) < DefaultFetchPageSize {
				break
			}
		}
	}

	for key := range w.watchingConfig {
		if _, exist := fetchedConfigs[key]; !exist {
			s := strings.Split(key, DefaultJoiner)
			err := w.unsubscribe(s[0], s[1])
			if err != nil {
				return err
			}
			delete(w.watchingConfig, key)
		}
	}

	wg := sync.WaitGroup{}
	subscribeFailed := atomic.NewBool(false)
	watchingKeys := make(chan string, len(fetchedConfigs))
	for key := range fetchedConfigs {
		s := strings.Split(key, DefaultJoiner)
		if _, exist := w.watchingConfig[key]; !exist {
			wg.Add(1)
			go func(k string) {
				err := w.subscribe(s[0], s[1])
				if err != nil {
					subscribeFailed.Store(true)
					mcpServerLog.Errorf("subscribe failed, group: %v, service: %v, errors: %v", s[0], s[1], err)
				} else {
					watchingKeys <- k
				}
				wg.Done()
			}(key)
		}
	}
	wg.Wait()
	close(watchingKeys)
	for key := range watchingKeys {
		w.watchingConfig[key] = true
	}
	if subscribeFailed.Load() {
		return errors.New("subscribe services failed")
	}
	return nil
}

func (w *watcher) unsubscribe(groupName string, dataId string) error {
	mcpServerLog.Infof("unsubscribe mcp server, groupName:%s, dataId:%s", groupName, dataId)
	defer w.UpdateService()

	err := w.configClient.CancelListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  groupName,
	})
	if err != nil {
		mcpServerLog.Errorf("unsubscribe mcp server error:%v, groupName:%s, dataId:%s", err, groupName, dataId)
		return err
	}
	key := strings.Join([]string{w.Name, w.NacosNamespace, groupName, dataId}, DefaultJoiner)
	w.configToConfigListener[key].Stop()
	delete(w.watchingConfigRefs, key)
	delete(w.configToConfigListener, key)
	// remove service for this config
	configKey := strings.Join([]string{groupName, dataId}, DefaultJoiner)
	svcInfo := w.configToService[configKey]
	split := strings.Split(svcInfo, DefaultJoiner)
	svcNamespace := split[0]
	svcGroup := split[1]
	svcName := split[2]
	if w.serviceCache[svcNamespace] != nil {
		err = w.serviceCache[svcNamespace].RemoveListener(svcGroup, svcName, configKey)
		if err != nil {
			mcpServerLog.Errorf("remove service listener error:%v, groupName:%s, dataId:%s", err, groupName, dataId)
		}
	}
	delete(w.configToService, configKey)

	w.cache.UpdateConfigCache(config.GroupVersionKind{}, key, nil, true)
	return nil
}

func (w *watcher) subscribe(groupName string, dataId string) error {
	mcpServerLog.Infof("subscribe mcp server, groupName:%s, dataId:%s", groupName, dataId)
	// first we get this config and callback manually
	content, err := w.configClient.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  groupName,
	})
	if err != nil {
		mcpServerLog.Errorf("get config %s/%s err: %v", groupName, dataId, err)
	} else {
		w.getConfigCallback(w.NacosNamespace, groupName, dataId, content)
	}
	// second, we set callback for this config
	err = w.configClient.ListenConfig(vo.ConfigParam{
		DataId:   dataId,
		Group:    groupName,
		OnChange: w.getConfigCallback,
	})
	if err != nil {
		mcpServerLog.Errorf("subscribe mcp server error:%v, groupName:%s, dataId:%s", err, groupName, dataId)
		return err
	}
	return nil
}

func (w *watcher) getConfigCallback(namespace, group, dataId, data string) {
	mcpServerLog.Infof("get config callback, namespace:%s, groupName:%s, dataId:%s", namespace, group, dataId)

	if data == "" {
		return
	}

	key := strings.Join([]string{w.Name, w.NacosNamespace, group, dataId}, DefaultJoiner)
	routeName := fmt.Sprintf("%s-%s-%s", provider.IstioMcpAutoGeneratedHttpRouteName, group, strings.TrimSuffix(dataId, ".json"))

	mcpServer := &provider.McpServer{}
	if err := json.Unmarshal([]byte(data), mcpServer); err != nil {
		mcpServerLog.Errorf("Unmarshal config data to mcp server error:%v, namespace:%s, groupName:%s, dataId:%s", err, namespace, group, dataId)
		return
	}
	if mcpServer.Protocol == provider.StdioProtocol || mcpServer.Protocol == provider.DubboProtocol || mcpServer.Protocol == provider.McpSSEProtocol {
		return
	}
	// process mcp service
	w.subMutex.Lock()
	defer w.subMutex.Unlock()
	if err := w.buildServiceEntryForMcpServer(mcpServer, group, dataId); err != nil {
		mcpServerLog.Errorf("build service entry for mcp server failed, namespace %v, group: %v, dataId %v, errors: %v", namespace, group, dataId, err)
	}
	// process mcp wasm
	// only generate wasm plugin for http protocol mcp server
	if mcpServer.Protocol != provider.HttpProtocol {
		return
	}
	if _, exist := w.configToConfigListener[key]; !exist {
		w.configToConfigListener[key] = NewMultiConfigListener(w.configClient, w.multiCallback(mcpServer, routeName, key))
	}
	if _, exist := w.watchingConfigRefs[key]; !exist {
		w.watchingConfigRefs[key] = sets.New[string]()
	}
	listener := w.configToConfigListener[key]

	curRef := sets.Set[string]{}
	// add description ref
	curRef.Insert(strings.Join([]string{provider.DefaultMcpToolsGroup, mcpServer.ToolsDescriptionRef}, DefaultJoiner))
	// add credential ref
	credentialNameMap := map[string]string{}
	for name, ref := range mcpServer.Credentials {
		credKey := strings.Join([]string{provider.DefaultMcpCredentialsGroup, ref.Ref}, DefaultJoiner)
		curRef.Insert(credKey)
		credentialNameMap[credKey] = name
	}
	w.callbackMutex.Lock()
	w.credentialKeyToName[key] = credentialNameMap
	w.callbackMutex.Unlock()

	toBeAdd := curRef.Difference(w.watchingConfigRefs[key])
	toBeDelete := w.watchingConfigRefs[key].Difference(curRef)

	var toBeListen, toBeUnListen []vo.ConfigParam
	for item, _ := range toBeAdd {
		split := strings.Split(item, DefaultJoiner)
		toBeListen = append(toBeListen, vo.ConfigParam{
			Group:  split[0],
			DataId: split[1],
		})
	}
	for item, _ := range toBeDelete {
		split := strings.Split(item, DefaultJoiner)
		toBeUnListen = append(toBeUnListen, vo.ConfigParam{
			Group:  split[0],
			DataId: split[1],
		})
	}

	// listen description and credential config
	if len(toBeListen) > 0 {
		if err := listener.StartListen(toBeListen); err != nil {
			mcpServerLog.Errorf("listen config ref failed, group: %v, dataId %v, errors: %v", group, dataId, err)
		}
	}
	// cancel listen description and credential config
	if len(toBeUnListen) > 0 {
		if err := listener.CancelListen(toBeUnListen); err != nil {
			mcpServerLog.Errorf("cancel listen config ref failed, group: %v, dataId %v, errors: %v", group, dataId, err)
		}
	}
}

func (w *watcher) multiCallback(server *provider.McpServer, routeName, configKey string) func(map[string]string) {
	callback := func(configs map[string]string) {
		defer w.UpdateService()

		mcpServerLog.Infof("callback, ref config changed: %s", configKey)
		rule := &provider.McpServerRule{
			MatchRoute: []string{routeName},
			Server: &provider.ServerConfig{
				Name:   server.Name,
				Config: map[string]interface{}{},
			},
		}

		// process mcp credential
		credentialConfig := map[string]interface{}{}
		for key, data := range configs {
			if strings.HasPrefix(key, provider.DefaultMcpToolsGroup) {
				// skip mcp tool description
				continue
			}
			var cred interface{}
			if err := json.Unmarshal([]byte(data), &cred); err != nil {
				mcpServerLog.Errorf("unmarshal credential data %v to map error:%v", key, err)
			}
			w.callbackMutex.Lock()
			name := w.credentialKeyToName[configKey][key]
			w.callbackMutex.Unlock()
			credentialConfig[name] = cred
		}
		rule.Server.Config["credentials"] = credentialConfig
		// process mcp tool description
		var allowTools []string
		for key, toolData := range configs {
			if strings.HasPrefix(key, provider.DefaultMcpCredentialsGroup) {
				// skip mcp credentials
				continue
			}
			toolsDescription := &provider.McpToolConfig{}
			if err := json.Unmarshal([]byte(toolData), toolsDescription); err != nil {
				mcpServerLog.Errorf("unmarshal toolsDescriptionRef to mcp tool config error:%v", err)
			}
			for _, t := range toolsDescription.Tools {
				convertTool := &provider.McpTool{Name: t.Name, Description: t.Description}

				toolMeta := toolsDescription.ToolsMeta[t.Name]
				if toolMeta != nil && toolMeta.Enabled {
					allowTools = append(allowTools, t.Name)
				}
				argsPosition, err := getArgsPositionFromToolMeta(toolMeta)
				if err != nil {
					mcpServerLog.Errorf("get args position from tool meta error:%v, tool name %v", err, t.Name)
				}

				requiredMap := sets.Set[string]{}
				for _, s := range t.InputSchema.Required {
					requiredMap.Insert(s)
				}

				for argsName, args := range t.InputSchema.Properties {
					convertArgs, err := parseMcpArgs(args)
					if err != nil {
						mcpServerLog.Errorf("parse mcp args error:%v, tool name %v, args name %v", err, t.Name, argsName)
						continue
					}
					convertArgs.Name = argsName
					convertArgs.Required = requiredMap.Contains(argsName)
					if pos, exist := argsPosition[argsName]; exist {
						convertArgs.Position = pos
					}
					convertTool.Args = append(convertTool.Args, convertArgs)
					mcpServerLog.Debugf("parseMcpArgs, toolArgs:%v", convertArgs)
				}

				requestTemplate, err := getRequestTemplateFromToolMeta(toolMeta)
				if err != nil {
					mcpServerLog.Errorf("get request template from tool meta error:%v, tool name %v", err, t.Name)
				} else {
					convertTool.RequestTemplate = requestTemplate
				}

				responseTemplate, err := getResponseTemplateFromToolMeta(toolMeta)
				if err != nil {
					mcpServerLog.Errorf("get response template from tool meta error:%v, tool name %v", err, t.Name)
				} else {
					convertTool.ResponseTemplate = responseTemplate
				}
				rule.Tools = append(rule.Tools, convertTool)
			}
		}

		rule.Server.AllowTools = allowTools
		wasmPluginConfig := &config.Config{
			Meta: config.Meta{
				GroupVersionKind: gvk.WasmPlugin,
				Namespace:        w.namespace,
			},
			Spec: rule,
		}
		w.cache.UpdateConfigCache(gvk.WasmPlugin, configKey, wasmPluginConfig, false)
	}
	return callback
}

func (w *watcher) buildServiceEntryForMcpServer(mcpServer *provider.McpServer, configGroup, dataId string) error {
	if mcpServer == nil || mcpServer.RemoteServerConfig == nil || mcpServer.RemoteServerConfig.ServiceRef == nil {
		return nil
	}
	mcpServerLog.Debugf("ServiceRef %v for %v", mcpServer.RemoteServerConfig.ServiceRef, dataId)
	configKey := strings.Join([]string{configGroup, dataId}, DefaultJoiner)

	serviceGroup := mcpServer.RemoteServerConfig.ServiceRef.GroupName
	serviceNamespace := mcpServer.RemoteServerConfig.ServiceRef.NamespaceId
	serviceName := mcpServer.RemoteServerConfig.ServiceRef.ServiceName
	if serviceNamespace == "" {
		serviceNamespace = provider.DefaultNacosServiceNamespace
	}
	// update config to service and unsubscribe old service
	curSvcKey := strings.Join([]string{serviceNamespace, serviceGroup, serviceName}, DefaultJoiner)
	if svcKey, exist := w.configToService[configKey]; exist && svcKey != curSvcKey {
		split := strings.Split(svcKey, DefaultJoiner)
		if svcCache, has := w.serviceCache[split[0]]; has {
			if err := svcCache.RemoveListener(split[1], split[2], configKey); err != nil {
				mcpServerLog.Errorf("remove listener error:%v", err)
			}
		}
	}
	w.configToService[configKey] = curSvcKey

	if _, exist := w.serviceCache[serviceNamespace]; !exist {
		namingConfig := constant.NewClientConfig(
			constant.WithTimeoutMs(DefaultNacosTimeout),
			constant.WithLogLevel(DefaultNacosLogLevel),
			constant.WithLogDir(DefaultNacosLogDir),
			constant.WithCacheDir(DefaultNacosCacheDir),
			constant.WithNotLoadCacheAtStart(DefaultNacosNotLoadCache),
			constant.WithLogRollingConfig(&constant.ClientLogRollingConfig{
				MaxAge: DefaultNacosLogMaxAge,
			}),
			constant.WithUpdateCacheWhenEmpty(w.updateCacheWhenEmpty),
			constant.WithNamespaceId(serviceNamespace),
			constant.WithAccessKey(w.NacosAccessKey),
			constant.WithSecretKey(w.NacosSecretKey),
		)
		client, err := clients.NewNamingClient(vo.NacosClientParam{
			ClientConfig:  namingConfig,
			ServerConfigs: w.serverConfig,
		})
		if err == nil {
			w.serviceCache[serviceNamespace] = NewServiceCache(client)
		} else {
			return fmt.Errorf("can not create naming client err:%v", err)
		}
	}
	svcCache := w.serviceCache[serviceNamespace]
	err := svcCache.AddListener(serviceGroup, serviceName, configKey, w.getServiceCallback(mcpServer, configGroup, dataId))
	if err != nil {
		return fmt.Errorf("add listener for dataId %v, service %s/%s error:%v", dataId, serviceGroup, serviceName, err)
	}
	return nil
}

func (w *watcher) getServiceCallback(server *provider.McpServer, configGroup, dataId string) func(services []model.Instance) {
	groupName := server.RemoteServerConfig.ServiceRef.GroupName
	if groupName == "DEFAULT_GROUP" {
		groupName = "DEFAULT-GROUP"
	}
	namespace := server.RemoteServerConfig.ServiceRef.NamespaceId
	serviceName := server.RemoteServerConfig.ServiceRef.ServiceName
	path := server.RemoteServerConfig.ExportPath
	protocol := server.Protocol
	host := getNacosServiceFullHost(groupName, namespace, serviceName)

	return func(services []model.Instance) {
		defer w.UpdateService()

		mcpServerLog.Infof("callback for %s/%s, serviceName : %s", configGroup, dataId, host)
		configKey := strings.Join([]string{w.Name, w.NacosNamespace, configGroup, dataId}, DefaultJoiner)
		if len(services) == 0 {
			mcpServerLog.Errorf("callback for %s return empty service instance list, skip generate config", host)
			return
		}

		serviceEntry := w.generateServiceEntry(host, services)
		se := &config.Config{
			Meta: config.Meta{
				GroupVersionKind: gvk.ServiceEntry,
				Name:             fmt.Sprintf("%s-%s-%s", provider.IstioMcpAutoGeneratedSeName, configGroup, strings.TrimSuffix(dataId, ".json")),
				Namespace:        w.namespace,
			},
			Spec: serviceEntry,
		}
		if protocol == provider.McpSSEProtocol {
			destinationRule := w.generateDrForSSEService(host)
			dr := &config.Config{
				Meta: config.Meta{
					GroupVersionKind: gvk.DestinationRule,
					Name:             fmt.Sprintf("%s-%s-%s", provider.IstioMcpAutoGeneratedDrName, configGroup, strings.TrimSuffix(dataId, ".json")),
					Namespace:        w.namespace,
				},
				Spec: destinationRule,
			}
			w.cache.UpdateConfigCache(gvk.DestinationRule, configKey, dr, false)
		}
		w.cache.UpdateConfigCache(gvk.ServiceEntry, configKey, se, false)
		vs := w.buildVirtualServiceForMcpServer(serviceEntry, configGroup, dataId, path, server.Name)
		w.cache.UpdateConfigCache(gvk.VirtualService, configKey, vs, false)
	}
}

func (w *watcher) buildVirtualServiceForMcpServer(serviceentry *v1alpha3.ServiceEntry, group, dataId, path, serverName string) *config.Config {
	if serviceentry == nil {
		return nil
	}
	hosts := w.McpServerExportDomains
	if len(hosts) == 0 {
		hosts = []string{"*"}
	}
	var gateways []string
	for _, host := range hosts {
		cleanHost := common2.CleanHost(host)
		// namespace/name, name format: (istio cluster id)-host
		gateways = append(gateways, w.namespace+"/"+
			common2.CreateConvertedName(w.clusterId, cleanHost),
			common2.CreateConvertedName(constants.IstioIngressGatewayName, cleanHost))
	}
	routeName := fmt.Sprintf("%s-%s-%s", provider.IstioMcpAutoGeneratedHttpRouteName, group, strings.TrimSuffix(dataId, ".json"))
	mergePath := "/" + serverName
	if w.McpServerBaseUrl != "/" {
		mergePath = strings.TrimSuffix(w.McpServerBaseUrl, "/") + mergePath
	}
	if path != "/" {
		mergePath = mergePath + "/" + strings.TrimPrefix(path, "/")
	}

	vs := &v1alpha3.VirtualService{
		Hosts:    hosts,
		Gateways: gateways,
		Http: []*v1alpha3.HTTPRoute{{
			Name: routeName,
			Match: []*v1alpha3.HTTPMatchRequest{{
				Uri: &v1alpha3.StringMatch{
					MatchType: &v1alpha3.StringMatch_Prefix{
						Prefix: mergePath,
					},
				},
			}},
			Rewrite: &v1alpha3.HTTPRewrite{
				Uri: path,
			},
			Route: []*v1alpha3.HTTPRouteDestination{{
				Destination: &v1alpha3.Destination{
					Host: serviceentry.Hosts[0],
					Port: &v1alpha3.PortSelector{
						Number: serviceentry.Ports[0].Number,
					},
				},
			}},
		}},
	}

	mcpServerLog.Debugf("construct virtualservice %v", vs)

	return &config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.VirtualService,
			Name:             fmt.Sprintf("%s-%s-%s", provider.IstioMcpAutoGeneratedVsName, group, dataId),
			Namespace:        w.namespace,
		},
		Spec: vs,
	}
}

func (w *watcher) generateServiceEntry(host string, services []model.Instance) *v1alpha3.ServiceEntry {
	portList := make([]*v1alpha3.ServicePort, 0)
	endpoints := make([]*v1alpha3.WorkloadEntry, 0)
	isDnsService := false

	for _, service := range services {
		protocol := common.HTTP
		if service.Metadata != nil && service.Metadata["protocol"] != "" {
			protocol = common.ParseProtocol(service.Metadata["protocol"])
		}
		port := &v1alpha3.ServicePort{
			Name:     protocol.String(),
			Number:   uint32(service.Port),
			Protocol: protocol.String(),
		}
		if len(portList) == 0 {
			portList = append(portList, port)
		}
		if !isValidIP(service.Ip) {
			isDnsService = true
		}
		endpoint := &v1alpha3.WorkloadEntry{
			Address: service.Ip,
			Ports:   map[string]uint32{port.Protocol: port.Number},
			Labels:  service.Metadata,
		}
		endpoints = append(endpoints, endpoint)
	}

	resolution := v1alpha3.ServiceEntry_STATIC
	if isDnsService {
		resolution = v1alpha3.ServiceEntry_DNS
	}
	se := &v1alpha3.ServiceEntry{
		Hosts:      []string{host},
		Ports:      portList,
		Location:   v1alpha3.ServiceEntry_MESH_INTERNAL,
		Resolution: resolution,
		Endpoints:  endpoints,
	}

	return se
}

func (w *watcher) generateDrForSSEService(host string) *v1alpha3.DestinationRule {
	dr := &v1alpha3.DestinationRule{
		Host: host,
		TrafficPolicy: &v1alpha3.TrafficPolicy{
			LoadBalancer: &v1alpha3.LoadBalancerSettings{
				LbPolicy: &v1alpha3.LoadBalancerSettings_ConsistentHash{
					ConsistentHash: &v1alpha3.LoadBalancerSettings_ConsistentHashLB{
						HashKey: &v1alpha3.LoadBalancerSettings_ConsistentHashLB_UseSourceIp{
							UseSourceIp: true,
						},
					},
				},
			},
		},
	}
	return dr
}

func parseMcpArgs(args interface{}) (*provider.ToolArgs, error) {
	argsData, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	toolArgs := &provider.ToolArgs{}
	if err = json.Unmarshal(argsData, toolArgs); err != nil {
		return nil, err
	}
	return toolArgs, nil
}

func getArgsPositionFromToolMeta(toolMeta *provider.ToolsMeta) (map[string]string, error) {
	result := map[string]string{}
	if toolMeta == nil {
		return result, nil
	}
	toolTemplate := toolMeta.Templates
	for kind, meta := range toolTemplate {
		switch kind {
		case provider.JsonGoTemplateType:
			templateData, err := json.Marshal(meta)
			if err != nil {
				return result, err
			}
			template := &provider.JsonGoTemplate{}
			if err = json.Unmarshal(templateData, template); err != nil {
				return result, err
			}
			result = mergeMaps(result, template.ArgsPosition)
		default:
			return result, fmt.Errorf("unsupport tool meta type %v", kind)
		}
	}
	return result, nil
}

func getRequestTemplateFromToolMeta(toolMeta *provider.ToolsMeta) (*provider.RequestTemplate, error) {
	if toolMeta == nil {
		return nil, nil
	}
	toolTemplate := toolMeta.Templates
	for kind, meta := range toolTemplate {
		switch kind {
		case provider.JsonGoTemplateType:
			templateData, err := json.Marshal(meta)
			if err != nil {
				return nil, err
			}
			template := &provider.JsonGoTemplate{}
			if err = json.Unmarshal(templateData, template); err != nil {
				return nil, err
			}
			return &template.RequestTemplate, nil
		default:
			return nil, fmt.Errorf("unsupport tool meta type")
		}
	}
	return nil, nil
}

func getResponseTemplateFromToolMeta(toolMeta *provider.ToolsMeta) (*provider.ResponseTemplate, error) {
	if toolMeta == nil {
		return nil, nil
	}
	toolTemplate := toolMeta.Templates
	for kind, meta := range toolTemplate {
		switch kind {
		case provider.JsonGoTemplateType:
			templateData, err := json.Marshal(meta)
			if err != nil {
				return nil, err
			}
			template := &provider.JsonGoTemplate{}
			if err = json.Unmarshal(templateData, template); err != nil {
				return nil, err
			}
			return &template.ResponseTemplate, nil
		default:
			return nil, fmt.Errorf("unsupport tool meta type")
		}
	}
	return nil, nil
}

func mergeMaps(maps ...map[string]string) map[string]string {
	if len(maps) == 0 {
		return nil
	}
	res := make(map[string]string, len(maps[0]))
	for _, m := range maps {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}

func getNacosServiceFullHost(groupName, namespace, serviceName string) string {
	suffix := strings.Join([]string{groupName, namespace, string(provider.Nacos)}, common.DotSeparator)
	host := strings.Join([]string{serviceName, suffix}, common.DotSeparator)
	return host
}

func (w *watcher) Stop() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	mcpServerLog.Infof("unsubscribe all configs")
	for key := range w.watchingConfig {
		s := strings.Split(key, DefaultJoiner)
		err := w.unsubscribe(s[0], s[1])
		if err == nil {
			delete(w.watchingConfig, key)
		}
	}
	mcpServerLog.Infof("stop all service nameing client")
	for _, client := range w.serviceCache {
		client.Stop()
	}

	w.isStop = true
	mcpServerLog.Infof("stop all config client")
	mcpServerLog.Infof("watcher %v stop", w.Name)

	close(w.stop)
	w.Ready(false)
}

func (w *watcher) IsHealthy() bool {
	return w.Status == provider.Healthy
}

func (w *watcher) GetRegistryType() string {
	return w.RegistryType.String()
}

func isValidIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	return ip != nil
}
