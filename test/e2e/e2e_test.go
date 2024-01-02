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

package test

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/alibaba/higress/test/e2e/conformance/tests"
	"github.com/alibaba/higress/test/e2e/conformance/utils/flags"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
)

func TestPrepareHigressConformanceTests(t *testing.T) {
	flag.Parse()

	cfg, err := config.GetConfig()
	require.NoError(t, err)

	client, err := client.New(cfg, client.Options{})
	require.NoError(t, err)

	require.NoError(t, v1.AddToScheme(client.Scheme()))

	cSuite := suite.New(suite.Options{
		Client:           client,
		IngressClassName: *flags.IngressClassName,
		Debug:            *flags.ShowDebug,
		WASMOptions: suite.WASMOptions{
			IsWasmPluginTest: *flags.IsWasmPluginTest,
			WasmPluginName:   *flags.WasmPluginName,
			WasmPluginType:   *flags.WasmPluginType,
		},
		GatewayAddress:             "localhost",
		EnableAllSupportedFeatures: true,
		IsEnvoyConfigTest:          *flags.IsEnvoyConfigTest,
	})

	cSuite.Prepare(t)
}

func TestRunHigressConformanceTests(t *testing.T) {
	flag.Parse()

	cfg, err := config.GetConfig()
	require.NoError(t, err)

	client, err := client.New(cfg, client.Options{})
	require.NoError(t, err)

	require.NoError(t, v1.AddToScheme(client.Scheme()))

	cSuite := suite.New(suite.Options{
		Client:           client,
		IngressClassName: *flags.IngressClassName,
		Debug:            *flags.ShowDebug,
		WASMOptions: suite.WASMOptions{
			IsWasmPluginTest: *flags.IsWasmPluginTest,
			WasmPluginName:   *flags.WasmPluginName,
			WasmPluginType:   *flags.WasmPluginType,
		},
		GatewayAddress:             "localhost",
		EnableAllSupportedFeatures: true,
		IsEnvoyConfigTest:          *flags.IsEnvoyConfigTest,
	})

	cSuite.Setup(t)
	cSuite.Run(t, tests.ConformanceTests)
}

func TestCleanHigressConformanceTests(t *testing.T) {
	flag.Parse()

	cfg, err := config.GetConfig()
	require.NoError(t, err)

	client, err := client.New(cfg, client.Options{})
	require.NoError(t, err)

	require.NoError(t, v1.AddToScheme(client.Scheme()))

	cSuite := suite.New(suite.Options{
		Client:           client,
		IngressClassName: *flags.IngressClassName,
		Debug:            *flags.ShowDebug,
		WASMOptions: suite.WASMOptions{
			IsWasmPluginTest: *flags.IsWasmPluginTest,
			WasmPluginName:   *flags.WasmPluginName,
			WasmPluginType:   *flags.WasmPluginType,
		},
		GatewayAddress:             "localhost",
		EnableAllSupportedFeatures: true,
		IsEnvoyConfigTest:          *flags.IsEnvoyConfigTest,
	})

	cSuite.Clean(t)
}
