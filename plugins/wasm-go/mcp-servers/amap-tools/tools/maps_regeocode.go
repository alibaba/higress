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

type ReGeocodeRequest struct {
	Location string `json:"location" jsonschema_description:"经纬度"`
}

func (t ReGeocodeRequest) Description() string {
	return "将一个高德经纬度坐标转换为行政区划地址信息"
}

func (t ReGeocodeRequest) InputSchema() map[string]any {
	return wrapper.ToInputSchema(&ReGeocodeRequest{})
}

func (t ReGeocodeRequest) Create(params []byte) wrapper.MCPTool[server.AmapMCPServer] {
	request := &ReGeocodeRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t ReGeocodeRequest) Call(ctx wrapper.HttpContext, config server.AmapMCPServer) error {
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

	url := fmt.Sprintf("http://restapi.amap.com/v3/geocode/regeo?location=%s&key=%s&source=ts_mcp", url.QueryEscape(t.Location), apiKey)
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				ctx.OnMCPToolCallError(fmt.Errorf("regeocode call failed, status: %d", statusCode))
				return
			}
			var response struct {
				Status string `json:"status"`
				Info   string `json:"info"`
				Regeocode struct {
					AddressComponent struct {
						Province string `json:"province"`
						City     string `json:"city"`
						District string `json:"district"`
					} `json:"addressComponent"`
				} `json:"regeocode"`
			}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				ctx.OnMCPToolCallError(fmt.Errorf("failed to parse regeocode response: %v", err))
				return
			}
			if response.Status != "1" {
				ctx.OnMCPToolCallError(fmt.Errorf("regeocode failed: %s", response.Info))
				return
			}
			result := fmt.Sprintf(`{"province": "%s", "city": "%s", "district": "%s"}`, response.Regeocode.AddressComponent.Province, response.Regeocode.AddressComponent.City, response.Regeocode.AddressComponent.District)
			ctx.SendMCPToolTextResult(result)
		})
}