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
	informerV1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const (
	workNum       = 1
	maxRetry      = 2
	configMapName = "higress-https"
)

type controller struct {
	namespace       string
	configmapLister v1.ConfigMapLister
	configmapSynced cache.InformerSynced
	client          kubernetes.Interface
	queue           workqueue.RateLimitingInterface
	configMgr       *ConfigMgr
	server          *Server
}

func (c *controller) addConfigmap(obj interface{}) {
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
func (c *controller) updateConfigmap(oldObj interface{}, newObj interface{}) {
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

func (c *controller) enqueue(name string) {
	c.queue.Add(name)
}

func (c *controller) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()
	klog.Info("Starting controller")
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.configmapSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	// Launch one workers to process configmap resources
	for i := 0; i < workNum; i++ {
		go wait.Until(c.worker, time.Minute, stopCh)
	}
	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

func (c *controller) worker() {
	for c.processNextItem() {

	}
}

func (c *controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(item)
	key := item.(string)
	klog.Infof("controller process item:%s", key)
	err := c.syncConfigmap(key)
	if err != nil {
		c.handleError(key, err)
	}
	return true
}

func (c *controller) syncConfigmap(key string) error {
	configmap, err := c.configmapLister.ConfigMaps(c.namespace).Get(key)
	if err != nil {
		return err
	}
	newConfig, err := c.configMgr.ParseConfigFromConfigmap(configmap)
	if err != nil {
		return err
	}
	oldConfig := c.configMgr.GetConfig()
	// reconcile old config and new config
	return c.server.Reconcile(context.Background(), oldConfig, newConfig)
}

func (c *controller) handleError(key string, err error) {
	//if c.queue.NumRequeues(key) <= maxRetry {
	//	c.queue.AddRateLimited(key)
	//	return
	//}
	runtime.HandleError(err)
	klog.Errorf("%+v", err)
	c.queue.Forget(key)
}

func NewController(server *Server, client kubernetes.Interface, namespace string, informer informerV1.ConfigMapInformer, configMgr *ConfigMgr) *controller {
	c := &controller{
		server:          server,
		configMgr:       configMgr,
		client:          client,
		namespace:       namespace,
		configmapLister: informer.Lister(),
		configmapSynced: informer.Informer().HasSynced,
		queue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingressManage"),
	}

	klog.Info("Setting up configmap informer event handlers")
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addConfigmap,
		UpdateFunc: c.updateConfigmap,
	})

	return c
}
