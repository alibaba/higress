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

package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"amap-tools/config"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/server"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/utils"
	"github.com/tidwall/gjson"
)

var _ server.Tool = TextSearchRequest{}

type TextSearchRequest struct {
	Keywords  string `json:"keywords" jsonschema_description:"搜索关键词"`
	City      string `json:"city" jsonschema_description:"查询城市"`
	Citylimit string `json:"citylimit" jsonschema_description:"是否强制限制在设置的城市内搜索，默认值为false"`
}

func (t TextSearchRequest) Description() string {
	return "关键词搜，根据用户传入关键词，搜索出相关的POI"
}

func (t TextSearchRequest) InputSchema() map[string]any {
	return server.ToInputSchema(&TextSearchRequest{})
}

func (t TextSearchRequest) Create(params []byte) server.Tool {
	request := &TextSearchRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t TextSearchRequest) Call(ctx server.HttpContext, s server.Server) error {
	serverConfig := &config.AmapServerConfig{}
	s.GetConfig(serverConfig)
	if serverConfig.ApiKey == "" {
		return errors.New("amap API-KEY is not configured")
	}

	url := fmt.Sprintf("http://restapi.amap.com/v3/place/text?key=%s&keywords=%s&city=%s&citylimit=%s&source=ts_mcp", serverConfig.ApiKey, url.QueryEscape(t.Keywords), url.QueryEscape(t.City), url.QueryEscape(t.Citylimit))
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				utils.OnMCPToolCallError(ctx, fmt.Errorf("text search call failed, status: %d", statusCode))
				return
			}
			var response struct {
				Status     string `json:"status"`
				Info       string `json:"info"`
				Suggestion struct {
					Keywords []string `json:"keywords"`
					Cities   []struct {
						Name string `json:"name"`
					} `json:"cities"`
				} `json:"suggestion"`
				Pois []struct {
					ID       string `json:"id"`
					Name     string `json:"name"`
					Address  string `json:"address"`
					Typecode string `json:"typecode"`
				} `json:"pois"`
			}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				utils.OnMCPToolCallError(ctx, fmt.Errorf("failed to parse text search response: %v", err))
				return
			}
			if response.Status != "1" {
				utils.OnMCPToolCallError(ctx, fmt.Errorf("text search failed: %s", response.Info))
				return
			}
			var cities []string
			for _, city := range response.Suggestion.Cities {
				cities = append(cities, city.Name)
			}
			result := fmt.Sprintf(`{"suggestion": {"keywords": %s, "cities": %s}, "pois": %s}`, string(responseBody), string(responseBody), gjson.GetBytes(responseBody, "pois").Raw)
			utils.SendMCPToolTextResult(ctx, result)
		})
}
