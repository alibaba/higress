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
	"net/url"

	"github.com/tidwall/gjson"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

const (
	ctxKeyEditorContext = "editorContext"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"traffic-editor",
		wrapper.ParseConfig(parseConfig),
		wrapper.ProcessRequestHeaders(onHttpRequestHeaders),
		wrapper.ProcessResponseHeaders(onHttpResponseHeaders),
	)
}

func parseConfig(json gjson.Result, config *PluginConfig) (err error) {
	if err := config.FromJson(json); err != nil {
		return fmt.Errorf("failed to parse plugin config: %v", err)
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
	editorContext := &EditorContext{}
	if headers, err := proxywasm.GetHttpRequestHeaders(); err == nil {
		editorContext.requestHeaders = headerSlice2Map(headers)
	} else {
		log.Errorf("failed to get request headers: %v", err)
	}
	if paths := editorContext.requestHeaders[pathHeader]; len(paths) == 0 || paths[0] == "" {
		log.Warn("the request has an empty path")
	} else {
		path := paths[0]
		editorContext.requestPath = path
		if queries, err := extractRequestQueries(path); err == nil {
			editorContext.requestQueries = queries
		} else {
			log.Errorf("failed to get request queries: %v", err)
		}
	}
	saveEditorContext(ctx, editorContext)

	effectiveCommandSet := findEffectiveCommandSet(editorContext, &config)
	if effectiveCommandSet == nil {
		log.Debugf("no effective command set found for request %s", ctx.Path())
		return types.ActionContinue
	}
	if len(effectiveCommandSet.Commands) == 0 {
		log.Debugf("the effective command set found for request %s is empty", ctx.Path())
		return types.ActionContinue
	}
	editorContext.effectiveCommandSet = effectiveCommandSet
	editorContext.commandExecutors = effectiveCommandSet.CreatExecutors()

	executeCommands(editorContext, StageRequestHeaders)

	if err := saveRequestMetaChanges(editorContext); err != nil {
		log.Errorf("failed to save request meta changes: %v", err)
	}

	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
	editorContext := loadEditorContext(ctx)
	if editorContext.effectiveCommandSet == nil {
		return types.ActionContinue
	}

	if headers, err := proxywasm.GetHttpResponseHeaders(); err == nil {
		editorContext.responseHeaders = headerSlice2Map(headers)
	} else {
		log.Errorf("failed to get response headers: %v", err)
	}

	executeCommands(editorContext, StageResponseHeaders)
	if err := saveResponseMetaChanges(editorContext); err != nil {
		log.Errorf("failed to save request meta changes: %v", err)
	}

	return types.ActionContinue
}

func findEffectiveCommandSet(editorContext *EditorContext, config *PluginConfig) *CommandSet {
	if config == nil {
		return nil
	}
	if len(config.ConditionalConfigs) != 0 {
		for _, conditionalConfig := range config.ConditionalConfigs {
			if conditionalConfig.Matches(editorContext) {
				return &conditionalConfig.CommandSet
			}
		}
	}
	return config.DefaultConfig
}

func extractRequestQueries(path string) (map[string][]string, error) {
	queries := make(map[string][]string)

	if path == "" {
		return queries, nil
	}

	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	for k, vs := range u.Query() {
		queries[k] = vs
	}
	return queries, nil
}

func executeCommands(editorContext *EditorContext, stage Stage) {
	for _, executor := range editorContext.commandExecutors {
		if err := executor.Run(editorContext, stage); err != nil {
			log.Errorf("failed to execute a %s command in stage %s: %v", executor.GetCommand().GetType(), Stage2String[stage], err)
		}
	}
}

func saveRequestMetaChanges(editorContext *EditorContext) error {
	needSetHeaders := false
	if editorContext.requestHeadersDirty {
		needSetHeaders = true
	}
	if editorContext.requestQueriesDirty {
		u, err := url.Parse(editorContext.requestPath)
		if err != nil {
			return fmt.Errorf("failed to build the new path for query string changes: %v", err)
		}

		query := url.Values{}
		for k, vs := range editorContext.requestQueries {
			for _, v := range vs {
				query.Add(k, v)
			}
		}
		u.RawQuery = query.Encode()
		editorContext.requestHeaders[pathHeader] = []string{u.String()}
	}
	if !needSetHeaders {
		return nil
	}
	headerSlice := headerMap2Slice(editorContext.requestHeaders)
	return proxywasm.ReplaceHttpRequestHeaders(headerSlice)
}

func saveResponseMetaChanges(editorContext *EditorContext) error {
	return nil
}

func loadEditorContext(ctx wrapper.HttpContext) *EditorContext {
	editorContext, _ := ctx.GetContext(ctxKeyEditorContext).(*EditorContext)
	return editorContext
}

func saveEditorContext(ctx wrapper.HttpContext, editorContext *EditorContext) {
	ctx.SetContext(ctxKeyEditorContext, editorContext)
}
