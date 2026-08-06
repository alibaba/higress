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
	"strings"

	"istio.io/istio/pilot/pkg/model/credentials"
	"istio.io/istio/pkg/util/sets"
	"k8s.io/apimachinery/pkg/types"
)

func extraSecret(name string) types.NamespacedName {
	result := types.NamespacedName{}
	res := strings.TrimPrefix(name, credentials.KubernetesIngressSecretTypeURI)
	split := strings.Split(res, "/")
	if len(split) != 3 {
		return result
	}

	return types.NamespacedName{
		Namespace: split[1],
		Name:      split[2],
	}
}

func splitBySeparator(content, separator string) []string {
	var result []string
	parts := strings.Split(content, separator)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func toSet(slice []string) sets.Set[string] {
	return sets.New[string](slice...)
}
