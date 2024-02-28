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

type ManifestArgs struct {
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
	// Devel if set true when version is latest, it will get latest version, otherwise it will get latest stable version
	Devel bool
}

func (a *ManifestArgs) String() string {
	var b strings.Builder
	b.WriteString("KubeConfigPath:   " + a.KubeConfigPath + "\n")
	b.WriteString("Context:          " + a.Context + "\n")
	b.WriteString("Set:              " + fmt.Sprint(a.Set) + "\n")
	b.WriteString("ManifestsPath:    " + a.ManifestsPath + "\n")
	return b.String()
}

// newManifestCmd generates a higress install manifest and applies it to a cluster
func newManifestCmd() *cobra.Command {
	iArgs := &ManifestArgs{}
	manifestCmd := &cobra.Command{
		Use:   "manifest",
		Short: "Generate higress manifests.",
		Long:  "The manifest command generates an higress install manifests.",
	}

	generate := newManifestGenerateCmd(iArgs)
	addManifestFlags(generate, iArgs)
	flags := generate.Flags()
	options.AddKubeConfigFlags(flags)
	manifestCmd.AddCommand(generate)

	return manifestCmd
}

func addManifestFlags(cmd *cobra.Command, args *ManifestArgs) {
	cmd.PersistentFlags().StringSliceVarP(&args.InFilenames, "filename", "f", nil, filenameFlagHelpStr)
	cmd.PersistentFlags().StringArrayVarP(&args.Set, "set", "s", nil, setFlagHelpStr)
	cmd.PersistentFlags().StringVarP(&args.ManifestsPath, "manifests", "d", "", manifestsFlagHelpStr)
	cmd.PersistentFlags().BoolVar(&args.Devel, "devel", false, "use development versions (alpha, beta, and release candidate releases), If version is set, this is ignored")
}

// newManifestGenerateCmd generates a higress install manifest and applies it to a cluster
func newManifestGenerateCmd(iArgs *ManifestArgs) *cobra.Command {
	installCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate higress manifests.",
		Long:  "The manifest generate command generates higress install manifests.",
		// nolint: lll
		Example: `  # Generate higress manifests
  hgctl manifest generate
`,
		Args: cobra.ExactArgs(0),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return generate(cmd.OutOrStdout(), iArgs)
		},
	}

	return installCmd
}

func generate(writer io.Writer, iArgs *ManifestArgs) error {
	setFlags := applyFlagAliases(iArgs.Set, iArgs.ManifestsPath)

	// check profileName
	psf := helm.GetValueForSetFlag(setFlags, "profile")
	if len(psf) == 0 {
		setFlags = append(setFlags, fmt.Sprintf("profile=%s", helm.InstallLocalK8s))
	}

	_, profile, _, err := helm.GenerateConfig(iArgs.InFilenames, setFlags)
	if err != nil {
		return fmt.Errorf("generate config: %v", err)
	}

	err = profile.Validate()
	if err != nil {
		return err
	}

	err = genManifests(profile, writer, iArgs.Devel)
	if err != nil {
		return fmt.Errorf("failed to install manifests: %v", err)
	}
	return nil
}

func genManifests(profile *helm.Profile, writer io.Writer, devel bool) error {
	cliClient, err := kubernetes.NewCLIClient(options.DefaultConfigFlags.ToRawKubeConfigLoader())
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client: %w", err)
	}

	op, err := installer.NewK8sInstaller(profile, cliClient, writer, true, devel, installer.InstallInstallerMode)
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

	if err := op.GenerateManifests(manifestMap); err != nil {
		return err
	}
	return nil
}
