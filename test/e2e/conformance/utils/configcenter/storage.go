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

package cc

import "strings"

type Storage interface {
	PublishConfig(kind, name, namespace, content string) error
	DeleteConfig(kind, name, namespace string) error
}

func GetDataId(kind, name string) string {
	switch strings.ToLower(kind) {
	case "configmap":
		kind = "configmaps"
	case "secret":
		kind = "secrets"
	case "ingress":
		kind = "ingresses"
	case "service":
		kind = "services"
	case "ingressclass":
		kind = "ingressclasses"
	case "mcpbridge":
		kind = "mcpbridges"
	case "wasmplugin":
		kind = "wasmplugins"
	case "http2rpc":
		kind = "http2rpcs"
	default:
	}
	return kind + "." + name
}
