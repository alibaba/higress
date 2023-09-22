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
	"sort"

	"github.com/alibaba/higress/pkg/cmd/hgctl/helm"
	"github.com/spf13/cobra"
)

type profileListArgs struct {
	// manifestsPath is a path to a charts and profiles directory in the local filesystem with a release tgz.
	manifestsPath string
}

func addProfileListFlags(cmd *cobra.Command, args *profileListArgs) {
	cmd.PersistentFlags().StringVarP(&args.manifestsPath, "manifests", "d", "", manifestsFlagHelpStr)
}

func profileListCmd(plArgs *profileListArgs) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lists available higress configuration profiles",
		Long:  "The list subcommand lists the available higress configuration profiles.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return profileList(cmd, plArgs)
		},
	}
}

// profileList list all the builtin profiles.
func profileList(cmd *cobra.Command, plArgs *profileListArgs) error {

	profiles, err := helm.ListProfiles(plArgs.manifestsPath)
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		cmd.Println("No profiles available.")
	} else {
		cmd.Println("higress configuration profiles:")
		sort.Strings(profiles)
		for _, profile := range profiles {
			cmd.Printf("    %s\n", profile)
		}
	}

	return nil
}
