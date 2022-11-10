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
	"istio.io/istio/pilot/pkg/networking/core/v1alpha3/mseingress"
)

const (
	whitelist = "whitelist-source-range"
)

var (
	_ Parser       = &ipAccessControl{}
	_ RouteHandler = &ipAccessControl{}
)

type IPAccessControl struct {
	isWhite  bool
	remoteIp []string
}

type IPAccessControlConfig struct {
	Route *IPAccessControl
}

type ipAccessControl struct{}

func (i ipAccessControl) Parse(annotations Annotations, config *Ingress, _ *GlobalContext) error {
	if !needIPAccessControlConfig(annotations) {
		return nil
	}

	ipConfig := &IPAccessControlConfig{}
	defer func() {
		config.IPAccessControl = ipConfig
	}()

	var route *IPAccessControl
	if rawWhitelist, err := annotations.ParseStringASAP(whitelist); err == nil {
		route = &IPAccessControl{
			isWhite:  true,
			remoteIp: splitStringWithSpaceTrim(rawWhitelist),
		}
	}

	if route != nil {
		ipConfig.Route = route
	}

	return nil
}

func (i ipAccessControl) ApplyVirtualServiceHandler(_ *networking.VirtualService, _ *Ingress) {
	// DO NOTHING
}

func (i ipAccessControl) ApplyRoute(route *networking.HTTPRoute, config *Ingress) {
	ac := config.IPAccessControl
	if ac == nil || ac.Route == nil {
		return
	}

	filter := &networking.IPAccessControl{}
	if ac.Route.isWhite {
		filter.RemoteIpBlocks = ac.Route.remoteIp
	} else {
		filter.NotRemoteIpBlocks = ac.Route.remoteIp
	}

	route.RouteHTTPFilters = append(route.RouteHTTPFilters, &networking.HTTPFilter{
		Name: mseingress.IPAccessControl,
		Filter: &networking.HTTPFilter_IpAccessControl{
			IpAccessControl: filter,
		},
	})
}

func needIPAccessControlConfig(annotations Annotations) bool {
	return annotations.HasASAP(whitelist)
}
