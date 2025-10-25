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
