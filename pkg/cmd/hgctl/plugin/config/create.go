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
	"fmt"
	"io"
	"os"

	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/utils"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func newCreateCommand() *cobra.Command {
	var target string

	createCmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"c"},
		Short:   "Create the WASM plugin configuration template file",
		Example: `  hgctl plugin config create`,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(create(cmd.OutOrStdout(), target))
		},
	}

	createCmd.PersistentFlags().StringVarP(&target, "target", "t", "./", "Directory where the configuration is generated")

	return createCmd
}

func create(w io.Writer, target string) error {
	target, err := utils.GetAbsolutePath(target)
	if err != nil {
		return errors.Wrap(err, "invalid target path")
	}
	if err = os.MkdirAll(target, 0755); err != nil {
		return err
	}
	if err = GenPluginConfYAML(configHelpTmpl, target); err != nil {
		return errors.Wrap(err, "failed to create configuration template")
	}

	fmt.Fprintf(w, "Created configuration template %q\n", fmt.Sprintf("%s/%s", target, "plugin-conf.yaml"))

	return nil
}

var configHelpTmpl = &PluginConf{
	Name:        "Plugin Name",
	Namespace:   "higress-system",
	Title:       "Display Name",
	Description: "Plugin Description",
	IconUrl:     "Plugin Icon",
	Version:     "0.1.0",
	Category:    "auth | security | protocol | flow-control | flow-monitor | custom",
	Phase:       "UNSPECIFIED_PHASE | AUTHN | AUTHZ | STATS",
	Priority:    0,
	Config:      "  Plugin Configuration",
	Url:         "Plugin Image URL",
}
