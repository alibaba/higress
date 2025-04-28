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

package mcpserver

import (
	"fmt"

	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type MultiConfigListener struct {
	configClient  config_client.IConfigClient
	onChange      func(map[string]string)
	configCache   map[string]string
	innerCallback func(string, string, string, string)
}

func NewMultiConfigListener(configClient config_client.IConfigClient, onChange func(map[string]string)) *MultiConfigListener {
	result := &MultiConfigListener{
		configClient: configClient,
		configCache:  make(map[string]string),
		onChange:     onChange,
	}

	result.innerCallback = func(namespace string, group string, dataId string, content string) {
		result.configCache[group+DefaultJoiner+dataId] = content
		result.onChange(result.configCache)
	}

	return result
}

func (l *MultiConfigListener) StartListen(configs []vo.ConfigParam) error {
	for _, config := range configs {
		content, err := l.configClient.GetConfig(vo.ConfigParam{
			DataId: config.DataId,
			Group:  config.Group,
		})

		if err != nil {
			return fmt.Errorf("get config %s/%s err: %v", config.Group, config.DataId, err)
		}
		l.configCache[config.Group+DefaultJoiner+config.DataId] = content
		err = l.configClient.ListenConfig(vo.ConfigParam{
			DataId:   config.DataId,
			Group:    config.Group,
			OnChange: l.innerCallback,
		})

		if err != nil {
			return fmt.Errorf("listener to config %s/%s error: %w", config.Group, config.DataId, err)
		}
	}

	l.onChange(l.configCache)
	return nil
}

func (l *MultiConfigListener) Stop() {
	l.configClient.CloseClient()
}

func (l *MultiConfigListener) CancelListen(configs []vo.ConfigParam) error {
	for _, config := range configs {
		if _, ok := l.configCache[config.Group+DefaultJoiner+config.DataId]; ok {
			err := l.configClient.CancelListenConfig(vo.ConfigParam{
				DataId: config.DataId,
				Group:  config.Group,
			})

			if err != nil {
				return fmt.Errorf("cancel config %s/%s error: %w", config.Group, config.DataId, err)
			}
			delete(l.configCache, config.Group+config.DataId)
		}
	}
	return nil
}

type ServiceCache struct {
	services map[string]*NacosServiceRef
	client   naming_client.INamingClient
}

type NacosServiceRef struct {
	refs      map[string]func([]model.Instance)
	callback  func(services []model.Instance, err error)
	instances *[]model.Instance
}

func NewServiceCache(client naming_client.INamingClient) *ServiceCache {
	return &ServiceCache{
		client:   client,
		services: make(map[string]*NacosServiceRef),
	}
}

func (c *ServiceCache) AddListener(group string, serviceName string, key string, callback func([]model.Instance)) error {
	uniqueServiceName := c.makeServiceUniqueName(group, serviceName)
	if _, ok := c.services[uniqueServiceName]; !ok {
		instances, err := c.client.SelectAllInstances(vo.SelectAllInstancesParam{
			GroupName:   group,
			ServiceName: serviceName,
		})

		if err != nil {
			return err
		}

		ref := &NacosServiceRef{
			refs:      map[string]func([]model.Instance){},
			instances: &instances,
		}

		ref.callback = func(services []model.Instance, err error) {
			ref.instances = &services
			for _, refCallback := range ref.refs {
				refCallback(*ref.instances)
			}
		}

		c.services[uniqueServiceName] = ref

		err = c.client.Subscribe(&vo.SubscribeParam{
			GroupName:         group,
			ServiceName:       serviceName,
			SubscribeCallback: ref.callback,
		})
		if err != nil {
			return err
		}
	}

	ref := c.services[uniqueServiceName]
	ref.refs[key] = callback
	callback(*ref.instances)
	return nil
}

func (c *ServiceCache) RemoveListener(group string, serviceName string, key string) error {
	if ref, ok := c.services[c.makeServiceUniqueName(group, serviceName)]; ok {
		delete(ref.refs, key)
		if len(ref.refs) == 0 {
			err := c.client.Unsubscribe(&vo.SubscribeParam{
				GroupName:         group,
				ServiceName:       serviceName,
				SubscribeCallback: ref.callback,
			})

			delete(c.services, c.makeServiceUniqueName(group, serviceName))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *ServiceCache) makeServiceUniqueName(group string, serviceName string) string {
	return fmt.Sprintf("%s-%s", group, serviceName)
}

func (c *ServiceCache) Stop() {
	c.client.CloseClient()
}
