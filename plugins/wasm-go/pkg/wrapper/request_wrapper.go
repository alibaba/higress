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
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

func GetRequestScheme() string {
	scheme, err := proxywasm.GetHttpRequestHeader(":scheme")
	if err != nil {
		proxywasm.LogError("parse request scheme failed")
		return ""
	}
	return scheme
}

func GetRequestHost() string {
	host, err := proxywasm.GetHttpRequestHeader(":authority")
	if err != nil {
		proxywasm.LogError("parse request host failed")
		return ""
	}
	return host
}

func GetRequestPath() string {
	path, err := proxywasm.GetHttpRequestHeader(":path")
	if err != nil {
		proxywasm.LogError("parse request path failed")
		return ""
	}
	return path
}

func GetRequestMethod() string {
	method, err := proxywasm.GetHttpRequestHeader(":method")
	if err != nil {
		proxywasm.LogError("parse request path failed")
		return ""
	}
	return method
}

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
