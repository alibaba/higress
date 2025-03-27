package registry

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/internal"
	"github.com/mark3labs/mcp-go/mcp"
)

const HTTP_URL_TEMPLATE = "%s://%s:%d%s"
const FIX_QUERY_TOKEN_KEY = "key"
const FIX_QUERY_TOKEN_VALUE = "value"
const PROTOCOL_HTTP = "http"
const PROTOCOL_HTTPS = "https"
const DEFAULT_HTTP_METHOD = "GET"
const DEFAULT_HTTP_PATH = "/"

func getHttpCredentialHandle(name string) (func(*CredentialInfo, *HttpRemoteCallHandle), error) {
	if name == "fixed-query-token" {
		return FixedQueryToken, nil
	}

	return nil, fmt.Errorf("Unknown credential type")
}

type CommonRemoteCallHandle struct {
	Instance *Instance
}

type HttpRemoteCallHandle struct {
	CommonRemoteCallHandle
	Protocol string
	Headers  http.Header
	Body     *string
	Query    map[string]string
	Path     string
	Method   string
}

// http credentials handles
func FixedQueryToken(cred *CredentialInfo, h *HttpRemoteCallHandle) {
	key, _ := cred.Credentials[FIX_QUERY_TOKEN_KEY]
	value, _ := cred.Credentials[FIX_QUERY_TOKEN_VALUE]
	h.Query[key.(string)] = value.(string)
}

func newHttpRemoteCallHandle(ctx *RpcContext) *HttpRemoteCallHandle {
	instance := selectOneInstance(ctx)
	method, ok := ctx.ToolMeta.InvokeContext["method"]
	if !ok {
		method = DEFAULT_HTTP_METHOD
	}

	path, ok := ctx.ToolMeta.InvokeContext["path"]
	if !ok {
		path = DEFAULT_HTTP_PATH
	}

	return &HttpRemoteCallHandle{
		CommonRemoteCallHandle: CommonRemoteCallHandle{
			Instance: &instance,
		},
		Protocol: ctx.Protocol,
		Headers:  http.Header{},
		Body:     nil,
		Query:    map[string]string{},
		Path:     path,
		Method:   method,
	}
}

// http remote handle implementation
func (h *HttpRemoteCallHandle) HandleToolCall(ctx *RpcContext, parameters map[string]any) (*mcp.CallToolResult, error) {
	if ctx.Credential != nil {
		credentialHandle, err := getHttpCredentialHandle(ctx.Credential.CredentialType)
		if err != nil {
			return nil, err
		}
		credentialHandle(ctx.Credential, h)
	}

	err := h.handleParamMapping(&ctx.ToolMeta.ParametersMapping, parameters)
	if err != nil {
		return nil, err
	}

	response, err := h.doHttpCall()
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	responseType := "text"
	if respType, ok := ctx.ToolMeta.InvokeContext["responseType"]; ok {
		responseType = respType
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: responseType,
				Text: string(body),
			},
		},
	}, nil
}

func (h *HttpRemoteCallHandle) handleParamMapping(mapInfo *map[string]ParameterMapInfo, params map[string]any) error {
	paramMapInfo := *mapInfo
	for param, value := range params {
		if info, ok := paramMapInfo[param]; ok {
			if info.Position == "Query" {
				h.Query[info.BackendName] = fmt.Sprintf("%s", value)
			} else if info.Position == "Header" {
				h.Headers[info.BackendName] = []string{fmt.Sprintf("%s", value)}
			} else {
				return fmt.Errorf("Unsupport position for args %s, pos is %s", param, info.Position)
			}
		} else {
			h.Query[param] = fmt.Sprintf("%s", value)
		}
	}
	return nil
}

func (h *HttpRemoteCallHandle) doHttpCall() (*http.Response, error) {
	pathPrefix := fmt.Sprintf(HTTP_URL_TEMPLATE, h.Protocol, h.Instance.Host, h.Instance.Port, h.Path)
	queryString := ""
	queryGroup := []string{}
	for queryKey, queryValue := range h.Query {
		queryGroup = append(queryGroup, url.QueryEscape(queryKey)+"="+url.QueryEscape(queryValue))
	}

	if len(queryGroup) > 0 {
		queryString = "?" + strings.Join(queryGroup, "&")
	}
	fullUrl, err := url.Parse(pathPrefix + queryString)
	if err != nil {
		return nil, fmt.Errorf("Parse url error , url is %s", pathPrefix+queryString)
	}
	request := http.Request{
		URL:    fullUrl,
		Method: h.Method,
		Header: h.Headers,
	}

	if h.Body != nil {
		request.Body = io.NopCloser(strings.NewReader(*h.Body))
	}

	return http.DefaultClient.Do(&request)
}

func selectOneInstance(ctx *RpcContext) Instance {
	instanceId := 0
	instances := *ctx.Instances
	if len(instances) != 1 {
		instanceId = rand.Intn(len(instances) - 1)
	}
	return instances[instanceId]
}

func getRemoteCallhandle(ctx *RpcContext) RemoteCallHandle {
	if ctx.Protocol == PROTOCOL_HTTP || ctx.Protocol == PROTOCOL_HTTPS {
		return newHttpRemoteCallHandle(ctx)
	} else {
		return nil
	}
}

// common remote call process
func CommonRemoteCall(reg McpServerRegistry, toolName string, parameters map[string]any) (*mcp.CallToolResult, error) {
	ctx, ok := reg.GetToolRpcContext(toolName)
	if !ok {
		return nil, fmt.Errorf("Unknown tool %s", toolName)
	}

	remoteHandle := getRemoteCallhandle(ctx)
	if remoteHandle == nil {
		return nil, fmt.Errorf("Unknown backend protocol %s", ctx.Protocol)
	}

	return remoteHandle.HandleToolCall(ctx, parameters)
}

func HandleRegistryToolsCall(reg McpServerRegistry) internal.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.Params.Arguments
		return CommonRemoteCall(reg, request.Params.Name, arguments)
	}
}
