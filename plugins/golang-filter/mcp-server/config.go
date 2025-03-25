package main

import (
	"fmt"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/common"
	_ "github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/gorm"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	envoyHttp "github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
)

const Name = "mcp-server"

func init() {
	envoyHttp.RegisterHttpFilterFactoryAndConfigParser(Name, filterFactory, &parser{})
}

type config struct {
	ssePathSuffix string
	redisClient   *common.RedisClient
	stopChan      chan struct{}
	servers       []*common.SSEServer
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

	redisConfig, err := common.ParseRedisConfig(redisConfigMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis config: %w", err)
	}

	redisClient, err := common.NewRedisClient(redisConfig, conf.stopChan)
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
		api.LogInfo("No servers are configured")
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
		server := common.GlobalRegistry.GetServer(serverType)

		if server == nil {
			return nil, fmt.Errorf("server %s is not registered", serverType)
		}
		server.ParseConfig(serverConfigMap)
		serverInstance, err := server.NewServer()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize DBServer: %w", err)
		}
		conf.servers = append(conf.servers, common.NewSSEServer(serverInstance,
			common.WithRedisClient(redisClient),
			common.WithSSEEndpoint(fmt.Sprintf("%s%s", serverPath, ssePathSuffix)),
			common.WithMessageEndpoint(serverPath)))
		api.LogInfo(fmt.Sprintf("Registered MCP Server: %s", serverType))
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
	if childConfig.ssePathSuffix != "" {
		newConfig.ssePathSuffix = childConfig.ssePathSuffix
	}
	if childConfig.servers != nil {
		newConfig.servers = append(newConfig.servers, childConfig.servers...)
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
