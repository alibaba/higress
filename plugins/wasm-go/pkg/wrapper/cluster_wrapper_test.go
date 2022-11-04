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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClusterAndHost(t *testing.T) {
	cases := []struct {
		name          string
		cluster       Cluster
		expectCluster string
		expectHost    string
	}{
		{
			name: "k8s",
			cluster: K8sCluster{
				ServiceName: "foo",
				Namespace:   "bar",
				Port:        8080,
				Version:     "1.0",
			},
			expectCluster: "outbound|8080|1.0|foo.bar.svc.cluster.local",
			expectHost:    "foo.bar.svc.cluster.local",
		},
		{
			name: "k8s default",
			cluster: K8sCluster{
				ServiceName: "foo",
				Port:        8080,
				Host:        "www.example.com",
			},
			expectCluster: "outbound|8080||foo.default.svc.cluster.local",
			expectHost:    "www.example.com",
		},
		{
			name: "nacos",
			cluster: NacosCluster{
				ServiceName: "foo",
				Group:       "DEFAULT_GROUP",
				NamespaceID: "xxxx",
				Port:        8080,
				Version:     "1.0",
			},
			expectCluster: "outbound|8080|1.0|foo.DEFAULT-GROUP.xxxx.nacos",
			expectHost:    "foo",
		},
		{
			name: "nacos ext",
			cluster: NacosCluster{
				ServiceName:   "foo",
				NamespaceID:   "xxxx",
				Port:          8080,
				IsExtRegistry: true,
				Host:          "www.test.com",
			},
			expectCluster: "outbound|8080||foo.DEFAULT-GROUP.xxxx.nacos-ext",
			expectHost:    "www.test.com",
		},
		{
			name: "static",
			cluster: StaticIpCluster{
				ServiceName: "foo",
				Port:        8080,
				Host:        "www.test.com",
			},
			expectCluster: "outbound|8080||foo.static",
			expectHost:    "www.test.com",
		},
		{
			name: "dns",
			cluster: DnsCluster{
				ServiceName: "foo",
				Port:        8080,
				Domain:      "www.test.com",
			},
			expectCluster: "outbound|8080||foo.dns",
			expectHost:    "www.test.com",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expectCluster, c.cluster.ClusterName())
			assert.Equal(t, c.expectHost, c.cluster.HostName())
		})
	}
}
