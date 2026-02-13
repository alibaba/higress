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

package suite

import "istio.io/istio/pkg/util/sets"

type SupportedFeature string

const (
	// core: http
	HTTPConformanceFeature SupportedFeature = "http"

	// extended: extensibility
	WASMGoConformanceFeature   SupportedFeature = "wasm-go"
	WASMCPPConformanceFeature  SupportedFeature = "wasm-cpp"
	WASMRustConformanceFeature SupportedFeature = "wasm-rust"

	// extended: service discovery
	DubboConformanceFeature  SupportedFeature = "dubbo"
	EurekaConformanceFeature SupportedFeature = "eureka"
	NacosConformanceFeature  SupportedFeature = "nacos"

	// extended: envoy config
	EnvoyConfigConformanceFeature SupportedFeature = "envoy-config"
)

var WasmPluginTypeMap = map[string]SupportedFeature{
	"":     WASMGoConformanceFeature, // default
	"GO":   WASMGoConformanceFeature,
	"CPP":  WASMCPPConformanceFeature,
	"RUST": WASMRustConformanceFeature,
}

var AllFeatures = sets.Set[string]{}.
	Insert(string(HTTPConformanceFeature)).
	Insert(string(DubboConformanceFeature)).
	Insert(string(EurekaConformanceFeature)).
	Insert(string(NacosConformanceFeature)).
	Insert(string(EnvoyConfigConformanceFeature))

var ExperimentFeatures = sets.Set[string]{}.
	Insert(string(WASMGoConformanceFeature)).
	Insert(string(WASMCPPConformanceFeature)).
	Insert(string(WASMRustConformanceFeature))
