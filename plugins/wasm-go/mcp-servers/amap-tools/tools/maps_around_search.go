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

type AroundSearchRequest struct {
	Location string `json:"location" jsonschema_description:"中心点经度纬度"`
	Radius   string `json:"radius" jsonschema_description:"搜索半径"`
	Keywords string `json:"keywords" jsonschema_description:"搜索关键词"`
}

func (t AroundSearchRequest) Description() string {
	return "周边搜，根据用户传入关键词以及坐标location，搜索出radius半径范围的POI"
}

func (t AroundSearchRequest) InputSchema() map[string]any {
	return wrapper.ToInputSchema(&AroundSearchRequest{})
}

func (t AroundSearchRequest) Create(params []byte) wrapper.MCPTool[server.AmapMCPServer] {
	request := &AroundSearchRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t AroundSearchRequest) Call(ctx wrapper.HttpContext, config server.AmapMCPServer) error {
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

	url := fmt.Sprintf("http://restapi.amap.com/v3/place/around?key=%s&location=%s&radius=%s&keywords=%s&source=ts_mcp", apiKey, url.QueryEscape(t.Location), url.QueryEscape(t.Radius), url.QueryEscape(t.Keywords))
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				ctx.OnMCPToolCallError(fmt.Errorf("around search call failed, status: %d", statusCode))
				return
			}
			var response struct {
				Status string `json:"status"`
				Info   string `json:"info"`
				Pois []struct {
					ID       string `json:"id"`
					Name     string `json:"name"`
					Address  string `json:"address"`
					Typecode string `json:"typecode"`
				} `json:"pois"`
			}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				ctx.OnMCPToolCallError(fmt.Errorf("failed to parse around search response: %v", err))
				return
			}
			if response.Status != "1" {
				ctx.OnMCPToolCallError(fmt.Errorf("around search failed: %s", response.Info))
				return
			}
			result := fmt.Sprintf(`{"pois": %s}`, string(responseBody))
			ctx.SendMCPToolTextResult(result)
		})
}