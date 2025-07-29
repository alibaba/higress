package mcp_server

import (
	"fmt"

	_ "github.com/alibaba/higress/plugins/golang-filter/mcp-server/registry/nacos"
	_ "github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/gorm"
	_ "github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/higress/higress-api"
	mcp_session "github.com/alibaba/higress/plugins/golang-filter/mcp-session"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"google.golang.org/protobuf/types/known/anypb"
)

const Name = "mcp-server"
const Version = "1.0.0"

type SSEServerWrapper struct {
	BaseServer *common.SSEServer
	DomainList []string
}

type config struct {
	servers []*SSEServerWrapper
}

func (c *config) Destroy() {
	for _, server := range c.servers {
		server.BaseServer.Close()
	}
}

type Parser struct {
}

func (p *Parser) Parse(any *anypb.Any, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	configStruct := &xds.TypedStruct{}
	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}
	v := configStruct.Value

	conf := &config{
		servers: make([]*SSEServerWrapper, 0),
	}

	serverConfigs, ok := v.AsMap()["servers"].([]interface{})
	if !ok {
		api.LogDebug("No servers are configured")
		return conf, nil
	}

	for _, serverConfig := range serverConfigs {
		serverConfigMap, ok := serverConfig.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("server config must be an object")
		}

		serverType, ok := serverConfigMap["type"].(string)
		if !ok {
			return nil, fmt.Errorf("server type is not set")
		}

		serverPath, ok := serverConfigMap["path"].(string)
		if !ok {
			return nil, fmt.Errorf("server %s path is not set", serverType)
		}

		serverDomainList := []string{}
		if domainList, ok := serverConfigMap["domain_list"].([]interface{}); ok {
			for _, domain := range domainList {
				if domainStr, ok := domain.(string); ok {
					serverDomainList = append(serverDomainList, domainStr)
				}
			}
		} else {
			serverDomainList = []string{"*"}
		}

		serverName, ok := serverConfigMap["name"].(string)
		if !ok {
			return nil, fmt.Errorf("server %s name is not set", serverType)
		}
		server := common.GlobalRegistry.GetServer(serverType)

		if server == nil {
			return nil, fmt.Errorf("server %s is not registered", serverType)
		}
		serverConfig, ok := serverConfigMap["config"].(map[string]interface{})
		if !ok {
			api.LogDebug(fmt.Sprintf("No config provided for server %s", serverType))
		}
		api.LogDebug(fmt.Sprintf("Server config: %+v", serverConfig))

		err := server.ParseConfig(serverConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse server config: %w", err)
		}

		serverInstance, err := server.NewServer(serverName)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize MCP Server: %w", err)
		}

		conf.servers = append(conf.servers, &SSEServerWrapper{
			BaseServer: common.NewSSEServer(serverInstance,
				common.WithSSEEndpoint(fmt.Sprintf("%s%s", serverPath, mcp_session.GlobalSSEPathSuffix)),
				common.WithMessageEndpoint(serverPath)),
			DomainList: serverDomainList,
		})
		api.LogDebug(fmt.Sprintf("Registered MCP Server: %s", serverType))
	}

	return conf, nil
}

func (p *Parser) Merge(parent interface{}, child interface{}) interface{} {
	parentConfig := parent.(*config)
	childConfig := child.(*config)

	newConfig := *parentConfig
	if childConfig.servers != nil {
		newConfig.servers = childConfig.servers
	}
	return &newConfig
}

func FilterFactory(c interface{}, callbacks api.FilterCallbackHandler) api.StreamFilter {
	conf, ok := c.(*config)
	if !ok {
		panic("unexpected config type")
	}
	return &filter{
		config:    conf,
		callbacks: callbacks,
	}
}
