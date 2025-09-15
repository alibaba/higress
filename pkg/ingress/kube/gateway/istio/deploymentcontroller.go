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
	corev1 "k8s.io/api/core/v1"
	gateway "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/alibaba/higress/v2/pkg/config/constants"
)

// classInfo holds information about a gateway class
type classInfo struct {
	// controller name for this class
	controller string
	// description for this class
	description string
	// The key in the templates to use for this class
	templates string

	// defaultServiceType sets the default service type if one is not explicit set
	defaultServiceType corev1.ServiceType

	// disableRouteGeneration, if set, will make it so the controller ignores this class.
	disableRouteGeneration bool
}

var classInfos = getClassInfos()

var builtinClasses = getBuiltinClasses()

func getBuiltinClasses() map[gateway.ObjectName]gateway.GatewayController {
	res := map[gateway.ObjectName]gateway.GatewayController{
		defaultClassName: constants.ManagedGatewayController,
		// Start - Commented by Higress
		// constants.RemoteGatewayClassName: constants.UnmanagedGatewayController,
		// End - Commented by Higress
	}
	// Start - Commented by Higress
	//if features.EnableAmbientControllers {
	//	res[constants.WaypointGatewayClassName] = constants.ManagedGatewayMeshController
	//}
	// End - Commented by Higress
	return res
}

func getClassInfos() map[gateway.GatewayController]classInfo {
	// Start - Updated by Higress
	m := map[gateway.GatewayController]classInfo{
		constants.ManagedGatewayController: {
			controller:         constants.ManagedGatewayController,
			description:        "The default Higress GatewayClass",
			templates:          "kube-gateway",
			defaultServiceType: corev1.ServiceTypeLoadBalancer,
		},
		//UnmanagedGatewayController: {
		//	// This represents a gateway that our control plane cannot discover directly via the API server.
		//	// We shouldn't generate Istio resources for it. We aren't programming this gateway.
		//	controller:             UnmanagedGatewayController,
		//	description:            "Remote to this cluster. Does not deploy or affect configuration.",
		//	disableRouteGeneration: true,
		//},
	}
	//if features.EnableAmbientControllers {
	//	m[constants.ManagedGatewayMeshController] = classInfo{
	//		controller:         constants.ManagedGatewayMeshController,
	//		description:        "The default Istio waypoint GatewayClass",
	//		templates:          "waypoint",
	//		defaultServiceType: corev1.ServiceTypeClusterIP,
	//	}
	//}
	// End - Updated by Higress
	return m
}

// DeploymentController is removed by Higress
