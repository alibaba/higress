// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"amap-tools/config"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/server"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/mcp/utils"
)

var _ server.Tool = IPLocationRequest{}

type IPLocationRequest struct {
	IP string `json:"ip" jsonschema_description:"IP地址"`
}

func (t IPLocationRequest) Description() string {
	return "IP 定位根据用户输入的 IP 地址，定位 IP 的所在位置"
}

func (t IPLocationRequest) InputSchema() map[string]any {
	return server.ToInputSchema(&IPLocationRequest{})
}

func (t IPLocationRequest) Create(params []byte) server.Tool {
	request := &IPLocationRequest{}
	json.Unmarshal(params, &request)
	return request
}

func (t IPLocationRequest) Call(ctx server.HttpContext, s server.Server) error {
	serverConfig := &config.AmapServerConfig{}
	s.GetConfig(serverConfig)
	if serverConfig.ApiKey == "" {
		return errors.New("amap API-KEY is not configured")
	}

	url := fmt.Sprintf("https://restapi.amap.com/v3/ip?ip=%s&key=%s&source=ts_mcp", url.QueryEscape(t.IP), serverConfig.ApiKey)
	return ctx.RouteCall(http.MethodGet, url,
		[][2]string{{"Accept", "application/json"}}, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != http.StatusOK {
				utils.OnMCPToolCallError(ctx, fmt.Errorf("ip location call failed, status: %d", statusCode))
				return
			}
			utils.SendMCPToolTextResult(ctx, string(responseBody))
		})
}
