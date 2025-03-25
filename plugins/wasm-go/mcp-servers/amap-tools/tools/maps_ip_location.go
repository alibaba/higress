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

type IPLocationRequest struct {
	IP string `json:"ip" jsonschema_description:"IP地址"`
}

func (t IPLocationRequest) Description() string {
	return "IP 定位根据用户输入的 IP 地址，定位 IP 的所在位置"
}

func (t IPLocationRequest) InputSchema() map[string]any {
	return wrapper.ToInputSchema(&IPLocationRequest{})
}

func (t IPLocationRequest) Create(params []byte) wrapper.MCPTool[server.AmapMCPServer] {
	request := &IPLocationRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t IPLocationRequest) Call(ctx wrapper.HttpContext, config server.AmapMCPServer) error {
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

	url := fmt.Sprintf("https://restapi.amap.com/v3/ip?ip=%s&key=%s&source=ts_mcp", url.QueryEscape(t.IP), apiKey)
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				ctx.OnMCPToolCallError(fmt.Errorf("ip location call failed, status: %d", statusCode))
				return
			}
			var response struct {
				Status string `json:"status"`
				Info   string `json:"info"`
				Province string `json:"province"`
				City     string `json:"city"`
				Adcode   string `json:"adcode"`
				Rectangle string `json:"rectangle"`
			}
			err := json.Unmarshal(responseBody, &response)
			if err != nil {
				ctx.OnMCPToolCallError(fmt.Errorf("failed to parse ip location response: %v", err))
				return
			}
			if response.Status != "1" {
				ctx.OnMCPToolCallError(fmt.Errorf("ip location failed: %s", response.Info))
				return
			}
			result := fmt.Sprintf(`{"province": "%s", "city": "%s", "adcode": "%s", "rectangle": "%s"}`, response.Province, response.City, response.Adcode, response.Rectangle)
			ctx.SendMCPToolTextResult(result)
		})
}