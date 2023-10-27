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
	"context"
	"fmt"
	"io"

	k8s "github.com/alibaba/higress/pkg/cmd/hgctl/kubernetes"
	"github.com/alibaba/higress/pkg/cmd/options"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func NewCommand() *cobra.Command {
	var (
		name string
		all  bool
	)

	uninstallCmd := &cobra.Command{
		Use:     "uninstall",
		Aliases: []string{"u", "uins"},
		Short:   "Uninstall WASM plugin",
		Example: `  # Uninstall WASM plugin using the WasmPlugin name
  hgctl plugin uninstall -p example-plugin-name

  # Uninstall all WASM plugins
  hgctl plugin uninstall -A
  `,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(uninstall(cmd.OutOrStdout(), name, all))
		},
	}

	flags := uninstallCmd.PersistentFlags()
	options.AddKubeConfigFlags(flags)
	k8s.AddHigressNamespaceFlags(flags)
	flags.StringVarP(&name, "name", "p", "", "Name of the WASM plugin you want to uninstall")
	flags.BoolVarP(&all, "all", "A", false, "Delete all installed WASM plugin")

	return uninstallCmd
}

func uninstall(w io.Writer, name string, all bool) error {
	dynCli, err := k8s.NewDynamicClient(options.DefaultConfigFlags.ToRawKubeConfigLoader())
	if err != nil {
		return errors.Wrap(err, "failed to build kubernetes dynamic client")
	}
	cli := k8s.NewWasmPluginClient(dynCli)

	ctx := context.TODO()
	plugins := make([]string, 0)
	if all {
		list, err := cli.List(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to get information of all wasm plugins")
		}
		for _, item := range list.Items {
			plugins = append(plugins, item.GetName())
		}
	} else {
		plugins = append(plugins, name)
	}

	for _, p := range plugins {
		err = deleteOne(ctx, w, cli, p)
		if err != nil {
			fmt.Fprintln(w, err.Error())
		}
	}

	return nil
}

func deleteOne(ctx context.Context, w io.Writer, cli *k8s.WasmPluginClient, name string) error {
	result, err := cli.Delete(ctx, name)
	if err != nil && k8serr.IsNotFound(err) {
		return errors.Errorf("wasm plugin %q is not found", fmt.Sprintf("%s/%s", k8s.HigressNamespace, name))
	} else if err != nil {
		return errors.Wrapf(err, "failed to uninstall wasm plugin %q", fmt.Sprintf("%s/%s", k8s.HigressNamespace, name))
	}

	fmt.Fprintf(w, "Uninstalled wasm plugin %q\n", fmt.Sprintf("%s/%s", result.GetNamespace(), result.GetName()))
	return nil
}
