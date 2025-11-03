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

package main

import (
	"errors"
	"strings"

	"github.com/higress-group/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

type OpaConfig struct {
	policy  string
	timeout uint32

	client wrapper.HttpClient
}

func Client(json gjson.Result) (wrapper.HttpClient, error) {
	serviceSource := strings.TrimSpace(json.Get("serviceSource").String())
	serviceName := strings.TrimSpace(json.Get("serviceName").String())
	servicePort := json.Get("servicePort").Int()

	host := strings.TrimSpace(json.Get("host").String())
	if host == "" {
		if serviceName == "" || servicePort == 0 {
			return nil, errors.New("invalid service config")
		}
	}

	var namespace string
	if serviceSource == "k8s" || serviceSource == "nacos" {
		if namespace = strings.TrimSpace(json.Get("namespace").String()); namespace == "" {
			return nil, errors.New("namespace not allow empty")
		}
	}

	switch serviceSource {
	case "k8s":
		return wrapper.NewClusterClient(wrapper.K8sCluster{
			ServiceName: serviceName,
			Namespace:   namespace,
			Port:        servicePort,
		}), nil
	case "nacos":
		return wrapper.NewClusterClient(wrapper.NacosCluster{
			ServiceName: serviceName,
			NamespaceID: namespace,
			Port:        servicePort,
		}), nil
	case "ip":
		return wrapper.NewClusterClient(wrapper.StaticIpCluster{
			ServiceName: serviceName,
			Host:        host,
			Port:        servicePort,
		}), nil
	case "dns":
		return wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: serviceName,
			Port:        servicePort,
			Domain:      json.Get("domain").String(),
		}), nil
	case "route":
		return wrapper.NewClusterClient(wrapper.RouteCluster{
			Host: host,
		}), nil
	}
	return nil, errors.New("unknown service source: " + serviceSource)
}
