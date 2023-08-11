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

package config

import (
	"bytes"
	"fmt"
	"io"
	"os"

	k8s "github.com/alibaba/higress/pkg/cmd/hgctl/kubernetes"
	"github.com/alibaba/higress/pkg/cmd/options"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/printers"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/cmd/util/editor"
)

func newEditCommand() *cobra.Command {
	var name string

	editCmd := &cobra.Command{
		Use:     "edit",
		Aliases: []string{"e"},
		Short:   "Edit the installed WasmPlugin configuration, similar to `kubectl edit`",
		Example: `  # Edit the installed WASM plugin 'request-block'
  hgctl plugin config edit -p request-block
  `,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(edit(cmd.OutOrStdout(), name))
		},
	}

	flags := editCmd.PersistentFlags()
	options.AddKubeConfigFlags(flags)

	editCmd.PersistentFlags().StringVarP(&name, "name", "p", "", "The name of WasmPlugin that needs to be edited")
	editCmd.PersistentFlags().StringVarP(&k8s.CustomHigressNamespace, "namespace", "n", k8s.HigressNamespace, "The namespace where Higress was installed")

	return editCmd
}

func edit(w io.Writer, name string) error {
	cli, err := k8s.NewDynamicClient(options.DefaultConfigFlags.ToRawKubeConfigLoader())
	if err != nil {
		return fmt.Errorf("failed to build kubernetes dynamic client: %w", err)
	}

	originalObj, err := k8s.GetWasmPlugin(cli, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("wasm plugin %q is not found", name)
		}
		return fmt.Errorf("failed to get wasm plugin %q: %w", name, err)
	}

	gvk := schema.GroupVersionKind{
		Group:   k8s.HigressExtGroup,
		Version: k8s.HigressExtVersion,
		Kind:    k8s.WasmPluginKind,
	}
	originalObj.SetGroupVersionKind(gvk)
	originalObj.SetManagedFields(nil) // TODO(WeixinX): should write back

	buf := &bytes.Buffer{}
	var wObj io.Writer = buf
	printer := printers.YAMLPrinter{}
	err = printer.PrintObj(originalObj.DeepCopyObject(), wObj)
	if err != nil {
		return err
	}
	original := buf.Bytes()

	e := editor.NewDefaultEditor(editorEnvs())
	edited, file, err := e.LaunchTempFile("higress-wasm-edit-", ".yaml", buf)
	if err != nil {
		return fmt.Errorf("failed to launch editor: %w", err)
	}
	defer os.Remove(file)

	// no change
	if bytes.Equal(cmdutil.StripComments(original), cmdutil.StripComments(edited)) {
		fmt.Fprintf(w, "edit %q canceled, no change\n", name)
		return nil
	}

	eBuf := bytes.NewReader(edited)
	dc := yaml.NewYAMLOrJSONDecoder(eBuf, 4096)
	var editedObj unstructured.Unstructured
	err = dc.Decode(&editedObj)
	if err != nil {
		return err
	}
	err = keepSameOriginal(&editedObj, originalObj)
	if err != nil {
		fmt.Fprintln(w, err)
	}

	_, err = k8s.UpdateWasmPlugin(cli, &editedObj)
	if err != nil {
		return fmt.Errorf("failed to update wasm plugin %q: %w", name, err)
	}

	fmt.Fprintf(w, "wasm plugin %q edited\n", name)

	return nil
}

func editorEnvs() []string {
	return []string{
		"KUBE_EDITOR",
		"EDITOR",
	}
}

// to avoid changing the apiVersion, kind, namespace and name, keep them the same as the original
func keepSameOriginal(edited, original *unstructured.Unstructured) error {
	if edited.GroupVersionKind().String() != original.GroupVersionKind().String() ||
		edited.GetNamespace() != original.GetNamespace() ||
		edited.GetName() != original.GetName() {

		edited.SetGroupVersionKind(original.GroupVersionKind())
		edited.SetNamespace(original.GetNamespace())
		edited.SetName(original.GetName())
		return fmt.Errorf("[WARNING] Ensure that the apiVersion, kind, namespace, and name are the same as the original and are automatically corrected")
	}

	return nil
}
