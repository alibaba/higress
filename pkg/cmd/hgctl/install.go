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
	"github.com/alibaba/higress/pkg/cmd/hgctl/kubernetes"
	"github.com/alibaba/higress/pkg/cmd/options"
	"github.com/spf13/cobra"
)

const (
	setFlagHelpStr = `Override an higress profile value, e.g. to choose a profile
(--set profile=local-k8s), or override profile values (--set gateway.replicas=2), or override helm values (--set values.global.proxy.resources.requsts.cpu=500m).`
	// manifestsFlagHelpStr is the command line description for --manifests
	manifestsFlagHelpStr = `Specify a path to a directory of profiles
(e.g. ~/Downloads/higress/manifests).`
	outputHelpstr = "Specify a file to write profile yaml"

	profileNameK8s      = "k8s"
	profileNameLocalK8s = "local-k8s"
)

type InstallArgs struct {
	InFilenames []string
	// KubeConfigPath is the path to kube config file.
	KubeConfigPath string
	// Context is the cluster context in the kube config
	Context string
	// Set is a string with element format "path=value" where path is an profile path and the value is a
	// value to set the node at that path to.
	Set []string
	// ManifestsPath is a path to a ManifestsPath and profiles directory in the local filesystem with a release tgz.
	ManifestsPath string
}

func (a *InstallArgs) String() string {
	var b strings.Builder
	b.WriteString("KubeConfigPath:   " + a.KubeConfigPath + "\n")
	b.WriteString("Context:          " + a.Context + "\n")
	b.WriteString("Set:              " + fmt.Sprint(a.Set) + "\n")
	b.WriteString("ManifestsPath:    " + a.ManifestsPath + "\n")
	return b.String()
}

func addInstallFlags(cmd *cobra.Command, args *InstallArgs) {
	cmd.PersistentFlags().StringArrayVarP(&args.Set, "set", "s", nil, setFlagHelpStr)
	cmd.PersistentFlags().StringVarP(&args.ManifestsPath, "manifests", "d", "", manifestsFlagHelpStr)
}

// --manifests is an alias for --set installPackagePath=
func applyFlagAliases(flags []string, manifestsPath string) []string {
	if manifestsPath != "" {
		flags = append(flags, fmt.Sprintf("installPackagePath=%s", manifestsPath))
	}
	return flags
}

// newInstallCmd generates a higress install manifest and applies it to a cluster
func newInstallCmd() *cobra.Command {
	iArgs := &InstallArgs{}
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Applies an higress manifest, installing or reconfiguring higress on a cluster.",
		Long:  "The install command generates an higress install manifest and applies it to a cluster.",
		// nolint: lll
		Example: `  # Apply a default higress installation
  hgctl install

  # Install higress on local kubernetes cluster 
  hgctl install --set profile=local-k8s 

  # To override profile setting
  hgctl install --set profile=local-k8s  --set global.enableIstioAPI=true --set gateway.replicas=2"

  # To override helm setting
  hgctl install --set profile=local-k8s  --set values.global.proxy.resources.requsts.cpu=500m"

`,
		Args: cobra.ExactArgs(0),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return Install(cmd.OutOrStdout(), iArgs)
		},
	}
	addInstallFlags(installCmd, iArgs)
	flags := installCmd.Flags()
	options.AddKubeConfigFlags(flags)
	return installCmd
}

func Install(writer io.Writer, iArgs *InstallArgs) error {
	setFlags := applyFlagAliases(iArgs.Set, iArgs.ManifestsPath)

	// check profileName
	psf := helm.GetValueForSetFlag(setFlags, "profile")
	if len(psf) == 0 {
		psf = promptProfileName(writer)
		setFlags = append(setFlags, fmt.Sprintf("profile=%s", psf))
	}

	if !promptInstall(writer, psf) {
		return nil
	}

	_, profile, profileName, err := helm.GenerateConfig(iArgs.InFilenames, setFlags)
	if err != nil {
		return fmt.Errorf("generate config: %v", err)
	}

	fmt.Fprintf(writer, "🧐 Validating Profile: \"%s\" \n", profileName)
	err = profile.Validate()
	if err != nil {
		return err
	}

	err = InstallManifests(profile, writer)
	if err != nil {
		return fmt.Errorf("failed to install manifests: %v", err)
	}
	return nil
}

func promptInstall(writer io.Writer, profileName string) bool {
	answer := ""
	for {
		fmt.Fprintf(writer, "\nThis will install Higress \"%s\" profile into the cluster. \nProceed? (y/N)", profileName)
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

func promptProfileName(writer io.Writer) string {
	answer := ""
	fmt.Fprintf(writer, "Please select higress install configration profile:\n")
	fmt.Fprintf(writer, "1.Install higress to local kubernetes cluster like kind etc.\n")
	fmt.Fprintf(writer, "2.Install higress to kubernetes cluster\n")
	for {
		fmt.Fprintf(writer, "Please input 1 or 2 to select, input your selection:")
		fmt.Scanln(&answer)
		if strings.TrimSpace(answer) == "1" {
			return profileNameLocalK8s
		}
		if strings.TrimSpace(answer) == "2" {
			return profileNameK8s
		}
	}

}

func InstallManifests(profile *helm.Profile, writer io.Writer) error {
	cliClient, err := kubernetes.NewCLIClient(options.DefaultConfigFlags.ToRawKubeConfigLoader())
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client: %w", err)
	}

	op, err := installer.NewInstaller(profile, cliClient, writer)
	if err != nil {
		return err
	}
	if err := op.Run(); err != nil {
		return err
	}

	manifestMap, err := op.RenderManifests()
	if err != nil {
		return err
	}

	fmt.Fprintf(writer, "\n⌛️ Processing installation... \n\n")
	if err := op.ApplyManifests(manifestMap); err != nil {
		return err
	}
	fmt.Fprintf(writer, "\n🎊 Install All Resources Complete!\n")
	return nil
}
