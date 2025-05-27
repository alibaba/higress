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

package mcpserver

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

const McpServerVersionGroup = "mcp-server-versions"
const McpServerSpecGroup = "mcp-server"
const McpToolSpecGroup = "mcp-tools"
const SystemConfigIdPrefix = "system-"
const CredentialPrefix = "credentials-"

type ServerSpecInfo struct {
	RemoteServerConfig *RemoteServerConfig `json:"remoteServerConfig"`
}

type RemoteServerConfig struct {
	ServiceRef *ServiceRef `json:"serviceRef"`
}

type ServiceRef struct {
	ServiceName string `json:"serviceName"`
	GroupName   string `json:"groupName"`
	NamespaceId string `json:"namespaceId"`
}

type NacosRegistryClient struct {
	namespaceId  string
	configClient config_client.IConfigClient
	namingClient naming_client.INamingClient
	servers      map[string]*ServerContext
}

type VersionedMcpServerInfo struct {
	serverInfo *BasicMcpServerInfo
	version    string
}

type ServerContext struct {
	id                     string
	versionedMcpServerInfo *VersionedMcpServerInfo
	serverChangeListener   McpServerListener
	configsMap             map[string]*ConfigListenerWrap
	serviceInfo            *model.Service
	namingCallBck          func(services []model.Instance, err error)
}

type McpServerConfig struct {
	ServerSpecConfig string
	ToolsSpecConfig  string
	ServiceInfo      *model.Service
	Credentials      map[string]string
}

type ConfigListenerWrap struct {
	dataId   string
	group    string
	data     string
	listener func(namespace, group, dataId, data string)
}

type BasicMcpServerInfo struct {
	Name          string `json:"name"`
	Id            string `json:"id"`
	FrontProtocol string `json:"frontProtocol"`
	Protocol      string `json:"protocol"`
}

type VersionsMcpServerInfo struct {
	BasicMcpServerInfo
	Versions []*VersionDetail `json:"versionDetails"`
}

type VersionDetail struct {
	Version  string `json:"version"`
	IsLatest bool   `json:"is_latest"`
}

type McpServerListener func(info *McpServerConfig)

func NewMcpRegistryClient(clientConfig *constant.ClientConfig, serverConfig []constant.ServerConfig, namespaceId string) (*NacosRegistryClient, error) {
	clientParam := vo.NacosClientParam{
		ClientConfig:  clientConfig,
		ServerConfigs: serverConfig,
	}
	configClient, err := clients.NewConfigClient(clientParam)
	namingClient, err := clients.NewNamingClient(clientParam)

	if err != nil {
		return nil, err
	}

	return &NacosRegistryClient{
		namespaceId:  namespaceId,
		configClient: configClient,
		namingClient: namingClient,
		servers:      map[string]*ServerContext{},
	}, nil
}

func (n *NacosRegistryClient) ListMcpServer() ([]BasicMcpServerInfo, error) {
	configPage, err := n.configClient.SearchConfig(vo.SearchConfigParam{
		Search:   "blur",
		DataId:   "",
		Group:    McpServerVersionGroup,
		PageNo:   1,
		PageSize: 1000,
	})

	if err != nil {
		return nil, err
	}

	var result []BasicMcpServerInfo
	for _, config := range configPage.PageItems {
		mcpServerBasicConfig, err := n.configClient.GetConfig(vo.ConfigParam{
			Group:  McpServerVersionGroup,
			DataId: config.DataId,
		})

		if err != nil {
			logger.Errorf("Get mcpserver version config error")
			continue
		}

		mcpServer := BasicMcpServerInfo{}
		err = json.Unmarshal([]byte(mcpServerBasicConfig), &mcpServer)
		if err != nil {
			logger.Errorf("Parse mcp server version config error %v", err)
			continue
		}

		result = append(result, mcpServer)
	}
	return result, nil
}

func (n *NacosRegistryClient) ListenToMcpServer(id string, listener McpServerListener) error {
	versionConfigId := fmt.Sprintf("%s-mcp-versions.json", id)
	serverVersionConfig, err := n.configClient.GetConfig(vo.ConfigParam{
		Group:  McpServerVersionGroup,
		DataId: versionConfigId,
	})

	versionConfigCallBack := func(namespace string, group string, dataId string, content string) {
		info := VersionsMcpServerInfo{}
		err = json.Unmarshal([]byte(content), &info)
		if err != nil {
			// todo handle err
		}

		var latestVersion string
		for _, data := range info.Versions {
			if data.IsLatest {
				latestVersion = data.Version
				break
			}
		}

		ctx := n.servers[id]
		if ctx.versionedMcpServerInfo == nil {
			ctx.versionedMcpServerInfo = &VersionedMcpServerInfo{}
		}
		ctx.versionedMcpServerInfo.serverInfo = &info.BasicMcpServerInfo

		if ctx.versionedMcpServerInfo.version != latestVersion {
			ctx.versionedMcpServerInfo.version = latestVersion
			n.onServerVersionChanged(ctx)
			n.triggerMcpServerChange(id)
		}
	}

	n.servers[id] = &ServerContext{
		id:                   id,
		serverChangeListener: listener,
		configsMap: map[string]*ConfigListenerWrap{
			McpServerVersionGroup: {
				dataId:   versionConfigId,
				group:    McpServerVersionGroup,
				listener: versionConfigCallBack,
			},
		},
	}

	versionConfigCallBack(n.namespaceId, McpServerVersionGroup, versionConfigId, serverVersionConfig)
	err = n.configClient.ListenConfig(vo.ConfigParam{
		Group:    McpServerVersionGroup,
		DataId:   versionConfigId,
		OnChange: versionConfigCallBack,
	})

	if err != nil {
		return err
	}

	return nil
}

func (n *NacosRegistryClient) onServerVersionChanged(ctx *ServerContext) {
	id := ctx.versionedMcpServerInfo.serverInfo.Id
	version := ctx.versionedMcpServerInfo.version
	configsMap := map[string]string{
		McpServerSpecGroup: fmt.Sprintf("%s-%s-mcp-server.json", id, version),
		McpToolSpecGroup:   fmt.Sprintf("%s-%s-mcp-tools.json", id, version),
	}

	for group, dataId := range configsMap {
		configsKey := fmt.Sprintf(SystemConfigIdPrefix+"%s@@%s", id, group)
		if data, exist := ctx.configsMap[configsKey]; exist {
			err := n.cancelListenToConfig(data)
			if err != nil {
				// todo handle error
			}
		}

		configListenerWrap, err := n.ListenToConfig(ctx, dataId, group)
		if err != nil {
			// todo handle error
		}
		ctx.configsMap[configsKey] = configListenerWrap
	}
}

func (n *NacosRegistryClient) triggerMcpServerChange(id string) {
	if context, exist := n.servers[id]; exist {
		if config := mapConfigMapToServerConfig(context); config != nil {
			context.serverChangeListener(config)
		}
	}
}

func mapConfigMapToServerConfig(ctx *ServerContext) *McpServerConfig {
	result := &McpServerConfig{
		Credentials: map[string]string{},
	}
	configMaps := ctx.configsMap
	for key, data := range configMaps {
		if strings.HasPrefix(key, SystemConfigIdPrefix) {
			group := strings.Split(key, "@@")[1]
			if group == McpServerSpecGroup {
				result.ServerSpecConfig = data.data
			} else if group == McpToolSpecGroup {
				result.ToolsSpecConfig = data.data
			}
		} else if strings.HasPrefix(key, CredentialPrefix) {
			credentialId := strings.ReplaceAll(key, CredentialPrefix, "")
			result.Credentials[credentialId] = data.data
		}
	}

	result.ServiceInfo = ctx.serviceInfo
	return result
}

func (n *NacosRegistryClient) exactConfigsFromContent(ctx *ServerContext, config *ConfigListenerWrap) []*ConfigListenerWrap {
	var result []*ConfigListenerWrap
	compile, _ := regexp.Compile("\\$\\{nacos\\.([a-zA-Z0-9-_:\\\\.]+/[a-zA-Z0-9-_:\\\\.]+)}")
	allConfigs := compile.FindAllString(config.data, 10)
	newContent := config.data
	for _, data := range allConfigs {
		dataIdAndGroup := strings.ReplaceAll(data, "${nacos.", "")
		dataIdAndGroup = dataIdAndGroup[0 : len(dataIdAndGroup)-1]
		dataIdAndGroupArray := strings.Split(dataIdAndGroup, "/")
		dataId := strings.TrimSpace(dataIdAndGroupArray[0])
		group := strings.TrimSpace(dataIdAndGroupArray[1])
		configWrap, err := n.ListenToConfig(ctx, dataId, group)
		if err != nil {
			// todo handle error
		}
		result = append(result, configWrap)
		newContent = strings.Replace(newContent, data, ".config.credentials."+group+"_"+dataId, 1)
	}

	config.data = newContent
	return result
}

func (n *NacosRegistryClient) resetNacosTemplateConfigs(ctx *ServerContext, config *ConfigListenerWrap) {
	configWraps := n.exactConfigsFromContent(ctx, config)
	for _, data := range configWraps {
		ctx.configsMap[CredentialPrefix+data.group+"_"+data.dataId] = data
	}
}

func (n *NacosRegistryClient) refreshServiceListenerIfNeeded(ctx *ServerContext, serverConfig string) {
	var serverInfo ServerSpecInfo
	err := json.Unmarshal([]byte(serverConfig), &serverInfo)
	if err != nil {
		// todo handle error
	}

	if serverInfo.RemoteServerConfig != nil && serverInfo.RemoteServerConfig.ServiceRef != nil {
		ref := serverInfo.RemoteServerConfig.ServiceRef
		if ctx.serviceInfo != nil {
			if ctx.serviceInfo.Name == ref.ServiceName && ctx.serviceInfo.GroupName == ref.GroupName {
				return
			}

			err := n.namingClient.Unsubscribe(&vo.SubscribeParam{
				GroupName:         ctx.serviceInfo.GroupName,
				ServiceName:       ctx.serviceInfo.Name,
				SubscribeCallback: ctx.namingCallBck,
			})
			if err != nil {
				// todo handle error
			}
		}

		service, err := n.namingClient.GetService(vo.GetServiceParam{
			GroupName:   ref.GroupName,
			ServiceName: ref.ServiceName,
		})

		if err != nil {
			// todo handle error
		}

		ctx.serviceInfo = &service

		if ctx.namingCallBck == nil {
			ctx.namingCallBck = func(services []model.Instance, err error) {
				if ctx.serviceInfo == nil {
					ctx.serviceInfo = &model.Service{
						GroupName: ctx.serviceInfo.GroupName,
						Name:      ctx.serviceInfo.Name,
					}
				}

				ctx.serviceInfo.Name = ref.ServiceName
				ctx.serviceInfo.GroupName = ref.GroupName
				ctx.serviceInfo.Hosts = services
				n.triggerMcpServerChange(ctx.id)
			}
		}

		err = n.namingClient.Subscribe(&vo.SubscribeParam{
			GroupName:         ctx.serviceInfo.GroupName,
			ServiceName:       ctx.serviceInfo.Name,
			SubscribeCallback: ctx.namingCallBck,
		})
		if err != nil {
			// todo handle error
		}
	}
}

func (n *NacosRegistryClient) ListenToConfig(ctx *ServerContext, dataId string, group string) (*ConfigListenerWrap, error) {
	wrap := ConfigListenerWrap{
		dataId: dataId,
		group:  group,
	}

	configListener := func(namespace, group, dataId, data string) {
		if group == McpToolSpecGroup {
			n.resetNacosTemplateConfigs(ctx, &wrap)
		}

		if group == McpServerSpecGroup {
			n.refreshServiceListenerIfNeeded(ctx, data)
		}

		if ctx.serverChangeListener != nil && wrap.data != data {
			wrap.data = data
			n.triggerMcpServerChange(ctx.versionedMcpServerInfo.serverInfo.Id)
		}
	}

	config, err := n.configClient.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})

	if err != nil {
		// todo handle error
	}
	wrap.listener = configListener
	wrap.data = config
	if group == McpToolSpecGroup {
		n.resetNacosTemplateConfigs(ctx, &wrap)
	}

	if group == McpServerSpecGroup {
		n.refreshServiceListenerIfNeeded(ctx, wrap.data)
	}

	err = n.configClient.ListenConfig(vo.ConfigParam{
		DataId:   dataId,
		Group:    group,
		OnChange: configListener,
	})
	if err != nil {
		return nil, err
	}

	return &wrap, nil
}

func (n *NacosRegistryClient) cancelListenToConfig(wrap *ConfigListenerWrap) error {
	return n.configClient.CancelListenConfig(vo.ConfigParam{
		DataId:   wrap.dataId,
		Group:    wrap.group,
		OnChange: wrap.listener,
	})
}

func (n *NacosRegistryClient) CancelListenToServer(id string) error {
	if server, exist := n.servers[id]; exist {
		for _, wrap := range server.configsMap {
			err := n.configClient.CancelListenConfig(vo.ConfigParam{
				DataId:   wrap.dataId,
				Group:    wrap.group,
				OnChange: wrap.listener,
			})

			if err != nil {
				// to do handle error
			}
		}

		err := n.namingClient.Unsubscribe(&vo.SubscribeParam{
			GroupName:         server.serviceInfo.GroupName,
			ServiceName:       server.serviceInfo.Name,
			SubscribeCallback: server.namingCallBck,
		})
		if err != nil {
			// todo handle error
			return err
		}
		delete(n.servers, id)
	}
	return nil
}

func (n *NacosRegistryClient) CloseClient() {
	n.namingClient.CloseClient()
	n.configClient.CloseClient()
}
