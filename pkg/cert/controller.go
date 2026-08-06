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

package cert

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	v1informer "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	workNum       = 1
	maxRetry      = 2
	configMapName = "higress-https"
)

type Controller struct {
	namespace         string
	ConfigMapInformer v1informer.ConfigMapInformer
	client            kubernetes.Interface
	queue             workqueue.RateLimitingInterface
	configMgr         *ConfigMgr
	server            *Server
	certMgr           *CertMgr
	factory           informers.SharedInformerFactory
}

func (c *Controller) addConfigmap(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	namespace, name, _ := cache.SplitMetaNamespaceKey(key)
	if namespace != c.namespace || name != configMapName {
		return
	}
	c.enqueue(name)

}
func (c *Controller) updateConfigmap(oldObj interface{}, newObj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(oldObj)
	if err != nil {
		return
	}
	namespace, name, _ := cache.SplitMetaNamespaceKey(key)
	if namespace != c.namespace || name != configMapName {
		return
	}
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	c.enqueue(name)
}

func (c *Controller) enqueue(name string) {
	c.queue.Add(name)
}

func (c *Controller) cachesSynced() bool {
	return c.ConfigMapInformer.Informer().HasSynced()
}

func (c *Controller) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()
	CertLog.Info("Waiting for informer caches to sync")
	c.factory.Start(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, c.cachesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	CertLog.Info("Starting controller")
	// Launch one workers to process configmap resources
	for i := 0; i < workNum; i++ {
		go wait.Until(c.worker, time.Minute, stopCh)
	}
	CertLog.Info("Started workers")
	<-stopCh
	CertLog.Info("Shutting down workers")

	return nil
}

func (c *Controller) worker() {
	for c.processNextItem() {

	}
}

func (c *Controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(item)
	key := item.(string)
	CertLog.Infof("controller process item:%s", key)
	err := c.syncConfigmap(key)
	if err != nil {
		c.handleError(key, err)
	}
	return true
}

func (c *Controller) syncConfigmap(key string) error {
	configmap, err := c.ConfigMapInformer.Lister().ConfigMaps(c.namespace).Get(key)
	if err != nil {
		return err
	}
	newConfig, err := c.configMgr.ParseConfigFromConfigmap(configmap)
	if err != nil {
		return err
	}
	oldConfig := c.configMgr.GetConfig()
	// reconcile old config and new config
	return c.certMgr.Reconcile(context.Background(), oldConfig, newConfig)
}

func (c *Controller) handleError(key string, err error) {
	runtime.HandleError(err)
	CertLog.Errorf("%+v", err)
	c.queue.Forget(key)
}

func NewController(client kubernetes.Interface, namespace string, certMgr *CertMgr, configMgr *ConfigMgr) (*Controller, error) {
	kubeInformerFactory := informers.NewSharedInformerFactoryWithOptions(client, 0, informers.WithNamespace(namespace))
	configmapInformer := kubeInformerFactory.Core().V1().ConfigMaps()
	c := &Controller{
		certMgr:           certMgr,
		configMgr:         configMgr,
		client:            client,
		namespace:         namespace,
		factory:           kubeInformerFactory,
		ConfigMapInformer: configmapInformer,
		queue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingressManage"),
	}

	CertLog.Info("Setting up configmap informer event handlers")
	configmapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addConfigmap,
		UpdateFunc: c.updateConfigmap,
	})

	return c, nil
}
