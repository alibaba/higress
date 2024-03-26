// Copyright (c) 2023 Alibaba Group Holding Ltd.
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

package credentials

import (
	"fmt"
	"strings"

	"istio.io/istio/pilot/pkg/features"
	"istio.io/istio/pilot/pkg/model/credentials"
	"istio.io/istio/pkg/cluster"
)

const (
	KubernetesIngressSecretType    = "kubernetes-ingress"
	KubernetesIngressSecretTypeURI = KubernetesIngressSecretType + "://"
)

func ToKubernetesIngressResource(clusterId, namespace, name string) string {
	if clusterId == "" {
		clusterId = features.ClusterName
	}
	return fmt.Sprintf("%s://%s/%s/%s", KubernetesIngressSecretType, clusterId, namespace, name)
}

func createSecretResourceForIngress(resourceName string) (credentials.SecretResource, error) {
	res := strings.TrimPrefix(resourceName, KubernetesIngressSecretTypeURI)
	split := strings.Split(res, "/")
	if len(split) != 3 {
		return credentials.SecretResource{}, fmt.Errorf("invalid resource name %q. Expected clusterId, namespace and name", resourceName)
	}
	clusterId := split[0]
	namespace := split[1]
	name := split[2]
	if len(clusterId) == 0 {
		return credentials.SecretResource{}, fmt.Errorf("invalid resource name %q. Expected clusterId", resourceName)
	}
	if len(namespace) == 0 {
		return credentials.SecretResource{}, fmt.Errorf("invalid resource name %q. Expected namespace", resourceName)
	}
	if len(name) == 0 {
		return credentials.SecretResource{}, fmt.Errorf("invalid resource name %q. Expected name", resourceName)
	}
	return credentials.SecretResource{ResourceType: KubernetesIngressSecretType, Name: name, Namespace: namespace, ResourceName: resourceName, Cluster: cluster.ID(clusterId)}, nil
}
