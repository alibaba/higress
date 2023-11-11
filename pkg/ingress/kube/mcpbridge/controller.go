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
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/kube/controllers"
	ktypes "istio.io/istio/pkg/kube/kubetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	listersv1 "github.com/alibaba/higress/client/pkg/listers/networking/v1"
	"github.com/alibaba/higress/pkg/ingress/kube/controller"
	kubeclient "github.com/alibaba/higress/pkg/kube"
)

var mcpbridgesResource = schema.GroupVersionResource{Group: "networking.higress.io", Version: "v1", Resource: "mcpbridges"}

type McpBridgeController controller.Controller[listersv1.McpBridgeLister]

func NewController(client kubeclient.Client, clusterId cluster.ID) McpBridgeController {
	opts := ktypes.InformerOptions{
		Namespace: metav1.NamespaceAll,
		Cluster:   clusterId,
	}
	informer := client.Informers().InformerFor(mcpbridgesResource, opts, func() cache.SharedIndexInformer {
		return client.HigressInformer().Networking().V1().McpBridges().Informer()
	})
	return controller.NewCommonController("mcpbridge", client.HigressInformer().Networking().V1().McpBridges().Lister(),
		informer, GetMcpBridge, clusterId)
}

func GetMcpBridge(lister listersv1.McpBridgeLister, namespacedName types.NamespacedName) (controllers.Object, error) {
	return lister.McpBridges(namespacedName.Namespace).Get(namespacedName.Name)
}
