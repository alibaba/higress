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

package uninstall

import (
	"fmt"
	"io"

	k8s "github.com/alibaba/higress/pkg/cmd/hgctl/kubernetes"
	"github.com/alibaba/higress/pkg/cmd/options"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func NewCommand() *cobra.Command {
	var (
		name string
		all  bool
	)

	uninstallCmd := &cobra.Command{
		Use:     "uninstall",
		Aliases: []string{"u", "unins"},
		Short:   "Uninstall WASM plugin",
		Example: `  # Uninstall WASM plugin using the WasmPlugin name
  hgctl plugin uninstall -p example-plugin

  # Uninstall all WASM plugins
  hgctl plugin uninstall -A
  `,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(uninstall(cmd.OutOrStdout(), name, all))
		},
	}

	flags := uninstallCmd.PersistentFlags()
	options.AddKubeConfigFlags(flags)

	uninstallCmd.PersistentFlags().StringVarP(&name, "name", "p", "", "Specify the WasmPlugin name to uninstall")
	uninstallCmd.PersistentFlags().BoolVarP(&all, "all", "A", false, "Delete all installed wasm plugin")
	uninstallCmd.PersistentFlags().StringVarP(&k8s.CustomHigressNamespace, "namespace", "n", k8s.HigressNamespace, "The namespace where Higress was installed")

	return uninstallCmd
}

func uninstall(w io.Writer, name string, all bool) error {
	cli, err := k8s.NewDynamicClient(options.DefaultConfigFlags.ToRawKubeConfigLoader())
	if err != nil {
		return fmt.Errorf("failed to build kubernetes dynamic client: %w", err)
	}

	if all {
		list, err := k8s.ListWasmPlugins(cli)
		if err != nil {
			return fmt.Errorf("failed to get informations of all wasm plugins: %w", err)
		}

		for _, item := range list.Items {
			err = deleteOne(w, cli, item.GetName())
			if err != nil {
				fmt.Fprintln(w, err.Error())
				continue
			}
		}

	} else {
		err := deleteOne(w, cli, name)
		if err != nil {
			fmt.Fprintln(w, err.Error())
		}
	}

	return nil
}

func deleteOne(w io.Writer, cli *k8s.DynamicClient, name string) error {
	result, err := k8s.DeleteWasmPlugin(cli, name)
	if err != nil && errors.IsNotFound(err) {
		return fmt.Errorf("wasm plugin %q is not found", name)
	} else if err != nil {
		return fmt.Errorf("failed to uninstall wasm plugin %q: %w", name, err)
	}

	fmt.Fprintf(w, "wasm plugin %q uninstalled\n", result.GetName())
	return nil
}
