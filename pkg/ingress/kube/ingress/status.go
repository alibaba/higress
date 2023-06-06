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

package ingress

import (
	"context"
	common2 "github.com/alibaba/higress/pkg/ingress/kube/common"
	. "github.com/alibaba/higress/pkg/ingress/log"
	"reflect"
	"sort"
	"time"

	kubelib "istio.io/istio/pkg/kube"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	ingresslister "k8s.io/client-go/listers/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// statusSyncer keeps the status IP in each Ingress resource updated
type statusSyncer struct {
	client     kubernetes.Interface
	controller *controller

	watchedNamespace string

	ingressLister ingresslister.IngressLister
	// search service in the mse vpc
	serviceLister listerv1.ServiceLister
}

// newStatusSyncer creates a new instance
func newStatusSyncer(localKubeClient, client kubelib.Client, controller *controller, namespace string) *statusSyncer {
	return &statusSyncer{
		client:           client,
		controller:       controller,
		watchedNamespace: namespace,
		ingressLister:    client.KubeInformer().Networking().V1beta1().Ingresses().Lister(),
		// search service in the mse vpc
		serviceLister: localKubeClient.KubeInformer().Core().V1().Services().Lister(),
	}
}

func (s *statusSyncer) run(stopCh <-chan struct{}) {
	cache.WaitForCacheSync(stopCh, s.controller.HasSynced)

	ticker := time.NewTicker(common2.DefaultStatusUpdateInterval)
	for {
		select {
		case <-stopCh:
			ticker.Stop()
			return
		case <-ticker.C:
			if err := s.runUpdateStatus(); err != nil {
				IngressLog.Errorf("update status task fail, err %v", err)
			}
		}
	}
}

func (s *statusSyncer) runUpdateStatus() error {
	svcList, err := s.serviceLister.Services(s.watchedNamespace).List(common2.SvcLabelSelector)
	if err != nil {
		return err
	}

	IngressLog.Debugf("found number %d of svc", len(svcList))

	lbStatusList := common2.GetLbStatusList(svcList)
	if len(lbStatusList) == 0 {
		return nil
	}

	return s.updateStatus(lbStatusList)
}

// updateStatus updates ingress status with the list of IP
func (s *statusSyncer) updateStatus(status []coreV1.LoadBalancerIngress) error {
	ingressList, err := s.ingressLister.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, ingress := range ingressList {
		shouldTarget, err := s.controller.shouldProcessIngress(ingress)
		if err != nil {
			IngressLog.Warnf("error determining whether should target ingress %s/%s within cluster %s for status update: %v",
				ingress.Namespace, ingress.Name, s.controller.options.ClusterId, err)
			return err
		}

		if !shouldTarget {
			continue
		}

		curIPs := ingress.Status.LoadBalancer.Ingress
		sort.SliceStable(curIPs, common2.SortLbIngressList(curIPs))

		if reflect.DeepEqual(status, curIPs) {
			IngressLog.Debugf("skipping update of Ingress %v/%v within cluster %s (no change)",
				ingress.Namespace, ingress.Name, s.controller.options.ClusterId)
			continue
		}

		ingress.Status.LoadBalancer.Ingress = status
		IngressLog.Infof("Update Ingress %v/%v within cluster %s status",
			ingress.Namespace, ingress.Name, s.controller.options.ClusterId)
		_, err = s.client.NetworkingV1beta1().Ingresses(ingress.Namespace).UpdateStatus(context.TODO(), ingress, metaV1.UpdateOptions{})
		if err != nil {
			IngressLog.Warnf("error updating ingress %s/%s within cluster %s status: %v",
				ingress.Namespace, ingress.Name, s.controller.options.ClusterId, err)
		}
	}

	return nil
}
