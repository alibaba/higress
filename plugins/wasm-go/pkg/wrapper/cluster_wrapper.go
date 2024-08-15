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

package wrapper

import (
	"fmt"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

type Cluster interface {
	ClusterName() string
	HostName() string
}

type RouteCluster struct {
	Host string
}

func (c RouteCluster) ClusterName() string {
	routeName, err := proxywasm.GetProperty([]string{"cluster_name"})
	if err != nil {
		proxywasm.LogErrorf("get route cluster failed, err:%v", err)
	}
	return string(routeName)
}

func (c RouteCluster) HostName() string {
	if c.Host != "" {
		return c.Host
	}
	return GetRequestHost()
}

type K8sCluster struct {
	ServiceName string
	Namespace   string
	Port        int64
	Version     string
	Host        string
}

func (c K8sCluster) ClusterName() string {
	namespace := "default"
	if c.Namespace != "" {
		namespace = c.Namespace
	}
	return fmt.Sprintf("outbound|%d|%s|%s.%s.svc.cluster.local",
		c.Port, c.Version, c.ServiceName, namespace)
}

func (c K8sCluster) HostName() string {
	if c.Host != "" {
		return c.Host
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", c.ServiceName, c.Namespace)
}

type NacosCluster struct {
	ServiceName string
	// use DEFAULT-GROUP by default
	Group       string
	NamespaceID string
	Port        int64
	// set true if use edas/sae registry
	IsExtRegistry bool
	Version       string
	Host          string
}

func (c NacosCluster) ClusterName() string {
	group := "DEFAULT-GROUP"
	if c.Group != "" {
		group = strings.ReplaceAll(c.Group, "_", "-")
	}
	tail := "nacos"
	if c.IsExtRegistry {
		tail += "-ext"
	}
	return fmt.Sprintf("outbound|%d|%s|%s.%s.%s.%s",
		c.Port, c.Version, c.ServiceName, group, c.NamespaceID, tail)
}

func (c NacosCluster) HostName() string {
	if c.Host != "" {
		return c.Host
	}
	return c.ServiceName
}

type StaticIpCluster struct {
	ServiceName string
	Port        int64
	Host        string
}

func (c StaticIpCluster) ClusterName() string {
	return fmt.Sprintf("outbound|%d||%s.static", c.Port, c.ServiceName)
}

func (c StaticIpCluster) HostName() string {
	if c.Host != "" {
		return c.Host
	}
	return c.ServiceName
}

type DnsCluster struct {
	ServiceName string
	Domain      string
	Port        int64
}

func (c DnsCluster) ClusterName() string {
	return fmt.Sprintf("outbound|%d||%s.dns", c.Port, c.ServiceName)
}

func (c DnsCluster) HostName() string {
	return c.Domain
}

type ConsulCluster struct {
	ServiceName string
	Datacenter  string
	Port        int64
	Host        string
}

func (c ConsulCluster) ClusterName() string {
	tail := "consul"
	return fmt.Sprintf("outbound|%d||%s.%s.%s",
		c.Port, c.ServiceName, c.Datacenter, tail)
}

func (c ConsulCluster) HostName() string {
	if c.Host != "" {
		return c.Host
	}
	return c.ServiceName
}

type FQDNCluster struct {
	FQDN string
	Host string
	Port int64
}

func (c FQDNCluster) ClusterName() string {
	return fmt.Sprintf("outbound|%d||%s", c.Port, c.FQDN)
}

func (c FQDNCluster) HostName() string {
	if c.Host != "" {
		return c.Host
	}
	return c.FQDN
}
