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
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin"
	"github.com/spf13/cobra"
	"os"
)

// GetRootCommand returns the root cobra command to be executed
// by hgctl main.
func GetRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "hgctl",
		Long:              "A command line utility for operating Higress",
		SilenceUsage:      true,
		DisableAutoGenTag: true,
	}

	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newConfigCommand())
	rootCmd.AddCommand(newInstallCmd())
	rootCmd.AddCommand(newUninstallCmd())
	rootCmd.AddCommand(newUpgradeCmd())
	rootCmd.AddCommand(newProfileCmd())
	rootCmd.AddCommand(newDashboardCmd())
	rootCmd.AddCommand(newManifestCmd())
	rootCmd.AddCommand(plugin.NewCommand())
	rootCmd.AddCommand(newCompletionCmd(os.Stdout))
	rootCmd.AddCommand(newCodeDebugCmd())

	return rootCmd
}
