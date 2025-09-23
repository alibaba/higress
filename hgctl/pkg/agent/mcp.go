package agent

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type MCPType string

const (
	HTTPTyp   string = "http"
	OPENAITyp string = "openai"
)

func NewMCPCmd() *cobra.Command {
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "for the mcp management",
	}

	mcpCmd.AddCommand(newMCPAddCmd())

	return mcpCmd
}

func newMCPAddCmd() *cobra.Command {
	// parameter
	var typ string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "to add mcp server including http and openai",
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(addMCPServer(cmd.OutOrStdout(), typ))
		},
	}

	cmd.PersistentFlags().StringVarP(&typ, "type", "t", HTTPTyp, "Determine the MCP Server's Type")
	return cmd
}

func addMCPServer(w io.Writer, typ string) error {
	fmt.Fprintln(w, "mcp added")
	return nil
}
