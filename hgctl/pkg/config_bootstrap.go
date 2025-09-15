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

package hgctl

import (
	"fmt"

	"github.com/alibaba/higress/hgctl/cmd/hgctl/config"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func bootstrapConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:     "bootstrap <pod-name>",
		Aliases: []string{"b"},
		Short:   "Retrieves bootstrap Envoy xDS resources from the specified Higress Gateway Pod",
		Long:    `Retrieves information about bootstrap Envoy xDS resources from the specified Higress Gateway Pod`,
		Example: `  # Retrieve summary about bootstrap configuration for a given pod from Envoy.
  hgctl gateway-config bootstrap <pod-name> -n <pod-namespace>

  # Retrieve full configuration dump as YAML
  hgctl gateway-config bootstrap <pod-name> -n <pod-namespace> -o yaml

  # Retrieve full configuration dump with short syntax
  hgctl gc b <pod-name> -n <pod-namespace>
`,
		Run: func(c *cobra.Command, args []string) {
			cmdutil.CheckErr(runBootstrapConfig(c, args))
		},
	}

	return configCmd
}

func runBootstrapConfig(c *cobra.Command, args []string) error {
	if len(args) != 0 {
		podName = args[0]
	}
	envoyConfig, err := config.GetEnvoyConfig(&config.GetEnvoyConfigOptions{
		PodName:         podName,
		PodNamespace:    podNamespace,
		BindAddress:     bindAddress,
		Output:          output,
		EnvoyConfigType: config.BootstrapEnvoyConfigType,
		IncludeEds:      true,
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(c.OutOrStdout(), string(envoyConfig))
	return err
}
