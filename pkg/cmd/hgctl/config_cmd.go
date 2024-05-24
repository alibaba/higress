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

	"github.com/alibaba/higress/cmd/hgctl/config"
	"github.com/alibaba/higress/pkg/cmd/options"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var (
	output       string
	podName      string
	podNamespace string
)

const (
	defaultProxyAdminPort = 15000
	containerName         = "envoy"
)

func newConfigCommand() *cobra.Command {
	cfgCommand := &cobra.Command{
		Use:     "gateway-config",
		Aliases: []string{"gc"},
		Short:   "Retrieve Higress Gateway configuration.",
		Long:    "Retrieve information about Higress Gateway Configuration.",
	}

	cfgCommand.AddCommand(allConfigCmd())
	cfgCommand.AddCommand(bootstrapConfigCmd())
	cfgCommand.AddCommand(clusterConfigCmd())
	cfgCommand.AddCommand(endpointConfigCmd())
	cfgCommand.AddCommand(listenerConfigCmd())
	cfgCommand.AddCommand(routeConfigCmd())

	flags := cfgCommand.Flags()
	options.AddKubeConfigFlags(flags)

	cfgCommand.PersistentFlags().StringVarP(&output, "output", "o", "json", "Output format: one of json|yaml|short")
	cfgCommand.PersistentFlags().StringVarP(&podNamespace, "namespace", "n", "higress-system", "Namespace where envoy proxy pod are installed.")

	return cfgCommand
}

func allConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "all <pod-name>",
		Short: "Retrieves all Envoy xDS resources from the specified Higress Gateway Pod",
		Long:  `Retrieves information about all Envoy xDS resources from the specified Higress Gateway Pod`,
		Example: `  # Retrieve summary about all configuration for a given pod from Envoy.
  hgctl gateway-config all <pod-name> -n <pod-namespace>

  # Retrieve full configuration dump as YAML
  hgctl gateway-config all <pod-name> -n <pod-namespace> -o yaml

  # Retrieve full configuration dump with short syntax
  hgctl gc all <pod-name> -n <pod-namespace>
`,
		Run: func(c *cobra.Command, args []string) {
			cmdutil.CheckErr(runAllConfig(c, args))
		},
	}

	return configCmd
}

func runAllConfig(c *cobra.Command, args []string) error {
	if len(args) != 0 {
		podName = args[0]
	}
	envoyConfig, err := config.GetEnvoyConfig(&config.GetEnvoyConfigOptions{
		PodName:         podName,
		PodNamespace:    podNamespace,
		BindAddress:     bindAddress,
		Output:          output,
		EnvoyConfigType: config.AllEnvoyConfigType,
		IncludeEds:      true,
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(c.OutOrStdout(), string(envoyConfig))
	return err
}
