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

package wasmplugin

import (
	"time"

	"istio.io/istio/pkg/kube/controllers"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "github.com/alibaba/higress/v2/client/pkg/apis/extensions/v1alpha1"
	"github.com/alibaba/higress/v2/client/pkg/clientset/versioned"
	informersv1 "github.com/alibaba/higress/v2/client/pkg/informers/externalversions/extensions/v1alpha1"
	listersv1 "github.com/alibaba/higress/v2/client/pkg/listers/extensions/v1alpha1"
	"github.com/alibaba/higress/v2/pkg/ingress/kube/common"
	"github.com/alibaba/higress/v2/pkg/ingress/kube/controller"
	kubeclient "github.com/alibaba/higress/v2/pkg/kube"
)

type WasmPluginController controller.Controller[listersv1.WasmPluginLister]
type WasmPluginMatchRuleController controller.Controller[listersv1.WasmPluginMatchRuleLister]

func NewController(client kubeclient.Client, options common.Options) WasmPluginController {
	var informer cache.SharedIndexInformer
	if options.WatchNamespace == "" {
		informer = client.HigressInformer().Extensions().V1alpha1().WasmPlugins().Informer()
	} else {
		informer = client.HigressInformer().InformerFor(&v1.WasmPlugin{}, func(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
			return informersv1.NewWasmPluginInformer(client, options.WatchNamespace, resyncPeriod, nil)
		})
	}
	return controller.NewCommonController("wasmplugin", listersv1.NewWasmPluginLister(informer.GetIndexer()), informer, GetWasmPlugin, options.ClusterId)
}

func GetWasmPlugin(lister listersv1.WasmPluginLister, namespacedName types.NamespacedName) (controllers.Object, error) {
	return lister.WasmPlugins(namespacedName.Namespace).Get(namespacedName.Name)
}

func NewMatchRuleController(client kubeclient.Client, options common.Options) WasmPluginMatchRuleController {
	var informer cache.SharedIndexInformer
	if options.WatchNamespace == "" {
		informer = client.HigressInformer().Extensions().V1alpha1().WasmPluginMatchRules().Informer()
	} else {
		informer = client.HigressInformer().InformerFor(&v1.WasmPluginMatchRule{}, func(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
			return informersv1.NewWasmPluginMatchRuleInformer(client, options.WatchNamespace, resyncPeriod, nil)
		})
	}
	return controller.NewCommonController("wasmpluginmatchrule", listersv1.NewWasmPluginMatchRuleLister(informer.GetIndexer()), informer, GetWasmPluginMatchRule, options.ClusterId)
}

func GetWasmPluginMatchRule(lister listersv1.WasmPluginMatchRuleLister, namespacedName types.NamespacedName) (controllers.Object, error) {
	return lister.WasmPluginMatchRules(namespacedName.Namespace).Get(namespacedName.Name)
}
