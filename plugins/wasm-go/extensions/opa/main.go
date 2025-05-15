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
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"opa",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
	)
}

type Metadata struct {
	Input map[string]interface{} `json:"input"`
}

func parseConfig(json gjson.Result, config *OpaConfig, log wrapper.Log) error {
	policy := json.Get("policy").String()
	if strings.TrimSpace(policy) == "" {
		return errors.New("policy not allow empty")
	}

	timeout := json.Get("timeout").String()
	if strings.TrimSpace(timeout) == "" {
		return errors.New("timeout not allow empty")
	}

	duration, err := time.ParseDuration(timeout)
	if err != nil {
		return errors.New("timeout parse fail: " + err.Error())
	}

	config.resultPath = json.Get("resultPath").String()
	if config.resultPath == "" {
		config.resultPath = defaultResultPath
	}

	config.skipHeader = json.Get("skipHeader").Bool()
	config.skipBody = json.Get("skipBody").Bool()

	config.denyCodePath = json.Get("denyCodePath").String()
	config.denyMappingMessages = make(map[string]string)
	if config.denyCodePath != "" {
		denyMappingMessages := json.Get("denyMappingMessages").Map()
		for k, v := range denyMappingMessages {
			config.denyMappingMessages[k] = v.String()
		}
		if len(config.denyMappingMessages) == 0 {
			return errors.New("denyMappingMessages not allow empty when denyCodePath not empty")
		}
		config.denyMessageContenType = json.Get("denyMessageContenType").String()
		if config.denyMessageContenType == "" {
			return errors.New("denyMessageContenType not allow empty when denyCodePath not empty")
		}
	}

	config.no200Message = json.Get("no200Message").String()
	if config.no200Message != "" {
		config.no200Code = uint32(json.Get("no200Code").Int())
		config.no200ContenType = json.Get("no200ContenType").String()
		if config.no200ContenType == "" {
			return errors.New("no200ContenType not allow empty when no200Message not empty")
		}
		if config.no200Code == 0 {
			return errors.New("no200Code not allow empty when no200Message not empty")
		}
	}

	config.extratHeaders = make(map[string]string)
	extratHeaders := json.Get("extratHeaders").Map()
	for k, v := range extratHeaders {
		config.extratHeaders[k] = v.String()
	}

	var uint32Duration uint32

	if duration.Milliseconds() > int64(^uint32(0)) {
	} else {
		uint32Duration = uint32(duration.Milliseconds())
	}
	config.timeout = uint32Duration

	client, err := Client(json)
	if err != nil {
		return err
	}
	config.client = client
	config.policy = policy

	return nil
}

const (
	OPACtxKeyHeaders = "headers"
	OPACtxKeyMethod  = "method"
	OPACtxKeyScheme  = "scheme"
	OPACtxKeyPath    = "path"
	OPACtxKeyQuery   = "query"
)

func setCtx(ctx wrapper.HttpContext) {
	p, _ := url.Parse(ctx.Path())
	headers, _ := proxywasm.GetHttpRequestHeaders()
	ctx.SetContext(OPACtxKeyMethod, ctx.Method())
	ctx.SetContext(OPACtxKeyScheme, ctx.Scheme())
	ctx.SetContext(OPACtxKeyPath, p.Path)
	ctx.SetContext(OPACtxKeyQuery, p.RawQuery)
	ctx.SetContext(OPACtxKeyQuery, p.RawQuery)
	ctx.SetContext(OPACtxKeyHeaders, headers)
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config OpaConfig, log wrapper.Log) types.Action {
	if config.skipHeader {
		if !config.skipBody {
			setCtx(ctx)
		}
		if len(config.extratHeaders) > 0 {
			return types.HeaderStopIteration
		}
		return types.ActionContinue
	}
	setCtx(ctx)
	return opaCall(ctx, config, nil, log)
}

func onHttpRequestBody(ctx wrapper.HttpContext, config OpaConfig, body []byte, log wrapper.Log) types.Action {
	if config.skipBody {
		return types.ActionContinue
	}
	return opaCall(ctx, config, body, log)
}

func opaCall(ctx wrapper.HttpContext, config OpaConfig, body []byte, log wrapper.Log) types.Action {
	request := make(map[string]interface{}, 6)
	request["headers"] = ctx.GetContext(OPACtxKeyHeaders)
	request["method"] = ctx.GetContext(OPACtxKeyMethod)
	request["scheme"] = ctx.GetContext(OPACtxKeyScheme)
	request["path"] = ctx.GetContext(OPACtxKeyPath)
	request["query"] = ctx.GetContext(OPACtxKeyQuery)

	if len(body) != 0 {
		request["body"] = body
	}

	data, _ := json.Marshal(Metadata{Input: map[string]interface{}{"request": request}})
	opaUrl := fmt.Sprintf("/v1/data/%s/allow", config.policy)
	if config.resultPath != "" {
		opaUrl = fmt.Sprintf("/v1/data/%s", config.policy)
	}
	if err := config.client.Post(opaUrl, [][2]string{{"Content-Type", "application/json"}}, data, config.rspCall, config.timeout); err != nil {
		log.Errorf("client opa fail %v", err)
		return types.ActionPause
	}
	return types.ActionPause
}
