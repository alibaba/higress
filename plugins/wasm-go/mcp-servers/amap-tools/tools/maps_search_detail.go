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

type SearchDetailRequest struct {
	ID string `json:"id" jsonschema_description:"关键词搜或者周边搜获取到的POI ID"`
}

func (t SearchDetailRequest) Description() string {
	return "查询关键词搜或者周边搜获取到的POI ID的详细信息"
}

func (t SearchDetailRequest) InputSchema() map[string]any {
	return wrapper.ToInputSchema(&SearchDetailRequest{})
}

func (t SearchDetailRequest) Create(params []byte) wrapper.MCPTool[server.AmapMCPServer] {
	request := &SearchDetailRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t SearchDetailRequest) Call(ctx wrapper.HttpContext, config server.AmapMCPServer) error {
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

	url := fmt.Sprintf("http://restapi.amap.com/v3/place/detail?id=%s&key=%s&source=ts_mcp", url.QueryEscape(t.ID), apiKey)
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				ctx.OnMCPToolCallError(fmt.Errorf("search detail call failed, status: %d", statusCode))
				return
			}
			var response struct {
				Status string `json:"status"`
				Info   string `json:"info"`
				Pois []struct {
					ID string `json:"id"`
					Name string `json:"name"`
					Location string `json:"location"`
					Address string `json:"address"`
					BusinessArea string `json:"business_area"`
					Cityname string `json:"cityname"`
					Type string `json:"type"`
					Alias string `json:"alias"`
					BizExt map[string]string `json:"biz_ext"`
				} `json:"pois"`
			}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				ctx.OnMCPToolCallError(fmt.Errorf("failed to parse search detail response: %v", err))
				return
			}
			if response.Status != "1" {
				ctx.OnMCPToolCallError(fmt.Errorf("search detail failed: %s", response.Info))
				return
			}
			poi := response.Pois[0]
			result := fmt.Sprintf(`{"id": "%s", "name": "%s", "location": "%s", "address": "%s", "business_area": "%s", "city": "%s", "type": "%s", "alias": "%s", "biz_ext": %s}`, poi.ID, poi.Name, poi.Location, poi.Address, poi.BusinessArea, poi.Cityname, poi.Type, poi.Alias, string(responseBody))
			ctx.SendMCPToolTextResult(result)
		})
}