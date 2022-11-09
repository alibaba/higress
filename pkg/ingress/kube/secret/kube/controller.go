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

package kube

import (
	"time"

	"istio.io/istio/pilot/pkg/model"
	kubeclient "istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/controllers"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/alibaba/higress/pkg/ingress/kube/common"
	"github.com/alibaba/higress/pkg/ingress/kube/secret"
	"github.com/alibaba/higress/pkg/ingress/kube/util"
	. "github.com/alibaba/higress/pkg/ingress/log"
)

var _ secret.Controller = &controller{}

type controller struct {
	queue     workqueue.RateLimitingInterface
	informer  cache.SharedIndexInformer
	lister    listersv1.SecretLister
	handler   func(util.ClusterNamespacedName)
	clusterId string
}

// NewController is copied from NewCredentialsController.
func NewController(client kubeclient.Client, options common.Options) secret.Controller {
	q := workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter())

	informer := client.KubeInformer().InformerFor(&v1.Secret{}, func(k kubernetes.Interface, resync time.Duration) cache.SharedIndexInformer {
		return informersv1.NewFilteredSecretInformer(
			k, metav1.NamespaceAll, resync, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
			func(options *metav1.ListOptions) {
				options.FieldSelector = fields.AndSelectors(
					fields.OneTermNotEqualSelector("type", "helm.sh/release.v1"),
					fields.OneTermNotEqualSelector("type", string(v1.SecretTypeServiceAccountToken)),
				).String()
			},
		)
	})

	handler := controllers.LatestVersionHandlerFuncs(controllers.EnqueueForSelf(q))
	informer.AddEventHandler(handler)

	return &controller{
		queue:     q,
		informer:  informer,
		lister:    listersv1.NewSecretLister(informer.GetIndexer()),
		clusterId: options.ClusterId,
	}
}

func (c *controller) Lister() listersv1.SecretLister {
	return c.lister
}

func (c *controller) Informer() cache.SharedIndexInformer {
	return c.informer
}

func (c *controller) AddEventHandler(f func(util.ClusterNamespacedName)) {
	c.handler = f
}

func (c *controller) Run(stop <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	if !cache.WaitForCacheSync(stop, c.HasSynced) {
		IngressLog.Errorf("Failed to sync secret controller cache")
		return
	}
	go wait.Until(c.worker, time.Second, stop)
	<-stop
}

func (c *controller) worker() {
	for c.processNextWorkItem() {
	}
}

func (c *controller) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	ingressNamespacedName := key.(types.NamespacedName)
	IngressLog.Debugf("secret %s push to queue", ingressNamespacedName)
	if err := c.onEvent(ingressNamespacedName); err != nil {
		IngressLog.Errorf("error processing secret item (%v) (retrying): %v", key, err)
		c.queue.AddRateLimited(key)
	} else {
		c.queue.Forget(key)
	}
	return true
}

func (c *controller) onEvent(namespacedName types.NamespacedName) error {
	_, err := c.lister.Secrets(namespacedName.Namespace).Get(namespacedName.Name)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}

	// We only care about add or update event.
	c.handler(util.ClusterNamespacedName{
		NamespacedName: model.NamespacedName{
			Namespace: namespacedName.Namespace,
			Name:      namespacedName.Name,
		},
		ClusterId: c.clusterId,
	})
	return nil
}

func (c *controller) HasSynced() bool {
	return c.informer.HasSynced()
}
