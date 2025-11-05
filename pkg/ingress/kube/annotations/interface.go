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

package annotations

import networking "istio.io/api/networking/v1alpha3"

type Parser interface {
	// Parse parses ingress annotations and puts result on config
	Parse(annotations Annotations, config *Ingress, globalContext *GlobalContext) error
}

type GatewayHandler interface {
	// ApplyGateway parsed ingress annotation config reflected on gateway
	ApplyGateway(gateway *networking.Gateway, config *Ingress)
}

type VirtualServiceHandler interface {
	// ApplyVirtualServiceHandler parsed ingress annotation config reflected on virtual host
	ApplyVirtualServiceHandler(virtualService *networking.VirtualService, config *Ingress)
}

type RouteHandler interface {
	// ApplyRoute parsed ingress annotation config reflected on route
	ApplyRoute(route *networking.HTTPRoute, config *Ingress)
}

type TrafficPolicyHandler interface {
	// ApplyTrafficPolicy parsed ingress annotation config reflected on traffic policy
	ApplyTrafficPolicy(trafficPolicy *networking.TrafficPolicy, portTrafficPolicy *networking.TrafficPolicy_PortTrafficPolicy, config *Ingress)
}
