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

package configmap

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/alibaba/higress/pkg/ingress/kube/util"
	. "github.com/alibaba/higress/pkg/ingress/log"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/gvk"
)

// RedisConfig defines the configuration for Redis connection
type RedisConfig struct {
	// The address of Redis server in the format of "host:port"
	Address string `json:"address,omitempty"`
	// The username for Redis authentication
	Username string `json:"username,omitempty"`
	// The password for Redis authentication
	Password string `json:"password,omitempty"`
	// The database index to use
	DB int `json:"db,omitempty"`
}

// SSEServer defines the configuration for Server-Sent Events (SSE) server
type SSEServer struct {
	// The name of the SSE server
	Name string `json:"name,omitempty"`
	// The path where the SSE server will be mounted, the full path is (PATH + SsePathSuffix)
	Path string `json:"path,omitempty"`
	// The type of the SSE server
	Type string `json:"type,omitempty"`
	// Additional Config parameters for the real MCP server implementation
	Config map[string]interface{} `json:"config,omitempty"`
}

// McpServer defines the configuration for MCP (Model Context Protocol) server
type McpServer struct {
	// Flag to control whether MCP server is enabled
	Enable bool `json:"enable,omitempty"`
	// Redis Config for MCP server
	Redis *RedisConfig `json:"redis,omitempty"`
	// The suffix to be appended to SSE paths, default is "/sse"
	SsePathSuffix string `json:"sse_path_suffix,omitempty"`
	// List of SSE servers Configs
	Servers []*SSEServer `json:"servers,omitempty"`
}

func NewDefaultMcpServer() *McpServer {
	return &McpServer{Enable: false}
}

const (
	higressMcpServerEnvoyFilterName = "higress-config-mcp-server"
)

func validMcpServer(m *McpServer) error {
	if m == nil {
		return nil
	}

	if m.Enable && m.Redis == nil {
		return errors.New("redis config cannot be empty when mcp server is enabled")
	}

	return nil
}

func compareMcpServer(old *McpServer, new *McpServer) (Result, error) {
	if old == nil && new == nil {
		return ResultNothing, nil
	}

	if new == nil {
		return ResultDelete, nil
	}

	if !reflect.DeepEqual(old, new) {
		return ResultReplace, nil
	}

	return ResultNothing, nil
}

func deepCopyMcpServer(mcp *McpServer) (*McpServer, error) {
	newMcp := NewDefaultMcpServer()
	newMcp.Enable = mcp.Enable

	if mcp.Redis != nil {
		newMcp.Redis = &RedisConfig{
			Address:  mcp.Redis.Address,
			Username: mcp.Redis.Username,
			Password: mcp.Redis.Password,
			DB:       mcp.Redis.DB,
		}
	}

	newMcp.SsePathSuffix = mcp.SsePathSuffix

	if len(mcp.Servers) > 0 {
		newMcp.Servers = make([]*SSEServer, len(mcp.Servers))
		for i, server := range mcp.Servers {
			newServer := &SSEServer{
				Name: server.Name,
				Path: server.Path,
				Type: server.Type,
			}
			if server.Config != nil {
				newServer.Config = make(map[string]interface{})
				for k, v := range server.Config {
					newServer.Config[k] = v
				}
			}
			newMcp.Servers[i] = newServer
		}
	}

	return newMcp, nil
}

type McpServerController struct {
	Namespace    string
	mcpServer    atomic.Value
	Name         string
	eventHandler ItemEventHandler
}

func NewMcpServerController(namespace string) *McpServerController {
	mcpController := &McpServerController{
		Namespace: namespace,
		mcpServer: atomic.Value{},
		Name:      "mcpServer",
	}
	mcpController.SetMcpServer(NewDefaultMcpServer())
	return mcpController
}

func (m *McpServerController) GetName() string {
	return m.Name
}

func (m *McpServerController) SetMcpServer(mcp *McpServer) {
	m.mcpServer.Store(mcp)
}

func (m *McpServerController) GetMcpServer() *McpServer {
	value := m.mcpServer.Load()
	if value != nil {
		if mcp, ok := value.(*McpServer); ok {
			return mcp
		}
	}
	return nil
}

func (m *McpServerController) AddOrUpdateHigressConfig(name util.ClusterNamespacedName, old *HigressConfig, new *HigressConfig) error {
	if err := validMcpServer(new.McpServer); err != nil {
		IngressLog.Errorf("data:%+v convert to mcp server, error: %+v", new.McpServer, err)
		return nil
	}

	result, _ := compareMcpServer(old.McpServer, new.McpServer)

	switch result {
	case ResultReplace:
		if newMcp, err := deepCopyMcpServer(new.McpServer); err != nil {
			IngressLog.Infof("mcp server deepcopy error:%v", err)
		} else {
			m.SetMcpServer(newMcp)
			IngressLog.Infof("AddOrUpdate Higress config mcp server")
			m.eventHandler(higressMcpServerEnvoyFilterName)
			IngressLog.Infof("send event with filter name:%s", higressMcpServerEnvoyFilterName)
		}
	case ResultDelete:
		m.SetMcpServer(NewDefaultMcpServer())
		IngressLog.Infof("Delete Higress config mcp server")
		m.eventHandler(higressMcpServerEnvoyFilterName)
		IngressLog.Infof("send event with filter name:%s", higressMcpServerEnvoyFilterName)
	}

	return nil
}

func (m *McpServerController) ValidHigressConfig(higressConfig *HigressConfig) error {
	if higressConfig == nil {
		return nil
	}
	if higressConfig.McpServer == nil {
		return nil
	}

	return validMcpServer(higressConfig.McpServer)
}

func (m *McpServerController) RegisterItemEventHandler(eventHandler ItemEventHandler) {
	m.eventHandler = eventHandler
}

func (m *McpServerController) ConstructEnvoyFilters() ([]*config.Config, error) {
	configs := make([]*config.Config, 0)
	mcpServer := m.GetMcpServer()
	namespace := m.Namespace

	if mcpServer == nil || !mcpServer.Enable {
		return configs, nil
	}

	mcpStruct := m.constructMcpServerStruct(mcpServer)
	if mcpStruct == "" {
		return configs, nil
	}

	config := &config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.EnvoyFilter,
			Name:             higressMcpServerEnvoyFilterName,
			Namespace:        namespace,
		},
		Spec: &networking.EnvoyFilter{
			ConfigPatches: []*networking.EnvoyFilter_EnvoyConfigObjectPatch{
				{
					ApplyTo: networking.EnvoyFilter_HTTP_FILTER,
					Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
						Context: networking.EnvoyFilter_GATEWAY,
						ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
							Listener: &networking.EnvoyFilter_ListenerMatch{
								FilterChain: &networking.EnvoyFilter_ListenerMatch_FilterChainMatch{
									Filter: &networking.EnvoyFilter_ListenerMatch_FilterMatch{
										Name: "envoy.filters.network.http_connection_manager",
										SubFilter: &networking.EnvoyFilter_ListenerMatch_SubFilterMatch{
											Name: "envoy.filters.http.cors",
										},
									},
								},
							},
						},
					},
					Patch: &networking.EnvoyFilter_Patch{
						Operation: networking.EnvoyFilter_Patch_INSERT_AFTER,
						Value:     util.BuildPatchStruct(mcpStruct),
					},
				},
			},
		},
	}

	configs = append(configs, config)
	return configs, nil
}

func (m *McpServerController) constructMcpServerStruct(mcp *McpServer) string {
	// 构建 servers 配置
	servers := "[]"
	if len(mcp.Servers) > 0 {
		serverConfigs := make([]string, len(mcp.Servers))
		for i, server := range mcp.Servers {
			serverConfig := fmt.Sprintf(`{
				"name": "%s",
				"path": "%s",
				"type": "%s"`,
				server.Name, server.Path, server.Type)

			if len(server.Config) > 0 {
				config, _ := json.Marshal(server.Config)
				serverConfig += fmt.Sprintf(`,
				"config": %s`, string(config))
			}

			serverConfig += "}"
			serverConfigs[i] = serverConfig
		}
		servers = fmt.Sprintf("[%s]", strings.Join(serverConfigs, ","))
	}

	// 构建完整的配置结构
	structFmt := `{
		"name": "envoy.filters.http.golang",
		"typed_config": {
			"@type": "type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config",
			"value": {
				"library_id": "mcp-server",
				"library_path": "/var/lib/istio/envoy/mcp-server.so",
				"plugin_name": "mcp-server",
				"plugin_config": {
					"@type": "type.googleapis.com/xds.type.v3.TypedStruct",
					"value": {
						"redis": {
							"address": "%s",
							"username": "%s",
							"password": "%s",
							"db": %d
						},
						"sse_path_suffix": "%s",
						"servers": %s
					}
				}
			}
		}
	}`

	return fmt.Sprintf(structFmt,
		mcp.Redis.Address,
		mcp.Redis.Username,
		mcp.Redis.Password,
		mcp.Redis.DB,
		mcp.SsePathSuffix,
		servers)
}
