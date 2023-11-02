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
	"io"
	"strings"

	"github.com/alibaba/higress/pkg/cmd/hgctl/helm"
	"github.com/alibaba/higress/pkg/cmd/hgctl/installer"
	"github.com/alibaba/higress/pkg/cmd/options"
	"github.com/spf13/cobra"
)

type upgradeArgs struct {
	*InstallArgs
}

func addUpgradeFlags(cmd *cobra.Command, args *upgradeArgs) {
	cmd.PersistentFlags().StringSliceVarP(&args.InFilenames, "filename", "f", nil, filenameFlagHelpStr)
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
			return upgrade(cmd.OutOrStdout(), upgradeArgs.InstallArgs)
		},
	}
	addUpgradeFlags(upgradeCmd, upgradeArgs)
	flags := upgradeCmd.Flags()
	options.AddKubeConfigFlags(flags)
	return upgradeCmd
}

// upgrade upgrade higress resources from the cluster.
func upgrade(writer io.Writer, iArgs *InstallArgs) error {
	setFlags := applyFlagAliases(iArgs.Set, iArgs.ManifestsPath)
	profileName, ok := installer.GetInstalledYamlPath()
	if !ok {
		fmt.Fprintf(writer, "\nHigress hasn't been installed yet!\n")
		return nil
	}

	valuesOverlay, err := helm.GetValuesOverylayFromFiles(iArgs.InFilenames)
	if err != nil {
		return err
	}

	_, profile, err := helm.GenProfile(profileName, valuesOverlay, setFlags)
	if err != nil {
		return err
	}

	fmt.Fprintf(writer, "🧐 Validating Profile: \"%s\" \n", profileName)
	err = profile.Validate()
	if err != nil {
		return err
	}

	if !promptUpgrade(writer) {
		return nil
	}

	err = upgradeManifests(profile, writer)
	if err != nil {
		return err
	}

	return nil
}

func promptUpgrade(writer io.Writer) bool {
	answer := ""
	for {
		fmt.Fprintf(writer, "All Higress resources will be upgraed from the cluster. \nProceed? (y/N)")
		fmt.Scanln(&answer)
		if strings.TrimSpace(answer) == "y" {
			fmt.Fprintf(writer, "\n")
			return true
		}
		if strings.TrimSpace(answer) == "N" {
			fmt.Fprintf(writer, "Cancelled.\n")
			return false
		}
	}
}

func upgradeManifests(profile *helm.Profile, writer io.Writer) error {
	installer, err := installer.NewInstaller(profile, writer, false)
	if err != nil {
		return err
	}

	err = installer.Upgrade()
	if err != nil {
		return err
	}

	return nil
}
