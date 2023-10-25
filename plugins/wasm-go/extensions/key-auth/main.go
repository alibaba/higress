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
	"key-auth/common"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"golang.org/x/exp/slices"
)

var (
	errMissingParams       = errors.New("key-auth: request missing params")
	errUnauthorized        = errors.New("key-auth: request unauthorized")
	errNotFoundKey         = errors.New("key-auth: request not found Key")
	errNotParseHeader      = errors.New("key-auth: request not parse headers")
	errKeyAuthNamesEmpty   = errors.New("key-auth: keys allow cannot be empty")
	errHeaderQueryAllFalse = errors.New("key-auth: must one of in_query and in_header be true")

	errRequestDeniedUnauthorizedConsumer = errors.New("key-auth: Request denied by Basic Auth check. Unauthorized consumer	")
)

const (
	defaultKeyAuthName = "x-api-key"
)

func main() {
	wrapper.SetCtx(
		"key-auth", // middleware name
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type Config struct {
	Keys     []string `yaml:"keys"` // key auth names
	InQuery  bool     `yaml:"in_query,omitempty"`
	InHeader bool     `yaml:"in_header,omitempty"`
	*common.Consumers
	*common.Rules
}

type Response struct {
	Message    string `json:"message"`
	StatusCode int    `json:"code"`
}

func parseConfig(json gjson.Result, config *Config, log wrapper.Log) error {
	err := common.ParseConsumersConfig(json, config.Consumers, log)
	if err != nil {
		return err
	}

	err = common.ParseRulesConfig(json, config.Rules, log)
	if err != nil {
		return err
	}

	names := json.Get("keys").Array()
	if len(names) == 0 {
		return errKeyAuthNamesEmpty
	}

	for _, name := range names {
		config.Keys = append(config.Keys, name.String())
	}

	in_query := json.Get("in_query").Bool()
	in_header := json.Get("in_header").Bool()

	if in_query || in_header {
		config.InHeader = in_header
		config.InQuery = in_query
	} else {
		return errHeaderQueryAllFalse
	}

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
	if len(config.Keys) <= 0 ||
		len(config.Consumers.Consumers) <= 0 ||
		(config.InHeader == false && config.InQuery == false) {
		return SendHttpResponse(http.StatusUnauthorized, errUnauthorized.Error(), nil)
	}

	if config.InHeader {
		token := ""
		for _, key := range config.Keys {
			value, err := proxywasm.GetHttpRequestHeader(key)
			if err != nil && value != "" {
				token = value
				break
			}
		}

		if token == "" {
			return SendHttpResponse(http.StatusUnauthorized, errNotFoundKey.Error(), nil)
		} else {
			ok, consumer := ParseTokenValid(token, config.Consumers.Consumers)
			if !ok {
				_ = proxywasm.ResumeHttpRequest()
				return types.ActionPause
			} else {
				ok, consumer := ParseRulesValid(ctx, consumer, config.Rules.Rules)
				if !ok {
					return SendHttpResponse(http.StatusForbidden, errRequestDeniedUnauthorizedConsumer.Error(), nil)
				} else {
					return Authenticated(consumer.Name)
				}
			}
		}

	} else if config.InQuery {
		token := ""
		for _, key := range config.Keys {
			value, err := proxywasm.GetHttpRequestTrailer(key)
			if err != nil && value != "" {
				token = value
				break
			}
		}

		if token == "" {
			return SendHttpResponse(http.StatusUnauthorized, errNotFoundKey.Error(), nil)
		} else {
			ok, consumer := ParseTokenValid(token, config.Consumers.Consumers)
			if !ok {
				_ = proxywasm.ResumeHttpRequest()
				return types.ActionPause
			} else {
				ok, consumer := ParseRulesValid(ctx, consumer, config.Rules.Rules)
				if !ok {
					return SendHttpResponse(http.StatusForbidden, errRequestDeniedUnauthorizedConsumer.Error(), nil)
				} else {
					return Authenticated(consumer.Name)
				}
			}
		}
	} else {
		return SendHttpResponse(http.StatusUnauthorized, errNotFoundKey.Error(), nil)
	}
	return types.ActionContinue
}

func ParseTokenValid(token string, consumers []common.Consumer) (bool, common.Consumer) {
	for _, consumer := range consumers {
		if consumer.Credential == token {
			return true, consumer
		}
	}
	return false, common.Consumer{}
}

func ParseRulesValid(ctx wrapper.HttpContext, consumer common.Consumer, rules []*common.Rule) (bool, common.Consumer) {
	if len(rules) <= 0 {
		return true, consumer
	}
	for _, rule := range rules {
		if len(rule.Allow) <= 0 || slices.Contains(rule.Allow, consumer.Name) {
			if len(rule.MatchDomain) > 0 {
				if slices.Contains(rule.MatchDomain, ctx.Host()) {
					return true, consumer
				}
			}

			if len(rule.MatchRoute) > 0 {
				if slices.Contains(rule.MatchRoute, ctx.Path()) {
					return true, consumer
				}
			}
		}
	}

	return false, common.Consumer{}
}

func SendHttpResponse(code int, message string, headers [][2]string) types.Action {
	var res Response
	res.StatusCode = code
	res.Message = message
	data, _ := json.Marshal(res)
	_ = proxywasm.SendHttpResponse(uint32(res.StatusCode), headers, data, -1)
	return types.ActionContinue
}

func Authenticated(name string) types.Action {
	_ = proxywasm.AddHttpRequestHeader("X-Mse-Consumer", name)
	return types.ActionContinue
}
