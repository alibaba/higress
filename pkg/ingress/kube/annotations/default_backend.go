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
	"strconv"

	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"

	"github.com/alibaba/higress/pkg/ingress/kube/util"
	. "github.com/alibaba/higress/pkg/ingress/log"
)

const (
	annDefaultBackend = "default-backend"
	customHTTPError   = "custom-http-errors"

	defaultRedirectUrl            = "http://example.com/"
	FallbackRouteNameSuffix       = "-fallback"
	FallbackInjectHeaderRouteName = "x-envoy-route-name"
	FallbackInjectFallbackService = "x-envoy-fallback-service"
)

var (
	_ Parser       = fallback{}
	_ RouteHandler = fallback{}
)

type FallbackConfig struct {
	DefaultBackend   model.NamespacedName
	Port             uint32
	customHTTPErrors []uint32
}

type fallback struct{}

func (f fallback) Parse(annotations Annotations, config *Ingress, globalContext *GlobalContext) error {
	if !needFallback(annotations) {
		return nil
	}

	fallBackConfig := &FallbackConfig{}
	svcName, err := annotations.ParseStringASAP(annDefaultBackend)
	if err != nil {
		IngressLog.Errorf("Parse annotation default backend err: %v", err)
		return nil
	}

	fallBackConfig.DefaultBackend = util.SplitNamespacedName(svcName)
	if fallBackConfig.DefaultBackend.Name == "" {
		IngressLog.Errorf("Annotation default backend within ingress %s/%s is invalid", config.Namespace, config.Name)
		return nil
	}
	// Use ingress namespace instead, if user don't specify the namespace for default backend svc.
	if fallBackConfig.DefaultBackend.Namespace == "" {
		fallBackConfig.DefaultBackend.Namespace = config.Namespace
	}

	serviceLister, exist := globalContext.ClusterServiceList[config.ClusterId]
	if !exist {
		IngressLog.Errorf("service lister of cluster %s doesn't exist", config.ClusterId)
		return nil
	}

	fallbackSvc, err := serviceLister.Services(fallBackConfig.DefaultBackend.Namespace).Get(fallBackConfig.DefaultBackend.Name)
	if err != nil {
		IngressLog.Errorf("Fallback service %s/%s within ingress %s/%s is not found",
			fallBackConfig.DefaultBackend.Namespace, fallBackConfig.DefaultBackend.Name, config.Namespace, config.Name)
		return nil
	}
	if len(fallbackSvc.Spec.Ports) == 0 {
		IngressLog.Errorf("Fallback service %s/%s within ingress %s/%s haven't ports",
			fallBackConfig.DefaultBackend.Namespace, fallBackConfig.DefaultBackend.Name, config.Namespace, config.Name)
		return nil
	}
	// Use the first port like nginx ingress.
	fallBackConfig.Port = uint32(fallbackSvc.Spec.Ports[0].Port)

	config.Fallback = fallBackConfig

	if codes, err := annotations.ParseStringASAP(customHTTPError); err == nil {
		codesStr := splitBySeparator(codes, ",")
		var codesUint32 []uint32
		for _, rawCode := range codesStr {
			code, err := strconv.ParseUint(rawCode, 10, 32)
			if err != nil {
				IngressLog.Errorf("Custom HTTP code %s within ingress %s/%s is invalid", rawCode, config.Namespace, config.Name)
				continue
			}
			codesUint32 = append(codesUint32, uint32(code))
		}
		fallBackConfig.customHTTPErrors = codesUint32
	}

	return nil
}

func (f fallback) ApplyRoute(route *networking.HTTPRoute, config *Ingress) {
	fallback := config.Fallback
	if fallback == nil {
		return
	}

	// Add fallback svc
	route.Route[0].FallbackClusters = []*networking.Destination{
		{
			Host: util.CreateServiceFQDN(fallback.DefaultBackend.Namespace, fallback.DefaultBackend.Name),
			Port: &networking.PortSelector{
				Number: fallback.Port,
			},
		},
	}

	if len(fallback.customHTTPErrors) > 0 {
		route.InternalActiveRedirect = &networking.HTTPInternalActiveRedirect{
			MaxInternalRedirects:  1,
			RedirectResponseCodes: fallback.customHTTPErrors,
			AllowCrossScheme:      true,
			Headers: &networking.Headers{
				Request: &networking.Headers_HeaderOperations{
					Add: map[string]string{
						FallbackInjectHeaderRouteName: route.Name + FallbackRouteNameSuffix,
						FallbackInjectFallbackService: fallback.DefaultBackend.String(),
					},
				},
			},
			RedirectUrlRewriteSpecifier: &networking.HTTPInternalActiveRedirect_RedirectUrl{
				RedirectUrl: defaultRedirectUrl,
			},
			ForcedUseOriginalHost:             true,
			ForcedAddHeaderBeforeRouteMatcher: true,
		}
	}
}

func needFallback(annotations Annotations) bool {
	return annotations.HasASAP(annDefaultBackend)
}
