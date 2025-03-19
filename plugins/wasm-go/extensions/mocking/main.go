// Copyright (c) 2023 Alibaba Group Holding Ltd.
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
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"net/url"
	"strings"
)

const (
	defaultResponseContentType = "application/json"
	defaultStatusdCode         = 200
	defaultRespBody            = "{\"hello\":\"world\"}"
)

func main() {
	wrapper.SetCtx(
		"mocking",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type MockConfig struct {
	Responses      []ResponseConfig `json:"responses"`
	WithMockHeader bool             `json:"with_mock_header"`
}

type ResponseConfig struct {
	Trigger    TriggerConfig `json:"trigger"`
	Body       string        `json:"body"`
	Headers    []HeaderPair  `json:"headers"`
	StatusCode int           `json:"status_code"`
}

type TriggerConfig struct {
	Headers []HeaderPair `json:"headers"`
	Query   []QueryPair  `json:"queries"`
}

type HeaderPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type QueryPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func parseConfig(json gjson.Result, config *MockConfig, log wrapper.Log) error {

	responses := json.Get("responses").Array()
	config.Responses = make([]ResponseConfig, 0, len(responses))

	for i, respJson := range responses {
		var resp ResponseConfig

		if respBody := respJson.Get("body"); respBody.Exists() {
			resp.Body = respBody.String()
		} else {
			resp.Body = defaultRespBody
		}

		if statusCode := respJson.Get("status_code"); statusCode.Exists() {
			if statusCode.Int() < 100 || statusCode.Int() >= 600 {
				return fmt.Errorf("response[%d]: invalid statusCode %d", i, resp.StatusCode)
			}
			resp.StatusCode = int(statusCode.Int())
		} else {
			resp.StatusCode = defaultStatusdCode
		}

		if responseHeaders := respJson.Get("headers"); responseHeaders.Exists() {
			headerArray := responseHeaders.Array()
			resp.Headers = make([]HeaderPair, 0, len(headerArray))
			for _, h := range headerArray {
				resp.Headers = append(resp.Headers, HeaderPair{
					Key:   h.Get("key").String(),
					Value: h.Get("value").String(),
				})
			}
		} else {
			resp.Headers = make([]HeaderPair, 0, 1)
			resp.Headers = append(resp.Headers, HeaderPair{
				Key:   "Content-Type",
				Value: defaultResponseContentType,
			})
		}

		if triggerJson := respJson.Get("trigger"); triggerJson.Exists() {
			resp.Trigger.Headers = parseHeaderPairs(triggerJson.Get("headers"))
			resp.Trigger.Query = parseQueryPairs(triggerJson.Get("queries"))
		}

		config.Responses = append(config.Responses, resp)
	}

	if len(config.Responses) == 0 {
		return fmt.Errorf("at least one response configuration is required")
	}

	if mockHeaderStatus := json.Get("with_mock_header"); mockHeaderStatus.Exists() {
		config.WithMockHeader = mockHeaderStatus.Bool()
	} else {
		config.WithMockHeader = true
	}

	return nil
}

func parseHeaderPairs(json gjson.Result) []HeaderPair {
	pairs := make([]HeaderPair, 0)
	for _, p := range json.Array() {
		pairs = append(pairs, HeaderPair{
			Key:   p.Get("key").String(),
			Value: p.Get("value").String(),
		})
	}
	return pairs
}

func parseQueryPairs(json gjson.Result) []QueryPair {
	pairs := make([]QueryPair, 0)
	for _, p := range json.Array() {
		pairs = append(pairs, QueryPair{
			Key:   p.Get("key").String(),
			Value: p.Get("value").String(),
		})
	}
	return pairs
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config MockConfig, log wrapper.Log) types.Action {
	for _, resp := range config.Responses {
		if matchTrigger(ctx, resp.Trigger) {
			sendMockResponse(ctx, config, resp, log)
			return types.ActionPause
		}
	}
	headers := make([][2]string, 0)
	if config.WithMockHeader {
		headers = append(headers, [2]string{"x-mock-by", "higress"})
	}
	headers = append(headers, [2]string{"Content-Type", defaultResponseContentType})
	err := proxywasm.SendHttpResponse(200, headers, []byte(defaultRespBody), -1)
	if err != nil {
		log.Errorf("send http response to client occurs error: %v", err)
	}
	return types.ActionPause
}

func matchTrigger(ctx wrapper.HttpContext, trigger TriggerConfig) bool {
	for _, h := range trigger.Headers {
		value, err := proxywasm.GetHttpRequestHeader(h.Key)
		if err != nil || !strings.EqualFold(value, h.Value) {
			return false
		}
	}

	parsedURL, err := url.Parse(ctx.Path())
	if err != nil {
		return false
	}
	query, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		return false
	}

	for _, q := range trigger.Query {
		values, exists := query[q.Key]
		if !exists {
			return false
		}

		valueMatched := false
		for _, v := range values {
			if strings.EqualFold(v, q.Value) {
				valueMatched = true
				break
			}
		}
		if !valueMatched {
			return false
		}
	}

	return true
}

func sendMockResponse(ctx wrapper.HttpContext, config MockConfig, resp ResponseConfig, log wrapper.Log) {
	headers := make([][2]string, 0, len(resp.Headers))

	for _, h := range resp.Headers {
		headers = append(headers, [2]string{h.Key, h.Value})
	}

	if config.WithMockHeader {
		headers = append(headers, [2]string{"x-mock-by", "higress"})
	}

	err := proxywasm.SendHttpResponse(uint32(resp.StatusCode), headers, []byte(resp.Body), -1)
	if err != nil {
		log.Errorf("send http response to client occurs error: %v", err)
		return
	}
}
