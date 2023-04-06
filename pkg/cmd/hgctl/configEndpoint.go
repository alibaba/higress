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

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func endpointConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:     "endpoint <pod-name>",
		Short:   "Retrieves endpoint Envoy xDS resources from the specified Higress Gateway Pod",
		Aliases: []string{"e"},
		Long:    `Retrieves information about endpoint Envoy xDS resources from the specified Higress Gateway Pod`,
		Example: `  # Retrieve summary about endpoint configuration for a given pod from Envoy.
  hgctl gateway-config endpoint <pod-name> -n <pod-namespace>

  # Retrieve configuration dump as YAML
  hgctl gateway-config endpoint <pod-name> -n <pod-namespace> -o yaml

  # Retrieve configuration dump with short syntax
  hgctl gc e <pod-name> -n <pod-namespace>
`,
		Run: func(c *cobra.Command, args []string) {
			cmdutil.CheckErr(runEndpointConfig(c, args))
		},
	}

	return configCmd
}

func runEndpointConfig(c *cobra.Command, args []string) error {
	configDump, err := retrieveConfigDump(args, true)
	if err != nil {
		return err
	}

	endpoint, err := GetXDSResource(EndpointEnvoyConfigType, configDump)
	if err != nil {
		return err
	}

	out, err := formatGatewayConfig(endpoint, output)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(c.OutOrStdout(), string(out))
	return err
}
