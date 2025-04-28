package mcp_session

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/handler"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	RedisNotEnabledResponseBody = "Redis is not enabled, SSE connection is not supported"
)

// The callbacks in the filter, like `DecodeHeaders`, can be implemented on demand.
// Because api.PassThroughStreamFilter provides a default implementation.
type filter struct {
	api.PassThroughStreamFilter

	callbacks api.FilterCallbackHandler
	path      string
	config    *config
	stopChan  chan struct{}

	req         *http.Request
	serverName  string
	proxyURL    *url.URL
	neepProcess bool

	userLevelConfig     bool
	mcpConfigHandler    *handler.MCPConfigHandler
	ratelimit           bool
	mcpRatelimitHandler *handler.MCPRatelimitHandler
}

// Callbacks which are called in request path
// The endStream is true if the request doesn't have body
func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	url := common.NewRequestURL(header)
	if url == nil {
		return api.Continue
	}
	f.path = url.ParsedURL.Path

	// Check if request matches any rule in match_list
	if !common.IsMatch(f.config.matchList, url.Host, f.path) {
		api.LogDebugf("Request does not match any rule in match_list: %s", url.ParsedURL.String())
		return api.Continue
	}
	f.neepProcess = true

	f.req = &http.Request{
		Method: url.Method,
		URL:    url.ParsedURL,
	}

	if strings.HasSuffix(f.path, ConfigPathSuffix) && f.config.enableUserLevelServer {
		if !url.InternalIP {
			api.LogWarnf("Access denied: non-Internal IP address %s", url.ParsedURL.String())
			f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusForbidden, "", nil, 0, "")
			return api.LocalReply
		}
		if strings.HasSuffix(f.path, ConfigPathSuffix) && url.Method == http.MethodGet {
			api.LogDebugf("Handling config request: %s", f.path)
			f.mcpConfigHandler.HandleConfigRequest(f.req, []byte{})
			return api.LocalReply
		}
		f.userLevelConfig = true
		if endStream {
			return api.Continue
		} else {
			return api.StopAndBuffer
		}
	}

	if !strings.HasSuffix(url.ParsedURL.Path, GlobalSSEPathSuffix) {
		f.proxyURL = url.ParsedURL
		if f.config.enableUserLevelServer {
			parts := strings.Split(url.ParsedURL.Path, "/")
			if len(parts) >= 3 {
				serverName := parts[1]
				uid := parts[2]
				// Get encoded config
				encodedConfig, _ := f.mcpConfigHandler.GetEncodedConfig(serverName, uid)
				if encodedConfig != "" {
					header.Set("x-higress-mcpserver-config", encodedConfig)
					api.LogDebugf("Set x-higress-mcpserver-config Header for %s:%s", serverName, uid)
				}
			}
			f.ratelimit = true
		}
		if endStream {
			return api.Continue
		} else {
			return api.StopAndBuffer
		}
	}

	if url.Method != http.MethodGet {
		f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusMethodNotAllowed, "Method not allowed", nil, 0, "")
	} else {
		f.config.defaultServer = common.NewSSEServer(common.NewMCPServer(DefaultServerName, Version),
			common.WithSSEEndpoint(GlobalSSEPathSuffix),
			common.WithMessageEndpoint(strings.TrimSuffix(url.ParsedURL.Path, GlobalSSEPathSuffix)),
			common.WithRedisClient(common.GlobalRedisClient))
		f.serverName = f.config.defaultServer.GetServerName()
		body := "SSE connection create"
		f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusOK, body, nil, 0, "")
	}
	return api.LocalReply
}

// DecodeData might be called multiple times during handling the request body.
// The endStream is true when handling the last piece of the body.
func (f *filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	if !f.neepProcess {
		return api.Continue
	}
	if !endStream {
		return api.StopAndBuffer
	}
	if f.userLevelConfig {
		// Handle config POST request
		api.LogDebugf("Handling config request: %s", f.path)
		f.mcpConfigHandler.HandleConfigRequest(f.req, buffer.Bytes())
		return api.LocalReply
	} else if f.ratelimit {
		if checkJSONRPCMethod(buffer.Bytes(), "tools/list") {
			api.LogDebugf("Not a tools call request, skipping ratelimit")
			return api.Continue
		}
		parts := strings.Split(f.req.URL.Path, "/")
		if len(parts) < 3 {
			api.LogWarnf("Access denied: no valid uid found")
			f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusForbidden, "", nil, 0, "")
			return api.LocalReply
		}
		serverName := parts[1]
		uid := parts[2]
		encodedConfig, err := f.mcpConfigHandler.GetEncodedConfig(serverName, uid)
		if err != nil {
			api.LogWarnf("Access denied: no valid config found for uid %s", uid)
			f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusForbidden, "", nil, 0, "")
			return api.LocalReply
		} else if encodedConfig == "" && checkJSONRPCMethod(buffer.Bytes(), "tools/call") {
			api.LogDebugf("Empty config found for %s:%s", serverName, uid)
			if !f.mcpRatelimitHandler.HandleRatelimit(f.req, buffer.Bytes()) {
				return api.LocalReply
			}
		}
	}
	return api.Continue
}

// Callbacks which are called in response path
// The endStream is true if the response doesn't have body
func (f *filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	if !f.neepProcess {
		return api.Continue
	}
	if f.serverName != "" {
		if common.GlobalRedisClient != nil {
			header.Set("Content-Type", "text/event-stream")
			header.Set("Cache-Control", "no-cache")
			header.Set("Connection", "keep-alive")
			header.Set("Access-Control-Allow-Origin", "*")
			header.Del("Content-Length")
		} else {
			header.Set("Content-Length", strconv.Itoa(len(RedisNotEnabledResponseBody)))
		}
		return api.Continue
	}
	return api.Continue
}

// EncodeData might be called multiple times during handling the response body.
// The endStream is true when handling the last piece of the body.
func (f *filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	if !f.neepProcess {
		return api.Continue
	}
	if !endStream {
		return api.StopAndBuffer
	}
	if f.proxyURL != nil && common.GlobalRedisClient != nil {
		sessionID := f.proxyURL.Query().Get("sessionId")
		if sessionID != "" {
			channel := common.GetSSEChannelName(sessionID)
			eventData := fmt.Sprintf("event: message\ndata: %s\n\n", buffer.String())
			publishErr := common.GlobalRedisClient.Publish(channel, eventData)
			if publishErr != nil {
				api.LogErrorf("Failed to publish wasm mcp server message to Redis: %v", publishErr)
			}
		}
	}

	if f.serverName != "" {
		if common.GlobalRedisClient != nil {
			// handle default server
			buffer.Reset()
			f.config.defaultServer.HandleSSE(f.callbacks, f.stopChan)
			return api.Running
		} else {
			buffer.SetString(RedisNotEnabledResponseBody)
			return api.Continue
		}
	}
	return api.Continue
}

// OnDestroy stops the goroutine
func (f *filter) OnDestroy(reason api.DestroyReason) {
	api.LogDebugf("OnDestroy: reason=%v", reason)
	if f.serverName != "" && f.stopChan != nil {
		select {
		case <-f.stopChan:
			return
		default:
			api.LogDebug("Stopping SSE connection")
			close(f.stopChan)
		}
	}
}

// check if the request is a tools/call request
func checkJSONRPCMethod(body []byte, method string) bool {
	var request mcp.CallToolRequest
	if err := json.Unmarshal(body, &request); err != nil {
		api.LogWarnf("Failed to unmarshal request body: %v, not a JSON RPC request", err)
		return true
	}

	return request.Method == method
}
