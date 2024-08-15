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
	"istio.io/istio/istioctl/pkg/writer/envoy/configdump"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func listenerConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:     "listener <pod-name>",
		Aliases: []string{"l"},
		Short:   "Retrieves listener Envoy xDS resources from the specified Higress Gateway Pod",
		Long:    `Retrieves information about listener Envoy xDS resources from the specified Higress Gateway Pod`,
		Example: `  # Retrieve summary about listener configuration for a given pod from Envoy.
  hgctl gateway-config listener <pod-name> -n <pod-namespace>

  # Retrieve full configuration dump as YAML
  hgctl gateway-config listener <pod-name> -n <pod-namespace> -o yaml

  # Retrieve full configuration dump with short syntax
  hgctl gc l <pod-name> -n <pod-namespace>
`,
		Run: func(c *cobra.Command, args []string) {
			cmdutil.CheckErr(runListenerConfig(c, args))
		},
	}

	return configCmd
}

func runListenerConfig(c *cobra.Command, args []string) error {
	if len(args) != 0 {
		podName = args[0]
	}
	configWriter, err := config.GetEnvoyConfigWriter(&config.GetEnvoyConfigOptions{
		PodName:         podName,
		PodNamespace:    podNamespace,
		BindAddress:     bindAddress,
		Output:          output,
		EnvoyConfigType: config.ListenerEnvoyConfigType,
		IncludeEds:      true,
	}, c.OutOrStdout())
	if err != nil {
		return err
	}
	switch output {
	case summaryOutput:
		return configWriter.PrintListenerSummary(configdump.ListenerFilter{Verbose: true})
	case jsonOutput, yamlOutput:
		return configWriter.PrintListenerDump(configdump.ListenerFilter{Verbose: true}, output)
	default:
		return fmt.Errorf("output format %q not supported", output)
	}
}
