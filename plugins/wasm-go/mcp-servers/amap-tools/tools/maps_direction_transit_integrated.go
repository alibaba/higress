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

type TransitIntegratedRequest struct {
	Origin      string `json:"origin" jsonschema_description:"出发点经纬度，坐标格式为：经度，纬度"`
	Destination string `json:"destination" jsonschema_description:"目的地经纬度，坐标格式为：经度，纬度"`
	City        string `json:"city" jsonschema_description:"公共交通规划起点城市"`
	Cityd       string `json:"cityd" jsonschema_description:"公共交通规划终点城市"`
}

func (t TransitIntegratedRequest) Description() string {
	return "公交路径规划 API 可以根据用户起终点经纬度坐标规划综合各类公共（火车、公交、地铁）交通方式的通勤方案，并且返回通勤方案的数据，跨城场景下必须传起点城市与终点城市"
}

func (t TransitIntegratedRequest) InputSchema() map[string]any {
	return wrapper.ToInputSchema(&TransitIntegratedRequest{})
}

func (t TransitIntegratedRequest) Create(params []byte) wrapper.MCPTool[server.AmapMCPServer] {
	request := &TransitIntegratedRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t TransitIntegratedRequest) Call(ctx wrapper.HttpContext, config server.AmapMCPServer) error {
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

	url := fmt.Sprintf("http://restapi.amap.com/v3/direction/transit/integrated?key=%s&origin=%s&destination=%s&city=%s&cityd=%s&source=ts_mcp", apiKey, url.QueryEscape(t.Origin), url.QueryEscape(t.Destination), url.QueryEscape(t.City), url.QueryEscape(t.Cityd))
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				ctx.OnMCPToolCallError(fmt.Errorf("transit integrated call failed, status: %d", statusCode))
				return
			}
			var response struct {
				Status string `json:"status"`
				Info   string `json:"info"`
				Route struct {
					Origin string `json:"origin"`
					Destination string `json:"destination"`
					Distance string `json:"distance"`
					Transits []struct {
						Duration string `json:"duration"`
						WalkingDistance string `json:"walking_distance"`
						Segments []struct {
							Walking struct {
								Origin string `json:"origin"`
								Destination string `json:"destination"`
								Distance string `json:"distance"`
								Duration string `json:"duration"`
								Steps []struct {
									Instruction string `json:"instruction"`
									Road string `json:"road"`
									Distance string `json:"distance"`
									Action string `json:"action"`
									AssistantAction string `json:"assistant_action"`
								} `json:"steps"`
							} `json:"walking"`
							Bus struct {
								Buslines []struct {
									Name string `json:"name"`
									DepartureStop struct {
										Name string `json:"name"`
									} `json:"departure_stop"`
									ArrivalStop struct {
										Name string `json:"name"`
									} `json:"arrival_stop"`
									Distance string `json:"distance"`
									Duration string `json:"duration"`
									ViaStops []struct {
										Name string `json:"name"`
									} `json:"via_stops"`
								} `json:"buslines"`
							} `json:"bus"`
							Entrance struct {
								Name string `json:"name"`
							} `json:"entrance"`
							Exit struct {
								Name string `json:"name"`
							} `json:"exit"`
							Railway struct {
								Name string `json:"name"`
								Trip string `json:"trip"`
							} `json:"railway"`
						} `json:"segments"`
					} `json:"transits"`
				} `json:"route"`
			}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				ctx.OnMCPToolCallError(fmt.Errorf("failed to parse transit integrated response: %v", err))
				return
			}
			if response.Status != "1" {
				ctx.OnMCPToolCallError(fmt.Errorf("transit integrated failed: %s", response.Info))
				return
			}
			result := fmt.Sprintf(`{"origin": "%s", "destination": "%s", "distance": "%s", "transits": %s}`, response.Route.Origin, response.Route.Destination, response.Route.Distance, string(responseBody))
			ctx.SendMCPToolTextResult(result)
		})
}