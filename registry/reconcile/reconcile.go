package reconcile

import (
	"errors"
	"path"
	"reflect"
	"sync"

	"istio.io/pkg/log"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
	v1 "github.com/alibaba/higress/client/pkg/apis/networking/v1"
	. "github.com/alibaba/higress/registry"
	"github.com/alibaba/higress/registry/memory"
	"github.com/alibaba/higress/registry/nacos"
	nacosv2 "github.com/alibaba/higress/registry/nacos/v2"
	"github.com/alibaba/higress/registry/zookeeper"
)

type Reconciler struct {
	memory.Cache
	registries    map[string]*apiv1.RegistryConfig
	watchers      map[string]Watcher
	serviceUpdate func()
}

func NewReconciler(serviceUpdate func()) *Reconciler {
	return &Reconciler{
		Cache:         memory.NewCache(),
		registries:    make(map[string]*apiv1.RegistryConfig),
		watchers:      make(map[string]Watcher),
		serviceUpdate: serviceUpdate,
	}
}

func (r *Reconciler) Reconcile(mcpbridge *v1.McpBridge) {
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
	for k, v := range toBeUpdated {
		go r.watchers[k].Stop()
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
	for k := range toBeDeleted {
		go r.watchers[k].Stop()
		delete(r.registries, k)
		delete(r.watchers, k)
	}
	if errHappened {
		log.Error("ReconcileRegistries failed, Init Watchers failed")
		return
	}
	wg.Wait()
	log.Infof("Registries is reconciled")
}

func (r *Reconciler) generateWatcherFromRegistryConfig(registry *apiv1.RegistryConfig, wg *sync.WaitGroup) (Watcher, error) {
	var watcher Watcher
	var err error

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
		)
	case string(Nacos2):
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
				log.Infof("Registry Watcher is  ready, type:%s, name:%s", registry.Type, registry.Name)
			}
		})
	})
	watcher.AppendServiceUpdateHandler(r.serviceUpdate)

	return watcher, nil
}
