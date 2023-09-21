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

type uninstallArgs struct {
	// purgeIstioCRD delete  all of Istio resources.
	purgeIstioCRD bool
	// istioNamespace is the target namespace of istio control plane.
	istioNamespace string
	// namespace is the namespace of higress installed .
	namespace string
	// verbose generates verbose output.
	verbose bool
}

func addUninstallFlags(cmd *cobra.Command, args *uninstallArgs) {
	cmd.PersistentFlags().StringVar(&args.istioNamespace, "istio-namespace", "istio-system",
		"The namespace of Istio Control Plane.")
	cmd.PersistentFlags().StringVarP(&args.namespace, "namespace", "n", "higress-system",
		"The namespace of higress")
	cmd.PersistentFlags().BoolVarP(&args.purgeIstioCRD, "purge-istio-crd", "p", false,
		"Delete  all of Istio resources")
}

// newUninstallCmd command uninstalls Istio from a cluster
func newUninstallCmd() *cobra.Command {
	uiArgs := &uninstallArgs{}
	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall higress from a cluster",
		Long:  "The uninstall command uninstalls higress from a cluster",
		Example: `  # Uninstall higress 
  hgctl uninstall 

  # Uninstall higress by special namespace
  hgctl uninstall --namespace=higress-system
  
  # Uninstall higress and istio CRD
  hgctl uninstall --purge-istio-crd  --istio-namespace=istio-system`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstall(cmd.OutOrStdout(), uiArgs)
		},
	}
	addUninstallFlags(uninstallCmd, uiArgs)
	flags := uninstallCmd.Flags()
	options.AddKubeConfigFlags(flags)
	return uninstallCmd
}

// uninstall uninstalls control plane by either pruning by target revision or deleting specified manifests.
func uninstall(writer io.Writer, uiArgs *uninstallArgs) error {
	if !promptUninstall(writer) {
		return nil
	}

	fmt.Fprintf(writer, "start to uninstall higress\n")
	setFlags := make([]string, 0)
	profileName := helm.GetUninstallProfileName()
	fmt.Fprintf(writer, "start to uninstall higress profile:%s\n", profileName)
	_, profile, err := helm.GenProfile(profileName, "", setFlags)
	if err != nil {
		return err
	}
	profile.Global.EnableIstioAPI = uiArgs.purgeIstioCRD
	profile.Global.Namespace = uiArgs.namespace
	profile.Global.IstioNamespace = uiArgs.istioNamespace
	err = UnInstallManifests(profile, writer)
	if err != nil {
		return err
	}
	return nil
}

func promptUninstall(writer io.Writer) bool {
	answer := ""
	fmt.Fprintf(writer, "Are you sure to uninstall higress?\n")
	for {
		fmt.Fprintf(writer, "Please input yes or no to select, input your selection:")
		fmt.Scanln(&answer)
		if strings.TrimSpace(answer) == "yes" {
			return true
		}
		if strings.TrimSpace(answer) == "no" {
			return false
		}
	}

	return false
}

func UnInstallManifests(profile *helm.Profile, writer io.Writer) error {
	fmt.Fprintf(writer, "start to check kubernetes cluster enviroment ......\n")
	cliClient, err := kubernetes.NewCLIClient(options.DefaultConfigFlags.ToRawKubeConfigLoader())
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client: %w", err)
	}
	fmt.Fprintf(writer, "start to init higress installer ......\n")
	op, err := installer.NewInstaller(profile, cliClient, writer)
	if err != nil {
		return err
	}
	fmt.Fprintf(writer, "start to run higress installer ......\n")
	if err := op.Run(); err != nil {
		return err
	}
	fmt.Fprintf(writer, "start to render manifests ......\n")
	manifestMap, err := op.RenderManifests()
	if err != nil {
		return err
	}
	fmt.Fprintf(writer, "start to delete manifests ......\n")
	if err := op.DeleteManifests(manifestMap); err != nil {
		return err
	}
	fmt.Fprintf(writer, "uninstall higress complete!\n")
	return nil
}
