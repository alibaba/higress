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

package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	wasm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/wasm/v3"
	httppb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/wasm/v3"
	"github.com/golang/protobuf/jsonpb"
	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/protobuf/types/known/anypb"
	extensions "istio.io/api/extensions/v1alpha1"
	networking "istio.io/api/networking/v1alpha3"
	istiotype "istio.io/api/type/v1beta1"
	"istio.io/istio/pilot/pkg/features"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/util/protoconv"
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/config/schema/collection"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/config/schema/kind"
	"istio.io/istio/pkg/log"
	"istio.io/istio/pkg/util/sets"
	v1 "k8s.io/api/core/v1"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	higressext "github.com/alibaba/higress/api/extensions/v1alpha1"
	higressv1 "github.com/alibaba/higress/api/networking/v1"
	extlisterv1 "github.com/alibaba/higress/client/pkg/listers/extensions/v1alpha1"
	netlisterv1 "github.com/alibaba/higress/client/pkg/listers/networking/v1"
	"github.com/alibaba/higress/pkg/cert"
	higressconst "github.com/alibaba/higress/pkg/config/constants"
	"github.com/alibaba/higress/pkg/ingress/kube/annotations"
	"github.com/alibaba/higress/pkg/ingress/kube/common"
	"github.com/alibaba/higress/pkg/ingress/kube/configmap"
	"github.com/alibaba/higress/pkg/ingress/kube/gateway"
	"github.com/alibaba/higress/pkg/ingress/kube/http2rpc"
	"github.com/alibaba/higress/pkg/ingress/kube/ingress"
	"github.com/alibaba/higress/pkg/ingress/kube/ingressv1"
	"github.com/alibaba/higress/pkg/ingress/kube/mcpbridge"
	"github.com/alibaba/higress/pkg/ingress/kube/secret"
	"github.com/alibaba/higress/pkg/ingress/kube/util"
	"github.com/alibaba/higress/pkg/ingress/kube/wasmplugin"
	. "github.com/alibaba/higress/pkg/ingress/log"
	"github.com/alibaba/higress/pkg/kube"
	"github.com/alibaba/higress/registry/memory"
	"github.com/alibaba/higress/registry/reconcile"
)

var (
	_                 istiomodel.ConfigStoreController = &IngressConfig{}
	_                 istiomodel.IngressStore          = &IngressConfig{}
	Http2RpcMethodMap                                  = func() map[string]string {
		return map[string]string{
			"GET":    "ALL_GET",
			"POST":   "ALL_POST",
			"PUT":    "ALL_PUT",
			"DELETE": "ALL_DELETE",
			"PATCH":  "ALL_PATCH",
		}
	}
	Http2RpcParamSourceMap = func() map[string]string {
		return map[string]string{
			"QUERY":  "ALL_QUERY_PARAMETER",
			"HEADER": "ALL_HEADER",
			"PATH":   "ALL_PATH",
			"BODY":   "ALL_BODY",
		}
	}
)

const (
	DefaultMcpbridgeName = "default"
)

type IngressConfig struct {
	remoteIngressControllers map[cluster.ID]common.IngressController
	remoteGatewayControllers map[cluster.ID]common.GatewayController
	mutex                    sync.RWMutex

	ingressRouteCache  istiomodel.IngressRouteCollection
	ingressDomainCache istiomodel.IngressDomainCollection

	localKubeClient kube.Client

	virtualServiceHandlers  []istiomodel.EventHandler
	gatewayHandlers         []istiomodel.EventHandler
	destinationRuleHandlers []istiomodel.EventHandler
	envoyFilterHandlers     []istiomodel.EventHandler
	serviceEntryHandlers    []istiomodel.EventHandler
	wasmPluginHandlers      []istiomodel.EventHandler
	watchErrorHandler       cache.WatchErrorHandler

	cachedEnvoyFilters []config.Config

	watchedSecretSet sets.Set[string]

	RegistryReconciler *reconcile.Reconciler

	mcpbridgeController mcpbridge.McpBridgeController

	mcpbridgeLister netlisterv1.McpBridgeLister

	wasmPluginController wasmplugin.WasmPluginController

	wasmPluginLister extlisterv1.WasmPluginLister

	wasmPlugins map[string]*extensions.WasmPlugin

	http2rpcController http2rpc.Http2RpcController

	http2rpcLister netlisterv1.Http2RpcLister

	http2rpcs map[string]*higressv1.Http2Rpc

	configmapMgr *configmap.ConfigmapMgr

	XDSUpdater istiomodel.XDSUpdater

	annotationHandler annotations.AnnotationHandler

	globalGatewayName string

	namespace string

	clusterId cluster.ID

	httpsConfigMgr *cert.ConfigMgr

	// templateProcessor processes template variables in config
	templateProcessor *TemplateProcessor

	// secretConfigMgr manages secret dependencies
	secretConfigMgr *SecretConfigMgr
}

// getSecretValue implements the getValue function for secret references
func (m *IngressConfig) getSecretValue(valueType, namespace, name, key string) (string, error) {
	if valueType != "secret" {
		return "", fmt.Errorf("unsupported value type: %s", valueType)
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, controller := range m.remoteIngressControllers {
		secret, err := controller.SecretLister().Secrets(namespace).Get(name)
		if err == nil {
			if value, exists := secret.Data[key]; exists {
				return string(value), nil
			}
			return "", fmt.Errorf("key %s not found in secret %s/%s", key, namespace, name)
		}
	}
	return "", fmt.Errorf("secret %s/%s not found", namespace, name)
}

func NewIngressConfig(localKubeClient kube.Client, xdsUpdater istiomodel.XDSUpdater, namespace string, options common.Options) *IngressConfig {
	clusterId := options.ClusterId
	if clusterId == "Kubernetes" {
		clusterId = ""
	}
	config := &IngressConfig{
		remoteIngressControllers: make(map[cluster.ID]common.IngressController),
		remoteGatewayControllers: make(map[cluster.ID]common.GatewayController),
		localKubeClient:          localKubeClient,
		XDSUpdater:               xdsUpdater,
		annotationHandler:        annotations.NewAnnotationHandlerManager(),
		clusterId:                clusterId,
		globalGatewayName:        namespace + "/" + common.CreateConvertedName(clusterId.String(), "global"),
		watchedSecretSet:         sets.New[string](),
		namespace:                namespace,
		wasmPlugins:              make(map[string]*extensions.WasmPlugin),
		http2rpcs:                make(map[string]*higressv1.Http2Rpc),
	}

	// Initialize secret config manager
	config.secretConfigMgr = NewSecretConfigMgr(xdsUpdater)

	// Initialize template processor with value getter function
	config.templateProcessor = NewTemplateProcessor(config.getSecretValue, namespace, config.secretConfigMgr)

	mcpbridgeController := mcpbridge.NewController(localKubeClient, options)
	mcpbridgeController.AddEventHandler(config.AddOrUpdateMcpBridge, config.DeleteMcpBridge)
	config.mcpbridgeController = mcpbridgeController
	config.mcpbridgeLister = mcpbridgeController.Lister()

	wasmPluginController := wasmplugin.NewController(localKubeClient, options)
	wasmPluginController.AddEventHandler(config.AddOrUpdateWasmPlugin, config.DeleteWasmPlugin)
	config.wasmPluginController = wasmPluginController
	config.wasmPluginLister = wasmPluginController.Lister()

	http2rpcController := http2rpc.NewController(localKubeClient, options)
	http2rpcController.AddEventHandler(config.AddOrUpdateHttp2Rpc, config.DeleteHttp2Rpc)
	config.http2rpcController = http2rpcController
	config.http2rpcLister = http2rpcController.Lister()

	higressConfigController := configmap.NewController(localKubeClient, clusterId, namespace)
	config.configmapMgr = configmap.NewConfigmapMgr(xdsUpdater, namespace, higressConfigController, higressConfigController.Lister())

	httpsConfigMgr, _ := cert.NewConfigMgr(namespace, localKubeClient.Kube())
	config.httpsConfigMgr = httpsConfigMgr

	return config
}

func (m *IngressConfig) RegisterEventHandler(kind config.GroupVersionKind, f istiomodel.EventHandler) {
	IngressLog.Infof("register resource %v", kind)
	switch kind {
	case gvk.VirtualService:
		m.virtualServiceHandlers = append(m.virtualServiceHandlers, f)

	case gvk.Gateway:
		m.gatewayHandlers = append(m.gatewayHandlers, f)

	case gvk.DestinationRule:
		m.destinationRuleHandlers = append(m.destinationRuleHandlers, f)

	case gvk.EnvoyFilter:
		m.envoyFilterHandlers = append(m.envoyFilterHandlers, f)

	case gvk.ServiceEntry:
		m.serviceEntryHandlers = append(m.serviceEntryHandlers, f)

	case gvk.WasmPlugin:
		m.wasmPluginHandlers = append(m.wasmPluginHandlers, f)
	}

	for _, remoteIngressController := range m.remoteIngressControllers {
		remoteIngressController.RegisterEventHandler(kind, f)
	}
	for _, remoteGatewayController := range m.remoteGatewayControllers {
		remoteGatewayController.RegisterEventHandler(kind, f)
	}
}

func (m *IngressConfig) AddLocalCluster(options common.Options) {
	secretController := secret.NewController(m.localKubeClient, options)
	secretController.AddEventHandler(m.ReflectSecretChanges)
	secretController.AddEventHandler(m.secretConfigMgr.HandleSecretChange)

	var ingressController common.IngressController
	v1 := common.V1Available(m.localKubeClient)
	if !v1 {
		ingressController = ingress.NewController(m.localKubeClient, m.localKubeClient, options, secretController)
	} else {
		ingressController = ingressv1.NewController(m.localKubeClient, m.localKubeClient, options, secretController)
	}
	m.remoteIngressControllers[options.ClusterId] = ingressController
	if features.EnableGatewayAPI {
		m.remoteGatewayControllers[options.ClusterId] = gateway.NewController(m.localKubeClient, options)
	}
}

func (m *IngressConfig) List(typ config.GroupVersionKind, namespace string) []config.Config {
	if typ != gvk.Gateway &&
		typ != gvk.VirtualService &&
		typ != gvk.DestinationRule &&
		typ != gvk.EnvoyFilter &&
		typ != gvk.ServiceEntry &&
		typ != gvk.WasmPlugin {
		return nil
	}
	var configs = make([]config.Config, 0)

	if configsFromIngress := m.listFromIngressControllers(typ, namespace); configsFromIngress != nil {
		// Process templates for ingress configs
		for i := range configsFromIngress {
			if err := m.templateProcessor.ProcessConfig(&configsFromIngress[i]); err != nil {
				IngressLog.Errorf("Failed to process template for config %s/%s: %v",
					configsFromIngress[i].Namespace, configsFromIngress[i].Name, err)
			}
		}
		configs = append(configs, configsFromIngress...)
	}

	if configsFromGateway := m.listFromGatewayControllers(typ, namespace); configsFromGateway != nil {
		// Process templates for gateway configs
		for i := range configsFromGateway {
			if err := m.templateProcessor.ProcessConfig(&configsFromGateway[i]); err != nil {
				IngressLog.Errorf("Failed to process template for config %s/%s: %v",
					configsFromGateway[i].Namespace, configsFromGateway[i].Name, err)
			}
		}
		configs = append(configs, configsFromGateway...)
	}

	return configs
}

func (m *IngressConfig) listFromIngressControllers(typ config.GroupVersionKind, namespace string) []config.Config {
	// Currently, only support list all namespaces gateways or virtualservices.
	if namespace != "" {
		IngressLog.Warnf("ingress store only support type %s of all namespace, request namespace: %s", typ, namespace)
		return nil
	}

	if typ == gvk.EnvoyFilter {
		m.mutex.RLock()
		defer m.mutex.RUnlock()
		var envoyFilters []config.Config
		// Build configmap envoy filters
		configmapEnvoyFilters, err := m.configmapMgr.ConstructEnvoyFilters()
		if err != nil {
			IngressLog.Errorf("Construct configmap EnvoyFilters error %v", err)
		} else {
			for _, envoyFilter := range configmapEnvoyFilters {
				envoyFilters = append(envoyFilters, *envoyFilter)
			}
			IngressLog.Infof("Append %d configmap EnvoyFilters", len(configmapEnvoyFilters))
		}
		if len(envoyFilters) == 0 {
			IngressLog.Infof("resource type %s, configs number %d", typ, len(m.cachedEnvoyFilters))
			return m.cachedEnvoyFilters
		}
		envoyFilters = append(envoyFilters, m.cachedEnvoyFilters...)
		IngressLog.Infof("resource type %s, configs number %d", typ, len(envoyFilters))
		return envoyFilters
	}

	var configs []config.Config
	m.mutex.RLock()
	for _, ingressController := range m.remoteIngressControllers {
		configs = append(configs, ingressController.List()...)
	}
	m.mutex.RUnlock()

	common.SortIngressByCreationTime(configs)
	wrapperConfigs := m.createWrapperConfigs(configs)

	var result []config.Config
	switch typ {
	case gvk.Gateway:
		result = m.convertGateways(wrapperConfigs)
	case gvk.VirtualService:
		result = m.convertVirtualService(wrapperConfigs)
	case gvk.DestinationRule:
		result = m.convertDestinationRule(wrapperConfigs)
	case gvk.ServiceEntry:
		result = m.convertServiceEntry(wrapperConfigs)
	case gvk.WasmPlugin:
		result = m.convertWasmPlugin(wrapperConfigs)
	}
	IngressLog.Infof("resource type %s, ingress number %d, convert configs number %d", typ, len(configs), len(result))
	return result
}

func (m *IngressConfig) listFromGatewayControllers(typ config.GroupVersionKind, namespace string) []config.Config {
	var configs []config.Config
	for _, gatewayController := range m.remoteGatewayControllers {
		if clusterConfigs := gatewayController.List(typ, namespace); clusterConfigs != nil {
			configs = append(configs, clusterConfigs...)
		}
	}
	return configs
}

func (m *IngressConfig) createWrapperConfigs(configs []config.Config) []common.WrapperConfig {
	var wrapperConfigs []common.WrapperConfig

	// Init global context
	clusterSecretListers := map[cluster.ID]listersv1.SecretLister{}
	clusterServiceListers := map[cluster.ID]listersv1.ServiceLister{}
	m.mutex.RLock()
	for clusterId, controller := range m.remoteIngressControllers {
		clusterSecretListers[clusterId] = controller.SecretLister()
		clusterServiceListers[clusterId] = controller.ServiceLister()
	}
	m.mutex.RUnlock()
	globalContext := &annotations.GlobalContext{
		WatchedSecrets:      sets.New[string](),
		ClusterSecretLister: clusterSecretListers,
		ClusterServiceList:  clusterServiceListers,
	}

	for idx := range configs {
		rawConfig := configs[idx]
		annotationsConfig := &annotations.Ingress{
			Meta: annotations.Meta{
				Namespace:    rawConfig.Namespace,
				Name:         rawConfig.Name,
				RawClusterId: common.GetRawClusterId(rawConfig.Annotations),
				ClusterId:    common.GetClusterId(rawConfig.Annotations),
			},
		}
		_ = m.annotationHandler.Parse(rawConfig.Annotations, annotationsConfig, globalContext)
		wrapperConfigs = append(wrapperConfigs, common.WrapperConfig{
			Config:            &rawConfig,
			AnnotationsConfig: annotationsConfig,
		})
	}

	m.mutex.Lock()
	m.watchedSecretSet = globalContext.WatchedSecrets
	m.mutex.Unlock()

	return wrapperConfigs
}

func (m *IngressConfig) convertGateways(configs []common.WrapperConfig) []config.Config {
	convertOptions := common.ConvertOptions{
		IngressDomainCache: common.NewIngressDomainCache(),
		Gateways:           map[string]*common.WrapperGateway{},
	}

	httpsCredentialConfig, err := m.httpsConfigMgr.GetConfigFromConfigmap()
	if err != nil {
		IngressLog.Errorf("Get higress https configmap err %v", err)
	}
	for idx := range configs {
		cfg := configs[idx]
		clusterId := common.GetClusterId(cfg.Config.Annotations)
		m.mutex.RLock()
		ingressController := m.remoteIngressControllers[clusterId]
		m.mutex.RUnlock()
		if ingressController == nil {
			continue
		}
		if err := ingressController.ConvertGateway(&convertOptions, &cfg, httpsCredentialConfig); err != nil {
			IngressLog.Errorf("Convert ingress %s/%s to gateway fail in cluster %s, err %v", cfg.Config.Namespace, cfg.Config.Name, clusterId, err)
		}
	}

	// apply annotation
	for _, wrapperGateway := range convertOptions.Gateways {
		m.annotationHandler.ApplyGateway(wrapperGateway.Gateway, wrapperGateway.WrapperConfig.AnnotationsConfig)
	}

	m.mutex.Lock()
	m.ingressDomainCache = convertOptions.IngressDomainCache.Extract()
	m.mutex.Unlock()

	out := make([]config.Config, 0, len(convertOptions.Gateways))
	for _, gateway := range convertOptions.Gateways {
		cleanHost := common.CleanHost(gateway.Host)
		out = append(out, config.Config{
			Meta: config.Meta{
				GroupVersionKind: gvk.Gateway,
				Name:             common.CreateConvertedName(constants.IstioIngressGatewayName, cleanHost),
				Namespace:        m.namespace,
				Annotations: map[string]string{
					common.ClusterIdAnnotation: gateway.ClusterId.String(),
					common.HostAnnotation:      gateway.Host,
				},
			},
			Spec: gateway.Gateway,
		})
	}
	return out
}

func (m *IngressConfig) convertVirtualService(configs []common.WrapperConfig) []config.Config {
	convertOptions := common.ConvertOptions{
		IngressRouteCache: common.NewIngressRouteCache(),
		VirtualServices:   map[string]*common.WrapperVirtualService{},
		HTTPRoutes:        map[string][]*common.WrapperHTTPRoute{},
		Route2Ingress:     map[string]*common.WrapperConfigWithRuleKey{},
	}

	// convert http route
	for idx := range configs {
		cfg := configs[idx]
		clusterId := common.GetClusterId(cfg.Config.Annotations)
		m.mutex.RLock()
		ingressController := m.remoteIngressControllers[clusterId]
		m.mutex.RUnlock()
		if ingressController == nil {
			continue
		}
		if err := ingressController.ConvertHTTPRoute(&convertOptions, &cfg); err != nil {
			IngressLog.Errorf("Convert ingress %s/%s to HTTP route fail in cluster %s, err %v", cfg.Config.Namespace, cfg.Config.Name, clusterId, err)
		}
	}

	// Apply annotation on routes
	for _, routes := range convertOptions.HTTPRoutes {
		for _, route := range routes {
			m.annotationHandler.ApplyRoute(route.HTTPRoute, route.WrapperConfig.AnnotationsConfig)
		}
	}

	// Apply canary ingress
	if len(configs) > len(convertOptions.CanaryIngresses) {
		m.applyCanaryIngresses(&convertOptions)
	}

	// Normalize weighted cluster to make sure the sum of weight is 100.
	for _, host := range convertOptions.HTTPRoutes {
		for _, route := range host {
			normalizeWeightedCluster(convertOptions.IngressRouteCache, route)
		}
	}

	// Apply spec default backend.
	if convertOptions.HasDefaultBackend {
		for idx := range configs {
			cfg := configs[idx]
			clusterId := common.GetClusterId(cfg.Config.Annotations)
			m.mutex.RLock()
			ingressController := m.remoteIngressControllers[clusterId]
			m.mutex.RUnlock()
			if ingressController == nil {
				continue
			}
			if err := ingressController.ApplyDefaultBackend(&convertOptions, &cfg); err != nil {
				IngressLog.Errorf("Apply default backend on ingress %s/%s fail in cluster %s, err %v", cfg.Config.Namespace, cfg.Config.Name, clusterId, err)
			}
		}
	}

	// Apply annotation on virtual services
	for _, virtualService := range convertOptions.VirtualServices {
		m.annotationHandler.ApplyVirtualServiceHandler(virtualService.VirtualService, virtualService.WrapperConfig.AnnotationsConfig)
	}

	// Apply app root for per host.
	m.applyAppRoot(&convertOptions)

	// Apply internal active redirect for error page.
	m.applyInternalActiveRedirect(&convertOptions)

	m.mutex.Lock()
	m.ingressRouteCache = convertOptions.IngressRouteCache.Extract()
	m.mutex.Unlock()

	// Convert http route to virtual service
	out := make([]config.Config, 0, len(convertOptions.HTTPRoutes))
	for host, routes := range convertOptions.HTTPRoutes {
		if len(routes) == 0 {
			continue
		}

		cleanHost := common.CleanHost(host)
		// namespace/name, name format: (istio cluster id)-host
		gateways := []string{m.namespace + "/" +
			common.CreateConvertedName(m.clusterId.String(), cleanHost),
			common.CreateConvertedName(constants.IstioIngressGatewayName, cleanHost)}

		wrapperVS, exist := convertOptions.VirtualServices[host]
		if !exist {
			IngressLog.Warnf("virtual service for host %s does not exist.", host)
		}
		vs := wrapperVS.VirtualService
		vs.Gateways = gateways

		// Sort, exact -> prefix -> regex
		common.SortHTTPRoutes(routes)

		for _, route := range routes {
			vs.Http = append(vs.Http, route.HTTPRoute)
		}

		firstRoute := routes[0]
		out = append(out, config.Config{
			Meta: config.Meta{
				GroupVersionKind: gvk.VirtualService,
				Name:             common.CreateConvertedName(constants.IstioIngressGatewayName, firstRoute.WrapperConfig.Config.Namespace, firstRoute.WrapperConfig.Config.Name, cleanHost),
				Namespace:        m.namespace,
				Annotations: map[string]string{
					common.ClusterIdAnnotation: firstRoute.ClusterId.String(),
				},
			},
			Spec: vs,
		})
	}

	// We generate some specific envoy filter here to avoid duplicated computation.
	m.convertEnvoyFilter(&convertOptions)
	return out
}

func (m *IngressConfig) convertEnvoyFilter(convertOptions *common.ConvertOptions) {
	var envoyFilters []config.Config
	mappings := map[string]*common.Rule{}

	initHttp2RpcGlobalConfig := true
	for _, routes := range convertOptions.HTTPRoutes {
		for _, route := range routes {
			if strings.HasSuffix(route.HTTPRoute.Name, "app-root") {
				continue
			}

			http2rpc := route.WrapperConfig.AnnotationsConfig.Http2Rpc
			if http2rpc != nil {
				IngressLog.Infof("Found http2rpc for name %s", http2rpc.Name)
				envoyFilter, err := m.constructHttp2RpcEnvoyFilter(http2rpc, route, m.namespace, initHttp2RpcGlobalConfig)
				if err != nil {
					IngressLog.Infof("Construct http2rpc EnvoyFilter error %v", err)
				} else {
					IngressLog.Infof("Append http2rpc EnvoyFilter for name %s", http2rpc.Name)
					envoyFilters = append(envoyFilters, *envoyFilter)
					initHttp2RpcGlobalConfig = false
				}
			}

			auth := route.WrapperConfig.AnnotationsConfig.Auth
			if auth == nil {
				continue
			}

			key := auth.AuthSecret.String() + "/" + auth.AuthRealm
			if rule, exist := mappings[key]; !exist {
				mappings[key] = &common.Rule{
					Realm:       auth.AuthRealm,
					MatchRoute:  []string{route.HTTPRoute.Name},
					Credentials: auth.Credentials,
					Encrypted:   true,
				}
			} else {
				rule.MatchRoute = append(rule.MatchRoute, route.HTTPRoute.Name)
			}
		}
	}

	IngressLog.Infof("Found %d number of basic auth", len(mappings))
	if len(mappings) > 0 {
		rules := &common.BasicAuthRules{}
		for _, rule := range mappings {
			rules.Rules = append(rules.Rules, rule)
		}

		basicAuth, err := constructBasicAuthEnvoyFilter(rules, m.namespace)
		if err != nil {
			IngressLog.Errorf("Construct basic auth filter error %v", err)
		} else {
			envoyFilters = append(envoyFilters, *basicAuth)
		}
	}

	// TODO Support other envoy filters

	IngressLog.Infof("Found %d number of envoyFilters", len(envoyFilters))
	m.mutex.Lock()
	m.cachedEnvoyFilters = envoyFilters
	m.mutex.Unlock()
}

func (m *IngressConfig) convertWasmPlugin([]common.WrapperConfig) []config.Config {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	out := make([]config.Config, 0, len(m.wasmPlugins))
	for name, wasmPlugin := range m.wasmPlugins {
		out = append(out, config.Config{
			Meta: config.Meta{
				GroupVersionKind: gvk.WasmPlugin,
				Name:             name,
				Namespace:        m.namespace,
			},
			Spec: wasmPlugin,
		})
	}
	return out
}

func (m *IngressConfig) convertServiceEntry([]common.WrapperConfig) []config.Config {
	if m.RegistryReconciler == nil {
		return nil
	}
	serviceEntries := m.RegistryReconciler.GetAllServiceWrapper()
	IngressLog.Infof("Found mcp serviceEntries %v", serviceEntries)
	out := make([]config.Config, 0, len(serviceEntries))
	for _, se := range serviceEntries {
		out = append(out, config.Config{
			Meta: config.Meta{
				GroupVersionKind:  gvk.ServiceEntry,
				Name:              se.ServiceEntry.Hosts[0],
				Namespace:         "mcp",
				CreationTimestamp: se.GetCreateTime(),
				Labels: map[string]string{
					higressconst.RegistryTypeLabelKey: se.RegistryType,
					higressconst.RegistryNameLabelKey: se.RegistryName,
				},
			},
			Spec: se.ServiceEntry,
		})
	}
	return out
}

func (m *IngressConfig) convertDestinationRule(configs []common.WrapperConfig) []config.Config {
	convertOptions := common.ConvertOptions{
		Service2TrafficPolicy: map[common.ServiceKey]*common.WrapperTrafficPolicy{},
	}

	// Convert destination from service within ingress rule.
	for idx := range configs {
		cfg := configs[idx]
		clusterId := common.GetClusterId(cfg.Config.Annotations)
		m.mutex.RLock()
		ingressController := m.remoteIngressControllers[clusterId]
		m.mutex.RUnlock()
		if ingressController == nil {
			continue
		}
		if err := ingressController.ConvertTrafficPolicy(&convertOptions, &cfg); err != nil {
			IngressLog.Errorf("Convert ingress %s/%s to destination rule fail in cluster %s, err %v", cfg.Config.Namespace, cfg.Config.Name, clusterId, err)
		}
	}

	IngressLog.Debugf("traffic policy number %d", len(convertOptions.Service2TrafficPolicy))

	for _, wrapperTrafficPolicy := range convertOptions.Service2TrafficPolicy {
		m.annotationHandler.ApplyTrafficPolicy(wrapperTrafficPolicy.TrafficPolicy, wrapperTrafficPolicy.PortTrafficPolicy, wrapperTrafficPolicy.WrapperConfig.AnnotationsConfig)
	}

	// Merge multi-port traffic policy per service into one destination rule.
	destinationRules := map[string]*common.WrapperDestinationRule{}
	for key, wrapperTrafficPolicy := range convertOptions.Service2TrafficPolicy {
		var serviceName string
		if key.ServiceFQDN != "" {
			serviceName = key.ServiceFQDN
		} else {
			serviceName = util.CreateServiceFQDN(key.Namespace, key.Name)
		}
		dr, exist := destinationRules[serviceName]
		if !exist {
			trafficPolicy := &networking.TrafficPolicy{}
			if wrapperTrafficPolicy.PortTrafficPolicy != nil {
				trafficPolicy.PortLevelSettings = []*networking.TrafficPolicy_PortTrafficPolicy{wrapperTrafficPolicy.PortTrafficPolicy}
			} else if wrapperTrafficPolicy.TrafficPolicy != nil {
				trafficPolicy = wrapperTrafficPolicy.TrafficPolicy
			}
			dr = &common.WrapperDestinationRule{
				DestinationRule: &networking.DestinationRule{
					Host:          serviceName,
					TrafficPolicy: trafficPolicy,
				},
				WrapperConfig: wrapperTrafficPolicy.WrapperConfig,
				ServiceKey:    key,
			}
		} else if wrapperTrafficPolicy.PortTrafficPolicy != nil {
			dr.DestinationRule.TrafficPolicy.PortLevelSettings = append(dr.DestinationRule.TrafficPolicy.PortLevelSettings, wrapperTrafficPolicy.PortTrafficPolicy)
		}

		destinationRules[serviceName] = dr
	}

	if m.RegistryReconciler != nil {
		drws := m.RegistryReconciler.GetAllDestinationRuleWrapper()
		for _, destinationRuleWrapper := range drws {
			serviceName := destinationRuleWrapper.ServiceKey.ServiceFQDN
			dr, exist := destinationRules[serviceName]
			if !exist {
				destinationRules[serviceName] = destinationRuleWrapper
			} else if dr.DestinationRule.TrafficPolicy != nil {
				portTrafficPolicy := destinationRuleWrapper.DestinationRule.TrafficPolicy.PortLevelSettings[0]
				portUpdated := false
				for _, policy := range dr.DestinationRule.TrafficPolicy.PortLevelSettings {
					if policy.Port.Number == portTrafficPolicy.Port.Number {
						policy.Tls = portTrafficPolicy.Tls
						portUpdated = true
						break
					}
				}
				if portUpdated {
					continue
				}
				dr.DestinationRule.TrafficPolicy.PortLevelSettings = append(dr.DestinationRule.TrafficPolicy.PortLevelSettings, portTrafficPolicy)
			}
		}
	}

	out := make([]config.Config, 0, len(destinationRules))
	for _, dr := range destinationRules {
		sort.SliceStable(dr.DestinationRule.TrafficPolicy.PortLevelSettings, func(i, j int) bool {
			portI := dr.DestinationRule.TrafficPolicy.PortLevelSettings[i].Port
			portJ := dr.DestinationRule.TrafficPolicy.PortLevelSettings[j].Port
			if portI == nil && portJ == nil {
				return true
			} else if portI == nil {
				return true
			} else if portJ == nil {
				return false
			}
			return portI.Number < portJ.Number
		})
		drName := util.CreateDestinationRuleName(m.clusterId, dr.ServiceKey.Namespace, dr.ServiceKey.Name)
		out = append(out, config.Config{
			Meta: config.Meta{
				GroupVersionKind: gvk.DestinationRule,
				Name:             common.CreateConvertedName(constants.IstioIngressGatewayName, drName),
				Namespace:        m.namespace,
			},
			Spec: dr.DestinationRule,
		})
	}

	return out
}

func (m *IngressConfig) applyAppRoot(convertOptions *common.ConvertOptions) {
	for host, wrapVS := range convertOptions.VirtualServices {
		if wrapVS.AppRoot != "" {
			route := &common.WrapperHTTPRoute{
				HTTPRoute: &networking.HTTPRoute{
					Name: common.CreateConvertedName(host, "app-root"),
					Match: []*networking.HTTPMatchRequest{
						{
							Uri: &networking.StringMatch{
								MatchType: &networking.StringMatch_Exact{
									Exact: "/",
								},
							},
						},
					},
					Redirect: &networking.HTTPRedirect{
						RedirectCode: 302,
						Uri:          wrapVS.AppRoot,
					},
				},
				WrapperConfig: wrapVS.WrapperConfig,
				ClusterId:     wrapVS.WrapperConfig.AnnotationsConfig.ClusterId,
			}
			convertOptions.HTTPRoutes[host] = append([]*common.WrapperHTTPRoute{route}, convertOptions.HTTPRoutes[host]...)
		}
	}
}

func (m *IngressConfig) applyInternalActiveRedirect(convertOptions *common.ConvertOptions) {
	for host, routes := range convertOptions.HTTPRoutes {
		var tempRoutes []*common.WrapperHTTPRoute
		for _, route := range routes {
			tempRoutes = append(tempRoutes, route)
			if route.HTTPRoute.InternalActiveRedirect != nil {
				fallbackConfig := route.WrapperConfig.AnnotationsConfig.Fallback
				if fallbackConfig == nil {
					continue
				}

				typedNamespace := fallbackConfig.DefaultBackend
				internalRedirectRoute := route.HTTPRoute.DeepCopy()
				internalRedirectRoute.Name = internalRedirectRoute.Name + annotations.FallbackRouteNameSuffix
				internalRedirectRoute.InternalActiveRedirect = nil
				internalRedirectRoute.Match = []*networking.HTTPMatchRequest{
					{
						Uri: &networking.StringMatch{
							MatchType: &networking.StringMatch_Exact{
								Exact: "/",
							},
						},
						Headers: map[string]*networking.StringMatch{
							annotations.FallbackInjectHeaderRouteName: {
								MatchType: &networking.StringMatch_Exact{
									Exact: internalRedirectRoute.Name,
								},
							},
							annotations.FallbackInjectFallbackService: {
								MatchType: &networking.StringMatch_Exact{
									Exact: typedNamespace.String(),
								},
							},
						},
					},
				}
				internalRedirectRoute.Route = []*networking.HTTPRouteDestination{
					{
						Destination: &networking.Destination{
							Host: util.CreateServiceFQDN(typedNamespace.Namespace, typedNamespace.Name),
							Port: &networking.PortSelector{
								Number: fallbackConfig.Port,
							},
						},
						Weight: 100,
					},
				}

				tempRoutes = append([]*common.WrapperHTTPRoute{{
					HTTPRoute:     internalRedirectRoute,
					WrapperConfig: route.WrapperConfig,
					ClusterId:     route.ClusterId,
				}}, tempRoutes...)
			}
		}
		convertOptions.HTTPRoutes[host] = tempRoutes
	}
}

func (m *IngressConfig) convertIstioWasmPlugin(obj *higressext.WasmPlugin) (*extensions.WasmPlugin, error) {
	result := &extensions.WasmPlugin{
		Selector: &istiotype.WorkloadSelector{
			MatchLabels: map[string]string{
				"higress": m.namespace + "-higress-gateway",
			},
		},
		Url:             obj.Url,
		Sha256:          obj.Sha256,
		ImagePullPolicy: extensions.PullPolicy(obj.ImagePullPolicy),
		ImagePullSecret: obj.ImagePullSecret,
		VerificationKey: obj.VerificationKey,
		PluginConfig:    obj.PluginConfig,
		PluginName:      obj.PluginName,
		Phase:           extensions.PluginPhase(obj.Phase),
		FailStrategy:    extensions.FailStrategy(obj.FailStrategy),
		Priority:        obj.Priority,
	}
	if obj.VmConfig != nil {
		result.VmConfig = &extensions.VmConfig{}
		for _, env := range obj.VmConfig.Env {
			result.VmConfig.Env = append(result.VmConfig.Env, &extensions.EnvVar{
				Name:      env.Name,
				ValueFrom: extensions.EnvValueSource(env.ValueFrom),
				Value:     env.Value,
			})
		}
	}
	if result.PluginConfig != nil {
		return result, nil
	}
	if !isBoolValueTrue(obj.DefaultConfigDisable) {
		result.PluginConfig = obj.DefaultConfig
	}
	hasValidRule := false
	if len(obj.MatchRules) > 0 {
		if result.PluginConfig == nil {
			result.PluginConfig = &_struct.Struct{
				Fields: map[string]*_struct.Value{},
			}
		}
		var ruleValues []*_struct.Value
		for _, rule := range obj.MatchRules {
			if isBoolValueTrue(rule.ConfigDisable) {
				continue
			}
			if rule.Config == nil {
				rule.Config = &_struct.Struct{
					Fields: map[string]*_struct.Value{},
				}
			}
			v := &_struct.Value_StructValue{
				StructValue: rule.Config,
			}

			validRule := false
			var matchItems []*_struct.Value
			// match ingress
			for _, ing := range rule.Ingress {
				matchItems = append(matchItems, &_struct.Value{
					Kind: &_struct.Value_StringValue{
						StringValue: ing,
					},
				})
			}
			if len(matchItems) > 0 {
				validRule = true
				v.StructValue.Fields["_match_route_"] = &_struct.Value{
					Kind: &_struct.Value_ListValue{
						ListValue: &_struct.ListValue{
							Values: matchItems,
						},
					},
				}
			}
			// match service
			matchItems = nil
			for _, service := range rule.Service {
				matchItems = append(matchItems, &_struct.Value{
					Kind: &_struct.Value_StringValue{
						StringValue: service,
					},
				})
			}
			if len(matchItems) > 0 {
				validRule = true
				v.StructValue.Fields["_match_service_"] = &_struct.Value{
					Kind: &_struct.Value_ListValue{
						ListValue: &_struct.ListValue{
							Values: matchItems,
						},
					},
				}
			}
			// match domain
			matchItems = nil
			for _, domain := range rule.Domain {
				matchItems = append(matchItems, &_struct.Value{
					Kind: &_struct.Value_StringValue{
						StringValue: domain,
					},
				})
			}
			if len(matchItems) > 0 {
				validRule = true
				v.StructValue.Fields["_match_domain_"] = &_struct.Value{
					Kind: &_struct.Value_ListValue{
						ListValue: &_struct.ListValue{
							Values: matchItems,
						},
					},
				}
			}
			if validRule {
				ruleValues = append(ruleValues, &_struct.Value{
					Kind: v,
				})
			} else {
				return nil, fmt.Errorf("invalid match rule has no match condition, rule:%v", rule)
			}
		}
		if len(ruleValues) > 0 {
			hasValidRule = true
			result.PluginConfig.Fields["_rules_"] = &_struct.Value{
				Kind: &_struct.Value_ListValue{
					ListValue: &_struct.ListValue{
						Values: ruleValues,
					},
				},
			}
		}
	}
	if !hasValidRule && isBoolValueTrue(obj.DefaultConfigDisable) {
		return nil, nil
	}
	return result, nil
}

func isBoolValueTrue(b *wrappers.BoolValue) bool {
	return b != nil && b.Value
}

func (m *IngressConfig) AddOrUpdateWasmPlugin(clusterNamespacedName util.ClusterNamespacedName) {
	if clusterNamespacedName.Namespace != m.namespace {
		return
	}
	wasmPlugin, err := m.wasmPluginLister.WasmPlugins(clusterNamespacedName.Namespace).Get(clusterNamespacedName.Name)
	if err != nil {
		IngressLog.Errorf("wasmPlugin is not found, namespace:%s, name:%s",
			clusterNamespacedName.Namespace, clusterNamespacedName.Name)
		return
	}
	metadata := config.Meta{
		Name:             clusterNamespacedName.Name + "-wasmplugin",
		Namespace:        clusterNamespacedName.Namespace,
		GroupVersionKind: gvk.WasmPlugin,
		// Set this label so that we do not compare configs and just push.
		Labels: map[string]string{constants.AlwaysPushLabel: "true"},
	}
	for _, f := range m.wasmPluginHandlers {
		IngressLog.Debug("WasmPlugin triggerd update")
		f(config.Config{Meta: metadata}, config.Config{Meta: metadata}, istiomodel.EventUpdate)
	}
	istioWasmPlugin, err := m.convertIstioWasmPlugin(&wasmPlugin.Spec)
	if err != nil {
		IngressLog.Errorf("invalid wasmPlugin:%s, err:%v", clusterNamespacedName.Name, err)
		return
	}
	if istioWasmPlugin == nil {
		IngressLog.Infof("wasmPlugin:%s will not be transferred to istio since config disabled",
			clusterNamespacedName.Name)
		m.mutex.Lock()
		delete(m.wasmPlugins, clusterNamespacedName.Name)
		m.mutex.Unlock()
		return
	}
	IngressLog.Debugf("wasmPlugin:%s convert to istioWasmPlugin:%v", clusterNamespacedName.Name, istioWasmPlugin)
	m.mutex.Lock()
	m.wasmPlugins[clusterNamespacedName.Name] = istioWasmPlugin
	m.mutex.Unlock()
}

func (m *IngressConfig) DeleteWasmPlugin(clusterNamespacedName util.ClusterNamespacedName) {
	if clusterNamespacedName.Namespace != m.namespace {
		return
	}
	var hit bool
	m.mutex.Lock()
	if _, ok := m.wasmPlugins[clusterNamespacedName.Name]; ok {
		delete(m.wasmPlugins, clusterNamespacedName.Name)
		hit = true
	}
	m.mutex.Unlock()
	if hit {
		metadata := config.Meta{
			Name:             clusterNamespacedName.Name + "-wasmplugin",
			Namespace:        clusterNamespacedName.Namespace,
			GroupVersionKind: gvk.WasmPlugin,
			// Set this label so that we do not compare configs and just push.
			Labels: map[string]string{constants.AlwaysPushLabel: "true"},
		}
		for _, f := range m.wasmPluginHandlers {
			IngressLog.Debug("WasmPlugin triggerd update")
			f(config.Config{Meta: metadata}, config.Config{Meta: metadata}, istiomodel.EventDelete)
		}
	}
}

func (m *IngressConfig) AddOrUpdateMcpBridge(clusterNamespacedName util.ClusterNamespacedName) {
	// TODO: get resource name from config
	if clusterNamespacedName.Name != DefaultMcpbridgeName || clusterNamespacedName.Namespace != m.namespace {
		return
	}
	mcpbridge, err := m.mcpbridgeLister.McpBridges(clusterNamespacedName.Namespace).Get(clusterNamespacedName.Name)
	if err != nil {
		IngressLog.Errorf("Mcpbridge is not found, namespace:%s, name:%s",
			clusterNamespacedName.Namespace, clusterNamespacedName.Name)
		return
	}
	if m.RegistryReconciler == nil {
		m.RegistryReconciler = reconcile.NewReconciler(func() {
			seMetadata := config.Meta{
				Name:             "mcpbridge-serviceentry",
				Namespace:        m.namespace,
				GroupVersionKind: gvk.ServiceEntry,
				// Set this label so that we do not compare configs and just push.
				Labels: map[string]string{constants.AlwaysPushLabel: "true"},
			}
			drMetadata := config.Meta{
				Name:             "mcpbridge-destinationrule",
				Namespace:        m.namespace,
				GroupVersionKind: gvk.DestinationRule,
				// Set this label so that we do not compare configs and just push.
				Labels: map[string]string{constants.AlwaysPushLabel: "true"},
			}
			for _, f := range m.serviceEntryHandlers {
				IngressLog.Debug("McpBridge triggerd serviceEntry update")
				f(config.Config{Meta: seMetadata}, config.Config{Meta: seMetadata}, istiomodel.EventUpdate)
			}
			for _, f := range m.destinationRuleHandlers {
				IngressLog.Debug("McpBridge triggerd destinationRule update")
				f(config.Config{Meta: drMetadata}, config.Config{Meta: drMetadata}, istiomodel.EventUpdate)
			}
		}, m.localKubeClient, m.namespace)
	}
	reconciler := m.RegistryReconciler
	err = reconciler.Reconcile(mcpbridge)
	if err != nil {
		IngressLog.Errorf("Mcpbridge reconcile failed, err:%v", err)
		return
	}
	IngressLog.Info("Mcpbridge reconciled")
}

func (m *IngressConfig) DeleteMcpBridge(clusterNamespacedName util.ClusterNamespacedName) {
	// TODO: get resource name from config
	if clusterNamespacedName.Name != "default" || clusterNamespacedName.Namespace != m.namespace {
		return
	}
	if m.RegistryReconciler != nil {
		go m.RegistryReconciler.Reconcile(nil)
		m.RegistryReconciler = nil
	}
}

func (m *IngressConfig) AddOrUpdateHttp2Rpc(clusterNamespacedName util.ClusterNamespacedName) {
	if clusterNamespacedName.Namespace != m.namespace {
		return
	}
	http2rpc, err := m.http2rpcLister.Http2Rpcs(clusterNamespacedName.Namespace).Get(clusterNamespacedName.Name)
	if err != nil {
		IngressLog.Errorf("http2rpc is not found, namespace:%s, name:%s",
			clusterNamespacedName.Namespace, clusterNamespacedName.Name)
		return
	}
	m.mutex.Lock()
	m.http2rpcs[clusterNamespacedName.Name] = &http2rpc.Spec
	m.mutex.Unlock()
	IngressLog.Infof("AddOrUpdateHttp2Rpc http2rpc ingress name %s", clusterNamespacedName.Name)
	push := func(gvk config.GroupVersionKind) {
		m.XDSUpdater.ConfigUpdate(&istiomodel.PushRequest{
			Full: true,
			ConfigsUpdated: map[istiomodel.ConfigKey]struct{}{{
				Kind:      kind.MustFromGVK(gvk),
				Name:      clusterNamespacedName.Name,
				Namespace: clusterNamespacedName.Namespace,
			}: {}},
			Reason: istiomodel.NewReasonStats("Http2Rpc-AddOrUpdate"),
		})
	}
	push(gvk.VirtualService)
	push(gvk.EnvoyFilter)
}

func (m *IngressConfig) DeleteHttp2Rpc(clusterNamespacedName util.ClusterNamespacedName) {
	IngressLog.Infof("Http2Rpc triggered deleted event %s", clusterNamespacedName.Name)
	if clusterNamespacedName.Namespace != m.namespace {
		return
	}
	var hit bool
	m.mutex.Lock()
	if _, ok := m.http2rpcs[clusterNamespacedName.Name]; ok {
		delete(m.http2rpcs, clusterNamespacedName.Name)
		hit = true
	}
	m.mutex.Unlock()
	if hit {
		IngressLog.Infof("Http2Rpc triggerd deleted event executed %s", clusterNamespacedName.Name)
		push := func(gvk config.GroupVersionKind) {
			m.XDSUpdater.ConfigUpdate(&istiomodel.PushRequest{
				Full: true,
				ConfigsUpdated: map[istiomodel.ConfigKey]struct{}{{
					Kind:      kind.MustFromGVK(gvk),
					Name:      clusterNamespacedName.Name,
					Namespace: clusterNamespacedName.Namespace,
				}: {}},
				Reason: istiomodel.NewReasonStats("Http2Rpc-Deleted"),
			})
		}
		push(gvk.VirtualService)
		push(gvk.EnvoyFilter)
	}
}

func (m *IngressConfig) ReflectSecretChanges(clusterNamespacedName util.ClusterNamespacedName) {
	var hit bool
	m.mutex.RLock()
	if m.watchedSecretSet.Contains(clusterNamespacedName.String()) {
		hit = true
	}
	m.mutex.RUnlock()

	if hit {
		push := func(gvk config.GroupVersionKind) {
			m.XDSUpdater.ConfigUpdate(&istiomodel.PushRequest{
				Full: true,
				ConfigsUpdated: map[istiomodel.ConfigKey]struct{}{{
					Kind:      kind.MustFromGVK(gvk),
					Name:      clusterNamespacedName.Name,
					Namespace: clusterNamespacedName.Namespace,
				}: {}},
				Reason: istiomodel.NewReasonStats("auth-secret-change"),
			})
		}
		push(gvk.VirtualService)
		push(gvk.EnvoyFilter)
	}
}

func normalizeWeightedCluster(cache *common.IngressRouteCache, route *common.WrapperHTTPRoute) {
	if len(route.HTTPRoute.Route) == 1 {
		route.HTTPRoute.Route[0].Weight = 100
		return
	}

	var weightTotal int32 = 0
	for idx, routeDestination := range route.HTTPRoute.Route {
		if idx == 0 {
			continue
		}

		weightTotal += routeDestination.Weight
	}

	if weightTotal < route.WeightTotal {
		weightTotal = route.WeightTotal
	}

	var sum int32
	for idx, routeDestination := range route.HTTPRoute.Route {
		if idx == 0 {
			continue
		}

		weight := float32(routeDestination.Weight) / float32(weightTotal)
		routeDestination.Weight = int32(weight * 100)

		sum += routeDestination.Weight
	}

	route.HTTPRoute.Route[0].Weight = 100 - sum

	// Update the recorded status in ingress builder
	if cache != nil {
		cache.Update(route)
	}
}

func (m *IngressConfig) applyCanaryIngresses(convertOptions *common.ConvertOptions) {
	if len(convertOptions.CanaryIngresses) == 0 {
		return
	}

	IngressLog.Infof("Found %d number of canary ingresses.", len(convertOptions.CanaryIngresses))
	for _, cfg := range convertOptions.CanaryIngresses {
		clusterId := common.GetClusterId(cfg.Config.Annotations)
		m.mutex.RLock()
		ingressController := m.remoteIngressControllers[clusterId]
		m.mutex.RUnlock()
		if ingressController == nil {
			continue
		}
		if err := ingressController.ApplyCanaryIngress(convertOptions, cfg); err != nil {
			IngressLog.Errorf("Apply canary ingress %s/%s fail in cluster %s, err %v", cfg.Config.Namespace, cfg.Config.Name, clusterId, err)
		}
	}
}

func (m *IngressConfig) constructHttp2RpcEnvoyFilter(http2rpcConfig *annotations.Http2RpcConfig, route *common.WrapperHTTPRoute, namespace string, initHttp2RpcGlobalConfig bool) (*config.Config, error) {
	mappings := m.http2rpcs
	IngressLog.Infof("Found http2rpc mappings %v", mappings)
	if _, exist := mappings[http2rpcConfig.Name]; !exist {
		IngressLog.Errorf("Http2RpcConfig name %s, not found Http2Rpc CRD", http2rpcConfig.Name)
		return nil, errors.New("invalid http2rpcConfig has no usable http2rpc")
	}
	http2rpcCRD := mappings[http2rpcConfig.Name]

	if http2rpcCRD.GetDubbo() == nil {
		IngressLog.Errorf("Http2RpcConfig name %s, only support Http2Rpc CRD Dubbo Service type", http2rpcConfig.Name)
		return nil, errors.New("invalid http2rpcConfig has no usable http2rpc")
	}

	httpRoute := route.HTTPRoute
	httpRouteDestination := httpRoute.Route[0]
	typeStruct, err := m.constructHttp2RpcMethods(http2rpcCRD.GetDubbo())
	if err != nil {
		return nil, errors.New(err.Error())
	}
	configPatches := []*networking.EnvoyFilter_EnvoyConfigObjectPatch{
		{
			ApplyTo: networking.EnvoyFilter_HTTP_ROUTE,
			Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
				Context: networking.EnvoyFilter_GATEWAY,
				ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_RouteConfiguration{
					RouteConfiguration: &networking.EnvoyFilter_RouteConfigurationMatch{
						Vhost: &networking.EnvoyFilter_RouteConfigurationMatch_VirtualHostMatch{
							Route: &networking.EnvoyFilter_RouteConfigurationMatch_RouteMatch{
								Name: httpRoute.Name,
							},
						},
					},
				},
			},
			Patch: &networking.EnvoyFilter_Patch{
				Operation: networking.EnvoyFilter_Patch_MERGE,
				Value:     typeStruct,
			},
		},
		{
			ApplyTo: networking.EnvoyFilter_CLUSTER,
			Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
				Context: networking.EnvoyFilter_GATEWAY,
				ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_Cluster{
					Cluster: &networking.EnvoyFilter_ClusterMatch{
						Service: httpRouteDestination.Destination.Host,
					},
				},
			},
			Patch: &networking.EnvoyFilter_Patch{
				Operation: networking.EnvoyFilter_Patch_MERGE,
				Value: buildPatchStruct(`{
							"upstream_config": {
								"name":"envoy.upstreams.http.dubbo_tcp",
								"typed_config":{
									"@type":"type.googleapis.com/udpa.type.v1.TypedStruct",
									"type_url":"type.googleapis.com/envoy.extensions.upstreams.http.dubbo_tcp.v3.DubboTcpConnectionPoolProto"
								}
							}
						}`),
			},
		},
	}
	if initHttp2RpcGlobalConfig {
		configPatches = append(configPatches, &networking.EnvoyFilter_EnvoyConfigObjectPatch{
			ApplyTo: networking.EnvoyFilter_HTTP_FILTER,
			Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
				Context: networking.EnvoyFilter_GATEWAY,
				ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
					Listener: &networking.EnvoyFilter_ListenerMatch{
						FilterChain: &networking.EnvoyFilter_ListenerMatch_FilterChainMatch{
							Filter: &networking.EnvoyFilter_ListenerMatch_FilterMatch{
								Name: "envoy.filters.network.http_connection_manager",
								SubFilter: &networking.EnvoyFilter_ListenerMatch_SubFilterMatch{
									Name: "envoy.filters.http.router",
								},
							},
						},
					},
				},
			},
			Patch: &networking.EnvoyFilter_Patch{
				Operation: networking.EnvoyFilter_Patch_INSERT_BEFORE,
				Value: buildPatchStruct(`{
							"name":"envoy.filters.http.http_dubbo_transcoder",
							"typed_config":{
								"@type":"type.googleapis.com/udpa.type.v1.TypedStruct",
								"type_url":"type.googleapis.com/envoy.extensions.filters.http.http_dubbo_transcoder.v3.HttpDubboTranscoder"
							}
						}`),
			},
		})
	}
	return &config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.EnvoyFilter,
			Name:             common.CreateConvertedName(constants.IstioIngressGatewayName, http2rpcConfig.Name),
			Namespace:        namespace,
		},
		Spec: &networking.EnvoyFilter{
			ConfigPatches: configPatches,
		},
	}, nil
}

func (m *IngressConfig) constructHttp2RpcMethods(dubbo *higressv1.DubboService) (*_struct.Struct, error) {
	httpRouterTemplate := `{
		"route": {
			"upgrade_configs": [
				{
					"connect_config": {
						"allow_post": true
					},
					"upgrade_type": "CONNECT"
				}
			]
		},
		"typed_per_filter_config": {
			"envoy.filters.http.http_dubbo_transcoder": {
				"@type": "type.googleapis.com/udpa.type.v1.TypedStruct",
				"type_url": "type.googleapis.com/envoy.extensions.filters.http.http_dubbo_transcoder.v3.HttpDubboTranscoder",
				"value": {
					"request_validation_options": {
						"reject_unknown_method": true,
						"reject_unknown_query_parameters": true
					},
					"services_mapping": %s,
					"url_unescape_spec": "ALL_CHARACTERS_EXCEPT_RESERVED"
				}
			}
		}
	}`
	var methods []interface{}
	for _, serviceMethod := range dubbo.GetMethods() {
		var method = make(map[string]interface{})
		method["name"] = serviceMethod.GetServiceMethod()
		var params []interface{}
		// paramFromEntireBody is for methods with single parameter. So when paramFromEntireBody exists, we just ignore params.
		var paramFromEntireBody = serviceMethod.GetParamFromEntireBody()
		if paramFromEntireBody != nil {
			var param = make(map[string]interface{})
			param["extract_key_spec"] = Http2RpcParamSourceMap()["BODY"]
			param["mapping_type"] = paramFromEntireBody.GetParamType()
			params = append(params, param)
		} else {
			for _, methodParam := range serviceMethod.GetParams() {
				var param = make(map[string]interface{})
				param["extract_key"] = methodParam.GetParamKey()
				param["extract_key_spec"] = Http2RpcParamSourceMap()[methodParam.GetParamSource()]
				param["mapping_type"] = methodParam.GetParamType()
				params = append(params, param)
			}
		}
		method["parameter_mapping"] = params
		var path_matcher = make(map[string]interface{})
		path_matcher["match_http_method_spec"] = Http2RpcMethodMap()[serviceMethod.HttpMethods[0]]
		path_matcher["match_pattern"] = serviceMethod.GetHttpPath()
		method["path_matcher"] = path_matcher
		var passthrough_setting = make(map[string]interface{})
		var headersAttach = serviceMethod.GetHeadersAttach()
		if headersAttach == "" {
			passthrough_setting["passthrough_all_headers"] = false
		} else if headersAttach == "*" {
			passthrough_setting["passthrough_all_headers"] = true
		} else {
			passthrough_setting["passthrough_headers"] = headersAttach
		}
		method["passthrough_setting"] = passthrough_setting
		methods = append(methods, method)
	}
	var serviceMapping = make(map[string]interface{})
	var dubboServiceGroup = dubbo.GetGroup()
	if dubboServiceGroup != "" {
		serviceMapping["group"] = dubboServiceGroup
	}
	serviceMapping["name"] = dubbo.GetService()
	serviceMapping["version"] = dubbo.GetVersion()
	serviceMapping["method_mapping"] = methods
	strBuffer := new(bytes.Buffer)
	serviceMappingJsonStr, _ := json.Marshal(serviceMapping)
	fmt.Fprintf(strBuffer, httpRouterTemplate, string(serviceMappingJsonStr))
	IngressLog.Infof("Found http2rpc buildHttp2RpcMethods %s", strBuffer.String())
	result := buildPatchStruct(strBuffer.String())
	return result, nil
}

func buildPatchStruct(config string) *_struct.Struct {
	val := &_struct.Struct{}
	err := jsonpb.Unmarshal(strings.NewReader(config), val)
	if err != nil {
		log.Errorf("jsonpb unmarshal failed: %s", config)
	}
	return val
}

func constructBasicAuthEnvoyFilter(rules *common.BasicAuthRules, namespace string) (*config.Config, error) {
	rulesStr, err := json.Marshal(rules)
	if err != nil {
		return nil, err
	}
	configuration := &wrappers.StringValue{
		Value: string(rulesStr),
	}

	wasm := &wasm.Wasm{
		Config: &v3.PluginConfig{
			Name:     "basic-auth",
			FailOpen: true,
			Vm: &v3.PluginConfig_VmConfig{
				VmConfig: &v3.VmConfig{
					Runtime: "envoy.wasm.runtime.null",
					Code: &corev3.AsyncDataSource{
						Specifier: &corev3.AsyncDataSource_Local{
							Local: &corev3.DataSource{
								Specifier: &corev3.DataSource_InlineString{
									InlineString: "envoy.wasm.basic_auth",
								},
							},
						},
					},
				},
			},
			Configuration: protoconv.MessageToAny(configuration),
		},
	}

	wasmAny, err := anypb.New(wasm)
	if err != nil {
		return nil, err
	}

	typedConfig := &httppb.HttpFilter{
		Name: "basic-auth",
		ConfigType: &httppb.HttpFilter_TypedConfig{
			TypedConfig: wasmAny,
		},
	}

	pbTypedConfig, err := util.MessageToStruct(typedConfig)
	if err != nil {
		return nil, err
	}

	return &config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.EnvoyFilter,
			Name:             common.CreateConvertedName(constants.IstioIngressGatewayName, "basic-auth"),
			Namespace:        namespace,
		},
		Spec: &networking.EnvoyFilter{
			ConfigPatches: []*networking.EnvoyFilter_EnvoyConfigObjectPatch{
				{
					ApplyTo: networking.EnvoyFilter_HTTP_FILTER,
					Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
						Context: networking.EnvoyFilter_GATEWAY,
						ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
							Listener: &networking.EnvoyFilter_ListenerMatch{
								FilterChain: &networking.EnvoyFilter_ListenerMatch_FilterChainMatch{
									Filter: &networking.EnvoyFilter_ListenerMatch_FilterMatch{
										Name: "envoy.filters.network.http_connection_manager",
										SubFilter: &networking.EnvoyFilter_ListenerMatch_SubFilterMatch{
											Name: "envoy.filters.http.cors",
										},
									},
								},
							},
						},
					},
					Patch: &networking.EnvoyFilter_Patch{
						Operation: networking.EnvoyFilter_Patch_INSERT_AFTER,
						Value:     pbTypedConfig,
					},
				},
			},
		},
	}, nil
}

func QueryByName(serviceEntries []*memory.ServiceWrapper, serviceName string) (*memory.ServiceWrapper, error) {
	IngressLog.Infof("Found http2rpc serviceEntries %s", serviceEntries)
	for _, se := range serviceEntries {
		if se.ServiceName == serviceName {
			return se, nil
		}
	}
	return nil, fmt.Errorf("can't find ServiceEntry by serviceName:%v", serviceName)
}

func QueryRpcServiceVersion(serviceEntry *memory.ServiceWrapper, serviceName string) (string, error) {
	IngressLog.Infof("Found http2rpc serviceEntry %s", serviceEntry)
	IngressLog.Infof("Found http2rpc ServiceEntry %s", serviceEntry.ServiceEntry)
	IngressLog.Infof("Found http2rpc WorkloadSelector %s", serviceEntry.ServiceEntry.WorkloadSelector)
	IngressLog.Infof("Found http2rpc Labels %s", serviceEntry.ServiceEntry.WorkloadSelector.Labels)
	labels := (*serviceEntry).ServiceEntry.WorkloadSelector.Labels
	for key, value := range labels {
		if key == "version" {
			return value, nil
		}
	}
	return "", fmt.Errorf("can't get RpcServiceVersion for serviceName:%v", serviceName)
}

func (m *IngressConfig) Run(stop <-chan struct{}) {
	for _, remoteIngressController := range m.remoteIngressControllers {
		_ = remoteIngressController.SetWatchErrorHandler(m.watchErrorHandler)
		go remoteIngressController.Run(stop)
	}
	for _, remoteGatewayController := range m.remoteGatewayControllers {
		_ = remoteGatewayController.SetWatchErrorHandler(m.watchErrorHandler)
		go remoteGatewayController.Run(stop)
	}
	go m.mcpbridgeController.Run(stop)
	go m.wasmPluginController.Run(stop)
	go m.http2rpcController.Run(stop)
	go m.configmapMgr.HigressConfigController.Run(stop)
}

func (m *IngressConfig) HasSynced() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, remoteIngressController := range m.remoteIngressControllers {
		if !remoteIngressController.HasSynced() {
			return false
		}
	}
	for _, remoteGatewayController := range m.remoteGatewayControllers {
		if !remoteGatewayController.HasSynced() {
			return false
		}
	}
	if !m.mcpbridgeController.HasSynced() {
		return false
	}
	if !m.wasmPluginController.HasSynced() {
		return false
	}
	if !m.http2rpcController.HasSynced() {
		return false
	}
	if !m.configmapMgr.HigressConfigController.HasSynced() {
		return false
	}
	IngressLog.Info("Ingress config controller synced.")
	return true
}

func (m *IngressConfig) SetWatchErrorHandler(f func(r *cache.Reflector, err error)) error {
	m.watchErrorHandler = f
	return nil
}

func (m *IngressConfig) GetIngressRoutes() istiomodel.IngressRouteCollection {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.ingressRouteCache
}

func (m *IngressConfig) GetIngressDomains() istiomodel.IngressDomainCollection {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.ingressDomainCache
}

func (m *IngressConfig) CheckIngress(clusterName string) istiomodel.CheckIngressResponse {
	return istiomodel.CheckIngressResponse{}
}

func (m *IngressConfig) Services(clusterName string) ([]*v1.Service, error) {
	return nil, nil
}

func (m *IngressConfig) IngressControllers() map[string]string {
	return nil
}

func (m *IngressConfig) Schemas() collection.Schemas {
	return common.IngressIR
}

func (m *IngressConfig) Get(config.GroupVersionKind, string, string) *config.Config {
	return nil
}

func (m *IngressConfig) Create(config.Config) (revision string, err error) {
	return "", common.ErrUnsupportedOp
}

func (m *IngressConfig) Update(config.Config) (newRevision string, err error) {
	return "", common.ErrUnsupportedOp
}

func (m *IngressConfig) UpdateStatus(config.Config) (newRevision string, err error) {
	return "", common.ErrUnsupportedOp
}

func (m *IngressConfig) Patch(config.Config, config.PatchFunc) (string, error) {
	return "", common.ErrUnsupportedOp
}

func (m *IngressConfig) Delete(config.GroupVersionKind, string, string, *string) error {
	return common.ErrUnsupportedOp
}
