package higressconfig

import (
	"github.com/alibaba/higress/pkg/ingress/kube/common"
	"sync"

	ingressconfig "github.com/alibaba/higress/pkg/ingress/config"
	. "github.com/alibaba/higress/pkg/ingress/log"
	"github.com/alibaba/higress/pkg/kube"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/collection"
	"istio.io/istio/pkg/config/schema/gvk"
	"k8s.io/client-go/tools/cache"
)

var (
	_ model.ConfigStoreCache = &HigressConfig{}
	_ model.IngressStore     = &HigressConfig{}
)

type HigressConfig struct {
	ingressconfig      *ingressconfig.IngressConfig
	kingressconfig     *ingressconfig.KIngressConfig
	mutex              sync.RWMutex
	higressRouteCache  model.IngressRouteCollection
	higressDomainCache model.IngressDomainCollection
}

func NewHigressConfig(localKubeClient kube.Client, XDSUpdater model.XDSUpdater, namespace, clusterId string) *HigressConfig {
	if clusterId == "Kubernetes" {
		clusterId = ""
	}
	config := &HigressConfig{
		ingressconfig:  ingressconfig.NewIngressConfig(localKubeClient, XDSUpdater, namespace, clusterId),
		kingressconfig: ingressconfig.NewKIngressConfig(localKubeClient, XDSUpdater, namespace, clusterId),
	}
	return config
}

func (m *HigressConfig) AddLocalCluster(options common.Options) (common.IngressController, common.KIngressController) {
	if m.kingressconfig == nil {
		return m.ingressconfig.AddLocalCluster(options), nil
	}
	return m.ingressconfig.AddLocalCluster(options), m.kingressconfig.AddLocalCluster(options)
}

func (m *HigressConfig) InitializeCluster(ingressController common.IngressController, kingressController common.KIngressController, stop <-chan struct{}) error {
	if err := m.ingressconfig.InitializeCluster(ingressController, stop); err != nil {
		return err
	}
	if kingressController == nil {
		return nil
	}
	if err := m.kingressconfig.InitializeCluster(kingressController, stop); err != nil {
		return err
	}
	return nil
}

func (m *HigressConfig) RegisterEventHandler(kind config.GroupVersionKind, f model.EventHandler) {
	m.ingressconfig.RegisterEventHandler(kind, f)
	if m.kingressconfig != nil {
		m.kingressconfig.RegisterEventHandler(kind, f)
	}
}

func (m *HigressConfig) HasSynced() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if !m.ingressconfig.HasSynced() {
		return false
	}
	if m.kingressconfig != nil {
		if !m.kingressconfig.HasSynced() {
			return false
		}
	}

	return true
}

func (m *HigressConfig) Run(stop <-chan struct{}) {
	go m.ingressconfig.Run(stop)
	if m.kingressconfig != nil {
		go m.kingressconfig.Run(stop)
	}
}

func (m *HigressConfig) SetWatchErrorHandler(f func(r *cache.Reflector, err error)) error {
	m.ingressconfig.SetWatchErrorHandler(f)
	if m.kingressconfig != nil {
		m.kingressconfig.SetWatchErrorHandler(f)
	}
	return nil
}

func (m *HigressConfig) GetIngressRoutes() model.IngressRouteCollection {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	ingressRouteCache := m.ingressconfig.GetIngressRoutes()
	m.higressRouteCache = model.IngressRouteCollection{}
	m.higressRouteCache.Invalid = append(m.higressRouteCache.Invalid, ingressRouteCache.Invalid...)
	m.higressRouteCache.Valid = append(m.higressRouteCache.Valid, ingressRouteCache.Valid...)
	if m.kingressconfig != nil {
		kingressRouteCache := m.kingressconfig.GetIngressRoutes()
		m.higressRouteCache.Invalid = append(m.higressRouteCache.Invalid, kingressRouteCache.Invalid...)
		m.higressRouteCache.Valid = append(m.higressRouteCache.Valid, kingressRouteCache.Valid...)
	}

	return m.higressRouteCache

}

func (m *HigressConfig) GetIngressDomains() model.IngressDomainCollection {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	ingressDomainCache := m.ingressconfig.GetIngressDomains()

	m.higressDomainCache = model.IngressDomainCollection{}
	m.higressDomainCache.Invalid = append(m.higressDomainCache.Invalid, ingressDomainCache.Invalid...)
	m.higressDomainCache.Valid = append(m.higressDomainCache.Valid, ingressDomainCache.Valid...)
	if m.kingressconfig != nil {
		kingressDomainCache := m.kingressconfig.GetIngressDomains()
		m.higressDomainCache.Invalid = append(m.higressDomainCache.Invalid, kingressDomainCache.Invalid...)
		m.higressDomainCache.Valid = append(m.higressDomainCache.Valid, kingressDomainCache.Valid...)
	}
	return m.higressDomainCache
}

func (m *HigressConfig) Schemas() collection.Schemas {
	return common.IngressIR
}

func (m *HigressConfig) Get(typ config.GroupVersionKind, name, namespace string) *config.Config {
	return nil
}

func (m *HigressConfig) List(typ config.GroupVersionKind, namespace string) ([]config.Config, error) {
	if typ != gvk.Gateway &&
		typ != gvk.VirtualService &&
		typ != gvk.DestinationRule &&
		typ != gvk.EnvoyFilter &&
		typ != gvk.ServiceEntry &&
		typ != gvk.WasmPlugin {
		return nil, common.ErrUnsupportedOp
	}

	// Currently, only support list all namespaces gateways or virtualservices.
	if namespace != "" {
		IngressLog.Warnf("ingress store only support type %s of all namespace.", typ)
		return nil, common.ErrUnsupportedOp
	}

	ingressConfig, err := m.ingressconfig.List(typ, namespace)
	if err != nil {
		return nil, err
	}
	var higressConfig []config.Config
	higressConfig = append(higressConfig, ingressConfig...)
	if m.kingressconfig != nil {
		kingressConfig, err := m.kingressconfig.List(typ, namespace)
		if err != nil {
			return nil, err
		}
		higressConfig = append(higressConfig, kingressConfig...)
	}
	return higressConfig, nil
}

func (m *HigressConfig) Create(config config.Config) (revision string, err error) {
	return "", common.ErrUnsupportedOp
}

func (m *HigressConfig) Update(config config.Config) (newRevision string, err error) {
	return "", common.ErrUnsupportedOp
}

func (m *HigressConfig) UpdateStatus(config config.Config) (newRevision string, err error) {
	return "", common.ErrUnsupportedOp
}

func (m *HigressConfig) Patch(orig config.Config, patchFn config.PatchFunc) (string, error) {
	return "", common.ErrUnsupportedOp
}

func (m *HigressConfig) Delete(typ config.GroupVersionKind, name, namespace string, resourceVersion *string) error {
	return common.ErrUnsupportedOp
}
