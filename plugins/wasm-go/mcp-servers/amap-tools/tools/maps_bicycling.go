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

type BicyclingRequest struct {
	Origin      string `json:"origin" jsonschema_description:"出发点经纬度，坐标格式为：经度，纬度"`
	Destination string `json:"destination" jsonschema_description:"目的地经纬度，坐标格式为：经度，纬度"`
}

func (t BicyclingRequest) Description() string {
	return "骑行路径规划用于规划骑行通勤方案，规划时会考虑天桥、单行线、封路等情况。最大支持 500km 的骑行路线规划"
}

func (t BicyclingRequest) InputSchema() map[string]any {
	return wrapper.ToInputSchema(&BicyclingRequest{})
}

func (t BicyclingRequest) Create(params []byte) wrapper.MCPTool[server.AmapMCPServer] {
	request := &BicyclingRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t BicyclingRequest) Call(ctx wrapper.HttpContext, config server.AmapMCPServer) error {
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

	url := fmt.Sprintf("http://restapi.amap.com/v4/direction/bicycling?key=%s&origin=%s&destination=%s&source=ts_mcp", apiKey, url.QueryEscape(t.Origin), url.QueryEscape(t.Destination))
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				ctx.OnMCPToolCallError(fmt.Errorf("bicycling call failed, status: %d", statusCode))
				return
			}
			var response struct {
				Errcode int `json:"errcode"`
				Data struct {
					Origin string `json:"origin"`
					Destination string `json:"destination"`
					Paths []struct {
						Distance string `json:"distance"`
						Duration string `json:"duration"`
						Steps []struct {
							Instruction string `json:"instruction"`
							Road string `json:"road"`
							Distance string `json:"distance"`
							Orientation string `json:"orientation"`
							Duration string `json:"duration"`
						} `json:"steps"`
					} `json:"paths"`
				} `json:"data"`
			}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				ctx.OnMCPToolCallError(fmt.Errorf("failed to parse bicycling response: %v", err))
				return
			}
			if response.Errcode != 0 {
				ctx.OnMCPToolCallError(fmt.Errorf("bicycling failed: %v", response))
				return
			}
			result := fmt.Sprintf(`{"origin": "%s", "destination": "%s", "paths": %s}`, response.Data.Origin, response.Data.Destination, string(responseBody))
			ctx.SendMCPToolTextResult(result)
		})
}