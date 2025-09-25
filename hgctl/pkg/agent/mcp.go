package agent

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type MCPType string

const (
	HTTP    string = "http"
	OPENAPI string = "OPENAPI"
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
	var spec string
	var noPublish bool

	cmd := &cobra.Command{
		Use:   "add",
		Short: "to add mcp server including http and openapi",
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(handleAddMCP(args, cmd.OutOrStdout(), typ, spec, noPublish))
		},
	}

	cmd.PersistentFlags().StringVarP(&typ, "type", "t", HTTP, "Determine the MCP Server's Type")
	cmd.PersistentFlags().StringVarP(&spec, "spec", "s", "", "Specification of the openapi api")
	cmd.PersistentFlags().BoolVar(&noPublish, "no-publish", false, "If set then the mcp server will not be plubished to higress")
	return cmd
}

func addHTTPMCP(args []string, w io.Writer, noPublish bool) error {
	// all we need to do is use the kode mcp functionality

	fmt.Println(args)

	return nil

}

func addOpenAPIMCP(args []string, w io.Writer, spec string) error {
	fmt.Println(args)
	return nil

}

func handleAddMCP(args []string, w io.Writer, typ string, spec string, noPublish bool) error {
	// spec -> OPENAPI
	// noPublish -> typ
	switch typ {
	case HTTP:
		return addHTTPMCP(args, w, noPublish)
	case OPENAPI:
		if spec == "" {
			return fmt.Errorf("--spec is required for openapi type")
		}
		if noPublish {
			return fmt.Errorf("--no-publish is not supported for openapi type")

		}
		return addOpenAPIMCP(args, w, spec)
	default:
		return fmt.Errorf("unsupported mcp type")
	}
}
