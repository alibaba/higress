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
	"net/http"
	"net/url"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"quark-mcp-server",
		wrapper.ParseConfig(parseConfig),
		wrapper.AddMCPTool("web_search", WebSearch{}),
	)
}

type QuarkMCPServer struct {
	apiKey string
}

func parseConfig(json gjson.Result, server *QuarkMCPServer) error {
	server.apiKey = json.Get("apiKey").String()
	if server.apiKey == "" {
		return errors.New("QuarkMCPServer need apikey")
	}
	return nil
}

type WebSearch struct {
	Query       string `json:"query" jsonschema_description:"Search query, please use Chinese" jsonschema:"example=黄金价格走势"`
	ContentMode string `json:"contentMode,omitempty" jsonschema_description:"Return the level of content detail, choose to use summary or full text" jsonschema:"enum=full,enum=summary,default=summary"`
	Number      uint32 `json:"number,omitempty" jsonschema_description:"Number of results" jsonschema:"default=5"`
}

func (t WebSearch) Description() string {
	return `Performs a web search using the Quark Search API, ideal for general queries, news, articles, and online content.
Use this for broad information gathering, recent events, or when you need diverse web sources.
Because Quark search performs poorly for English searches, please use Chinese for the query parameters.`
}

func (t WebSearch) InputSchema() map[string]any {
	return wrapper.ToInputSchema(&WebSearch{})
}

func (t WebSearch) Create(params []byte) wrapper.MCPTool[QuarkMCPServer] {
	webSearch := &WebSearch{
		ContentMode: "summary",
		Number:      5,
	}
	json.Unmarshal(params, &webSearch)
	return webSearch
}

type SearchResult struct {
	Title   string
	Link    string
	Content string
}

func (result SearchResult) Valid() bool {
	return result.Title != "" && result.Link != "" && result.Content != ""
}

func (result SearchResult) Format() string {
	return fmt.Sprintf(`
## Title: %s

### Reference URL
%s

### Content
%s
`, result.Title, result.Link, result.Content)
}

func (t WebSearch) Call(ctx wrapper.HttpContext, server QuarkMCPServer) error {
	return ctx.RouteCall(http.MethodGet, fmt.Sprintf("https://cloud-iqs.aliyuncs.com/search/genericSearch?query=%s", url.QueryEscape(t.Query)),
		[][2]string{{"Accept", "application/json"},
			{"X-API-Key", server.apiKey}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				ctx.OnMCPToolCallError(fmt.Errorf("quark search call failed, status: %d", statusCode))
				return
			}
			jsonObj := gjson.ParseBytes(responseBody)
			var results []string
			for index, item := range jsonObj.Get("pageItems").Array() {
				var content string
				if t.ContentMode == "full" {
					content = item.Get("markdownText").String()
					if content == "" {
						content = item.Get("mainText").String()
					}
				} else if t.ContentMode == "summary" {
					content = item.Get("snippet").String()
				}
				result := SearchResult{
					Title:   item.Get("title").String(),
					Link:    item.Get("link").String(),
					Content: content,
				}
				if result.Valid() && index < int(t.Number) {
					results = append(results, result.Format())
				}
			}
			ctx.SendMCPToolTextResult(fmt.Sprintf("# Search Results\n\n%s", strings.Join(results, "\n\n")))
		})
}
