package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"amap-tools/server"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

type WeatherRequest struct {
	City string `json:"city" jsonschema_description:"城市名称或者adcode"`
}

func (t WeatherRequest) Description() string {
	return "根据城市名称或者标准adcode查询指定城市的天气"
}

func (t WeatherRequest) InputSchema() map[string]any {
	return wrapper.ToInputSchema(&WeatherRequest{})
}

func (t WeatherRequest) Create(params []byte) wrapper.MCPTool[server.AmapMCPServer] {
	request := &WeatherRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t WeatherRequest) Call(ctx wrapper.HttpContext, config server.AmapMCPServer) error {
	err := server.ParseFromRequest(ctx, &config)
	if err != nil {
		log.Errorf("parse config from request failed, err:%s", err)
		return err
	}
	err = config.ConfigHasError()
	if err != nil {
		return err
	}

	apiKey := config.ApiKey
	if apiKey == "" {
		return fmt.Errorf("amap API-KEY is not set")
	}

	url := fmt.Sprintf("http://restapi.amap.com/v3/weather/weatherInfo?city=%s&key=%s&source=ts_mcp&extensions=all", url.QueryEscape(t.City), apiKey)
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				ctx.OnMCPToolCallError(fmt.Errorf("weather call failed, status: %d", statusCode))
				return
			}
			var response struct {
				Status string `json:"status"`
				Info   string `json:"info"`
				Forecasts []struct {
					City string `json:"city"`
					Casts []struct {
						Date string `json:"date"`
						Week string `json:"week"`
						DayWeather string `json:"dayweather"`
						NightWeather string `json:"nightweather"`
						DayTemp string `json:"daytemp"`
						NightTemp string `json:"nighttemp"`
						DayWind string `json:"daywind"`
						NightWind string `json:"nightwind"`
						DayPower string `json:"daypower"`
						NightPower string `json:"nightpower"`
						Humidity string `json:"humidity"`
					} `json:"casts"`
				} `json:"forecasts"`
			}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				ctx.OnMCPToolCallError(fmt.Errorf("failed to parse weather response: %v", err))
				return
			}
			if response.Status != "1" {
				ctx.OnMCPToolCallError(fmt.Errorf("weather failed: %s", response.Info))
				return
			}
			forecasts := response.Forecasts[0]
			result := fmt.Sprintf(`{"city": "%s", "forecasts": %s}`, forecasts.City, string(responseBody))
			ctx.SendMCPToolTextResult(result)
		})
}