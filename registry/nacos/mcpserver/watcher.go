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
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
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
	DefaultJoiner               = "@@"
)

var mcpServerLog = log.RegisterScope("McpServer", "Nacos Mcp Server Watcher process.")

type watcher struct {
	provider.BaseWatcher
	apiv1.RegistryConfig
	watchingConfig       map[string]bool
	RegistryType         provider.ServiceRegistryType
	Status               provider.WatcherStatus
	registryClient       *NacosRegistryClient
	cache                memory.Cache
	mutex                *sync.Mutex
	stop                 chan struct{}
	isStop               bool
	updateCacheWhenEmpty bool
	namespace            string
	clusterId            string
	authOption           provider.AuthOption
}

type WatcherOption func(w *watcher)

func NewWatcher(cache memory.Cache, opts ...WatcherOption) (provider.Watcher, error) {
	w := &watcher{
		watchingConfig: make(map[string]bool),
		RegistryType:   "nacos3",
		Status:         provider.UnHealthy,
		cache:          cache,
		mutex:          &sync.Mutex{},
		stop:           make(chan struct{}),
	}

	w.NacosRefreshInterval = int64(DefaultRefreshInterval)

	for _, opt := range opts {
		opt(w)
	}

	// The nacos mcp server uses these restricted namespaces and groups, and may be adjusted in the future.
	w.NacosNamespace = "nacos-default-mcp"
	w.NacosNamespaceId = w.NacosNamespace
	w.NacosGroups = []string{"mcp-server"}

	mcpServerLog.Infof("new nacos mcp server watcher with config Name:%s", w.Name)

	clientConfig := constant.NewClientConfig(
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
		constant.WithUsername(w.authOption.NacosUsername),
		constant.WithPassword(w.authOption.NacosPassword),
	)

	initTimer := time.NewTimer(DefaultInitTimeout)
	serverConfig := []constant.ServerConfig{
		*constant.NewServerConfig(w.Domain, uint64(w.Port)),
	}

	success := make(chan struct{})
	go func() {
		client, err := NewMcpRegistryClient(clientConfig, serverConfig, w.NacosNamespaceId)
		if err == nil {
			w.registryClient = client
			close(success)
		} else {
			mcpServerLog.Errorf("can not create registry client, err:%v", err)
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

func WithAuthOption(authOption provider.AuthOption) WatcherOption {
	return func(w *watcher) {
		w.authOption = authOption
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

	mcpConfigs, err := w.registryClient.ListMcpServer()
	if err != nil {
		return fmt.Errorf("list mcp server failed ,error %s", err.Error())
	}

	fetchedConfigs := map[string]bool{}
	for _, c := range mcpConfigs {
		fetchedConfigs[c.Id] = true
	}

	for key := range w.watchingConfig {
		if _, exist := fetchedConfigs[key]; !exist {
			if err = w.registryClient.CancelListenToConfig(key); err != nil {
				return fmt.Errorf("cancel listen mcp server config %s failed, error %s", key, err.Error())
			}
			mcpServerLog.Infof("cancel listen mcp server config %s success", key)
			delete(w.watchingConfig, key)
			// clean cache for this config
			w.cache.UpdateConfigCache(config.GroupVersionKind{}, key, nil, true)
		}
	}

	wg := sync.WaitGroup{}
	subscribeFailed := atomic.NewBool(false)
	watchingKeys := make(chan string, len(fetchedConfigs))
	for key := range fetchedConfigs {
		if _, exist := w.watchingConfig[key]; !exist {
			wg.Add(1)
			go func(k string) {
				err = w.registryClient.ListenToMcpServer(key, w.mcpServerListener(key))
				if err != nil {
					mcpServerLog.Errorf("subscribe mcp server failed, dataId %v, errors: %v", key, err)
					subscribeFailed.Store(true)
				} else {
					mcpServerLog.Infof("subscribe mcp server success, dataId:%s", key)
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

func (w *watcher) mcpServerListener(dataId string) func(info *McpServerConfig) {
	return func(info *McpServerConfig) {
		defer w.UpdateService()

		mcpServerLog.Infof("mcp server config callback, dataId %s", dataId)
		mcpServer := &provider.McpServer{}
		if err := json.Unmarshal([]byte(info.serverSpecConfig), mcpServer); err != nil {
			mcpServerLog.Errorf("unmarshal config data to mcp server error:%v, dataId:%s", err, dataId)
		}
		// TODO support stdio and dubbo protocol
		if mcpServer.Protocol == provider.StdioProtocol || mcpServer.Protocol == provider.DubboProtocol {
			return
		}
		if err := w.processServerConfig(dataId, mcpServer); err != nil {
			mcpServerLog.Errorf("process mcp server config error:%v, dataId:%s", err, dataId)
		}
		if err := w.processToolConfig(dataId, info.toolsSpecConfig, info.Credentials, mcpServer); err != nil {
			mcpServerLog.Errorf("process tool config error:%v, dataId:%s", err, dataId)
		}
	}
}

func (w *watcher) processServerConfig(dataId string, mcpServer *provider.McpServer) error {
	serviceHost := getServiceFullHostFromMcpServer(mcpServer)
	// generate vs for mcp server
	vs := w.buildVirtualServiceForMcpServer(mcpServer, dataId, serviceHost)
	w.cache.UpdateConfigCache(gvk.VirtualService, dataId, vs, false)
	// if protocol is sse, we should apply ConsistentHash policy for this service
	if mcpServer.Protocol == provider.McpSSEProtocol {
		destinationRule := w.generateDrForSSEService(serviceHost)
		dr := &config.Config{
			Meta: config.Meta{
				GroupVersionKind: gvk.DestinationRule,
				Name:             fmt.Sprintf("%s-%s", provider.IstioMcpAutoGeneratedDrName, strings.TrimSuffix(dataId, ".json")),
				Namespace:        w.namespace,
			},
			Spec: destinationRule,
		}
		w.cache.UpdateConfigCache(gvk.DestinationRule, dataId, dr, false)
	}
	return nil
}

func (w *watcher) processToolConfig(dataId, data string, credentials map[string]string, server *provider.McpServer) error {
	if server.Protocol != provider.HttpProtocol && server.Protocol != provider.HttpsProtocol {
		return nil
	}
	toolsDescription := &provider.McpToolConfig{}
	if err := json.Unmarshal([]byte(data), toolsDescription); err != nil {
		return fmt.Errorf("unmarshal toolsDescriptionRef to mcp tool config error:%v", err)
	}

	routeName := fmt.Sprintf("%s-%s", provider.IstioMcpAutoGeneratedHttpRouteName, strings.TrimSuffix(dataId, ".json"))
	rule := &provider.McpServerRule{
		MatchRoute: []string{routeName},
		Server: &provider.ServerConfig{
			Name:   server.Name,
			Config: map[string]interface{}{},
		},
	}
	rule.Server.Config["credentials"] = credentials

	var allowTools []string
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

	rule.Server.AllowTools = allowTools
	wasmPluginConfig := &config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.WasmPlugin,
			Namespace:        w.namespace,
		},
		Spec: rule,
	}
	w.cache.UpdateConfigCache(gvk.WasmPlugin, dataId, wasmPluginConfig, false)
	return nil
}

func (w *watcher) buildVirtualServiceForMcpServer(server *provider.McpServer, dataId, serviceName string) *config.Config {
	if server == nil {
		return nil
	}
	// if there is no export domain, use default *
	hosts := w.McpServerExportDomains
	if len(hosts) == 0 {
		hosts = []string{"*"}
	}
	// find gateway resources by host
	var gateways []string
	for _, host := range hosts {
		cleanHost := common2.CleanHost(host)
		// namespace/name, name format: (istio cluster id)-host
		gateways = append(gateways, w.namespace+"/"+
			common2.CreateConvertedName(w.clusterId, cleanHost),
			common2.CreateConvertedName(constants.IstioIngressGatewayName, cleanHost))
	}
	routeName := fmt.Sprintf("%s-%s", provider.IstioMcpAutoGeneratedHttpRouteName, strings.TrimSuffix(dataId, ".json"))
	// path format: {base-path}/{mcp-server-name}/{export-path}
	var exportPath, mergePath string
	if server.RemoteServerConfig != nil {
		exportPath = server.RemoteServerConfig.ExportPath
	}

	mergePath = "/" + server.Name
	if w.McpServerBaseUrl != "/" {
		mergePath = strings.TrimSuffix(w.McpServerBaseUrl, "/") + mergePath
	}
	if exportPath != "/" {
		mergePath = mergePath + "/" + strings.TrimPrefix(exportPath, "/")
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
			Route: []*v1alpha3.HTTPRouteDestination{{
				Destination: &v1alpha3.Destination{
					Host: serviceName,
				},
			}},
		}},
	}

	if server.Protocol == provider.McpStreambleProtocol {
		vs.Http[0].Rewrite = &v1alpha3.HTTPRewrite{
			Uri: exportPath,
		}
	}

	mcpServerLog.Debugf("construct virtualservice %v", vs)

	return &config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.VirtualService,
			Name:             fmt.Sprintf("%s-%s", provider.IstioMcpAutoGeneratedVsName, dataId),
			Namespace:        w.namespace,
		},
		Spec: vs,
	}
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

func getServiceFullHostFromMcpServer(server *provider.McpServer) string {
	if server == nil || server.RemoteServerConfig == nil || server.RemoteServerConfig.ServiceRef == nil {
		return ""
	}
	groupName := server.RemoteServerConfig.ServiceRef.GroupName
	if groupName == "DEFAULT_GROUP" {
		groupName = "DEFAULT-GROUP"
	}
	namespace := server.RemoteServerConfig.ServiceRef.NamespaceId
	serviceName := server.RemoteServerConfig.ServiceRef.ServiceName
	suffix := strings.Join([]string{groupName, namespace, string(provider.Nacos)}, common.DotSeparator)
	host := strings.Join([]string{serviceName, suffix}, common.DotSeparator)
	return host
}

func (w *watcher) Stop() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	for key := range w.watchingConfig {
		err := w.registryClient.CancelListenToConfig(key)
		if err == nil {
			delete(w.watchingConfig, key)
			w.cache.UpdateConfigCache(config.GroupVersionKind{}, key, nil, true)
			mcpServerLog.Infof("cancel listen to mcp server config %v", key)
		}
	}

	w.isStop = true

	close(w.stop)
	w.Ready(false)
}

func (w *watcher) IsHealthy() bool {
	return w.Status == provider.Healthy
}

func (w *watcher) GetRegistryType() string {
	return w.RegistryType.String()
}
