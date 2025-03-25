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

type DistanceRequest struct {
	Origins     string `json:"origins" jsonschema_description:"起点经度，纬度，可以传多个坐标，使用分号隔离，比如120,30;120,31，坐标格式为：经度，纬度"`
	Destination string `json:"destination" jsonschema_description:"终点经度，纬度，坐标格式为：经度，纬度"`
	Type        string `json:"type" jsonschema_description:"距离测量类型,1代表驾车距离测量，0代表直线距离测量，3步行距离测量"`
}

func (t DistanceRequest) Description() string {
	return "距离测量 API 可以测量两个经纬度坐标之间的距离,支持驾车、步行以及球面距离测量"
}

func (t DistanceRequest) InputSchema() map[string]any {
	return wrapper.ToInputSchema(&DistanceRequest{})
}

func (t DistanceRequest) Create(params []byte) wrapper.MCPTool[server.AmapMCPServer] {
	request := &DistanceRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t DistanceRequest) Call(ctx wrapper.HttpContext, config server.AmapMCPServer) error {
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

	url := fmt.Sprintf("http://restapi.amap.com/v3/distance?key=%s&origins=%s&destination=%s&type=%s&source=ts_mcp", apiKey, url.QueryEscape(t.Origins), url.QueryEscape(t.Destination), url.QueryEscape(t.Type))
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				ctx.OnMCPToolCallError(fmt.Errorf("distance call failed, status: %d", statusCode))
				return
			}
			var response struct {
				Status string `json:"status"`
				Info   string `json:"info"`
				Results []struct {
					OriginID string `json:"origin_id"`
					DestID   string `json:"dest_id"`
					Distance string `json:"distance"`
					Duration string `json:"duration"`
				} `json:"results"`
			}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				ctx.OnMCPToolCallError(fmt.Errorf("failed to parse distance response: %v", err))
				return
			}
			if response.Status != "1" {
				ctx.OnMCPToolCallError(fmt.Errorf("distance failed: %s", response.Info))
				return
			}
			result := fmt.Sprintf(`{"results": %s}`, string(responseBody))
			ctx.SendMCPToolTextResult(result)
		})
}