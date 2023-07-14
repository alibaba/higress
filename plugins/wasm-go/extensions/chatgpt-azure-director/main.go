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
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

var pathMatchers = []*PathMatcher{
	NewPathMatcher(`^/v1/models$`, "/openai/models"),
	NewPathMatcher(`^/v1/models/(?P<model>[^/]+)$`, "/openai/models/${ path.model }"),
	NewPathMatcher(`^/v1/completions$`, "/openai/developments/${ config.DevelopmentName }/completions"),
	NewPathMatcher(`^/v1/chat/completions$`, "/openai/developments/${ config.DevelopmentName }/chat/completions"),
}

func main() {
	wrapper.SetCtx(
		"chatgpt-azure-director",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type Config struct {
	AllowedKeys     map[string]struct{}
	APIKey          string
	APIVersion      string
	DevelopmentName string
	Scheme          string
	AzureHost       string
}

func parseConfig(json gjson.Result, config *Config, log wrapper.Log) error {
	if !json.Get("allowedKeys").IsArray() {
		return errors.New("allowedKeys type error, it should be an array")
	}
	array := json.Get("allowedKeys").Array()
	if len(array) == 0 {
		return errors.New("at least one key in allowedKeys")
	}
	if config.AllowedKeys == nil {
		config.AllowedKeys = make(map[string]struct{})
	}
	for _, item := range array {
		config.AllowedKeys[item.String()] = struct{}{}
	}

	config.APIKey = json.Get("apiKey").String()
	if config.APIKey == "" {
		return errors.New("invalid apiKey")
	}

	config.APIVersion = json.Get("apiVersion").String()
	if config.APIVersion == "" {
		config.APIVersion = "2023-03-15-preview"
	}

	config.DevelopmentName = json.Get("developmentName").String()
	if config.DevelopmentName == "" {
		return errors.New("invalid developmentName")
	}

	config.Scheme = json.Get("scheme").String()
	if config.Scheme != "http" {
		config.Scheme = "https"
	}

	config.AzureHost = json.Get("azureHost").String()
	if config.AzureHost == "" {
		config.AzureHost = config.DevelopmentName + ".openai.azure.com"
	}

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
	u, err := url.Parse(ctx.Path())
	if err != nil {
		_ = proxywasm.SendHttpResponse(400, nil, []byte("invalid path"), -1)
		return types.ActionContinue
	}
	for _, matcher := range pathMatchers {
		if matcher.Match(u.Path) {
			action := rewrite(ctx, config, matcher, u)
			scheme, _ := proxywasm.GetHttpRequestHeader(":scheme")
			host, _ := proxywasm.GetHttpRequestHeader(":host")
			path, _ := proxywasm.GetHttpRequestHeader(":path")
			log.Debugf("the directed request: %s://%s%s\n", scheme, host, path)
			return action
		}
	}
	return types.ActionContinue
}

func rewrite(ctx wrapper.HttpContext, config Config, matcher *PathMatcher, u *url.URL) types.Action {
	// check authorization
	authorization, err := proxywasm.GetHttpRequestHeader("Authorization")
	if err != nil || authorization == "" {
		_ = proxywasm.SendHttpResponse(401, nil, []byte("Authorization need"), -1)
		return types.ActionContinue
	}
	if _, ok := config.AllowedKeys[strings.TrimPrefix(authorization, "Bearer ")]; !ok {
		_ = proxywasm.SendHttpResponse(403, nil, []byte("Authorization is invalid"), -1)
		return types.ActionContinue
	}

	// reset authorization
	_ = proxywasm.RemoveHttpRequestHeader("Authorization")
	_ = proxywasm.RemoveHttpRequestHeader("Api-Key")
	_ = proxywasm.AddHttpResponseHeader("Api-Key", config.APIKey)

	// set api-version in queries
	query := u.Query()
	if query.Get("api-version") == "" {
		query.Set("api-version", config.APIVersion)
	}
	u.RawQuery = query.Encode()

	// set rewritten path
	var rewrite = matcher.AzurePathPattern
	rewrite = strings.ReplaceAll(rewrite, "${ path.model }", matcher.Values["model"])
	rewrite = strings.ReplaceAll(rewrite, "${ config.DevelopmentName }", config.DevelopmentName)
	u.Path = rewrite

	// reset uri
	_ = proxywasm.ReplaceHttpRequestHeader(":path", u.RequestURI())

	// rewrite host
	_ = proxywasm.ReplaceHttpRequestHeader(":host", config.AzureHost)

	// rewrite scheme
	_ = proxywasm.ReplaceHttpRequestHeader(":scheme", config.Scheme)

	return types.ActionContinue
}

func NewPathMatcher(openaiPathExpr, azurePathPattern string) *PathMatcher {
	re := regexp.MustCompile(openaiPathExpr)
	var pm = PathMatcher{
		OpenaiPathExpr:   openaiPathExpr,
		AzurePathPattern: azurePathPattern,
		Values:           make(map[string]string),
	}
	pm.match = func(path string) bool {
		if !re.MatchString(path) {
			return false
		}
		matches := re.FindAllStringSubmatch(path, -1)
		for _, subs := range matches {
			for i, name := range re.SubexpNames() {
				if i > 0 && i < len(subs) {
					pm.Values[name] = subs[i]
				}
			}
		}
		return true
	}
	return &pm
}

type PathMatcher struct {
	OpenaiPathExpr   string
	AzurePathPattern string
	Values           map[string]string

	match func(path string) bool
}

func (p *PathMatcher) Match(path string) bool {
	return p.match(path)
}
