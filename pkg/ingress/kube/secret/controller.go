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
	"github.com/alibaba/higress/v2/pkg/ingress/kube/common"
	"github.com/alibaba/higress/v2/pkg/ingress/kube/controller"
	"istio.io/istio/pkg/config/schema/gvr"
	schemakubeclient "istio.io/istio/pkg/config/schema/kubeclient"
	kubeclient "istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/controllers"
	ktypes "istio.io/istio/pkg/kube/kubetypes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	listersv1 "k8s.io/client-go/listers/core/v1"
)

type SecretController controller.Controller[listersv1.SecretLister]

func NewController(client kubeclient.Client, options common.Options) SecretController {
	opts := ktypes.InformerOptions{
		Namespace: options.WatchNamespace,
		Cluster:   options.ClusterId,
		FieldSelector: fields.AndSelectors(
			fields.OneTermNotEqualSelector("type", "helm.sh/release.v1"),
			fields.OneTermNotEqualSelector("type", string(v1.SecretTypeServiceAccountToken)),
		).String(),
	}
	informer := schemakubeclient.GetInformerFilteredFromGVR(client, opts, gvr.Secret)
	return controller.NewCommonController("secret", listersv1.NewSecretLister(informer.Informer.GetIndexer()), informer.Informer, GetSecret, options.ClusterId)
}

func GetSecret(lister listersv1.SecretLister, namespacedName types.NamespacedName) (controllers.Object, error) {
	return lister.Secrets(namespacedName.Namespace).Get(namespacedName.Name)
}
