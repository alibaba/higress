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

package mcpbridge

import (
	"time"

	"istio.io/istio/pkg/kube/controllers"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "github.com/alibaba/higress/client/pkg/apis/networking/v1"
	"github.com/alibaba/higress/client/pkg/clientset/versioned"
	informersv1 "github.com/alibaba/higress/client/pkg/informers/externalversions/networking/v1"
	listersv1 "github.com/alibaba/higress/client/pkg/listers/networking/v1"
	"github.com/alibaba/higress/pkg/ingress/kube/common"
	"github.com/alibaba/higress/pkg/ingress/kube/controller"
	kubeclient "github.com/alibaba/higress/pkg/kube"
)

type McpBridgeController controller.Controller[listersv1.McpBridgeLister]

func NewController(client kubeclient.Client, options common.Options) McpBridgeController {
	var informer cache.SharedIndexInformer
	if options.WatchNamespace == "" {
		informer = client.HigressInformer().Networking().V1().McpBridges().Informer()
	} else {
		informer = client.HigressInformer().InformerFor(&v1.McpBridge{}, func(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
			return informersv1.NewMcpBridgeInformer(client, options.WatchNamespace, resyncPeriod, nil)
		})
	}
	return controller.NewCommonController("mcpbridge", listersv1.NewMcpBridgeLister(informer.GetIndexer()), informer, GetMcpBridge, options.ClusterId)
}

func GetMcpBridge(lister listersv1.McpBridgeLister, namespacedName types.NamespacedName) (controllers.Object, error) {
	return lister.McpBridges(namespacedName.Namespace).Get(namespacedName.Name)
}
