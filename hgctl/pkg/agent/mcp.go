package agent

import (
	"fmt"
	"io"

	// "github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress"
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

type MCPAddHandler struct {
	c   *KodeClient
	arg MCPAddArg
	w   io.Writer
}

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

func newHanlder(c *KodeClient, arg MCPAddArg, w io.Writer) *MCPAddHandler {
	return &MCPAddHandler{
		c,
		arg,
		w,
	}
}

func (h *MCPAddHandler) addHTTPMCP() error {
	if h.arg.noPublish {
		fmt.Printf("%s is set to be noPublish\n", h.arg.name)
	}

	if err := h.c.AddMCPServer(h.arg.name, h.arg.url); err != nil {
		return fmt.Errorf("mcp add failed: %w", err)
	}

	// TODO: Publish to higress
	publishToHigress(h.arg.name, h.arg.url, "http", nil)

	return nil

}

func (h *MCPAddHandler) addOpenAPIMCP() error {
	fmt.Printf("get mcp server %s spec %s\n", h.arg.name, h.arg.spec)
	// TODO: OpenAPI transfer
	return nil
}

func handleAddMCP(w io.Writer, arg MCPAddArg) error {
	client := getClient()
	h := newHanlder(client, arg, w)
	// spec -> OPENAPI
	// noPublish -> typ
	switch arg.typ {
	case HTTP:
		return h.addHTTPMCP()
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
		return h.addOpenAPIMCP()
	default:
		return fmt.Errorf("unsupported mcp type")
	}
}

func publishToHigress(name, url, serverType string, config interface{}) error {

	// add service
	respBody, err := client.Post("/v1/service-sources")

	// add route
	// add MCP

	// mcpServer := &MCPServerConfig{
	// 	Name:   name,
	// 	Type:   serverType,
	// 	URL:    url,
	// 	Config: config,
	// }

	return client.CreateMCPServer()
}
