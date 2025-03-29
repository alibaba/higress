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
)

var _ server.Tool = GeoRequest{}

type GeoRequest struct {
	Address string `json:"address" jsonschema_description:"待解析的结构化地址信息"`
	City    string `json:"city" jsonschema_description:"指定查询的城市"`
}

func (t GeoRequest) Description() string {
	return "将详细的结构化地址转换为经纬度坐标。支持对地标性名胜景区、建筑物名称解析为经纬度坐标"
}

func (t GeoRequest) InputSchema() map[string]any {
	return server.ToInputSchema(&GeoRequest{})
}

func (t GeoRequest) Create(params []byte) server.Tool {
	request := &GeoRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t GeoRequest) Call(ctx server.HttpContext, s server.Server) error {
	serverConfig := &config.AmapServerConfig{}
	s.GetConfig(serverConfig)
	if serverConfig.ApiKey == "" {
		return errors.New("amap API-KEY is not configured")
	}

	apiKey := serverConfig.ApiKey
	url := fmt.Sprintf("https://restapi.amap.com/v3/geocode/geo?key=%s&address=%s&city=%s&source=ts_mcp", apiKey, url.QueryEscape(t.Address), url.QueryEscape(t.City))
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				utils.OnMCPToolCallError(ctx, fmt.Errorf("geo call failed, status: %d", statusCode))
				return
			}
			var response struct {
				Status   string `json:"status"`
				Info     string `json:"info"`
				Geocodes []struct {
					Country  string `json:"country"`
					Province string `json:"province"`
					City     string `json:"city"`
					Citycode string `json:"citycode"`
					District string `json:"district"`
					Street   string `json:"street"`
					Number   string `json:"number"`
					Adcode   string `json:"adcode"`
					Location string `json:"location"`
					Level    string `json:"level"`
				} `json:"geocodes"`
			}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				utils.OnMCPToolCallError(ctx, fmt.Errorf("failed to parse geo response: %v", err))
				return
			}
			if response.Status != "1" {
				utils.OnMCPToolCallError(ctx, fmt.Errorf("geo failed: %s", response.Info))
				return
			}
			var results []map[string]string
			for _, geo := range response.Geocodes {
				result := map[string]string{
					"country":  geo.Country,
					"province": geo.Province,
					"city":     geo.City,
					"citycode": geo.Citycode,
					"district": geo.District,
					"street":   geo.Street,
					"number":   geo.Number,
					"adcode":   geo.Adcode,
					"location": geo.Location,
					"level":    geo.Level,
				}
				results = append(results, result)
			}
			result, _ := json.Marshal(results)
			utils.SendMCPToolTextResult(ctx, string(result))
		})
}
