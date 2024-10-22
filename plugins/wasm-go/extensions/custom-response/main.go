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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"custom-response",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
	)
}

type CustomResponseConfig struct {
	statusCode     uint32
	headers        [][2]string
	body           string
	enableOnStatus []uint32
	contentType    string
}

func parseConfig(gjson gjson.Result, config *CustomResponseConfig, log wrapper.Log) error {
	headersArray := gjson.Get("headers").Array()
	config.headers = make([][2]string, 0, len(headersArray))
	for _, v := range headersArray {
		kv := strings.SplitN(v.String(), "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			if strings.EqualFold(key, "content-type") {
				config.contentType = value
			} else if strings.EqualFold(key, "content-length") {
				continue
			} else {
				config.headers = append(config.headers, [2]string{key, value})
			}
		} else {
			return fmt.Errorf("invalid header pair format: %s", v.String())
		}
	}

	config.body = gjson.Get("body").String()
	if config.contentType == "" && config.body != "" {
		if json.Valid([]byte(config.body)) {
			config.contentType = "application/json; charset=utf-8"
		} else {
			config.contentType = "text/plain; charset=utf-8"
		}
	}
	config.headers = append(config.headers, [2]string{"content-type", config.contentType})

	config.statusCode = 200
	if gjson.Get("status_code").Exists() {
		statusCode := gjson.Get("status_code")
		parsedStatusCode, err := strconv.Atoi(statusCode.String())
		if err != nil {
			return fmt.Errorf("invalid status code value: %s", statusCode.String())
		}
		config.statusCode = uint32(parsedStatusCode)
	}

	enableOnStatusArray := gjson.Get("enable_on_status").Array()
	config.enableOnStatus = make([]uint32, 0, len(enableOnStatusArray))
	for _, v := range enableOnStatusArray {
		parsedEnableOnStatus, err := strconv.Atoi(v.String())
		if err != nil {
			return fmt.Errorf("invalid enable_on_status value: %s", v.String())
		}
		config.enableOnStatus = append(config.enableOnStatus, uint32(parsedEnableOnStatus))
	}

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config CustomResponseConfig, log wrapper.Log) types.Action {
	if len(config.enableOnStatus) != 0 {
		return types.ActionContinue
	}
	err := proxywasm.SendHttpResponseWithDetail(config.statusCode, "custom-response", config.headers, []byte(config.body), -1)
	if err != nil {
		log.Errorf("send http response failed: %v", err)
	}

	return types.ActionPause
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config CustomResponseConfig, log wrapper.Log) types.Action {
	// enableOnStatus is not empty, compare the status code.
	// if match the status code, mock the response.
	statusCodeStr, err := proxywasm.GetHttpResponseHeader(":status")
	if err != nil {
		log.Errorf("get http response status code failed: %v", err)
		return types.ActionContinue
	}
	statusCode, err := strconv.ParseUint(statusCodeStr, 10, 32)
	if err != nil {
		log.Errorf("parse http response status code failed: %v", err)
		return types.ActionContinue
	}

	for _, v := range config.enableOnStatus {
		if uint32(statusCode) == v {
			err = proxywasm.SendHttpResponseWithDetail(config.statusCode, "custom-response", config.headers, []byte(config.body), -1)
			if err != nil {
				log.Errorf("send http response failed: %v", err)
			}
		}
	}

	return types.ActionContinue
}
