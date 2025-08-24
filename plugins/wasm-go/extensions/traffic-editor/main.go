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

	"github.com/alibaba/higress/plugins/wasm-go/extensions/traffic-editor/pkg"
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
	log.Debugf("onHttpRequestHeaders called with config")

	editorContext := pkg.NewEditorContext()
	if headers, err := proxywasm.GetHttpRequestHeaders(); err == nil {
		editorContext.SetRequestHeaders(headerSlice2Map(headers))
	} else {
		log.Errorf("failed to get request headers: %v", err)
	}
	if paths := editorContext.GetRequestHeader(pathHeader); len(paths) == 0 || paths[0] == "" {
		log.Warn("the request has an empty path")
	} else {
		path := paths[0]
		editorContext.SetRequestPath(path)
		if queries, err := extractRequestQueries(path); err == nil {
			editorContext.SetRequestQueries(queries)
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
	log.Debugf("an effective command set found for request %s with %d commands", ctx.Path(), len(effectiveCommandSet.Commands))
	editorContext.SetEffectiveCommandSet(effectiveCommandSet)
	editorContext.SetCommandExecutors(effectiveCommandSet.CreatExecutors())

	executeCommands(editorContext, pkg.StageRequestHeaders)

	if err := saveRequestMetaChanges(editorContext); err != nil {
		log.Errorf("failed to save request meta changes: %v", err)
	}

	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config PluginConfig) types.Action {
	log.Debugf("onHttpResponseHeaders called with config")

	editorContext := loadEditorContext(ctx)
	if editorContext.GetEffectiveCommandSet() == nil {
		log.Debugf("no effective command set found for request %s", ctx.Path())
		return types.ActionContinue
	}

	if headers, err := proxywasm.GetHttpResponseHeaders(); err == nil {
		editorContext.SetResponseHeaders(headerSlice2Map(headers))
	} else {
		log.Errorf("failed to get response headers: %v", err)
	}

	executeCommands(editorContext, pkg.StageResponseHeaders)
	if err := saveResponseMetaChanges(editorContext); err != nil {
		log.Errorf("failed to save request meta changes: %v", err)
	}

	return types.ActionContinue
}

func findEffectiveCommandSet(editorContext pkg.EditorContext, config *PluginConfig) *pkg.CommandSet {
	if config == nil {
		return nil
	}
	if len(config.ConditionalConfigs) != 0 {
		for i, conditionalConfig := range config.ConditionalConfigs {
			log.Debugf("Evaluating conditional config %d: %+v", i, conditionalConfig)
			if conditionalConfig.Matches(editorContext) {
				log.Debugf("Use the conditional command set %d", i)
				return &conditionalConfig.CommandSet
			}
		}
	}
	log.Debugf("Use the default command set")
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

func executeCommands(editorContext pkg.EditorContext, stage pkg.Stage) {
	for _, executor := range editorContext.GetCommandExecutors() {
		if err := executor.Run(editorContext, stage); err != nil {
			log.Errorf("failed to execute a %s command in stage %s: %v", executor.GetCommand().GetType(), pkg.Stage2String[stage], err)
		}
	}
}

func saveRequestMetaChanges(editorContext pkg.EditorContext) error {
	needSetHeaders := false
	if editorContext.IsRequestHeadersDirty() {
		needSetHeaders = true
	}
	if editorContext.IsRequestQueriesDirty() {
		u, err := url.Parse(editorContext.GetRequestPath())
		if err != nil {
			return fmt.Errorf("failed to build the new path for query string changes: %v", err)
		}

		query := url.Values{}
		for k, vs := range editorContext.GetRequestQueries() {
			for _, v := range vs {
				query.Add(k, v)
			}
		}
		u.RawQuery = query.Encode()
		editorContext.SetRefValue(&pkg.Ref{Type: pkg.RefTypeRequestHeader, Name: pathHeader}, u.String())
		needSetHeaders = true
	}
	if !needSetHeaders {
		log.Debugf("no request meta changes to save")
		return nil
	}
	log.Debugf("saving request meta changes: %v", editorContext.GetRequestHeaders())
	headerSlice := headerMap2Slice(editorContext.GetRequestHeaders())
	return proxywasm.ReplaceHttpRequestHeaders(headerSlice)
}

func saveResponseMetaChanges(editorContext pkg.EditorContext) error {
	if !editorContext.IsResponseHeadersDirty() {
		log.Debugf("no response meta changes to save")
		return nil
	}
	log.Debugf("saving response meta changes: %v", editorContext.GetResponseHeaders())
	headerSlice := headerMap2Slice(editorContext.GetResponseHeaders())
	return proxywasm.ReplaceHttpResponseHeaders(headerSlice)
}

func loadEditorContext(ctx wrapper.HttpContext) pkg.EditorContext {
	editorContext, _ := ctx.GetContext(ctxKeyEditorContext).(pkg.EditorContext)
	return editorContext
}

func saveEditorContext(ctx wrapper.HttpContext, editorContext pkg.EditorContext) {
	ctx.SetContext(ctxKeyEditorContext, editorContext)
}
