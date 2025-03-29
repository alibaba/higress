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

var _ server.Tool = WeatherRequest{}

type WeatherRequest struct {
	City string `json:"city" jsonschema_description:"城市名称或者adcode"`
}

func (t WeatherRequest) Description() string {
	return "根据城市名称或者标准adcode查询指定城市的天气"
}

func (t WeatherRequest) InputSchema() map[string]any {
	return server.ToInputSchema(&WeatherRequest{})
}

func (t WeatherRequest) Create(params []byte) server.Tool {
	request := &WeatherRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t WeatherRequest) Call(ctx server.HttpContext, s server.Server) error {
	serverConfig := &config.AmapServerConfig{}
	s.GetConfig(serverConfig)
	if serverConfig.ApiKey == "" {
		return errors.New("amap API-KEY is not configured")
	}

	url := fmt.Sprintf("http://restapi.amap.com/v3/weather/weatherInfo?city=%s&key=%s&source=ts_mcp&extensions=all", url.QueryEscape(t.City), serverConfig.ApiKey)
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				utils.OnMCPToolCallError(ctx, fmt.Errorf("weather call failed, status: %d", statusCode))
				return
			}
			var response struct {
				Status    string `json:"status"`
				Info      string `json:"info"`
				Forecasts []struct {
					City  string `json:"city"`
					Casts []struct {
						Date         string `json:"date"`
						Week         string `json:"week"`
						DayWeather   string `json:"dayweather"`
						NightWeather string `json:"nightweather"`
						DayTemp      string `json:"daytemp"`
						NightTemp    string `json:"nighttemp"`
						DayWind      string `json:"daywind"`
						NightWind    string `json:"nightwind"`
						DayPower     string `json:"daypower"`
						NightPower   string `json:"nightpower"`
						Humidity     string `json:"humidity"`
					} `json:"casts"`
				} `json:"forecasts"`
			}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				utils.OnMCPToolCallError(ctx, fmt.Errorf("failed to parse weather response: %v", err))
				return
			}
			if response.Status != "1" {
				utils.OnMCPToolCallError(ctx, fmt.Errorf("weather failed: %s", response.Info))
				return
			}
			forecasts := response.Forecasts[0]
			result, _ := json.MarshalIndent(forecasts, "", "  ")
			utils.SendMCPToolTextResult(ctx, string(result))
		})
}
