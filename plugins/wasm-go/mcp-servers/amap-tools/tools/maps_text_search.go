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

type TextSearchRequest struct {
	Keywords   string `json:"keywords" jsonschema_description:"搜索关键词"`
	City       string `json:"city" jsonschema_description:"查询城市"`
	Citylimit  string `json:"citylimit" jsonschema_description:"是否强制限制在设置的城市内搜索，默认值为false"`
}

func (t TextSearchRequest) Description() string {
	return "关键词搜，根据用户传入关键词，搜索出相关的POI"
}

func (t TextSearchRequest) InputSchema() map[string]any {
	return wrapper.ToInputSchema(&TextSearchRequest{})
}

func (t TextSearchRequest) Create(params []byte) wrapper.MCPTool[server.AmapMCPServer] {
	request := &TextSearchRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t TextSearchRequest) Call(ctx wrapper.HttpContext, config server.AmapMCPServer) error {
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

	url := fmt.Sprintf("http://restapi.amap.com/v3/place/text?key=%s&keywords=%s&city=%s&citylimit=%s&source=ts_mcp", apiKey, url.QueryEscape(t.Keywords), url.QueryEscape(t.City), url.QueryEscape(t.Citylimit))
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				ctx.OnMCPToolCallError(fmt.Errorf("text search call failed, status: %d", statusCode))
				return
			}
			var response struct {
				Status string `json:"status"`
				Info   string `json:"info"`
				Suggestion struct {
					Keywords []string `json:"keywords"`
					Cities   []struct {
						Name string `json:"name"`
					} `json:"cities"`
				} `json:"suggestion"`
				Pois []struct {
					ID       string `json:"id"`
					Name     string `json:"name"`
					Address  string `json:"address"`
					Typecode string `json:"typecode"`
				} `json:"pois"`
			}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				ctx.OnMCPToolCallError(fmt.Errorf("failed to parse text search response: %v", err))
				return
			}
			if response.Status != "1" {
				ctx.OnMCPToolCallError(fmt.Errorf("text search failed: %s", response.Info))
				return
			}
			var cities []string
			for _, city := range response.Suggestion.Cities {
				cities = append(cities, city.Name)
			}
			result := fmt.Sprintf(`{"suggestion": {"keywords": %s, "cities": %s}, "pois": %s}`, string(responseBody), string(responseBody), string(responseBody))
			ctx.SendMCPToolTextResult(result)
		})
}