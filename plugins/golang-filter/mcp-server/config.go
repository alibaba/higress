package main

import (
	"fmt"

	"net/http"
	_ "net/http/pprof"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/handler"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/internal"
	_ "github.com/alibaba/higress/plugins/golang-filter/mcp-server/registry/nacos"
	_ "github.com/alibaba/higress/plugins/golang-filter/mcp-server/servers/gorm"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	envoyHttp "github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
)

const Name = "mcp-session"
const Version = "1.0.0"
const DefaultServerName = "defaultServer"
const ConfigPathSuffix = "/config"

func init() {
	envoyHttp.RegisterHttpFilterFactoryAndConfigParser(Name, filterFactory, &parser{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				api.LogErrorf("Pprof server recovered from panic: %v", r)
			}
		}()
		api.LogError(http.ListenAndServe("localhost:6060", nil).Error())
	}()
}

type config struct {
	ssePathSuffix         string
	redisClient           *internal.RedisClient
	servers               []*internal.SSEServer
	defaultServer         *internal.SSEServer
	matchList             []internal.MatchRule
	enableUserLevelServer bool
}

func (c *config) Destroy() {
	if c.redisClient != nil {
		api.LogDebug("Closing Redis client")
		c.redisClient.Close()
	}
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

	conf := &config{
		matchList: make([]internal.MatchRule, 0),
		servers:   make([]*internal.SSEServer, 0),
	}

	// Parse match_list if exists
	if matchList, ok := v.AsMap()["match_list"].([]interface{}); ok {
		for _, item := range matchList {
			if ruleMap, ok := item.(map[string]interface{}); ok {
				rule := internal.MatchRule{}
				if domain, ok := ruleMap["match_rule_domain"].(string); ok {
					rule.MatchRuleDomain = domain
				}
				if path, ok := ruleMap["match_rule_path"].(string); ok {
					rule.MatchRulePath = path
				}
				if ruleType, ok := ruleMap["match_rule_type"].(string); ok {
					rule.MatchRuleType = internal.RuleType(ruleType)
				}
				conf.matchList = append(conf.matchList, rule)
			}
		}
	}

	// Redis configuration is optional
	if redisConfigMap, ok := v.AsMap()["redis"].(map[string]interface{}); ok {
		redisConfig, err := internal.ParseRedisConfig(redisConfigMap)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis config: %w", err)
		}

		redisClient, err := internal.NewRedisClient(redisConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize RedisClient: %w", err)
		}
		conf.redisClient = redisClient
		api.LogDebug("Redis client initialized")
	} else {
		api.LogDebug("Redis configuration not provided, running without Redis")
	}

	enableUserLevelServer, ok := v.AsMap()["enable_user_level_server"].(bool)
	if !ok {
		enableUserLevelServer = false
		if conf.redisClient == nil {
			return nil, fmt.Errorf("redis configuration is not provided, enable_user_level_server is true")
		}
	}
	conf.enableUserLevelServer = enableUserLevelServer

	ssePathSuffix, ok := v.AsMap()["sse_path_suffix"].(string)
	if !ok || ssePathSuffix == "" {
		return nil, fmt.Errorf("sse path suffix is not set or empty")
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

		err := server.ParseConfig(serverConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse server config: %w", err)
		}

		serverInstance, err := server.NewServer(serverName)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize DBServer: %w", err)
		}

		conf.servers = append(conf.servers, internal.NewSSEServer(serverInstance,
			internal.WithRedisClient(conf.redisClient),
			internal.WithSSEEndpoint(fmt.Sprintf("%s%s", serverPath, ssePathSuffix)),
			internal.WithMessageEndpoint(serverPath)))
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
		newConfig.servers = childConfig.servers
	}
	if childConfig.defaultServer != nil {
		newConfig.defaultServer = childConfig.defaultServer
	}
	if childConfig.matchList != nil {
		newConfig.matchList = childConfig.matchList
	}
	return &newConfig
}

func filterFactory(c interface{}, callbacks api.FilterCallbackHandler) api.StreamFilter {
	conf, ok := c.(*config)
	if !ok {
		panic("unexpected config type")
	}
	return &filter{
		callbacks:        callbacks,
		config:           conf,
		stopChan:         make(chan struct{}),
		mcpConfigHandler: handler.NewMCPConfigHandler(conf.redisClient, callbacks),
	}
}

func main() {}
