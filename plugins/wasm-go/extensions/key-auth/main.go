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

// The 'Basic' HTTP Authentication Scheme: https://datatracker.ietf.org/doc/html/rfc7617

package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"golang.org/x/exp/slices"
)

var (
	errMissingParams      = errors.New("key-auth: request missing params")
	errUnauthorized       = errors.New("key-auth: request missing params")
	errKeyAuthTokensEmpty = errors.New("key-auth: tokens allow cannot be empty")
)

const (
	defaultKeyAuthName = "X-API-KEY"
)

func main() {
	wrapper.SetCtx(
		"key-auth", // middleware name
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type Config struct {
	KeyAuthName   string   // key auth name
	KeyAuthTokens []string // key auth tokens
}

type Response struct {
	Message    string `json:"message"`
	StatusCode int    `json:"code"`
}

func parseConfig(json gjson.Result, config *Config, log wrapper.Log) error {
	name := json.Get("key_auth_name").String()
	if name == "" {
		config.KeyAuthName = name
	} else {
		config.KeyAuthName = defaultKeyAuthName
	}
	tokens := json.Get("key_auth_tokens").Array()
	if len(tokens) == 0 {
		return errKeyAuthTokensEmpty
	}

	for _, token := range tokens {
		config.KeyAuthTokens = append(config.KeyAuthTokens, token.String())
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
	var res Response
	if config.KeyAuthName == "" || len(config.KeyAuthTokens) <= 0 {
		res.StatusCode = http.StatusUnauthorized
		res.Message = errUnauthorized.Error()
		data, _ := json.Marshal(res)
		_ = proxywasm.SendHttpResponse(uint32(res.StatusCode), nil, data, -1)
		return types.ActionContinue
	}

	token, err := proxywasm.GetHttpRequestHeader(config.KeyAuthName)
	if err != nil {
		res.StatusCode = http.StatusUnauthorized
		res.Message = errUnauthorized.Error()
		data, _ := json.Marshal(res)
		_ = proxywasm.SendHttpResponse(uint32(res.StatusCode), nil, data, -1)
		return types.ActionContinue
	}
	valid := ParseTokenValid(token, config.KeyAuthTokens)
	if valid {
		_ = proxywasm.ResumeHttpRequest()
		return types.ActionPause
	} else {
		res.StatusCode = http.StatusUnauthorized
		res.Message = errUnauthorized.Error()
		data, _ := json.Marshal(res)
		_ = proxywasm.SendHttpResponse(uint32(res.StatusCode), nil, data, -1)
		return types.ActionContinue
	}
}

func ParseTokenValid(token string, tokens []string) bool {
	return slices.Contains(tokens, token)
}
