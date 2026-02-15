// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wrapper

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

// Deprecated: Use HttpContext.Scheme() instead.
// This function calls proxywasm.GetHttpRequestHeader which will fail outside the header phase.
// The new method uses cached values from the header phase and can be called at any time.
func GetRequestScheme() string {
	scheme, err := proxywasm.GetHttpRequestHeader(":scheme")
	if err != nil {
		proxywasm.LogErrorf("get request scheme failed: %v", err)
		return ""
	}
	return scheme
}

// Deprecated: Use HttpContext.Host() instead.
// This function calls proxywasm.GetHttpRequestHeader which will fail outside the header phase.
// The new method uses cached values from the header phase and can be called at any time.
func GetRequestHost() string {
	host, err := proxywasm.GetHttpRequestHeader(":authority")
	if err != nil {
		proxywasm.LogErrorf("get request host failed: %v", err)
		return ""
	}
	return host
}

// Deprecated: Use HttpContext.Path() instead.
// This function calls proxywasm.GetHttpRequestHeader which will fail outside the header phase.
// The new method uses cached values from the header phase and can be called at any time.
func GetRequestPath() string {
	path, err := proxywasm.GetHttpRequestHeader(":path")
	if err != nil {
		proxywasm.LogErrorf("get request path failed: %v", err)
		return ""
	}
	return path
}

// Deprecated: Use url.Parse(HttpContext.Path()).Path instead.
// This function depends on GetRequestPath() which will fail outside the header phase.
func GetRequestPathWithoutQuery() string {
	rawPath := GetRequestPath()
	if rawPath == "" {
		return ""
	}
	path, err := url.Parse(rawPath)
	if err != nil {
		proxywasm.LogErrorf("failed to parse request path '%s': %v", rawPath, err)
		return ""
	}
	return path.Path
}

// Deprecated: Use HttpContext.Method() instead.
// This function calls proxywasm.GetHttpRequestHeader which will fail outside the header phase.
// The new method uses cached values from the header phase and can be called at any time.
func GetRequestMethod() string {
	method, err := proxywasm.GetHttpRequestHeader(":method")
	if err != nil {
		proxywasm.LogErrorf("get request method failed: %v", err)
		return ""
	}
	return method
}

// Deprecated: Use HttpContext.IsWebsocket() instead.
// This function calls proxywasm.GetHttpRequestHeader which will fail outside the header phase.
// The new method uses cached values from the header phase and can be called at any time.
func IsWebsocket() bool {
	connection, _ := proxywasm.GetHttpRequestHeader("connection")
	upgrade, _ := proxywasm.GetHttpRequestHeader("upgrade")
	if strings.EqualFold(connection, "upgrade") && strings.EqualFold(upgrade, "websocket") {
		return true
	}
	return false
}

// Deprecated: Use HttpContext.IsBinaryRequestBody() instead.
// This function calls proxywasm.GetHttpRequestHeader which will fail outside the header phase.
// The new method uses cached values from the header phase and can be called at any time.
func IsBinaryRequestBody() bool {
	contentType, _ := proxywasm.GetHttpRequestHeader("content-type")
	if strings.Contains(contentType, "octet-stream") ||
		strings.Contains(contentType, "grpc") {
		return true
	}
	encoding, _ := proxywasm.GetHttpRequestHeader("content-encoding")
	if encoding != "" {
		return true
	}
	return false
}

// Deprecated: Use HttpContext.IsBinaryResponseBody() instead.
// This function calls proxywasm.GetHttpResponseHeader which will fail outside the header phase.
// The new method uses cached values from the header phase and can be called at any time.
func IsBinaryResponseBody() bool {
	contentType, _ := proxywasm.GetHttpResponseHeader("content-type")
	if strings.Contains(contentType, "octet-stream") ||
		strings.Contains(contentType, "grpc") {
		return true
	}
	encoding, _ := proxywasm.GetHttpResponseHeader("content-encoding")
	if encoding != "" {
		return true
	}
	return false
}

// Deprecated: Use HttpContext.HasRequestBody() instead.
// This function only checks headers and may give incorrect results.
// The new method also considers whether endOfStream was received in the header phase.
func HasRequestBody() bool {
	contentTypeStr, _ := proxywasm.GetHttpRequestHeader("content-type")
	contentLengthStr, _ := proxywasm.GetHttpRequestHeader("content-length")
	transferEncodingStr, _ := proxywasm.GetHttpRequestHeader("transfer-encoding")
	proxywasm.LogDebugf("check has request body: contentType:%s, contentLengthStr:%s, transferEncodingStr:%s",
		contentTypeStr, contentLengthStr, transferEncodingStr)
	if contentLengthStr != "" {
		contentLength, err := strconv.Atoi(contentLengthStr)
		if err == nil {
			if contentLength == 0 {
				return false
			}
			return true
		}
	}
	if contentTypeStr != "" {
		return true
	}
	return strings.Contains(transferEncodingStr, "chunked")
}

func IsApplicationJson() bool {
	contentTypeStr, _ := proxywasm.GetHttpResponseHeader("content-type")
	return strings.Contains(contentTypeStr, "application/json")
}

// Deprecated: Use HttpContext.HasResponseBody() instead.
// This function only checks headers and may give incorrect results.
// The new method also considers whether endOfStream was received in the header phase.
func HasResponseBody() bool {
	contentTypeStr, _ := proxywasm.GetHttpResponseHeader("content-type")
	contentLengthStr, _ := proxywasm.GetHttpResponseHeader("content-length")
	transferEncodingStr, _ := proxywasm.GetHttpResponseHeader("transfer-encoding")
	proxywasm.LogDebugf("check has request body: contentType:%s, contentLengthStr:%s, transferEncodingStr:%s",
		contentTypeStr, contentLengthStr, transferEncodingStr)
	if contentLengthStr != "" {
		contentLength, err := strconv.Atoi(contentLengthStr)
		if err == nil {
			if contentLength == 0 {
				return false
			}
			return true
		}
	}
	if contentTypeStr != "" {
		return true
	}
	return strings.Contains(transferEncodingStr, "chunked")
}
