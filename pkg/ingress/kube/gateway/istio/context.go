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
	"fmt"
	"sort"
	"strconv"
	"strings"

	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/config/host"
	"istio.io/istio/pkg/util/sets"
	corev1 "k8s.io/api/core/v1"
)

// GatewayContext contains a minimal subset of push context functionality to be exposed to GatewayAPIControllers
type GatewayContext struct {
	ps *model.PushContext
	si *serviceIndex
}

func NewGatewayContext(ps *model.PushContext, si *serviceIndex) GatewayContext {
	return GatewayContext{ps, si}
}

// ResolveGatewayInstances attempts to resolve all instances that a gateway will be exposed on.
// Note: this function considers *all* instances of the service; its possible those instances will not actually be properly functioning
// gateways, so this is not 100% accurate, but sufficient to expose intent to users.
// The actual configuration generation is done on a per-workload basis and will get the exact set of matched instances for that workload.
// Four sets are exposed:
// * Internal addresses (eg istio-ingressgateway.istio-system.svc.cluster.local:80).
// * External addresses (eg 1.2.3.4), this comes from LoadBalancer services. There may be multiple in some cases (especially multi cluster).
// * Pending addresses (eg istio-ingressgateway.istio-system.svc), are LoadBalancer-type services with pending external addresses.
// * Warnings for references that could not be resolved. These are intended to be user facing.
func (gc GatewayContext) ResolveGatewayInstances(
	namespace string,
	gwsvcs []string,
	// Start - Updated by Higress
	gatewaySelector map[string]string,
	// End - Updated by Higress
	servers []*networking.Server,
) (internal, external, pending, warns []string) {
	ports := map[int]struct{}{}
	for _, s := range servers {
		ports[int(s.Port.Number)] = struct{}{}
	}
	foundInternal := sets.New[string]()
	foundExternal := sets.New[string]()
	foundPending := sets.New[string]()
	warnings := []string{}
	// Start - Added by Higress
	if gatewaySelector != nil && len(gatewaySelector) != 0 {
		gwsvcs = append([]string{}, gwsvcs...)
		for _, svc := range gc.si.all {
			matches := true
			for k, v := range gatewaySelector {
				if svc.Attributes.Labels[k] != v {
					matches = false
					break
				}
			}
			if matches {
				gwsvcs = append(gwsvcs, string(svc.Hostname))
			}
		}
	}
	// End - Added by Higress
	for _, g := range gwsvcs {
		// Start - Updated by Higress
		svc, f := gc.si.HostnameAndNamespace[host.Name(g)][namespace]
		// End - Updated by Higress
		if !f {
			otherNamespaces := []string{}
			for ns := range gc.si.HostnameAndNamespace[host.Name(g)] {
				otherNamespaces = append(otherNamespaces, `"`+ns+`"`) // Wrap in quotes for output
			}
			if len(otherNamespaces) > 0 {
				sort.Strings(otherNamespaces)
				warnings = append(warnings, fmt.Sprintf("hostname %q not found in namespace %q, but it was found in namespace(s) %v",
					g, namespace, strings.Join(otherNamespaces, ", ")))
			} else {
				warnings = append(warnings, fmt.Sprintf("hostname %q not found", g))
			}
			continue
		}
		svcKey := svc.Key()
		for port := range ports {
			// Start - Updated by Higress
			instances := gc.si.ServiceInstancesByPort(svc, port, nil)
			// End - Updated by Higress
			if len(instances) > 0 {
				foundInternal.Insert(fmt.Sprintf("%s:%d", g, port))
				if svc.Attributes.ClusterExternalAddresses.Len() > 0 {
					// Fetch external IPs from all clusters
					svc.Attributes.ClusterExternalAddresses.ForEach(func(c cluster.ID, externalIPs []string) {
						foundExternal.InsertAll(externalIPs...)
					})
				} else if corev1.ServiceType(svc.Attributes.Type) == corev1.ServiceTypeLoadBalancer {
					if !foundPending.Contains(g) {
						warnings = append(warnings, fmt.Sprintf("address pending for hostname %q", g))
						foundPending.Insert(g)
					}
				}
			} else {
				// Start - Updated by Higress
				instancesByPort := gc.si.ServiceInstances(svcKey)
				// End - Updated by Higress
				if instancesEmpty(instancesByPort) {
					warnings = append(warnings, fmt.Sprintf("no instances found for hostname %q", g))
				} else {
					hintPort := sets.New[string]()
					for _, instances := range instancesByPort {
						for _, i := range instances {
							if i.Endpoint.EndpointPort == uint32(port) {
								hintPort.Insert(strconv.Itoa(i.ServicePort.Port))
							}
						}
					}
					if hintPort.Len() > 0 {
						warnings = append(warnings, fmt.Sprintf(
							"port %d not found for hostname %q (hint: the service port should be specified, not the workload port. Did you mean one of these ports: %v?)",
							port, g, sets.SortedList(hintPort)))
					} else {
						warnings = append(warnings, fmt.Sprintf("port %d not found for hostname %q", port, g))
					}
				}
			}
		}
	}
	sort.Strings(warnings)
	return sets.SortedList(foundInternal), sets.SortedList(foundExternal), sets.SortedList(foundPending), warnings
}

func (gc GatewayContext) GetService(hostname, namespace string) *model.Service {
	// Start - Updated by Higress
	return gc.si.HostnameAndNamespace[host.Name(hostname)][namespace]
	// End - Updated by Higress
}

func instancesEmpty(m map[int][]*model.ServiceInstance) bool {
	for _, instances := range m {
		if len(instances) > 0 {
			return false
		}
	}
	return true
}
