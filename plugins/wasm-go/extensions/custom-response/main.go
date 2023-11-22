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

package main

import (
	"strconv"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"custom-response",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

type CustomResponseConfig struct {
	statusCode     uint32            `yaml:"status_code"`
	headers        map[string]string `yaml:"headers"`
	body           string            `yaml:"body"`
	enableOnStatus uint32            `yaml:"enable_on_status"`
}

func parseConfig(json gjson.Result, config *CustomResponseConfig, log wrapper.Log) error {
	headers := json.Get("headers")
	if len(headers.Raw) > 0 {
		config.headers = make(map[string]string)
		for _, v := range headers.Array() {
			kv := strings.Split(v.String(), "=")
			config.headers[kv[0]] = kv[1]
		}
	}
	config.body = json.Get("body").String()
	config.statusCode = uint32(json.Get("status_code").Int())
	config.enableOnStatus = uint32(json.Get("enable_on_status").Int())

	return nil
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config CustomResponseConfig, log wrapper.Log) types.Action {
	if config.enableOnStatus != 0 {
		statusCodeStr, err := proxywasm.GetHttpResponseHeader(":status")
		if err != nil {
			log.Warnf("get http response status code failed: %v", err)
			return types.ActionContinue
		}
		statusCode, err := strconv.ParseUint(statusCodeStr, 10, 32)
		if err != nil {
			log.Warnf("parse http response status code failed: %v", err)
			return types.ActionContinue
		}
		if uint32(statusCode) != config.enableOnStatus {
			return types.ActionContinue
		}
	}

	// Add custom HTTP response header
	for k, v := range config.headers {
		proxywasm.AddHttpResponseHeader(k, v)
	}
	// When we modify the response body, we need to remove the content-length header.
	// Otherwise, the wrong content-length is sent to the upstream and that might result in client crash,
	// if the size of the data differs from the original size.
	if config.body != "" {
		proxywasm.RemoveHttpResponseHeader("content-length")
	}

	// Modify HTTP response status code
	if config.statusCode != 0 {
		headers, err := proxywasm.GetHttpResponseHeaders()
		if err != nil {
			log.Warnf("get http response headers failed: %v", err)
			return types.ActionContinue
		}
		proxywasm.SendHttpResponse(config.statusCode, headers, []byte(config.body), -1)
	}

	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config CustomResponseConfig, body []byte, log wrapper.Log) types.Action {
	// Modify HTTP response body
	if config.body != "" && config.statusCode == 0 {
		err := proxywasm.ReplaceHttpResponseBody([]byte(config.body))
		if err != nil {
			log.Warnf("replace http response body failed: %v", err)
			return types.ActionContinue
		}
	}

	return types.ActionContinue
}
