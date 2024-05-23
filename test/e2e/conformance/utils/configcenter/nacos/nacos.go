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

package nacos

import (
	"net/url"
	"strconv"
	"strings"

	cc "github.com/alibaba/higress/test/e2e/conformance/utils/configcenter"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type storage struct {
	client config_client.IConfigClient
}

func NewClient(addr string) (cc.Storage, error) {
	clientConfig := constant.NewClientConfig(
		constant.WithNamespaceId(""),
		constant.WithUsername(""),
		constant.WithPassword(""),
		constant.WithLogLevel("info"),
	)

	serverUrl, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	rawPort := serverUrl.Port()
	var port uint64
	if rawPort != "" {
		port, err = strconv.ParseUint(rawPort, 10, 0)
		if err != nil || port < 1 || port > 65535 {
			return nil, err
		}
	} else {
		port = 80
	}
	path := serverUrl.Path
	if strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr:      serverUrl.Hostname(),
			ContextPath: path,
			Port:        port,
			Scheme:      serverUrl.Scheme,
		},
	}

	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, err
	}
	return storage{
		client: client,
	}, nil
}

func (s storage) PublishConfig(kind, name, namespace, content string) error {
	dataId := cc.GetDataId(kind, name)
	group := namespace
	_, err := s.client.PublishConfig(vo.ConfigParam{
		DataId:  dataId,
		Group:   group,
		Content: content,
	})
	return err
}

func (s storage) DeleteConfig(kind, name, namespace string) error {
	dataId := cc.GetDataId(kind, name)
	group := namespace
	_, err := s.client.DeleteConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	return err
}
