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
	"github.com/alibaba/higress/pkg/cmd/options"
	"github.com/spf13/cobra"
)

type upgradeArgs struct {
	*InstallArgs
}

func addUpgradeFlags(cmd *cobra.Command, args *upgradeArgs) {
	cmd.PersistentFlags().StringArrayVarP(&args.Set, "set", "s", nil, setFlagHelpStr)
	cmd.PersistentFlags().StringVarP(&args.ManifestsPath, "manifests", "d", "", manifestsFlagHelpStr)
}

// newUpgradeCmd upgrades Istio control plane in-place with eligibility checks.
func newUpgradeCmd() *cobra.Command {
	upgradeArgs := &upgradeArgs{
		InstallArgs: &InstallArgs{},
	}
	upgradeCmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade Higress in-place",
		Long: "The upgrade command is an alias for the install command" +
			" that performs additional upgrade-related checks.",
		RunE: func(cmd *cobra.Command, args []string) (e error) {
			return Install(cmd.OutOrStdout(), upgradeArgs.InstallArgs)
		},
	}
	addUpgradeFlags(upgradeCmd, upgradeArgs)
	flags := upgradeCmd.Flags()
	options.AddKubeConfigFlags(flags)
	return upgradeCmd
}
