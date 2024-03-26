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

package name

// Kubernetes Kind strings.
const (
	CRDStr                            = "CustomResourceDefinition"
	ClusterRoleStr                    = "ClusterRole"
	ClusterRoleBindingStr             = "ClusterRoleBinding"
	CMStr                             = "ConfigMap"
	DaemonSetStr                      = "DaemonSet"
	DeploymentStr                     = "Deployment"
	EndpointStr                       = "Endpoints"
	HPAStr                            = "HorizontalPodAutoscaler"
	IngressStr                        = "Ingress"
	IstioOperator                     = "IstioOperator"
	MutatingWebhookConfigurationStr   = "MutatingWebhookConfiguration"
	NamespaceStr                      = "Namespace"
	PVCStr                            = "PersistentVolumeClaim"
	PodStr                            = "Pod"
	PDBStr                            = "PodDisruptionBudget"
	ReplicationControllerStr          = "ReplicationController"
	ReplicaSetStr                     = "ReplicaSet"
	RoleStr                           = "Role"
	RoleBindingStr                    = "RoleBinding"
	SAStr                             = "ServiceAccount"
	ServiceStr                        = "Service"
	SecretStr                         = "Secret"
	StatefulSetStr                    = "StatefulSet"
	ValidatingWebhookConfigurationStr = "ValidatingWebhookConfiguration"
)

// Istio Kind strings
const (
	EnvoyFilterStr        = "EnvoyFilter"
	GatewayStr            = "Gateway"
	DestinationRuleStr    = "DestinationRule"
	MeshPolicyStr         = "MeshPolicy"
	PeerAuthenticationStr = "PeerAuthentication"
	VirtualServiceStr     = "VirtualService"
	IstioOperatorStr      = "IstioOperator"
)

// Istio API Group Names
const (
	AuthenticationAPIGroupName = "authentication.istio.io"
	ConfigAPIGroupName         = "config.istio.io"
	NetworkingAPIGroupName     = "networking.istio.io"
	OperatorAPIGroupName       = "operator.istio.io"
	SecurityAPIGroupName       = "security.istio.io"
)
