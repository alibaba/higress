package main

import (
	"fmt"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/internal"
	_ "github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/gorm"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	envoyHttp "github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
)

const Name = "mcp-server"
const Version = "1.0.0"
const DefaultServerName = "defaultServer"
const MessageEndpoint = "/message"

func init() {
	envoyHttp.RegisterHttpFilterFactoryAndConfigParser(Name, filterFactory, &parser{})
}

type config struct {
	ssePathSuffix string
	redisClient   *internal.RedisClient
	stopChan      chan struct{}
	servers       []*internal.SSEServer
	defaultServer *internal.SSEServer
}

type parser struct {
}

// Parse the filter configuration
func (p *parser) Parse(any *anypb.Any, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	configStruct := &xds.TypedStruct{}
	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}
	v := configStruct.Value

	conf := &config{}
	conf.stopChan = make(chan struct{})

	redisConfigMap, ok := v.AsMap()["redis"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("redis config is not set")
	}

	redisConfig, err := internal.ParseRedisConfig(redisConfigMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis config: %w", err)
	}

	redisClient, err := internal.NewRedisClient(redisConfig, conf.stopChan)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RedisClient: %w", err)
	}
	conf.redisClient = redisClient

	ssePathSuffix, ok := v.AsMap()["sse_path_suffix"].(string)
	if !ok {
		return nil, fmt.Errorf("sse path suffix is not set")
	}
	conf.ssePathSuffix = ssePathSuffix

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
		serverName, ok := serverConfigMap["name"].(string)
		if !ok {
			return nil, fmt.Errorf("server %s name is not set", serverType)
		}
		server := internal.GlobalRegistry.GetServer(serverType)

		if server == nil {
			return nil, fmt.Errorf("server %s is not registered", serverType)
		}
		serverConfig, ok := serverConfigMap["config"].(map[string]interface{})
		if !ok {
			api.LogDebug(fmt.Sprintf("No config provided for server %s", serverType))
		}
		api.LogDebug(fmt.Sprintf("Server config: %+v", serverConfig))

		err = server.ParseConfig(serverConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse server config: %w", err)
		}

		serverInstance, err := server.NewServer(serverName)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize DBServer: %w", err)
		}

		conf.servers = append(conf.servers, internal.NewSSEServer(serverInstance,
			internal.WithRedisClient(redisClient),
			internal.WithSSEEndpoint(fmt.Sprintf("%s%s", serverPath, ssePathSuffix)),
			internal.WithMessageEndpoint(fmt.Sprintf("%s%s", serverPath, MessageEndpoint))))
		api.LogDebug(fmt.Sprintf("Registered MCP Server: %s", serverType))
	}
	return conf, nil
}

func (p *parser) Merge(parent interface{}, child interface{}) interface{} {
	parentConfig := parent.(*config)
	childConfig := child.(*config)

	newConfig := *parentConfig
	if childConfig.redisClient != nil {
		newConfig.redisClient = childConfig.redisClient
	}
	if childConfig.ssePathSuffix != "" {
		newConfig.ssePathSuffix = childConfig.ssePathSuffix
	}
	if childConfig.servers != nil {
		newConfig.servers = append(newConfig.servers, childConfig.servers...)
	}
	if childConfig.defaultServer != nil {
		newConfig.defaultServer = childConfig.defaultServer
	}
	return &newConfig
}

func filterFactory(c interface{}, callbacks api.FilterCallbackHandler) api.StreamFilter {
	conf, ok := c.(*config)
	if !ok {
		panic("unexpected config type")
	}
	return &filter{
		callbacks: callbacks,
		config:    conf,
	}
}

func main() {}
