package agent

import (
	"fmt"
	"io"
	"net"
	"net/url"

	"github.com/alibaba/higress/hgctl/pkg/agent/services"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type MCPType string

const (
	HTTP    string = "http"
	OPENAPI string = "openapi"
)

type MCPAddArg struct {
	// higress console auth arg
	baseURL  string
	username string
	password string

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

	cmd.PersistentFlags().StringVar(&arg.baseURL, "base-url", "", "The BaseURL of higress console")
	cmd.PersistentFlags().StringVar(&arg.username, "user", "", "The username of higress console")
	cmd.PersistentFlags().StringVarP(&arg.password, "password", "p", "", "The password of higress console")
	return cmd
}

func newHanlder(c *KodeClient, arg MCPAddArg, w io.Writer) *MCPAddHandler {
	return &MCPAddHandler{c, arg, w}
}

func (h *MCPAddHandler) validateArg() error {
	if !h.arg.noPublish {
		if h.arg.baseURL == "" || h.arg.username == "" || h.arg.password == "" {
			fmt.Println("--username, --base-url, --password must be provided")
			return fmt.Errorf("invalid args")
		}
	}
	return nil

}

func (h *MCPAddHandler) addHTTPMCP() error {

	if err := h.c.AddMCPServer(h.arg.name, h.arg.url); err != nil {
		return fmt.Errorf("mcp add failed: %w", err)
	}

	if !h.arg.noPublish {
		fmt.Printf("%s is set to not be noPublish\n", h.arg.name)
		return publishToHigress(h.arg, nil)
	}
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
	if err := h.validateArg(); err != nil {
		return err
	}

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

func publishToHigress(arg MCPAddArg, config interface{}) error {
	client := services.NewHigressClient(arg.baseURL, arg.username, arg.password)
	// add service
	// handle the url
	res, err := url.Parse(arg.url)
	if err != nil {
		return err
	}

	srvType := ""
	srvPort := ""
	srvName := fmt.Sprintf("hgctl-%s", arg.name)
	srvPath := res.Path

	if ip := net.ParseIP(res.Hostname()); ip == nil {
		srvType = "dns"
	} else {
		srvType = "static"
	}

	if res.Port() == "" && res.Scheme == "http" {
		srvPort = "80"
	} else if res.Port() == "" && res.Scheme == "https" {
		srvPort = "443"
	} else {
		srvPort = res.Port()
	}

	_, err = services.HandleAddServiceSource(client, map[string]interface{}{
		"domain":        res.Host,
		"type":          srvType,
		"port":          srvPort,
		"name":          srvName,
		"domainForEdit": res.Host,
		"protocol":      res.Scheme,
	})
	if err != nil {
		return err
	}

	resp, err := services.HandleAddMCPServer(client, map[string]interface{}{
		"name": arg.name,
		//   "description": "",
		"type":               "DIRECT_ROUTE",
		"service":            fmt.Sprintf("%s.%s:%s", srvName, srvType, srvPort),
		"upstreamPathPrefix": srvPath,
		"services": []map[string]interface{}{{
			"name":    srvName,
			"port":    srvPort,
			"version": "1.0",
			"weight":  100,
		}},
	})
	if err != nil {
		return err
	}
	fmt.Printf("%v", resp)

	// return client.CreateMCPServer()
	return nil
}
