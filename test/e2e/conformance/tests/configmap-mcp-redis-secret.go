// Copyright (c) 2025 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alibaba/higress/v2/pkg/ingress/kube/configmap"
	"github.com/alibaba/higress/v2/test/e2e/conformance/utils/envoy"
	"github.com/alibaba/higress/v2/test/e2e/conformance/utils/kubernetes"
	"github.com/alibaba/higress/v2/test/e2e/conformance/utils/suite"
)

func init() {
	Register(ConfigMapMcpRedisSecret)
}

var ConfigMapMcpRedisSecret = suite.ConformanceTest{
	ShortName:   "ConfigMapMcpRedisSecret",
	Description: "Envoy MCP session filter should resolve Redis password from Kubernetes secret and react to updates",
	Manifests:   []string{"tests/configmap-mcp-redis-secret.yaml"},
	Features:    []suite.SupportedFeature{suite.EnvoyConfigConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		const (
			configMapNamespace = "higress-system"
			configMapName      = "higress-config"
			configMapKey       = "higress"
			secretNamespace    = "higress-system"
			secretName         = "redis-credentials"
			secretKey          = "password"

			initialSecretValue = "InitialSecretFromSecret123"
			updatedSecretValue = "UpdatedSecretFromSecret456"
		)

		higressCfg := &configmap.HigressConfig{
			McpServer: &configmap.McpServer{
				Enable: true,
				Redis: &configmap.RedisConfig{
					Address: "redis:6379",
					PasswordSecret: &configmap.SecretKeyReference{
						Name: secretName,
						Key:  secretKey,
					},
				},
			},
		}

		err := kubernetes.ApplyConfigmapDataWithYaml(t, suite.Client, configMapNamespace, configMapName, configMapKey, higressCfg)
		require.NoErrorf(t, err, "failed to update %s/%s", configMapNamespace, configMapName)

		assertRedisPassword := func(password string) {
			envoy.AssertEnvoyConfig(t, suite.TimeoutConfig, envoy.Assertion{
				Path: `configs.#(@type=="type.googleapis.com/envoy.admin.v3.EcdsConfigDump").` +
					`ecds_filters.#(ecds_filter.name=="golang-filter-mcp-session").` +
					`ecds_filter.typed_config.plugin_config.value.redis`,
				CheckType:       envoy.CheckTypeMatch,
				TargetNamespace: configMapNamespace,
				ExpectEnvoyConfig: map[string]interface{}{
					"password": password,
				},
			})
		}

		assertRedisPassword(initialSecretValue)

		err = kubernetes.ApplySecret(t, suite.Client, secretNamespace, secretName, secretKey, updatedSecretValue)
		require.NoErrorf(t, err, "failed to update %s/%s secret", secretNamespace, secretName)

		assertRedisPassword(updatedSecretValue)
	},
}
