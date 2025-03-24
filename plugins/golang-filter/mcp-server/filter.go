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

	req        *http.Request
	sse        bool
	message    bool
	bodyBuffer []byte
}

// Callbacks which are called in request path
// The endStream is true if the request doesn't have body
func (f *filter) DecodeHeaders(header api.RequestHeaderMap, endStream bool) api.StatusType {
	fullPath, _ := header.Get(":path")
	parsedURL, _ := url.Parse(fullPath)
	f.path = parsedURL.Path
	method, _ := header.Get(":method")
	for _, server := range f.config.servers {
		if f.path == server.GetSSEEndpoint() {
			if method != http.MethodGet {
				f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusMethodNotAllowed, "Method not allowed", nil, 0, "")
			} else {
				f.sse = true
				body := "SSE connection create"
				f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusOK, body, nil, 0, "")
			}
			api.LogInfof("%s SSE connection started", server.GetServerName())
			return api.LocalReply
		} else if f.path == server.GetMessageEndpoint() {
			if method != http.MethodPost {
				f.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusMethodNotAllowed, "Method not allowed", nil, 0, "")
			}
			// Create a new http.Request object
			f.req = &http.Request{
				Method: method,
				URL:    parsedURL,
				Header: make(http.Header),
			}
			api.LogDebugf("Message request: %v", parsedURL)
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
	if endStream {
		return api.Continue
	} else {
		return api.StopAndBuffer
	}
}

// DecodeData might be called multiple times during handling the request body.
// The endStream is true when handling the last piece of the body.
func (f *filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	if f.message {
		f.bodyBuffer = append(f.bodyBuffer, buffer.Bytes()...)

		if endStream {
			for _, server := range f.config.servers {
				if f.path == server.GetMessageEndpoint() {
					// Create a response recorder to capture the response
					recorder := httptest.NewRecorder()
					// Call the handleMessage method of SSEServer with complete body
					server.HandleMessage(recorder, f.req, f.bodyBuffer)
					f.message = false
					// clear buffer
					f.bodyBuffer = nil
					f.callbacks.DecoderFilterCallbacks().SendLocalReply(recorder.Code, recorder.Body.String(), recorder.Header(), 0, "")
					return api.LocalReply
				}
			}
		}
		return api.StopAndBuffer
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
		return api.Continue
	}
	return api.Continue
}

// EncodeData might be called multiple times during handling the response body.
// The endStream is true when handling the last piece of the body.
func (f *filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	for _, server := range f.config.servers {
		if f.sse {
			buffer.Reset()
			server.HandleSSE(f.callbacks)
			f.sse = false
			return api.Running
		}
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
