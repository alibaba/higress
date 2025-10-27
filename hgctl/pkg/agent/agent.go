// Copyright (c) 2025 Alibaba Group Holding Ltd.
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

package agent

import (
	"io"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func NewAgentCmd() *cobra.Command {
	agentCmd := &cobra.Command{
		Use:   "agent",
		Short: "start the interactive agent window",
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(handleAgentInvoke(cmd.OutOrStdout()))
		},
	}

	return agentCmd
}

func handleAgentInvoke(w io.Writer) error {

	return getAgent().Start()
}

// Sub-Agent1:
// 1. Parse the url provided by user to MCP server configuration.
// 2. Publish the parsed MCP Server to Higress
func addPrequisiteSubAgent() error {
	return nil
}
