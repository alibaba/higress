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

package install

import (
	"fmt"
	"io"
	"os"

	k8s "github.com/alibaba/higress/pkg/cmd/hgctl/kubernetes"
	"github.com/alibaba/higress/pkg/cmd/options"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func NewCommand() *cobra.Command {
	var filename string

	installCmd := &cobra.Command{
		Use:     "install",
		Aliases: []string{"i", "ins"},
		Short:   "Install WASM plugin",
		Example: `  # Install WASM plugin using a WasmPlugin manifest
  hgctl plugin install -f example.yaml
  `,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(install(cmd.OutOrStdout(), filename))
		},
	}

	flags := installCmd.PersistentFlags()
	options.AddKubeConfigFlags(flags)

	installCmd.PersistentFlags().StringVarP(&filename, "filename", "f", "", "Specify the WasmPlugin manifest to install")
	installCmd.PersistentFlags().StringVarP(&k8s.CustomHigressNamespace, "namespace", "n", k8s.HigressNamespace, "The namespace where Higress was installed")

	return installCmd
}

func install(w io.Writer, filename string) error {
	cli, err := k8s.NewDynamicClient(options.DefaultConfigFlags.ToRawKubeConfigLoader())
	if err != nil {
		return fmt.Errorf("failed to build kubernetes dynamic client: %w", err)
	}

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	dc := yaml.NewYAMLOrJSONDecoder(f, 4096)
	for {
		obj := &unstructured.Unstructured{}
		if err = dc.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to get WasmPlugin object from manifest: %w", err)
		}

		if !isValidAPIVersionAndKind(obj) {
			fmt.Fprintf(w, "[WARNING] wasm plugin %q has invalid apiVersion or kind, expected %q %q, but got %q %q\n",
				obj.GetName(), k8s.HigressExtAPIVersion, k8s.WasmPluginKind, obj.GetAPIVersion(), obj.GetKind())
			continue
		}
		if !isValidNamespace(obj) {
			fmt.Fprintf(w, "[WARNING] wasm plugin %q has invalid namespace, automatically modified: %q -> %q\n",
				obj.GetName(), obj.GetNamespace(), k8s.CustomHigressNamespace)
			obj.SetNamespace(k8s.CustomHigressNamespace)
		}

		result, err := k8s.CreateWasmPlugin(cli, obj)
		if err != nil && errors.IsAlreadyExists(err) {
			fmt.Fprintf(w, "wasm plugin %q already exists\n", obj.GetName())
			continue
		} else if err != nil {
			return fmt.Errorf("failed to install wasm plugin %q: %w\n", obj.GetName(), err)
		}
		fmt.Fprintf(w, "wasm plugin %q installed\n", result.GetName())
	}

	return nil
}

func isValidAPIVersionAndKind(obj *unstructured.Unstructured) bool {
	return obj.GetAPIVersion() == k8s.HigressExtAPIVersion && obj.GetKind() == k8s.WasmPluginKind
}

func isValidNamespace(obj *unstructured.Unstructured) bool {
	return obj.GetNamespace() == k8s.CustomHigressNamespace
}
