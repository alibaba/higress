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

type WalkingRequest struct {
	Origin      string `json:"origin" jsonschema_description:"出发点经度，纬度，坐标格式为：经度，纬度"`
	Destination string `json:"destination" jsonschema_description:"目的地经纬度，坐标格式为：经度，纬度"`
}

func (t WalkingRequest) Description() string {
	return "步行路径规划 API 可以根据输入起点终点经纬度坐标规划100km 以内的步行通勤方案，并且返回通勤方案的数据"
}

func (t WalkingRequest) InputSchema() map[string]any {
	return wrapper.ToInputSchema(&WalkingRequest{})
}

func (t WalkingRequest) Create(params []byte) wrapper.MCPTool[server.AmapMCPServer] {
	request := &WalkingRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t WalkingRequest) Call(ctx wrapper.HttpContext, config server.AmapMCPServer) error {
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

	url := fmt.Sprintf("http://restapi.amap.com/v3/direction/walking?key=%s&origin=%s&destination=%s&source=ts_mcp", apiKey, url.QueryEscape(t.Origin), url.QueryEscape(t.Destination))
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				ctx.OnMCPToolCallError(fmt.Errorf("walking call failed, status: %d", statusCode))
				return
			}
			var response struct {
				Status string `json:"status"`
				Info   string `json:"info"`
				Route struct {
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
				} `json:"route"`
			}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				ctx.OnMCPToolCallError(fmt.Errorf("failed to parse walking response: %v", err))
				return
			}
			if response.Status != "1" {
				ctx.OnMCPToolCallError(fmt.Errorf("walking failed: %s", response.Info))
				return
			}
			result := fmt.Sprintf(`{"origin": "%s", "destination": "%s", "paths": %s}`, response.Route.Origin, response.Route.Destination, string(responseBody))
			ctx.SendMCPToolTextResult(result)
		})
}