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
	"fmt"
	"io"
	"net"
	"net/url"
	"os"

	"github.com/alibaba/higress/hgctl/pkg/agent/services"
	"github.com/fatih/color"
	"github.com/higress-group/openapi-to-mcpserver/pkg/models"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type MCPType string

const (
	HTTP         string = "http"
	SSE          string = "sse"
	OPENAPI      string = "openapi"
	DIRECT_ROUTE string = "DIRECT_ROUTE"
	OPEN_API     string = "OPEN_API"
)

type MCPAddArg struct {
	// higress console auth arg
	baseURL  string
	username string
	password string

	name      string
	url       string
	transport string
	spec      string
	scope     string
	noPublish bool
	// TODO: support mcp env
	// env string

}

type MCPAddHandler struct {
	core *AgenticCore
	arg  MCPAddArg
	w    io.Writer
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
			color.Cyan("Tip: Try doing 'kubectl port-forward' and add the server to the agent manually, if MCP Server connection failed")
		},
		Args: cobra.ExactArgs(1),
	}

	cmd.PersistentFlags().StringVarP(&arg.transport, "transport", "t", HTTP, "Determine the MCP Server's Type")
	cmd.PersistentFlags().StringVarP(&arg.url, "url", "u", "", "MCP server URL")
	cmd.PersistentFlags().StringVarP(&arg.scope, "scope", "s", "project", `Configuration scope (project or global)`)
	cmd.PersistentFlags().StringVar(&arg.spec, "spec", "", "Specification of the openapi api")
	cmd.PersistentFlags().BoolVar(&arg.noPublish, "no-publish", false, "If set then the mcp server will not be plubished to higress")

	cmd.PersistentFlags().StringVar(&arg.baseURL, "base-url", "", "The BaseURL of higress console")
	cmd.PersistentFlags().StringVar(&arg.username, "user", "", "The username of higress console")
	cmd.PersistentFlags().StringVarP(&arg.password, "password", "p", "", "The password of higress console")
	return cmd
}

func newHanlder(c *AgenticCore, arg MCPAddArg, w io.Writer) *MCPAddHandler {
	return &MCPAddHandler{c, arg, w}
}

func (h *MCPAddHandler) validateArg() error {
	if !h.arg.noPublish {
		if h.arg.baseURL == "" || h.arg.username == "" || h.arg.password == "" {
			fmt.Println("--user, --base-url, --password must be provided")
			return fmt.Errorf("invalid args")
		}
	}
	return nil

}

func (h *MCPAddHandler) addHTTPMCP() error {
	if err := h.core.AddMCPServer(h.arg.name, h.arg.url); err != nil {
		return fmt.Errorf("mcp add failed: %w", err)
	}

	if !h.arg.noPublish {
		fmt.Printf("%s is set to not be noPublish\n", h.arg.name)
		return publishToHigress(h.arg, nil)
	}
	return nil

}

// hgctl mcp add -t openapi --name test-name --spec openapi.json
func (h *MCPAddHandler) addOpenAPIMCP() error {
	// fmt.Printf("get mcp server: %s openapi-spec-file: %s\n", h.arg.name, h.arg.spec)
	config := h.parseOpenapiSpec()

	// fmt.Printf("get config struct: %v", config)

	// publish to higress
	if err := publishToHigress(h.arg, config); err != nil {
		return err
	}

	// add mcp server to agent
	gatewayIP, err := GetHigressGatewayServiceIP()
	if err != nil {
		color.Red(
			"failed to add mcp server [%s] while getting higress-gateway ip due to: %v \n You may try to do port-forward and add it to agent manually", h.arg.name, err)
		return err
	}
	mcpURL := fmt.Sprintf("http://%s/mcp-servers/%s", gatewayIP, h.arg.name)
	return h.core.AddMCPServer(h.arg.name, mcpURL)
}

func (h *MCPAddHandler) parseOpenapiSpec() *models.MCPConfig {
	return parseOpenapi2MCP(h.arg)
}

func handleAddMCP(w io.Writer, arg MCPAddArg) error {
	client := getAgent()
	h := newHanlder(client, arg, w)
	if err := h.validateArg(); err != nil {
		return err
	}

	// spec -> OPENAPI
	// noPublish -> typ
	switch arg.transport {
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

func publishToHigress(arg MCPAddArg, config *models.MCPConfig) error {
	// 1. parse the raw http url
	// 2. add service source
	// 3. add MCP server request
	client := services.NewHigressClient(arg.baseURL, arg.username, arg.password)

	// mcp server's url
	rawURL := arg.url
	// DIRECT_ROUTE or OPEN_API
	mcpType := DIRECT_ROUTE

	if config != nil {
		// TODO: here use tools's url directly, need to be considered
		rawURL = config.Tools[0].RequestTemplate.URL
		mcpType = OPEN_API
	}

	res, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	// add service source
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

	srvField := []map[string]interface{}{{
		"name":    fmt.Sprintf("%s.%s", srvName, srvType),
		"port":    srvPort,
		"version": "1.0",
		"weight":  100,
	}}

	// generete mcp server add request body
	body := map[string]interface{}{
		"name": arg.name,
		//   "description": "",
		"type":               mcpType,
		"service":            fmt.Sprintf("%s.%s:%s", srvName, srvType, srvPort),
		"upstreamPathPrefix": srvPath,
		"services":           srvField,
	}

	// fmt.Printf("request body: %v", body)

	_, err = services.HandleAddMCPServer(client, body)
	if err != nil {
		return err
	}

	if mcpType == OPEN_API {
		addMCPToolConfig(client, config, srvField)
	}

	return nil
}

func addMCPToolConfig(client *services.HigressClient, config *models.MCPConfig, srvField []map[string]interface{}) {
	body := map[string]interface{}{
		"name": config.Server.Name,
		//	  "description": "",
		"services":          srvField,
		"type":              OPEN_API,
		"rawConfigurations": convertMCPConfigToStr(config),
		"mcpServerName":     config.Server.Name,
	}

	_, err := services.HandleAddOpenAPITool(client, body)
	if err != nil {
		fmt.Printf("add openapi tools failed: %v\n", err)
		os.Exit(1)
	}
	// fmt.Println("get openapi tools add response: ", string(resp))
}
