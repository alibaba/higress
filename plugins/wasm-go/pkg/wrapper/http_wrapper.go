// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wrapper

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

type ResponseCallback func(statusCode int, responseHeaders http.Header, responseBody []byte)

type HttpClient interface {
	Get(rawURL string, headers [][2]string, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Head(rawURL string, headers [][2]string, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Options(rawURL string, headers [][2]string, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Post(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Put(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Patch(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Delete(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Connect(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Trace(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
	Call(method, rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error
}

type ClusterClient[C Cluster] struct {
	cluster C
}

func NewClusterClient[C Cluster](cluster C) *ClusterClient[C] {
	return &ClusterClient[C]{cluster: cluster}
}

func (c ClusterClient[C]) Get(rawURL string, headers [][2]string, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, http.MethodGet, rawURL, headers, nil, cb, timeoutMillisecond...)
}
func (c ClusterClient[C]) Head(rawURL string, headers [][2]string, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, http.MethodHead, rawURL, headers, nil, cb, timeoutMillisecond...)
}
func (c ClusterClient[C]) Options(rawURL string, headers [][2]string, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, http.MethodOptions, rawURL, headers, nil, cb, timeoutMillisecond...)
}
func (c ClusterClient[C]) Post(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, http.MethodPost, rawURL, headers, body, cb, timeoutMillisecond...)
}
func (c ClusterClient[C]) Put(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, http.MethodPut, rawURL, headers, body, cb, timeoutMillisecond...)
}
func (c ClusterClient[C]) Patch(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, http.MethodPatch, rawURL, headers, body, cb, timeoutMillisecond...)
}
func (c ClusterClient[C]) Delete(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, http.MethodDelete, rawURL, headers, body, cb, timeoutMillisecond...)
}
func (c ClusterClient[C]) Connect(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, http.MethodConnect, rawURL, headers, body, cb, timeoutMillisecond...)
}
func (c ClusterClient[C]) Trace(rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, http.MethodTrace, rawURL, headers, body, cb, timeoutMillisecond...)
}

func (c ClusterClient[C]) Call(method, rawURL string, headers [][2]string, body []byte, cb ResponseCallback, timeoutMillisecond ...uint32) error {
	return HttpCall(c.cluster, method, rawURL, headers, body, cb, timeoutMillisecond...)
}

func HttpCall(cluster Cluster, method, rawURL string, headers [][2]string, body []byte,
	callback ResponseCallback, timeoutMillisecond ...uint32) error {
	for i := len(headers) - 1; i >= 0; i-- {
		key := headers[i][0]
		if key == ":method" || key == ":path" || key == ":authority" {
			headers = append(headers[:i], headers[i+1:]...)
		}
	}
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		proxywasm.LogCriticalf("invalid rawURL:%s", rawURL)
		return err
	}
	authority := cluster.HostName()
	if parsedURL.Host != "" {
		authority = parsedURL.Host
	}
	path := "/" + strings.TrimPrefix(parsedURL.Path, "/")
	if parsedURL.RawQuery != "" {
		path = fmt.Sprintf("%s?%s", path, parsedURL.RawQuery)
	}
	// default timeout is 500ms
	var timeout uint32 = 500
	if len(timeoutMillisecond) > 0 {
		timeout = timeoutMillisecond[0]
	}
	headers = append(headers, [2]string{":method", method}, [2]string{":path", path}, [2]string{":authority", authority})
	requestID := uuid.New().String()
	_, err = proxywasm.DispatchHttpCall(cluster.ClusterName(), headers, body, nil, timeout, func(numHeaders, bodySize, numTrailers int) {
		respBody, err := proxywasm.GetHttpCallResponseBody(0, bodySize)
		if err != nil {
			proxywasm.LogCriticalf("failed to get response body: %v", err)
		}
		respHeaders, err := proxywasm.GetHttpCallResponseHeaders()
		if err != nil {
			proxywasm.LogCriticalf("failed to get response headers: %v", err)
		}
		code := http.StatusBadGateway
		var normalResponse bool
		headers := make(http.Header)
		for _, h := range respHeaders {
			if h[0] == ":status" {
				code, err = strconv.Atoi(h[1])
				if err != nil {
					proxywasm.LogErrorf("failed to parse status: %v", err)
					code = http.StatusInternalServerError
				} else {
					normalResponse = true
				}
			}
			headers.Add(h[0], h[1])
		}
		proxywasm.LogDebugf("http call end, id: %s, code: %d, normal: %t, body: %s",
			requestID, code, normalResponse, respBody)
		callback(code, headers, respBody)
	})
	proxywasm.LogDebugf("http call start, id: %s, cluster: %s, method: %s, url: %s, headers: %#v, body: %s, timeout: %d",
		requestID, cluster.ClusterName(), method, rawURL, headers, body, timeout)
	return err
}
