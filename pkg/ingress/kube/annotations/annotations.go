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

import (
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/util/sets"
	listersv1 "k8s.io/client-go/listers/core/v1"
)

type GlobalContext struct {
	// secret key is cluster/namespace/name
	WatchedSecrets sets.Set

	ClusterSecretLister map[string]listersv1.SecretLister

	ClusterServiceList map[string]listersv1.ServiceLister
}

type Meta struct {
	Namespace    string
	Name         string
	RawClusterId string
	ClusterId    string
}

// Ingress defines the valid annotations present in one NGINX Ingress.
type Ingress struct {
	Meta

	Cors *CorsConfig

	Rewrite *RewriteConfig

	Redirect *RedirectConfig

	UpstreamTLS *UpstreamTLSConfig

	DownstreamTLS *DownstreamTLSConfig

	Canary *CanaryConfig

	IPAccessControl *IPAccessControlConfig

	Retry *RetryConfig

	LoadBalance *LoadBalanceConfig

	Fallback *FallbackConfig

	Auth *AuthConfig

	Destination *DestinationConfig

	IgnoreCase *IgnoreCaseConfig

	Match *MatchConfig

	HeaderControl *HeaderControlConfig
}

func (i *Ingress) NeedRegexMatch() bool {
	if i.Rewrite == nil {
		return false
	}

	return i.Rewrite.RewriteTarget != "" || i.Rewrite.UseRegex
}

func (i *Ingress) IsCanary() bool {
	if i.Canary == nil {
		return false
	}

	return i.Canary.Enabled
}

// CanaryKind return byHeader, byWeight
func (i *Ingress) CanaryKind() (bool, bool) {
	if !i.IsCanary() {
		return false, false
	}

	// first header, cookie
	if i.Canary.Header != "" || i.Canary.Cookie != "" {
		return true, false
	}

	// then weight
	return false, true
}

func (i *Ingress) NeedTrafficPolicy() bool {
	return i.UpstreamTLS != nil ||
		i.LoadBalance != nil
}

type AnnotationHandler interface {
	Parser
	GatewayHandler
	VirtualServiceHandler
	RouteHandler
	TrafficPolicyHandler
}

type AnnotationHandlerManager struct {
	parsers                []Parser
	gatewayHandlers        []GatewayHandler
	virtualServiceHandlers []VirtualServiceHandler
	routeHandlers          []RouteHandler
	trafficPolicyHandlers  []TrafficPolicyHandler
}

func NewAnnotationHandlerManager() AnnotationHandler {
	return &AnnotationHandlerManager{
		parsers: []Parser{
			canary{},
			cors{},
			downstreamTLS{},
			redirect{},
			rewrite{},
			upstreamTLS{},
			ipAccessControl{},
			retry{},
			loadBalance{},
			fallback{},
			auth{},
			destination{},
			ignoreCaseMatching{},
			match{},
			headerControl{},
		},
		gatewayHandlers: []GatewayHandler{
			downstreamTLS{},
		},
		virtualServiceHandlers: []VirtualServiceHandler{
			ipAccessControl{},
		},
		routeHandlers: []RouteHandler{
			cors{},
			redirect{},
			rewrite{},
			ipAccessControl{},
			retry{},
			fallback{},
			ignoreCaseMatching{},
			match{},
			headerControl{},
		},
		trafficPolicyHandlers: []TrafficPolicyHandler{
			upstreamTLS{},
			loadBalance{},
		},
	}
}

func (h *AnnotationHandlerManager) Parse(annotations Annotations, config *Ingress, globalContext *GlobalContext) error {
	for _, parser := range h.parsers {
		_ = parser.Parse(annotations, config, globalContext)
	}

	return nil
}

func (h *AnnotationHandlerManager) ApplyGateway(gateway *networking.Gateway, config *Ingress) {
	for _, handler := range h.gatewayHandlers {
		handler.ApplyGateway(gateway, config)
	}
}

func (h *AnnotationHandlerManager) ApplyVirtualServiceHandler(virtualService *networking.VirtualService, config *Ingress) {
	for _, handler := range h.virtualServiceHandlers {
		handler.ApplyVirtualServiceHandler(virtualService, config)
	}
}

func (h *AnnotationHandlerManager) ApplyRoute(route *networking.HTTPRoute, config *Ingress) {
	for _, handler := range h.routeHandlers {
		handler.ApplyRoute(route, config)
	}
}

func (h *AnnotationHandlerManager) ApplyTrafficPolicy(trafficPolicy *networking.TrafficPolicy_PortTrafficPolicy, config *Ingress) {
	for _, handler := range h.trafficPolicyHandlers {
		handler.ApplyTrafficPolicy(trafficPolicy, config)
	}
}
