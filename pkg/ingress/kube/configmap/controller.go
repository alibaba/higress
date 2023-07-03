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
	"github.com/alibaba/higress/pkg/ingress/kube/util"
	. "github.com/alibaba/higress/pkg/ingress/log"
	"istio.io/istio/pkg/config"
	kubeclient "istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/controllers"
	"k8s.io/apimachinery/pkg/types"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"reflect"
	"sigs.k8s.io/yaml"
	"sync/atomic"
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

type ItemController interface {
	GetName() string
	AddOrUpdateHigressConfig(name util.ClusterNamespacedName, old *HigressConfig, new *HigressConfig) error
	ValidHigressConfig(higressConfig *HigressConfig) error
	ConstructEnvoyFilters() ([]*config.Config, error)
}

type ConfigmapMgr struct {
	Namespace               string
	HigressConfigController HigressConfigController
	HigressConfigLister     listersv1.ConfigMapNamespaceLister
	higressConfig           atomic.Value
	ItemControllers         []ItemController
}

func NewConfigmapMgr(namespace string, higressConfigController HigressConfigController, higressConfigLister listersv1.ConfigMapNamespaceLister) *ConfigmapMgr {

	configmapMgr := &ConfigmapMgr{
		Namespace:               namespace,
		HigressConfigController: higressConfigController,
		HigressConfigLister:     higressConfigLister,
		higressConfig:           atomic.Value{},
	}
	configmapMgr.HigressConfigController.AddEventHandler(configmapMgr.AddOrUpdateHigressConfig)
	configmapMgr.SetHigressConfig(NewDefaultHigressConfig())

	tracingController := NewTracingController(namespace)
	configmapMgr.AddItemControllers(tracingController)

	return configmapMgr
}

func (c *ConfigmapMgr) SetHigressConfig(higressConfig *HigressConfig) {
	c.higressConfig.Store(higressConfig)
}

func (c *ConfigmapMgr) GetHigressConfig() *HigressConfig {
	value := c.higressConfig.Load()
	if value != nil {
		if higressConfig, ok := value.(*HigressConfig); ok {
			return higressConfig
		}
	}
	return nil
}

func (c *ConfigmapMgr) AddItemControllers(controllers ...ItemController) {
	c.ItemControllers = append(c.ItemControllers, controllers...)
}

func (c *ConfigmapMgr) AddOrUpdateHigressConfig(name util.ClusterNamespacedName) {
	if name.Namespace != c.Namespace || name.Name != HigressConfigMapName {
		return
	}
	higressConfigmap, err := c.HigressConfigLister.Get(HigressConfigMapName)
	if err != nil {
		IngressLog.Errorf("higress-config configmap is not found, namespace:%s, name:%s",
			name.Namespace, name.Name)
		return
	}

	if _, ok := higressConfigmap.Data[HigressConfigMapKey]; !ok {
		return
	}

	newHigressConfig := NewDefaultHigressConfig()
	if err = yaml.Unmarshal([]byte(higressConfigmap.Data[HigressConfigMapKey]), newHigressConfig); err != nil {
		IngressLog.Errorf("data:%s,  convert to higressconfig error, error: %+v", higressConfigmap.Data[HigressConfigMapKey], err)
		return
	}

	for _, itemController := range c.ItemControllers {
		if itemErr := itemController.ValidHigressConfig(newHigressConfig); itemErr != nil {
			IngressLog.Errorf("configmap %s controller valid higressconfig error, error: %+v", itemController.GetName(), itemErr)
			return
		}
	}

	oldHigressConfig := c.GetHigressConfig()
	result, _ := c.CompareHigressConfig(oldHigressConfig, newHigressConfig)

	if result == ResultDelete {
		newHigressConfig = NewDefaultHigressConfig()
	}

	if result == ResultReplace || result == ResultDelete {
		// Pass AddOrUpdateHigressConfig to itemControllers
		for _, itemController := range c.ItemControllers {
			IngressLog.Infof("configmap %s controller AddOrUpdateHigressConfig", itemController.GetName())
			if itemErr := itemController.AddOrUpdateHigressConfig(name, oldHigressConfig, newHigressConfig); itemErr != nil {
				IngressLog.Errorf("configmap %s controller AddOrUpdateHigressConfig error, error: %+v", itemController.GetName(), itemErr)
			}
		}
		c.SetHigressConfig(newHigressConfig)
		IngressLog.Infof("configmapMgr higress config AddOrUpdate success, reuslt is %d", result)
	}

}

func (c *ConfigmapMgr) ConstructEnvoyFilters() ([]*config.Config, error) {
	configs := make([]*config.Config, 0)
	for _, itemController := range c.ItemControllers {
		IngressLog.Infof("controller name:%s AConstructEnvoyFilters", itemController.GetName())
		if itemConfigs, err := itemController.ConstructEnvoyFilters(); err != nil {
			IngressLog.Errorf("controller name:%s ConstructEnvoyFilters error, error: %+v", itemController.GetName(), err)
		} else {
			configs = append(configs, itemConfigs...)
		}
	}
	return configs, nil
}

func (c *ConfigmapMgr) CompareHigressConfig(old *HigressConfig, new *HigressConfig) (Result, error) {
	if old == nil || new == nil {
		return ResultNothing, nil
	}

	if !reflect.DeepEqual(old, new) {
		return ResultReplace, nil
	}

	return ResultNothing, nil
}
