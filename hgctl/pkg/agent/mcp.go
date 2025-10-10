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
	OPENAPI string = "openapi"
)

type MCPAddArg struct {
	name      string
	url       string
	typ       string
	spec      string
	scope     string
	noPublish bool
	// TODO: support mcp env
	// env string
}

var KClient = NewKodeClient("")

func NewMCPCmd() *cobra.Command {
	setup()
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "for the mcp management",
	}

	mcpCmd.AddCommand(newMCPAddCmd())

	return mcpCmd
}

func newMCPAddCmd() *cobra.Command {
	// parameter
	arg := &MCPAddArg{}

	cmd := &cobra.Command{
		Use:   "add [name]",
		Short: "add mcp server including http and openapi",
		Run: func(cmd *cobra.Command, args []string) {
			arg.name = args[0]
			cmdutil.CheckErr(handleAddMCP(cmd.OutOrStdout(), *arg))
		},
		Args: cobra.ExactArgs(1),
	}

	cmd.PersistentFlags().StringVarP(&arg.typ, "type", "t", HTTP, "Determine the MCP Server's Type")
	cmd.PersistentFlags().StringVarP(&arg.url, "url", "u", "", "MCP server URL")
	cmd.PersistentFlags().StringVarP(&arg.scope, "scope", "s", "project", `Configuration scope (project or global)`)
	cmd.PersistentFlags().StringVar(&arg.spec, "spec", "", "Specification of the openapi api")
	cmd.PersistentFlags().BoolVar(&arg.noPublish, "no-publish", false, "If set then the mcp server will not be plubished to higress")
	return cmd
}

func addHTTPMCP(w io.Writer, arg MCPAddArg) error {
	if arg.noPublish {
		fmt.Printf("%s is set to be noPublish\n", arg.name)
	}

	if err := KClient.AddMCPServer(arg.name, arg.url); err != nil {
		return fmt.Errorf("mcp add failed: %w", err)
	}

	// TODO: Publish to higress

	return nil

}

func addOpenAPIMCP(w io.Writer, arg MCPAddArg) error {
	fmt.Printf("get mcp server %s spec %s\n", arg.name, arg.spec)
	// TODO: OpenAPI transfer
	return nil
}

func handleAddMCP(w io.Writer, arg MCPAddArg) error {
	// spec -> OPENAPI
	// noPublish -> typ
	switch arg.typ {
	case HTTP:
		return addHTTPMCP(w, arg)
	case OPENAPI:
		if arg.spec == "" {
			return fmt.Errorf("--spec is required for openapi type")
		}
		if arg.noPublish {
			return fmt.Errorf("--no-publish is not supported for openapi type")
		}
		if arg.url != "" {
			return fmt.Errorf("--url is not supported for openapi type")
		}
		return addOpenAPIMCP(w, arg)
	default:
		return fmt.Errorf("unsupported mcp type")
	}
}
