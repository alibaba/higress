package gateway

import (
	"strings"
	"sync/atomic"

	"istio.io/istio/pilot/pkg/bootstrap"
	"istio.io/istio/pilot/pkg/config/kube/crdclient"
	istiogateway "istio.io/istio/pilot/pkg/config/kube/gateway"
	"istio.io/istio/pilot/pkg/model"
	kubecontroller "istio.io/istio/pilot/pkg/serviceregistry/kube/controller"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/config/schema/collection"
	"istio.io/istio/pkg/config/schema/collections"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/kube"
	"k8s.io/client-go/tools/cache"

	"github.com/alibaba/higress/pkg/ingress/kube/common"
	"github.com/alibaba/higress/pkg/ingress/kube/util"
	. "github.com/alibaba/higress/pkg/ingress/log"
)

type gatewayController struct {
	virtualServiceHandlers  []model.EventHandler
	gatewayHandlers         []model.EventHandler
	destinationRuleHandlers []model.EventHandler
	envoyFilterHandlers     []model.EventHandler

	cache           model.ConfigStoreCache
	istioController *istiogateway.Controller

	resourceUpToDate atomic.Bool
}

func NewController(client kube.Client) (common.GatewayController, error) {
	domainSuffix := util.GetDomainSuffix()
	cache, err := crdclient.New(client, bootstrap.Revision, domainSuffix)
	if err != nil {
		return nil, err
	}
	istioController := istiogateway.NewController(client, cache, kubecontroller.Options{DomainSuffix: domainSuffix})
	return &gatewayController{cache: cache, istioController: istioController}, nil
}

func (g *gatewayController) Schemas() collection.Schemas {
	return g.istioController.Schemas()
}

func (g *gatewayController) Get(typ config.GroupVersionKind, name, namespace string) *config.Config {
	return g.istioController.Get(typ, name, namespace)
}

func (g *gatewayController) List(typ config.GroupVersionKind, namespace string) ([]config.Config, error) {
	if g.resourceUpToDate.CompareAndSwap(false, true) {
		err := g.istioController.Recompute(model.NewGatewayContext(model.NewPushContext()))
		if err != nil {
			IngressLog.Errorf("failed to recompute Gateway API resources: %v", err)
		}
	}
	configs, err := g.istioController.List(typ, namespace)
	if err != nil && strings.HasPrefix(err.Error(), "unsupported") {
		// Normalize unsupported error
		err = common.ErrUnsupportedOp
	}
	return configs, err
}

func (g *gatewayController) Create(config config.Config) (revision string, err error) {
	return g.istioController.Create(config)
}

func (g *gatewayController) Update(config config.Config) (newRevision string, err error) {
	return g.istioController.Update(config)
}

func (g *gatewayController) UpdateStatus(config config.Config) (newRevision string, err error) {
	return g.istioController.UpdateStatus(config)
}

func (g *gatewayController) Patch(orig config.Config, patchFn config.PatchFunc) (string, error) {
	return g.istioController.Patch(orig, patchFn)
}

func (g *gatewayController) Delete(typ config.GroupVersionKind, name, namespace string, resourceVersion *string) error {
	return g.istioController.Delete(typ, name, namespace, resourceVersion)
}

func (g *gatewayController) RegisterEventHandler(kind config.GroupVersionKind, f model.EventHandler) {
	switch kind {
	case gvk.VirtualService:
		g.virtualServiceHandlers = append(g.virtualServiceHandlers, f)
	case gvk.Gateway:
		g.gatewayHandlers = append(g.gatewayHandlers, f)
	case gvk.DestinationRule:
		g.destinationRuleHandlers = append(g.destinationRuleHandlers, f)
	case gvk.EnvoyFilter:
		g.envoyFilterHandlers = append(g.envoyFilterHandlers, f)
	}
}

func (g *gatewayController) Run(stop <-chan struct{}) {
	for _, schema := range collections.PilotGatewayAPI.All() {
		resource := schema.Resource()
		if resource.Group() == gvk.GatewayClass.Group {
			g.cache.RegisterEventHandler(resource.GroupVersionKind(), g.onEvent)
		}
	}
	go g.cache.Run(stop)
	go g.istioController.Run(stop)
}

func (g *gatewayController) SetWatchErrorHandler(f func(r *cache.Reflector, err error)) error {
	if err := g.cache.SetWatchErrorHandler(f); err != nil {
		return err
	}
	if err := g.istioController.SetWatchErrorHandler(f); err != nil {
		return err
	}
	return nil
}

func (g *gatewayController) HasSynced() bool {
	ret := g.istioController.HasSynced()
	if ret {
		err := g.istioController.Recompute(model.NewGatewayContext(model.NewPushContext()))
		if err != nil {
			IngressLog.Errorf("failed to recompute Gateway API resources: %v", err)
		}
	}
	return ret
}

func (g *gatewayController) onEvent(prev config.Config, curr config.Config, event model.Event) {
	g.resourceUpToDate.Store(false)

	name := "gateway-api"
	namespace := curr.Namespace

	vsMetadata := config.Meta{
		Name:             name + "-" + "virtualservice",
		Namespace:        namespace,
		GroupVersionKind: gvk.VirtualService,
		// Set this label so that we do not compare configs and just push.
		Labels: map[string]string{constants.AlwaysPushLabel: "true"},
	}
	gatewayMetadata := config.Meta{
		Name:             name + "-" + "gateway",
		Namespace:        namespace,
		GroupVersionKind: gvk.Gateway,
		// Set this label so that we do not compare configs and just push.
		Labels: map[string]string{constants.AlwaysPushLabel: "true"},
	}

	for _, f := range g.virtualServiceHandlers {
		f(config.Config{Meta: vsMetadata}, config.Config{Meta: vsMetadata}, event)
	}

	for _, f := range g.gatewayHandlers {
		f(config.Config{Meta: gatewayMetadata}, config.Config{Meta: gatewayMetadata}, event)
	}
}
