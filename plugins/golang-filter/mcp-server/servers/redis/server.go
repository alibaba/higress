package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/mark3labs/mcp-go/mcp"
)

const Version = "1.0.0"

// 在init函数中注册Redis服务器
func init() {
	common.GlobalRegistry.RegisterServer("redis", &RedisConfig{})
}

// RedisConfig 实现 Server 接口
type RedisConfig struct {
	address     string
	username    string
	password    string
	db          int
	description string
	secret      string
}

// ParseConfig 解析配置
func (c *RedisConfig) ParseConfig(config map[string]any) error {
	address, ok := config["address"].(string)
	if !ok {
		return fmt.Errorf("missing address")
	}

	c.address = address

	if username, ok := config["username"].(string); ok {
		c.username = username

	}

	if password, ok := config["password"].(string); ok {
		c.password = password

	}

	if db, ok := config["db"].(float64); ok {
		c.db = int(db)
	}

	if secret, ok := config["secret"].(string); ok {
		c.secret = secret

	}

	if description, ok := config["description"].(string); ok {
		c.description = description

	}

	api.LogDebugf("RedisConfig ParseConfig: %+v", config)
	return nil
}

// NewServer 创建新的Redis MCPServer
func (c *RedisConfig) NewServer(serverName string) (*common.MCPServer, error) {
	mcpServer := common.NewMCPServer(serverName, Version,
		common.WithInstructions(fmt.Sprintf("This is a Redis server with connection to %s", c.address)),
		common.WithToolCapabilities(true),
	)

	// 创建Redis配置
	redisConfig := &common.RedisConfig{

		address: c.address,

		username: c.username,

		password: c.password,

		db: c.db,

		secret: c.secret,
	}

	// 创建Redis客户端
	redisClient, err := common.NewRedisClient(redisConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %w", err)
	}

	// 添加Redis工具
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("get", "Get value from Redis by key", GetSchema()),
		HandleGetTool(redisClient),

	)
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("set", "Set value in Redis with optional expiration", SetSchema()),
		HandleSetTool(redisClient),

	)
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("del", "Delete keys from Redis", DelSchema()),
		HandleDelTool(redisClient),

	)
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("keys", "Find all keys matching the pattern", KeysSchema()),
		HandleKeysTool(redisClient),

	)
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("exists", "Check if keys exist in Redis", ExistsSchema()),
		HandleExistsTool(redisClient),

	)
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("expire", "Set expiration time for a key", ExpireSchema()),
		HandleExpireTool(redisClient),

	)
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("ttl", "Get time-to-live for a key", TTLSchema()),
		HandleTTLTool(redisClient),

	)
	mcpServer.AddTool(
		mcp.NewToolWithRawSchema("publish", "Publish message to a channel", PublishSchema()),
		HandlePublishTool(redisClient),

	)

	return mcpServer, nil
}

// Get工具模式定义
func GetSchema() string {
	return `{  
        "type": "object",  
        "properties": {  
            "key": {  
                "type": "string",  
                "description": "Redis key to get value for"  
            }  
        },  
        "required": ["key"]  
    }`
}

// Set工具模式定义
func SetSchema() string {
	return `{  
        "type": "object",  
        "properties": {  
            "key": {  
                "type": "string",  
                "description": "Redis key to set"  
            },  
            "value": {  
                "type": "string",  
                "description": "Value to set"  
            },  
            "expiration": {  
                "type": "integer",  
                "description": "Expiration time in seconds (optional)"  
            }  
        },  
        "required": ["key", "value"]  
    }`
}

// Del工具模式定义
func DelSchema() string {
	return `{  
        "type": "object",  
        "properties": {  
            "keys": {  
                "type": "array",  
                "items": {  
                    "type": "string"  
                },  
                "description": "Redis keys to delete"  
            }  
        },  
        "required": ["keys"]  
    }`
}

// Keys工具模式定义
func KeysSchema() string {
	return `{  
        "type": "object",  
        "properties": {  
            "pattern": {  
                "type": "string",  
                "description": "Pattern to match keys"  
            }  
        },  
        "required": ["pattern"]  
    }`
}

// Exists工具模式定义
func ExistsSchema() string {
	return `{  
        "type": "object",  
        "properties": {  
            "keys": {  
                "type": "array",  
                "items": {  
                    "type": "string"  
                },  
                "description": "Redis keys to check"  
            }  
        },  
        "required": ["keys"]  
    }`
}

// Expire工具模式定义
func ExpireSchema() string {
	return `{  
        "type": "object",  
        "properties": {  
            "key": {  
                "type": "string",  
                "description": "Redis key to set expiration on"  
            },  
            "seconds": {  
                "type": "integer",  
                "description": "Time to expiration in seconds"  
            }  
        },  
        "required": ["key", "seconds"]  
    }`
}

// TTL工具模式定义
func TTLSchema() string {
	return `{  
        "type": "object",  
        "properties": {  
            "key": {  
                "type": "string",  
                "description": "Redis key to get TTL for"  
            }  
        },  
        "required": ["key"]  
    }`
}

// Publish工具模式定义
func PublishSchema() string {
	return `{  
        "type": "object",  
        "properties": {  
            "channel": {  
                "type": "string",  
                "description": "Channel to publish message to"  
            },  
            "message": {  
                "type": "string",  
                "description": "Message to publish"  
            }  
        },  
        "required": ["channel", "message"]  
    }`
}

// Get工具处理函数
func HandleGetTool(client *common.RedisClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := struct {
			Key string `json:"key"`
		}{}

		if err := request.ParseParams(&params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
		value, err := client.Get(params.Key)
		if err != nil {
			return &mcp.CallToolResult{
				Output: fmt.Sprintf("Error getting key: %v", err),
			}, nil
		}

		return &mcp.CallToolResult{
			Output: value,
		}, nil
	}
}

// Set工具处理函数
func HandleSetTool(client *common.RedisClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := struct {
			Key        string `json:"key"`
			Value      string `json:"value"`
			Expiration int    `json:"expiration,omitempty"`
		}{}

		if err := request.ParseParams(&params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}

		exp := time.Duration(0)
		if params.Expiration > 0 {
			exp = time.Duration(params.Expiration) * time.Second

		}
		err := client.Set(params.Key, params.Value, exp)
		if err != nil {
			return &mcp.CallToolResult{
				Output: fmt.Sprintf("Error setting key: %v", err),
			}, nil
		}

		return &mcp.CallToolResult{
			Output: "OK",
		}, nil
	}
}

// Del工具处理函数
func HandleDelTool(client *common.RedisClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := struct {
			Keys []string `json:"keys"`
		}{}

		if err := request.ParseParams(&params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
		count, err := client.Del(params.Keys...)
		if err != nil {
			return &mcp.CallToolResult{
				Output: fmt.Sprintf("Error deleting keys: %v", err),
			}, nil
		}

		return &mcp.CallToolResult{
			Output: fmt.Sprintf("Deleted %d keys", count),
		}, nil
	}
}

// Keys工具处理函数
func HandleKeysTool(client *common.RedisClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := struct {
			Pattern string `json:"pattern"`
		}{}

		if err := request.ParseParams(&params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
		keys, err := client.Keys(params.Pattern)
		if err != nil {
			return &mcp.CallToolResult{
				Output: fmt.Sprintf("Error listing keys: %v", err),
			}, nil
		}

		return &mcp.CallToolResult{
			Output: fmt.Sprintf("%v", keys),
		}, nil
	}
}

// Exists工具处理函数
func HandleExistsTool(client *common.RedisClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := struct {
			Keys []string `json:"keys"`
		}{}

		if err := request.ParseParams(&params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
		count, err := client.Exists(params.Keys...)
		if err != nil {
			return &mcp.CallToolResult{
				Output: fmt.Sprintf("Error checking keys: %v", err),
			}, nil
		}

		return &mcp.CallToolResult{
			Output: fmt.Sprintf("%d", count),
		}, nil
	}
}

// Expire工具处理函数
func HandleExpireTool(client *common.RedisClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := struct {
			Key     string `json:"key"`
			Seconds int    `json:"seconds"`
		}{}

		if err := request.ParseParams(&params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
		success, err := client.Expire(params.Key, time.Duration(params.Seconds)*time.Second)
		if err != nil {
			return &mcp.CallToolResult{
				Output: fmt.Sprintf("Error setting expiration: %v", err),
			}, nil
		}

		if !success {
			return &mcp.CallToolResult{
				Output: "Key does not exist",
			}, nil
		}

		return &mcp.CallToolResult{
			Output: "OK",
		}, nil
	}
}

// TTL工具处理函数
func HandleTTLTool(client *common.RedisClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := struct {
			Key string `json:"key"`
		}{}

		if err := request.ParseParams(&params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
		ttl, err := client.TTL(params.Key)
		if err != nil {
			return &mcp.CallToolResult{
				Output: fmt.Sprintf("Error getting TTL: %v", err),
			}, nil
		}

		if ttl < 0 {
			if ttl == -1 {
				return &mcp.CallToolResult{
					Output: "Key exists but has no expiration",
				}, nil
			} else {
				return &mcp.CallToolResult{
					Output: "Key does not exist",
				}, nil
			}

		}

		return &mcp.CallToolResult{
			Output: fmt.Sprintf("%d", int(ttl.Seconds())),
		}, nil
	}
}

// Publish工具处理函数
func HandlePublishTool(client *common.RedisClient) common.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := struct {
			Channel string `json:"channel"`
			Message string `json:"message"`
		}{}

		if err := request.ParseParams(&params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
		err := client.Publish(params.Channel, params.Message)
		if err != nil {
			return &mcp.CallToolResult{
				Output: fmt.Sprintf("Error publishing message: %v", err),
			}, nil
		}

		return &mcp.CallToolResult{
			Output: "Message published",
		}, nil
	}
}
