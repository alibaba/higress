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

package secret

import (
	"time"

	kubeclient "istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/controllers"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/alibaba/higress/pkg/ingress/kube/controller"
)

type SecretController controller.Controller[listersv1.SecretLister]

func NewController(client kubeclient.Client, clusterId string) SecretController {
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
	return controller.NewCommonController("secret", listersv1.NewSecretLister(informer.GetIndexer()), informer, GetSecret, clusterId)
}

func GetSecret(lister listersv1.SecretLister, namespacedName types.NamespacedName) (controllers.Object, error) {
	return lister.Secrets(namespacedName.Namespace).Get(namespacedName.Name)
}
