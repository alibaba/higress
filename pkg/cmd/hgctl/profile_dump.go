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
	"os"

	"github.com/alibaba/higress/pkg/cmd/hgctl/helm"
	"github.com/spf13/cobra"
)

type profileDumpArgs struct {
	// output write profile to file
	output string
	// manifestsPath is a path to a charts and profiles directory in the local filesystem with a release tgz.
	manifestsPath string
}

func addProfileDumpFlags(cmd *cobra.Command, args *profileDumpArgs) {
	cmd.PersistentFlags().StringVarP(&args.output, "output", "o", "", outputHelpstr)
	cmd.PersistentFlags().StringVarP(&args.manifestsPath, "manifests", "d", "", manifestsFlagHelpStr)
}

func profileDumpCmd(pdArgs *profileDumpArgs) *cobra.Command {
	return &cobra.Command{
		Use:   "dump [<profile>]",
		Short: "Dumps a higress configuration profile",
		Long:  "The dump subcommand dumps the values in a higress configuration profile.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("too many positional arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return profileDump(cmd, args, pdArgs)
		},
	}
}

func profileDump(cmd *cobra.Command, args []string, pdArgs *profileDumpArgs) error {
	profileName := helm.DefaultProfileName
	if len(args) == 1 {
		profileName = args[0]
	}
	yaml, err := helm.ReadProfileYAML(profileName, pdArgs.manifestsPath)
	if err != nil {
		return err
	}
	if len(pdArgs.output) > 0 {
		err2 := os.WriteFile(pdArgs.output, []byte(yaml), 0644)
		if err2 != nil {
			return err2
		}
	} else {
		cmd.Println(yaml)
	}
	return nil
}
