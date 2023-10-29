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
	"context"
	"fmt"
	"io"
	"os"

	k8s "github.com/alibaba/higress/pkg/cmd/hgctl/kubernetes"
	"github.com/alibaba/higress/pkg/cmd/options"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
		Short:   "Edit the installed WASM plugin configuration",
		Example: `  # Edit the installed WASM plugin 'request-block'
  hgctl plugin config edit -p request-block
  `,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(edit(cmd.OutOrStdout(), name))
		},
	}

	flags := editCmd.PersistentFlags()
	options.AddKubeConfigFlags(flags)
	k8s.AddHigressNamespaceFlags(flags)
	flags.StringVarP(&name, "name", "p", "", "Name of the WASM plugin that needs to be edited")

	return editCmd
}

func edit(w io.Writer, name string) error {
	// TODO(WeixinX): Use WasmPlugin Object type instead of Unstructured
	dynCli, err := k8s.NewDynamicClient(options.DefaultConfigFlags.ToRawKubeConfigLoader())
	if err != nil {
		return errors.Wrap(err, "failed to build kubernetes dynamic client")
	}
	cli := k8s.NewWasmPluginClient(dynCli)

	originalObj, err := cli.Get(context.TODO(), name)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return errors.Errorf("wasm plugin %q is not found", fmt.Sprintf("%s/%s", k8s.HigressNamespace, name))
		}
		return errors.Wrapf(err, "failed to get wasm plugin %q", fmt.Sprintf("%s/%s", k8s.HigressNamespace, name))
	}

	originalObj.SetGroupVersionKind(k8s.WasmPluginGVK)
	originalObj.SetManagedFields(nil) // TODO(WeixinX): Managed Fields should be written back

	buf := &bytes.Buffer{}
	var wObj io.Writer = buf
	printer := printers.YAMLPrinter{}
	if err = printer.PrintObj(originalObj.DeepCopyObject(), wObj); err != nil {
		return err
	}
	original := buf.Bytes()
	e := editor.NewDefaultEditor(editorEnvs())
	edited, file, err := e.LaunchTempFile("higress-wasm-edit-", ".yaml", buf)
	if err != nil {
		return errors.Wrap(err, "failed to launch editor")
	}
	defer os.Remove(file)

	if bytes.Equal(cmdutil.StripComments(original), cmdutil.StripComments(edited)) { // no change
		fmt.Fprintf(w, "edit %q canceled, no change\n",
			fmt.Sprintf("%s/%s", originalObj.GetNamespace(), originalObj.GetName()))
		return nil
	}

	var editedObj unstructured.Unstructured
	eBuf := bytes.NewReader(edited)
	dc := yaml.NewYAMLOrJSONDecoder(eBuf, 4096)
	if err = dc.Decode(&editedObj); err != nil {
		return err
	}
	if !keepSameMeta(&editedObj, originalObj) {
		fmt.Fprintln(w, "Warning: ensure that the apiVersion, kind, namespace, and name are the same as the original and are automatically corrected")
	}

	ret, err := cli.Update(context.TODO(), &editedObj)
	if err != nil {
		return errors.Wrapf(err, "failed to update wasm plugin %q",
			fmt.Sprintf("%s/%s", originalObj.GetNamespace(), originalObj.GetName()))
	}

	fmt.Fprintf(w, "Edited wasm plugin %q\n", fmt.Sprintf("%s/%s", ret.GetNamespace(), ret.GetName()))

	return nil
}

func editorEnvs() []string {
	return []string{
		"KUBE_EDITOR",
		"EDITOR",
	}
}

// to avoid changing the apiVersion, kind, namespace and name, keep them the same as the original
func keepSameMeta(edited, original *unstructured.Unstructured) bool {
	same := true
	if edited.GroupVersionKind().String() != original.GroupVersionKind().String() {
		edited.SetGroupVersionKind(original.GroupVersionKind())
		same = false
	}
	if edited.GetNamespace() != original.GetNamespace() {
		edited.SetNamespace(original.GetNamespace())
		same = false
	}
	if edited.GetName() != original.GetName() {
		edited.SetName(original.GetName())
		same = false
	}
	return same
}
