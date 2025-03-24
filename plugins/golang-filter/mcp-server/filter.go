package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

// The callbacks in the filter, like `DecodeHeaders`, can be implemented on demand.
// Because api.PassThroughStreamFilter provides a default implementation.
type filter struct {
	api.PassThroughStreamFilter

	callbacks api.FilterCallbackHandler
	path      string
	config    *config

	req     *http.Request
	sse     bool
	message bool
}

// Callbacks which are called in request path
// The endStream is true if the request doesn't have body
func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	fullPath, _ := header.Get(":path")
	parsedURL, _ := url.Parse(fullPath)
	f.path = parsedURL.Path
	method, _ := header.Get(":method")
	api.LogInfo(f.path)
	if f.path == f.config.SSEServer.SSEEndpoint {
		if method != http.MethodGet {
			f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusMethodNotAllowed, "Method not allowed", nil, 0, "")
		} else {
			f.sse = true
			body := "SSE connection create"
			f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusOK, body, nil, 0, "")
		}
		api.LogInfo("SSE connection started")
		return api.LocalReply
	} else if f.path == f.config.SSEServer.MessageEndpoint {
		if method != http.MethodPost {
			f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusMethodNotAllowed, "Method not allowed", nil, 0, "")
		}
		// Create a new http.Request object
		f.req = &http.Request{
			Method: method,
			URL:    parsedURL,
			Header: make(http.Header),
		}
		api.LogInfof("Message request: %v", parsedURL)
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
	if endStream {
		return api.Continue
	} else {
		return api.StopAndBuffer
	}
}

// DecodeData might be called multiple times during handling the request body.
// The endStream is true when handling the last piece of the body.
func (f *filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	api.LogInfo("Message DecodeData")
	// support suspending & resuming the filter in a background goroutine
	api.LogInfof("DecodeData: {%v}", buffer)
	if f.message {
		// Create a response recorder to capture the response
		recorder := httptest.NewRecorder()
		// Call the handleMessage method of SSEServer
		f.config.SSEServer.HandleMessage(recorder, f.req, buffer.Bytes())
		f.message = false
		api.LogInfof("Message DecodeData SendLocalReply %v", recorder)
		f.callbacks.DecoderFilterCallbacks().SendLocalReply(recorder.Code, recorder.Body.String(), recorder.Header(), 0, "")
		return api.LocalReply
	}
	return api.Continue
}

// Callbacks which are called in response path
// The endStream is true if the response doesn't have body
func (f *filter) EncodeHeaders(header api.ResponseHeaderMap, endStream bool) api.StatusType {
	if f.sse {
		header.Set("Content-Type", "text/event-stream")
		header.Set("Cache-Control", "no-cache")
		header.Set("Connection", "keep-alive")
		header.Set("Access-Control-Allow-Origin", "*")
		header.Del("Content-Length")
		api.LogInfo("SSE connection header set")
		return api.Continue
	}
	return api.Continue
}

// TODO: 连接多种数据库
// TODO: 多种存储类型
// TODO: 数据库多个实例
// EncodeData might be called multiple times during handling the response body.
// The endStream is true when handling the last piece of the body.
func (f *filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	if f.sse {
		//TODO: buffer cleanup
		f.config.SSEServer.HandleSSE(f.callbacks)
		f.sse = false
		return api.Running
	}
	return api.Continue
}

// OnDestroy 或 OnStreamComplete 中停止 goroutine
func (f *filter) OnDestroy(reason api.DestroyReason) {
	if f.sse && f.config.stopChan != nil {
		api.LogInfo("Stopping SSE connection")
		close(f.config.stopChan)
	}
}
