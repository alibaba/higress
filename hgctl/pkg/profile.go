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
	"github.com/spf13/cobra"
)

// ProfileCmd is a group of commands related to profile listing, dumping and diffing.
func newProfileCmd() *cobra.Command {
	pc := &cobra.Command{
		Use:   "profile",
		Short: "Commands related to higress configuration profiles",
		Long:  "The profile command lists, dumps higress configuration profiles.",
		Example: "hgctl profile list\n" +
			"hgctl install --set profile=local-k8s  # Use a profile from the list",
	}

	pdArgs := &profileDumpArgs{}
	plArgs := &profileListArgs{}

	plc := profileListCmd(plArgs)
	pdc := profileDumpCmd(pdArgs)

	addProfileDumpFlags(pdc, pdArgs)
	addProfileListFlags(plc, plArgs)

	pc.AddCommand(plc)
	pc.AddCommand(pdc)

	return pc
}
