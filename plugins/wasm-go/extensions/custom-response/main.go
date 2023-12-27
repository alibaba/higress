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
	"fmt"
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
	)
}

type CustomResponseConfig struct {
	statusCode     uint32      `yaml:"status_code"`
	headers        [][2]string `yaml:"headers"`
	body           string      `yaml:"body"`
	enableOnStatus []uint32    `yaml:"enable_on_status"`
}

func parseConfig(json gjson.Result, config *CustomResponseConfig, log wrapper.Log) error {
	headersArray := json.Get("headers").Array()
	config.headers = make([][2]string, 0, len(headersArray))
	for _, v := range headersArray {
		kv := strings.Split(v.String(), "=")
		if len(kv) == 2 {
			config.headers = append(config.headers, [2]string{kv[0], kv[1]})
		} else {
			return fmt.Errorf("invalid header pair format: %s", v.String())
		}
	}

	config.body = json.Get("body").String()

	config.statusCode = 200
	if json.Get("status_code").Exists() {
		statusCode := json.Get("status_code")
		parsedStatusCode, err := strconv.Atoi(statusCode.String())
		if err != nil {
			return fmt.Errorf("invalid status code value: %s", statusCode.String())
		}
		config.statusCode = uint32(parsedStatusCode)
	}

	enableOnStatusArray := json.Get("enable_on_status").Array()
	config.enableOnStatus = make([]uint32, len(enableOnStatusArray))
	for _, v := range enableOnStatusArray {
		parsedEnableOnStatus, err := strconv.Atoi(v.String())
		if err != nil {
			return fmt.Errorf("invalid enable_on_status value: %s", v.String())
		}
		config.enableOnStatus = append(config.enableOnStatus, uint32(parsedEnableOnStatus))
	}

	return nil
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config CustomResponseConfig, log wrapper.Log) types.Action {
	if len(config.enableOnStatus) == 0 {
		proxywasm.SendHttpResponse(config.statusCode, config.headers, []byte(config.body), -1)
		return types.ActionContinue
	}

	// enableOnStatus is not empty, compare the status code.
	// if match the status code, mock the response.
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
	for _, v := range config.enableOnStatus {
		if uint32(statusCode) == v {
			proxywasm.SendHttpResponse(config.statusCode, config.headers, []byte(config.body), -1)
		}
	}
	return types.ActionContinue
}
