// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Updated based on Istio codebase by Higress

package istio

import (
	"istio.io/istio/pilot/pkg/credentials"
	"istio.io/istio/pilot/pkg/model"
	creds "istio.io/istio/pilot/pkg/model/credentials"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/host"
	"istio.io/istio/pkg/config/labels"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/util/sets"
	corev1 "k8s.io/api/core/v1"
	k8s "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/alibaba/higress/pkg/config/constants"
)

const (
	// Start - Updated by Higress
	defaultClassName             = constants.DefaultGatewayClass
	gatewayAliasForAnnotationKey = "gateway.higress.io/alias-for"
	gatewayTLSTerminateModeKey   = "gateway.higress.io/tls-terminate-mode"
	gatewayNameOverride          = "gateway.higress.io/name-override"
	gatewaySAOverride            = "gateway.higress.io/service-account"
	serviceTypeOverride          = "networking.higress.io/service-type"
	// End - Updated by Higress
)

// GatewayResources stores all gateway resources used for our conversion.
type GatewayResources struct {
	GatewayClass   []config.Config
	Gateway        []config.Config
	HTTPRoute      []config.Config
	TCPRoute       []config.Config
	TLSRoute       []config.Config
	ReferenceGrant []config.Config
	// Namespaces stores all namespace in the cluster, keyed by name
	Namespaces map[string]*corev1.Namespace
	// Credentials stores all credentials in the cluster
	Credentials credentials.Controller

	// Start - Added by Higress
	DefaultGatewaySelector map[string]string
	// End - Added by Higress

	// Domain for the cluster. Typically, cluster.local
	Domain  string
	Context GatewayContext
}

type Grants struct {
	AllowAll     bool
	AllowedNames sets.String
}

type AllowedReferences map[Reference]map[Reference]*Grants

func (refs AllowedReferences) SecretAllowed(resourceName string, namespace string) bool {
	p, err := creds.ParseResourceName(resourceName, "", "", "")
	if err != nil {
		log.Warnf("failed to parse resource name %q: %v", resourceName, err)
		return false
	}
	from := Reference{Kind: gvk.KubernetesGateway, Namespace: k8s.Namespace(namespace)}
	to := Reference{Kind: gvk.Secret, Namespace: k8s.Namespace(p.Namespace)}
	allow := refs[from][to]
	if allow == nil {
		return false
	}
	return allow.AllowAll || allow.AllowedNames.Contains(p.Name)
}

func (refs AllowedReferences) BackendAllowed(
	k config.GroupVersionKind,
	backendName k8s.ObjectName,
	backendNamespace k8s.Namespace,
	routeNamespace string,
) bool {
	from := Reference{Kind: k, Namespace: k8s.Namespace(routeNamespace)}
	to := Reference{Kind: gvk.Service, Namespace: backendNamespace}
	allow := refs[from][to]
	if allow == nil {
		return false
	}
	return allow.AllowAll || allow.AllowedNames.Contains(string(backendName))
}

// IstioResources stores all outputs of our conversion
type IstioResources struct {
	Gateway        []config.Config
	VirtualService []config.Config
	// AllowedReferences stores all allowed references, from Reference -> to Reference(s)
	AllowedReferences AllowedReferences
	// ReferencedNamespaceKeys stores the label key of all namespace selections. This allows us to quickly
	// determine if a namespace update could have impacted any Gateways. See namespaceEvent.
	ReferencedNamespaceKeys sets.String

	// ResourceReferences stores all resources referenced by gateway-api resources. This allows us to quickly
	// determine if a resource update could have impacted any Gateways.
	// key: referenced resources(e.g. secrets), value: gateway-api resources(e.g. gateways)
	ResourceReferences map[model.ConfigKey][]model.ConfigKey
}

// Reference stores a reference to a namespaced GVK, as used by ReferencePolicy
type Reference struct {
	Kind      config.GroupVersionKind
	Namespace k8s.Namespace
}

// Start - Added by Higress - Based on istio/pilot/pkg/model/push_context.go
// serviceIndex is an index of all services by various fields for easy access during push.
type serviceIndex struct {
	// privateByNamespace are services that can reachable within the same namespace, with exportTo "."
	privateByNamespace map[string][]*model.Service
	// public are services reachable within the mesh with exportTo "*"
	public []*model.Service
	// exportedToNamespace are services that were made visible to this namespace
	// by an exportTo explicitly specifying this namespace.
	exportedToNamespace map[string][]*model.Service

	// HostnameAndNamespace has all services, indexed by hostname then namespace.
	HostnameAndNamespace map[host.Name]map[string]*model.Service `json:"-"`

	all []*model.Service

	// instancesByPort contains a map of service key and instances by port. It is stored here
	// to avoid recomputations during push. This caches instanceByPort calls with empty labels.
	// Call InstancesByPort directly when instances need to be filtered by actual labels.
	instancesByPort map[string]map[int][]*model.ServiceInstance
}

func newServiceIndex() *serviceIndex {
	return &serviceIndex{
		all:                  []*model.Service{},
		public:               []*model.Service{},
		privateByNamespace:   map[string][]*model.Service{},
		exportedToNamespace:  map[string][]*model.Service{},
		HostnameAndNamespace: map[host.Name]map[string]*model.Service{},
		instancesByPort:      map[string]map[int][]*model.ServiceInstance{},
	}
}

// ServiceInstancesByPort returns the cached instances by port if it exists.
func (si *serviceIndex) ServiceInstancesByPort(svc *model.Service, port int, labels labels.Instance) []*model.ServiceInstance {
	out := []*model.ServiceInstance{}
	if instances, exists := si.instancesByPort[svc.Key()][port]; exists {
		// Use cached version of instances by port when labels are empty.
		if len(labels) == 0 {
			return instances
		}
		// If there are labels,	we will filter instances by pod labels.
		for _, instance := range instances {
			// check that one of the input labels is a subset of the labels
			if labels.SubsetOf(instance.Endpoint.Labels) {
				out = append(out, instance)
			}
		}
	}

	return out
}

// ServiceInstances returns the cached instances by svc if exists.
func (si *serviceIndex) ServiceInstances(svcKey string) map[int][]*model.ServiceInstance {
	if instances, exists := si.instancesByPort[svcKey]; exists {
		return instances
	}
	return nil
}

// End - Added by Higress
