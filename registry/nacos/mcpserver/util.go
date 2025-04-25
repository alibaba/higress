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
