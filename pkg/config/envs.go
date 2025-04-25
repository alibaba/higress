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

package config

import "istio.io/pkg/env"

var (
	PodNamespace = env.RegisterStringVar("POD_NAMESPACE", "higress-system", "").Get()
	PodName      = env.RegisterStringVar("POD_NAME", "", "").Get()
	GatewayName  = env.RegisterStringVar("GATEWAY_NAME", "higress-gateway", "").Get()
	// Revision is the value of the Istio control plane revision, e.g. "canary",
	// and is the value used by the "istio.io/rev" label.
	Revision              = env.Register("REVISION", "", "").Get()
	McpWasmPluginImageUrl = env.RegisterStringVar("MCP_WASM_PLUGIN_IMAGE_URL", "", "").Get()
)
