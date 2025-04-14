package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/handler"
	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/internal"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
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

	req        *http.Request
	serverName string
	message    bool
	proxyURL   *url.URL
	skip       bool

	userLevelConfig     bool
	mcpConfigHandler    *handler.MCPConfigHandler
	mcpRatelimitHandler *handler.MCPRatelimitHandler
}

type RequestURL struct {
	method     string
	scheme     string
	host       string
	path       string
	baseURL    string
	parsedURL  *url.URL
	internalIP bool
}

func NewRequestURL(header api.RequestHeaderMap) *RequestURL {
	method, _ := header.Get(":method")
	scheme, _ := header.Get(":scheme")
	host, _ := header.Get(":authority")
	path, _ := header.Get(":path")
	internalIP, _ := header.Get("x-envoy-internal")
	baseURL := fmt.Sprintf("%s://%s", scheme, host)
	parsedURL, _ := url.Parse(path)
	api.LogDebugf("RequestURL: method=%s, scheme=%s, host=%s, path=%s", method, scheme, host, path)
	return &RequestURL{method: method, scheme: scheme, host: host, path: path, baseURL: baseURL, parsedURL: parsedURL, internalIP: internalIP == "true"}
}

// Callbacks which are called in request path
// The endStream is true if the request doesn't have body
func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	url := NewRequestURL(header)
	f.path = url.parsedURL.Path

	// Check if request matches any rule in match_list
	if !internal.IsMatch(f.config.matchList, url.host, f.path) {
		f.skip = true
		api.LogDebugf("Request does not match any rule in match_list: %s", url.parsedURL.String())
		return api.Continue
	}

	for _, server := range f.config.servers {
		if f.path == server.GetSSEEndpoint() {
			if url.method != http.MethodGet {
				f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusMethodNotAllowed, "Method not allowed", nil, 0, "")
			} else {
				f.serverName = server.GetServerName()
				body := "SSE connection create"
				f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusOK, body, nil, 0, "")
			}
			api.LogDebugf("%s SSE connection started", server.GetServerName())
			return api.LocalReply
		} else if f.path == server.GetMessageEndpoint() {
			if f.path == server.GetMessageEndpoint() && url.method != http.MethodPost {
				f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusMethodNotAllowed, "Method not allowed", nil, 0, "")
				return api.LocalReply
			}
			// Create a new http.Request object
			f.req = &http.Request{
				Method: url.method,
				URL:    url.parsedURL,
				Header: make(http.Header),
			}
			api.LogDebugf("Message request: %v", url.parsedURL)
			// Copy headers from api.RequestHeaderMap to http.Header
			header.Range(func(key, value string) bool {
				f.req.Header.Add(key, value)
				return true
			})
			f.message = true
			if endStream {
				return api.Continue
			} else {
				return api.StopAndBuffer
			}
		}
	}

	if strings.HasSuffix(f.path, ConfigPathSuffix) && f.config.enableUserLevelServer {
		if !url.internalIP {
			f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusForbidden, "Access denied", nil, 0, "")
			return api.LocalReply
		}
		if strings.HasSuffix(f.path, ConfigPathSuffix) && url.method == http.MethodGet {
			api.LogDebugf("Handling config request: %s", f.path)
			f.mcpConfigHandler.HandleConfigRequest(f.path, url.method, []byte{})
			return api.LocalReply
		}
		f.req = &http.Request{
			Method: url.method,
			URL:    url.parsedURL,
		}
		f.userLevelConfig = true
		if endStream {
			return api.Continue
		} else {
			return api.StopAndBuffer
		}
	}

	if !strings.HasSuffix(url.parsedURL.Path, f.config.ssePathSuffix) {
		f.proxyURL = url.parsedURL
		if f.config.enableUserLevelServer {
			parts := strings.Split(url.parsedURL.Path, "/")
			if len(parts) < 3 {
				api.LogDebugf("Access denied: missing uid in path %s", url.parsedURL.Path)
				f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusForbidden, "Access denied: missing uid", nil, 0, "")
				return api.LocalReply
			}
			serverName := parts[1]
			uid := parts[2]
			// Get encoded config
			encodedConfig, err := f.mcpConfigHandler.GetEncodedConfig(serverName, uid)
			if err != nil {
				api.LogWarnf("Access denied: no valid config found for uid %s", uid)
				f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusForbidden, "", nil, 0, "")
				return api.LocalReply
			} else if encodedConfig != "" {
				header.Set("x-higress-mcpserver-config", encodedConfig)
				api.LogDebugf("Set x-higress-mcpserver-config Header for %s:%s", serverName, uid)
			} else {
				api.LogDebugf("Empty config found for %s:%s", serverName, uid)
				if !f.mcpRatelimitHandler.HandleRatelimit(url.parsedURL.Path, url.method, []byte{}) {
					return api.LocalReply
				}
			}
		}
		return api.Continue
	}

	if url.method != http.MethodGet {
		f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusMethodNotAllowed, "Method not allowed", nil, 0, "")
	} else {
		f.config.defaultServer = internal.NewSSEServer(internal.NewMCPServer(DefaultServerName, Version),
			internal.WithSSEEndpoint(f.config.ssePathSuffix),
			internal.WithMessageEndpoint(strings.TrimSuffix(url.parsedURL.Path, f.config.ssePathSuffix)),
			internal.WithRedisClient(f.config.redisClient))
		f.serverName = f.config.defaultServer.GetServerName()
		body := "SSE connection create"
		f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusOK, body, nil, 0, "")
	}
	return api.LocalReply
}

// DecodeData might be called multiple times during handling the request body.
// The endStream is true when handling the last piece of the body.
func (f *filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	if f.skip {
		return api.Continue
	}
	if f.message {
		if endStream {
			for _, server := range f.config.servers {
				if f.path == server.GetMessageEndpoint() {
					// Create a response recorder to capture the response
					recorder := httptest.NewRecorder()
					// Call the handleMessage method of SSEServer with complete body
					server.HandleMessage(recorder, f.req, buffer.Bytes())
					f.message = false
					f.callbacks.DecoderFilterCallbacks().SendLocalReply(recorder.Code, recorder.Body.String(), recorder.Header(), 0, "")
					return api.LocalReply
				}
			}
		}
		return api.StopAndBuffer
	} else if f.userLevelConfig {
		// Handle config POST request
		api.LogDebugf("Handling config request: %s", f.path)
		f.mcpConfigHandler.HandleConfigRequest(f.path, f.req.Method, buffer.Bytes())
		return api.LocalReply
	}
	return api.Continue
}

// Callbacks which are called in response path
// The endStream is true if the response doesn't have body
func (f *filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	if f.skip {
		return api.Continue
	}
	if f.serverName != "" {
		if f.config.redisClient != nil {
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
	if f.skip {
		return api.Continue
	}
	if !endStream {
		return api.StopAndBuffer
	}
	if f.proxyURL != nil && f.config.redisClient != nil {
		sessionID := f.proxyURL.Query().Get("sessionId")
		if sessionID != "" {
			channel := internal.GetSSEChannelName(sessionID)
			eventData := fmt.Sprintf("event: message\ndata: %s\n\n", buffer.String())
			publishErr := f.config.redisClient.Publish(channel, eventData)
			if publishErr != nil {
				api.LogErrorf("Failed to publish wasm mcp server message to Redis: %v", publishErr)
			}
		}
	}

	if f.serverName != "" {
		if f.config.redisClient != nil {
			// handle specific server
			for _, server := range f.config.servers {
				if f.serverName == server.GetServerName() {
					buffer.Reset()
					server.HandleSSE(f.callbacks, f.stopChan)
					return api.Running
				}
			}
			// handle default server
			if f.serverName == f.config.defaultServer.GetServerName() {
				buffer.Reset()
				f.config.defaultServer.HandleSSE(f.callbacks, f.stopChan)
				return api.Running
			}
			return api.Continue
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
