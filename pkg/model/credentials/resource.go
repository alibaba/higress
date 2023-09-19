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
