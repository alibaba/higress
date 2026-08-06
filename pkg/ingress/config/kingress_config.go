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
	"sync"

	networking "istio.io/api/networking/v1alpha3"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/config/schema/collection"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/util/sets"
	v1 "k8s.io/api/core/v1"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/alibaba/higress/v2/pkg/ingress/kube/annotations"
	"github.com/alibaba/higress/v2/pkg/ingress/kube/common"
	"github.com/alibaba/higress/v2/pkg/ingress/kube/kingress"
	"github.com/alibaba/higress/v2/pkg/ingress/kube/secret"
	"github.com/alibaba/higress/v2/pkg/ingress/kube/util"
	. "github.com/alibaba/higress/v2/pkg/ingress/log"
	"github.com/alibaba/higress/v2/pkg/kube"
	"github.com/alibaba/higress/v2/registry/reconcile"
)

var (
	_ istiomodel.ConfigStoreController = &KIngressConfig{}
	_ istiomodel.IngressStore          = &KIngressConfig{}
)

type KIngressConfig struct {
	remoteIngressControllers map[cluster.ID]common.KIngressController
	mutex                    sync.RWMutex

	ingressRouteCache  istiomodel.IngressRouteCollection
	ingressDomainCache istiomodel.IngressDomainCollection

	localKubeClient        kube.Client
	virtualServiceHandlers []istiomodel.EventHandler
	gatewayHandlers        []istiomodel.EventHandler
	envoyFilterHandlers    []istiomodel.EventHandler
	watchErrorHandler      cache.WatchErrorHandler

	cachedEnvoyFilters []config.Config

	watchedSecretSet sets.Set[string]

	RegistryReconciler *reconcile.Reconciler

	XDSUpdater istiomodel.XDSUpdater

	annotationHandler annotations.AnnotationHandler

	globalGatewayName string

	namespace string

	clusterId cluster.ID
}

func NewKIngressConfig(localKubeClient kube.Client, XDSUpdater istiomodel.XDSUpdater, namespace string, options common.Options) *KIngressConfig {
	if localKubeClient.KIngressInformer() == nil {
		return nil
	}
	clusterId := options.ClusterId
	if clusterId == "Kubernetes" {
		clusterId = ""
	}
	config := &KIngressConfig{
		remoteIngressControllers: make(map[cluster.ID]common.KIngressController),
		localKubeClient:          localKubeClient,
		XDSUpdater:               XDSUpdater,
		annotationHandler:        annotations.NewAnnotationHandlerManager(),
		clusterId:                clusterId,
		globalGatewayName:        namespace + "/" + common.CreateConvertedName(clusterId.String(), "global"),
		watchedSecretSet:         sets.New[string](),
		namespace:                namespace,
	}
	return config
}

func (m *KIngressConfig) RegisterEventHandler(kind config.GroupVersionKind, f istiomodel.EventHandler) {
	IngressLog.Infof("register resource %v", kind)
	switch kind {
	case gvk.VirtualService:
		m.virtualServiceHandlers = append(m.virtualServiceHandlers, f)

	case gvk.Gateway:
		m.gatewayHandlers = append(m.gatewayHandlers, f)

	case gvk.EnvoyFilter:
		m.envoyFilterHandlers = append(m.envoyFilterHandlers, f)
	}

	for _, remoteIngressController := range m.remoteIngressControllers {
		remoteIngressController.RegisterEventHandler(kind, f)
	}
}

func (m *KIngressConfig) AddLocalCluster(options common.Options) common.KIngressController {
	secretController := secret.NewController(m.localKubeClient, options)
	secretController.AddEventHandler(m.ReflectSecretChanges)

	var ingressController common.KIngressController

	ingressController = kingress.NewController(m.localKubeClient, m.localKubeClient, options, secretController)

	m.remoteIngressControllers[options.ClusterId] = ingressController
	return ingressController
}

func (m *KIngressConfig) List(typ config.GroupVersionKind, namespace string) []config.Config {
	if typ == gvk.EnvoyFilter || typ == gvk.DestinationRule || typ == gvk.WasmPlugin || typ == gvk.ServiceEntry {
		return nil
	}
	if typ != gvk.Gateway && typ != gvk.VirtualService {
		return nil
	}

	// Currently, only support list all namespaces gateways or virtualservices.
	if namespace != "" {
		IngressLog.Warnf("ingress store only support type %s of all namespace.", typ)
		return nil
	}

	var configs []config.Config
	m.mutex.RLock()
	for _, ingressController := range m.remoteIngressControllers {
		configs = append(configs, ingressController.List()...)
	}
	m.mutex.RUnlock()

	common.SortIngressByCreationTime(configs)
	wrapperConfigs := m.createWrapperConfigs(configs)

	IngressLog.Infof("resource type %s, configs number %d", typ, len(wrapperConfigs))
	switch typ {
	case gvk.Gateway:
		return m.convertGateways(wrapperConfigs)
	case gvk.VirtualService:
		return m.convertVirtualService(wrapperConfigs)
	}
	return nil
}

func (m *KIngressConfig) createWrapperConfigs(configs []config.Config) []common.WrapperConfig {
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

func (m *KIngressConfig) convertGateways(configs []common.WrapperConfig) []config.Config {
	convertOptions := common.ConvertOptions{
		IngressDomainCache: common.NewIngressDomainCache(),
		Gateways:           map[string]*common.WrapperGateway{},
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
		if err := ingressController.ConvertGateway(&convertOptions, &cfg); err != nil {
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

func (m *KIngressConfig) convertVirtualService(configs []common.WrapperConfig) []config.Config {
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

	// Normalize weighted cluster to make sure the sum of weight is 100.
	for _, host := range convertOptions.HTTPRoutes {
		for _, route := range host {
			normalizeWeightedKCluster(convertOptions.IngressRouteCache, route)
		}
	}

	// Apply annotation on virtual services Only IP-control and do nothing
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
		gateways := []string{
			m.namespace + "/" +
				common.CreateConvertedName(m.clusterId.String(), cleanHost),
			common.CreateConvertedName(constants.IstioIngressGatewayName, cleanHost),
		}

		wrapperVS, exist := convertOptions.VirtualServices[host]
		if !exist {
			IngressLog.Warnf("virtual service for host %s does not exist.", host)
		}
		vs := wrapperVS.VirtualService
		vs.Gateways = gateways

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

	return out
}

// Make sure that the sum of traffic split ratio is 100, if it is not 100, it will be normalized
func normalizeWeightedKCluster(cache *common.IngressRouteCache, route *common.WrapperHTTPRoute) {
	if len(route.HTTPRoute.Route) == 1 {
		route.HTTPRoute.Route[0].Weight = 100
		return
	}

	var weightTotal int32 = 0
	for _, routeDestination := range route.HTTPRoute.Route {
		weightTotal += routeDestination.Weight
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

func (m *KIngressConfig) applyAppRoot(convertOptions *common.ConvertOptions) {
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

func (m *KIngressConfig) applyInternalActiveRedirect(convertOptions *common.ConvertOptions) {
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

func (m *KIngressConfig) ReflectSecretChanges(clusterNamespacedName util.ClusterNamespacedName) {
	var hit bool
	m.mutex.RLock()
	if m.watchedSecretSet.Contains(clusterNamespacedName.String()) {
		hit = true
	}
	m.mutex.RUnlock()

	if hit {
		push := func(GVK config.GroupVersionKind) {
			m.XDSUpdater.ConfigUpdate(&istiomodel.PushRequest{
				Full: true,
				ConfigsUpdated: map[istiomodel.ConfigKey]struct{}{{
					Kind:      gvk.MustToKind(GVK),
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

func (m *KIngressConfig) Run(stop <-chan struct{}) {
	for _, remoteIngressController := range m.remoteIngressControllers {
		_ = remoteIngressController.SetWatchErrorHandler(m.watchErrorHandler)
		go remoteIngressController.Run(stop)
	}
}

func (m *KIngressConfig) HasSynced() bool {
	IngressLog.Info("In Kingress Synced.")
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, remoteIngressController := range m.remoteIngressControllers {
		IngressLog.Info("In Kingress Synced.")
		if !remoteIngressController.HasSynced() {
			return false
		}
	}
	IngressLog.Info("KIngress config controller synced.")
	return true
}

func (m *KIngressConfig) SetWatchErrorHandler(f func(r *cache.Reflector, err error)) error {
	m.watchErrorHandler = f
	return nil
}

func (m *KIngressConfig) GetIngressRoutes() istiomodel.IngressRouteCollection {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.ingressRouteCache
}

func (m *KIngressConfig) GetIngressDomains() istiomodel.IngressDomainCollection {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.ingressDomainCache
}

func (m *KIngressConfig) CheckIngress(clusterName string) istiomodel.CheckIngressResponse {
	return istiomodel.CheckIngressResponse{}
}

func (m *KIngressConfig) Services(clusterName string) ([]*v1.Service, error) {
	return nil, nil
}

func (m *KIngressConfig) IngressControllers() map[string]string {
	return nil
}

func (m *KIngressConfig) Schemas() collection.Schemas {
	return common.IngressIR
}

func (m *KIngressConfig) Get(config.GroupVersionKind, string, string) *config.Config {
	return nil
}

func (m *KIngressConfig) Create(config.Config) (revision string, err error) {
	return "", common.ErrUnsupportedOp
}

func (m *KIngressConfig) Update(config.Config) (newRevision string, err error) {
	return "", common.ErrUnsupportedOp
}

func (m *KIngressConfig) UpdateStatus(config.Config) (newRevision string, err error) {
	return "", common.ErrUnsupportedOp
}

func (m *KIngressConfig) Patch(config.Config, config.PatchFunc) (string, error) {
	return "", common.ErrUnsupportedOp
}

func (m *KIngressConfig) Delete(config.GroupVersionKind, string, string, *string) error {
	return common.ErrUnsupportedOp
}
