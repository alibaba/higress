package main

import (
	"errors"
	"fmt"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	envoyHttp "github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
	"github.com/envoyproxy/envoy/examples/golang-http/simple/internal"
	"github.com/envoyproxy/envoy/examples/golang-http/simple/servers/gorm"
)

const Name = "mcp-server"
const SCHEME_PATH = "scheme"

func init() {
	envoyHttp.RegisterHttpFilterFactoryAndConfigParser(Name, filterFactory, &parser{})
}

type config struct {
	echoBody string
	// other fields
	dbClient    *gorm.DBClient
	redisClient *internal.RedisClient
	stopChan    chan struct{}
	SSEServer   *internal.SSEServer
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

	dsn, ok := v.AsMap()["dsn"].(string)
	if !ok {
		return nil, errors.New("missing dsn")
	}

	dbType, ok := v.AsMap()["dbType"].(string)
	if !ok {
		return nil, errors.New("missing database type")
	}

	dbClient, err := gorm.NewDBClient(dsn, dbType)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DBClient: %w", err)
	}
	conf.dbClient = dbClient

	conf.stopChan = make(chan struct{})
	redisClient, err := internal.NewRedisClient("localhost:6379", conf.stopChan)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RedisClient: %w", err)
	}
	conf.redisClient = redisClient

	conf.SSEServer = internal.NewSSEServer(NewServer(conf.dbClient), internal.WithRedisClient(redisClient))
	return conf, nil
}

func (p *parser) Merge(parent interface{}, child interface{}) interface{} {
	parentConfig := parent.(*config)
	childConfig := child.(*config)

	newConfig := *parentConfig
	if childConfig.echoBody != "" {
		newConfig.echoBody = childConfig.echoBody
	}
	if childConfig.dbClient != nil {
		newConfig.dbClient = childConfig.dbClient
	}
	if childConfig.redisClient != nil {
		newConfig.redisClient = childConfig.redisClient
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

func NewServer(dbClient *gorm.DBClient) *internal.MCPServer {
	mcpServer := internal.NewMCPServer(
		"mcp-server-envoy-poc",
		"1.0.0",
	)

	// Add query tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("query", "Run a read-only SQL query in clickhouse database with repository git data", gorm.GetQueryToolSchema()),
		gorm.HandleQueryTool(dbClient),
	)
	api.LogInfo("Added query tool successfully")

	// Add favorite files tool
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("author_favorite_files", "Favorite files for an author", gorm.GetFavoriteToolSchema()),
		gorm.HandleFavoriteTool(dbClient),
	)
	return mcpServer
}

func main() {}
