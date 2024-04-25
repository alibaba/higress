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

package bootstrap

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/alibaba/higress/pkg/cert"
	"github.com/alibaba/higress/pkg/ingress/kube/common"
	"github.com/alibaba/higress/pkg/ingress/mcp"
	"github.com/alibaba/higress/pkg/ingress/translation"
	higresskube "github.com/alibaba/higress/pkg/kube"
	prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"istio.io/api/mesh/v1alpha1"
	configaggregate "istio.io/istio/pilot/pkg/config/aggregate"
	"istio.io/istio/pilot/pkg/features"
	istiogrpc "istio.io/istio/pilot/pkg/grpc"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/server"
	"istio.io/istio/pilot/pkg/serviceregistry/aggregate"
	kubecontroller "istio.io/istio/pilot/pkg/serviceregistry/kube/controller"
	"istio.io/istio/pilot/pkg/xds"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/config/mesh"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/keepalive"
	istiokube "istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/security"
	"istio.io/istio/security/pkg/server/ca/authenticate"
	"istio.io/istio/security/pkg/server/ca/authenticate/kubeauth"
	"istio.io/pkg/env"
	"istio.io/pkg/ledger"
	"istio.io/pkg/log"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type XdsOptions struct {
	// DebounceAfter is the delay added to events to wait after a registry/config event for debouncing.
	// This will delay the push by at least this interval, plus the time getting subsequent events. If no change is
	// detected the push will happen, otherwise we'll keep delaying until things settle.
	DebounceAfter time.Duration
	// DebounceMax is the maximum time to wait for events while debouncing. Defaults to 10 seconds. If events keep
	// showing up with no break for this time, we'll trigger a push.
	DebounceMax time.Duration
	// EnableEDSDebounce indicates whether EDS pushes should be debounced.
	EnableEDSDebounce bool
}

// RegistryOptions provide configuration options for the configuration controller. If FileDir is set, that directory will
// be monitored for CRD yaml files and will update the controller as those files change (This is used for testing
// purposes). Otherwise, a CRD client is created based on the configuration.
type RegistryOptions struct {
	// If FileDir is set, the below kubernetes options are ignored
	FileDir string

	Registries []string

	// Kubernetes controller options
	KubeOptions kubecontroller.Options
	// ClusterRegistriesNamespace specifies where the multi-cluster secret resides
	ClusterRegistriesNamespace string
	KubeConfig                 string

	// DistributionTracking control
	DistributionCacheRetention time.Duration

	// DistributionTracking control
	DistributionTrackingEnabled bool
}

type ServerArgs struct {
	Debug       bool
	MeshId      string
	RegionId    string
	NativeIstio bool
	HttpAddress string
	GrpcAddress string

	// IngressClass filters which ingress resources the higress controller watches.
	// The default ingress class is higress.
	// There are some special cases for special ingress class.
	// 1. When the ingress class is set as nginx, the higress controller will watch ingress
	// resources with the nginx ingress class or without any ingress class.
	// 2. When the ingress class is set empty, the higress controller will watch all ingress
	// resources in the k8s cluster.
	IngressClass         string
	EnableStatus         bool
	WatchNamespace       string
	GrpcKeepAliveOptions *keepalive.Options
	XdsOptions           XdsOptions
	RegistryOptions      RegistryOptions
	KeepStaleWhenEmpty   bool
	GatewaySelectorKey   string
	GatewaySelectorValue string
	GatewayHttpPort      uint32
	GatewayHttpsPort     uint32
	EnableAutomaticHttps bool
	AutomaticHttpsEmail  string
	CertHttpAddress      string
}

type readinessProbe func() (bool, error)

type ServerInterface interface {
	Start(stop <-chan struct{}) error
	WaitUntilCompletion()
}

type Server struct {
	*ServerArgs
	environment      *model.Environment
	kubeClient       higresskube.Client
	configController model.ConfigStoreCache
	configStores     []model.ConfigStoreCache
	httpServer       *http.Server
	httpMux          *http.ServeMux
	grpcServer       *grpc.Server
	xdsServer        *xds.DiscoveryServer
	server           server.Instance
	readinessProbes  map[string]readinessProbe
	certServer       *cert.Server
}

var (
	PodNamespace = env.RegisterStringVar("POD_NAMESPACE", "higress-system", "").Get()
	PodName      = env.RegisterStringVar("POD_NAME", "", "").Get()
)

func NewServer(args *ServerArgs) (*Server, error) {
	e := &model.Environment{
		PushContext:  model.NewPushContext(),
		DomainSuffix: constants.DefaultKubernetesDomain,
		MCPMode:      true,
	}
	e.SetLedger(buildLedger(args.RegistryOptions))

	ac := aggregate.NewController(aggregate.Options{
		MeshHolder: e,
	})
	e.ServiceDiscovery = ac
	s := &Server{
		ServerArgs:      args,
		httpMux:         http.NewServeMux(),
		environment:     e,
		readinessProbes: make(map[string]readinessProbe),
		server:          server.New(),
	}
	s.environment.Watcher = mesh.NewFixedWatcher(&v1alpha1.MeshConfig{})
	s.environment.Init()
	initFuncList := []func() error{
		s.initKubeClient,
		s.initXdsServer,
		s.initHttpServer,
		s.initConfigController,
		s.initRegistryEventHandlers,
		s.initAuthenticators,
		s.initAutomaticHttps,
	}

	for _, f := range initFuncList {
		if err := f(); err != nil {
			return nil, err
		}
	}

	s.server.RunComponent(func(stop <-chan struct{}) error {
		s.kubeClient.RunAndWait(stop)
		return nil
	})

	s.readinessProbes["xds"] = func() (bool, error) {
		return s.xdsServer.IsServerReady(), nil
	}

	return s, nil
}

// initRegistryEventHandlers sets up event handlers for config updates
func (s *Server) initRegistryEventHandlers() error {
	log.Info("initializing registry event handlers")
	configHandler := func(prev config.Config, curr config.Config, event model.Event) {
		// For update events, trigger push only if spec has changed.
		pushReq := &model.PushRequest{
			Full: true,
			ConfigsUpdated: map[model.ConfigKey]struct{}{{
				Kind:      curr.GroupVersionKind,
				Name:      curr.Name,
				Namespace: curr.Namespace,
			}: {}},
			Reason: []model.TriggerReason{model.ConfigUpdate},
		}
		s.xdsServer.ConfigUpdate(pushReq)
	}
	schemas := common.IngressIR.All()
	for _, schema := range schemas {
		s.configController.RegisterEventHandler(schema.Resource().GroupVersionKind(), configHandler)
	}
	return nil
}

func (s *Server) initConfigController() error {
	ns := PodNamespace
	options := common.Options{
		Enable:               true,
		ClusterId:            string(s.RegistryOptions.KubeOptions.ClusterID),
		IngressClass:         s.IngressClass,
		WatchNamespace:       s.WatchNamespace,
		EnableStatus:         s.EnableStatus,
		SystemNamespace:      ns,
		GatewaySelectorKey:   s.GatewaySelectorKey,
		GatewaySelectorValue: s.GatewaySelectorValue,
		GatewayHttpPort:      s.GatewayHttpPort,
		GatewayHttpsPort:     s.GatewayHttpsPort,
	}
	if options.ClusterId == "Kubernetes" {
		options.ClusterId = ""
	}

	ingressConfig := translation.NewIngressTranslation(s.kubeClient, s.xdsServer, ns, options.ClusterId)
	ingressController, kingressController := ingressConfig.AddLocalCluster(options)

	s.configStores = append(s.configStores, ingressConfig)

	// Wrap the config controller with a cache.
	aggregateConfigController, err := configaggregate.MakeCache(s.configStores)
	if err != nil {
		return err
	}
	s.configController = aggregateConfigController

	// Create the config store.
	s.environment.IstioConfigStore = model.MakeIstioStore(s.configController)

	s.environment.IngressStore = ingressConfig

	// Defer starting the controller until after the service is created.
	s.server.RunComponent(func(stop <-chan struct{}) error {
		if err := ingressConfig.InitializeCluster(ingressController, kingressController, stop); err != nil {
			return err
		}
		go s.configController.Run(stop)
		return nil
	})
	return nil
}

func (s *Server) Start(stop <-chan struct{}) error {
	if err := s.server.Start(stop); err != nil {
		return err
	}
	if !s.waitForCacheSync(stop) {
		return fmt.Errorf("failed to sync cache")
	}
	// Inform Discovery Server so that it can start accepting connections.
	s.xdsServer.CachesSynced()
	grpcListener, err := net.Listen("tcp", s.GrpcAddress)
	if err != nil {
		return err
	}
	go func() {
		log.Infof("starting gRPC discovery service at %s", grpcListener.Addr())
		if err := s.grpcServer.Serve(grpcListener); err != nil {
			log.Errorf("error serving GRPC server: %v", err)
		}
	}()
	httpListener, err := net.Listen("tcp", s.HttpAddress)
	if err != nil {
		return err
	}
	go func() {
		log.Infof("starting HTTP service at %s", httpListener.Addr())
		if err := s.httpServer.Serve(httpListener); err != nil {
			log.Errorf("error serving http server: %v", err)
		}
	}()

	if s.EnableAutomaticHttps {
		go func() {
			log.Infof("starting Automatic Cert HTTP service at %s", s.CertHttpAddress)
			if err := s.certServer.Run(stop); err != nil {
				log.Errorf("error serving Automatic Cert HTTP server: %v", err)
			}
		}()
	}

	s.waitForShutDown(stop)
	return nil
}

func (s *Server) waitForShutDown(stop <-chan struct{}) {
	go func() {
		<-stop

		stopped := make(chan struct{})
		go func() {
			// Some grpcServer implementations do not support GracefulStop. Unfortunately, this is not
			// exposed; they just panic. To avoid this, we will recover and do a standard Stop when its not
			// support.
			defer func() {
				if r := recover(); r != nil {
					s.grpcServer.Stop()
					close(stopped)
				}
			}()
			s.grpcServer.GracefulStop()
			close(stopped)
		}()

		timer := time.NewTimer(time.Second * 2)
		select {
		case <-timer.C:
			s.grpcServer.Stop()
		case <-stopped:
			timer.Stop()
		}

		s.xdsServer.Shutdown()
	}()
}

func (s *Server) WaitUntilCompletion() {
	s.server.Wait()
}

func (s *Server) initXdsServer() error {
	log.Info("init xds server")
	s.xdsServer = xds.NewDiscoveryServer(s.environment, nil, PodName, PodNamespace, s.RegistryOptions.KubeOptions.ClusterAliases)
	s.xdsServer.McpGenerators[gvk.WasmPlugin.String()] = &mcp.WasmpluginGenerator{Server: s.xdsServer}
	s.xdsServer.McpGenerators[gvk.DestinationRule.String()] = &mcp.DestinationRuleGenerator{Server: s.xdsServer}
	s.xdsServer.McpGenerators[gvk.EnvoyFilter.String()] = &mcp.EnvoyFilterGenerator{Server: s.xdsServer}
	s.xdsServer.McpGenerators[gvk.Gateway.String()] = &mcp.GatewayGenerator{Server: s.xdsServer}
	s.xdsServer.McpGenerators[gvk.VirtualService.String()] = &mcp.VirtualServiceGenerator{Server: s.xdsServer}
	s.xdsServer.McpGenerators[gvk.ServiceEntry.String()] = &mcp.ServiceEntryGenerator{Server: s.xdsServer}
	s.xdsServer.ProxyNeedsPush = func(proxy *model.Proxy, req *model.PushRequest) bool {
		return true
	}
	s.server.RunComponent(func(stop <-chan struct{}) error {
		log.Infof("Starting ADS server")
		s.xdsServer.Start(stop)
		return nil
	})
	return s.initGrpcServer()
}

func (s *Server) initGrpcServer() error {
	interceptors := []grpc.UnaryServerInterceptor{
		// setup server prometheus monitoring (as final interceptor in chain)
		prometheus.UnaryServerInterceptor,
	}
	grpcOptions := istiogrpc.ServerOptions(s.GrpcKeepAliveOptions, interceptors...)
	s.grpcServer = grpc.NewServer(grpcOptions...)
	s.xdsServer.Register(s.grpcServer)
	reflection.Register(s.grpcServer)
	return nil
}

func (s *Server) initAuthenticators() error {
	authenticators := []security.Authenticator{
		&authenticate.ClientCertAuthenticator{},
	}
	authenticators = append(authenticators,
		kubeauth.NewKubeJWTAuthenticator(s.environment.Watcher, s.kubeClient, s.RegistryOptions.KubeOptions.ClusterID, nil, features.JwtPolicy))
	if features.XDSAuth {
		s.xdsServer.Authenticators = authenticators
	}
	return nil
}

func (s *Server) initAutomaticHttps() error {
	certOption := &cert.Option{
		Namespace:     PodNamespace,
		ServerAddress: s.CertHttpAddress,
		Email:         s.AutomaticHttpsEmail,
	}
	certServer, err := cert.NewServer(s.kubeClient.Kube(), certOption)
	if err != nil {
		return err
	}
	s.certServer = certServer
	log.Infof("init cert default config")
	s.certServer.InitDefaultConfig()
	if !s.EnableAutomaticHttps {
		log.Info("automatic https is disabled")
		return nil
	}
	return s.certServer.InitServer()
}

func (s *Server) initKubeClient() error {
	if s.kubeClient != nil {
		// Already initialized by startup arguments
		return nil
	}
	kubeRestConfig, err := istiokube.DefaultRestConfig(s.RegistryOptions.KubeConfig, "", func(config *rest.Config) {
		config.QPS = s.RegistryOptions.KubeOptions.KubernetesAPIQPS
		config.Burst = s.RegistryOptions.KubeOptions.KubernetesAPIBurst
	})
	if err != nil {
		return fmt.Errorf("failed creating kube config: %v", err)
	}
	s.kubeClient, err = higresskube.NewClient(istiokube.NewClientConfigForRestConfig(kubeRestConfig))
	if err != nil {
		return fmt.Errorf("failed creating kube client: %v", err)
	}
	return nil
}

func (s *Server) initHttpServer() error {
	s.httpServer = &http.Server{
		Addr:        s.HttpAddress,
		Handler:     s.httpMux,
		IdleTimeout: 90 * time.Second, // matches http.DefaultTransport keep-alive timeout
		ReadTimeout: 30 * time.Second,
	}
	s.xdsServer.AddDebugHandlers(s.httpMux, nil, true, nil)
	s.httpMux.HandleFunc("/ready", s.readyHandler)
	s.httpMux.HandleFunc("/registry/watcherStatus", s.registryWatcherStatusHandler)
	return nil
}

// readyHandler checks whether the http server is ready
func (s *Server) readyHandler(w http.ResponseWriter, _ *http.Request) {
	for name, fn := range s.readinessProbes {
		if ready, err := fn(); !ready {
			log.Warnf("%s is not ready: %v", name, err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) registryWatcherStatusHandler(w http.ResponseWriter, _ *http.Request) {
	ingressTranslation, ok := s.environment.IngressStore.(*translation.IngressTranslation)
	if !ok {
		http.Error(w, "IngressStore not found", http.StatusNotFound)
		return
	}

	ingressConfig := ingressTranslation.GetIngressConfig()
	if ingressConfig == nil {
		http.Error(w, "IngressConfig not found", http.StatusNotFound)
		return
	}

	registryReconciler := ingressConfig.RegistryReconciler
	if registryReconciler == nil {
		http.Error(w, "RegistryReconciler not found", http.StatusNotFound)
		return
	}

	watcherStatusList := registryReconciler.GetRegistryWatcherStatusList()
	writeJSON(w, watcherStatusList)
}

func writeJSON(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	b, err := config.ToJSON(obj)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, err = w.Write(b)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// cachesSynced checks whether caches have been synced.
func (s *Server) cachesSynced() bool {
	return s.configController.HasSynced()
}

func (s *Server) waitForCacheSync(stop <-chan struct{}) bool {
	start := time.Now()
	log.Info("Waiting for caches to be synced")
	if !cache.WaitForCacheSync(stop, s.cachesSynced) {
		log.Errorf("Failed waiting for cache sync")
		return false
	}
	log.Infof("All controller caches have been synced up in %v", time.Since(start))

	// At this point, we know that all update events of the initial state-of-the-world have been
	// received. We wait to ensure we have committed at least this many updates. This avoids a race
	// condition where we are marked ready prior to updating the push context, leading to incomplete
	// pushes.
	expected := s.xdsServer.InboundUpdates.Load()
	if !cache.WaitForCacheSync(stop, func() bool { return s.pushContextReady(expected) }) {
		log.Errorf("Failed waiting for push context initialization")
		return false
	}

	return true
}

// pushContextReady indicates whether pushcontext has processed all inbound config updates.
func (s *Server) pushContextReady(expected int64) bool {
	committed := s.xdsServer.CommittedUpdates.Load()
	if committed < expected {
		log.Debugf("Waiting for pushcontext to process inbound updates, inbound: %v, committed : %v", expected, committed)
		return false
	}
	return true
}

func buildLedger(ca RegistryOptions) ledger.Ledger {
	var result ledger.Ledger
	if ca.DistributionTrackingEnabled {
		result = ledger.Make(ca.DistributionCacheRetention)
	} else {
		result = &model.DisabledLedger{}
	}
	return result
}
