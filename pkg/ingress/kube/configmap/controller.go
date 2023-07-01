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

package configmap

import (
	"github.com/alibaba/higress/pkg/ingress/kube/controller"
	kubeclient "istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/controllers"
	"k8s.io/apimachinery/pkg/types"
	listersv1 "k8s.io/client-go/listers/core/v1"
)

type HigressConfigController controller.Controller[listersv1.ConfigMapNamespaceLister]

func NewController(client kubeclient.Client, clusterId string, namespace string) HigressConfigController {
	informer := client.KubeInformer().Core().V1().ConfigMaps().Informer()
	return controller.NewCommonController("higressConfig", client.KubeInformer().Core().V1().ConfigMaps().Lister().ConfigMaps(namespace),
		informer, GetConfigmap, clusterId)
}

func GetConfigmap(lister listersv1.ConfigMapNamespaceLister, namespacedName types.NamespacedName) (controllers.Object, error) {
	return lister.Get(namespacedName.Name)
}
