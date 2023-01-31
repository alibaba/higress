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

package common

import (
	"istio.io/istio/pkg/config/schema/collection"
	"istio.io/istio/pkg/config/schema/collections"
)

var IngressIR = collection.NewSchemasBuilder().
	MustAdd(collections.IstioNetworkingV1Alpha3Destinationrules).
	MustAdd(collections.IstioNetworkingV1Alpha3Envoyfilters).
	MustAdd(collections.IstioNetworkingV1Alpha3Gateways).
	MustAdd(collections.IstioNetworkingV1Alpha3Serviceentries).
	MustAdd(collections.IstioNetworkingV1Alpha3Virtualservices).
	MustAdd(collections.IstioExtensionsV1Alpha1Wasmplugins).
	Build()
