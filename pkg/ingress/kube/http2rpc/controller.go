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

package http2rpc

import (
	"time"

	"istio.io/istio/pkg/kube/controllers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "github.com/alibaba/higress/client/pkg/apis/networking/v1"
	"github.com/alibaba/higress/client/pkg/clientset/versioned"
	informersv1 "github.com/alibaba/higress/client/pkg/informers/externalversions/networking/v1"
	listersv1 "github.com/alibaba/higress/client/pkg/listers/networking/v1"
	"github.com/alibaba/higress/pkg/ingress/kube/controller"
	kubeclient "github.com/alibaba/higress/pkg/kube"
)

type Http2RpcController controller.Controller[listersv1.Http2RpcLister]

func NewController(client kubeclient.Client, clusterId string) Http2RpcController {
	informer := client.HigressInformer().InformerFor(&v1.Http2Rpc{}, func(k versioned.Interface, resync time.Duration) cache.SharedIndexInformer {
		return informersv1.NewHttp2RpcInformer(k, metav1.NamespaceAll, resync,
			cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
	return controller.NewCommonController("http2rpc", listersv1.NewHttp2RpcLister(informer.GetIndexer()),
		informer, GetHttp2Rpc, clusterId)
}

func GetHttp2Rpc(lister listersv1.Http2RpcLister, namespacedName types.NamespacedName) (controllers.Object, error) {
	return lister.Http2Rpcs(namespacedName.Namespace).Get(namespacedName.Name)
}
