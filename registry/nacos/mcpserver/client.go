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
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

const McpServerVersionGroup = "mcp-server-versions"
const McpServerSpecGroup = "mcp-server"
const McpToolSpecGroup = "mcp-tools"
const SystemConfigIdPrefix = "system-"
const CredentialPrefix = "credentials-"
const DefaultNacosListConfigMode = "blur"

const ListMcpServeConfigIdPattern = "*mcp-versions.json"

const DefaultNacosListConfigPageSize = 50

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
	namingCallBack         func(services []model.Instance, err error)
}

type McpServerConfig struct {
	ServerSpecConfig string
	ToolsSpecConfig  string
	ServiceInfo      *model.Service
	Credentials      map[string]interface{}
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
	LatestPublishedVersion string           `json:"latestPublishedVersion"`
	Versions               []*VersionDetail `json:"versionDetails"`
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

func (n *NacosRegistryClient) listMcpServerConfigs() ([]model.ConfigItem, error) {
	currentPageNum := 1
	result := make([]model.ConfigItem, 0)
	for {
		configPage, err := n.configClient.SearchConfig(vo.SearchConfigParam{
			Search:   DefaultNacosListConfigMode,
			DataId:   ListMcpServeConfigIdPattern,
			Group:    McpServerVersionGroup,
			PageNo:   currentPageNum,
			PageSize: DefaultNacosListConfigPageSize,
		})

		if err != nil {
			mcpServerLog.Errorf("List mcp server configs for page size %d, page number %d error %v", currentPageNum, DefaultNacosListConfigPageSize)
		}

		if configPage == nil {
			mcpServerLog.Errorf("List mcp server configs for page size %d, page number %d null %v", currentPageNum, DefaultNacosListConfigPageSize)
			continue
		}

		result = append(result, configPage.PageItems...)

		if configPage.PageNumber >= configPage.PagesAvailable {
			break
		}

		currentPageNum += 1
	}
	return result, nil
}

// ListMcpServer List all mcp server from nacos mcp registry /**
func (n *NacosRegistryClient) ListMcpServer() ([]BasicMcpServerInfo, error) {
	configs, err := n.listMcpServerConfigs()

	if err != nil {
		return nil, err
	}

	var result []BasicMcpServerInfo
	for _, config := range configs {
		mcpServerBasicConfig, err := n.configClient.GetConfig(vo.ConfigParam{
			Group:  McpServerVersionGroup,
			DataId: config.DataId,
		})

		if err != nil {
			mcpServerLog.Errorf("Get mcp server version config (dataId: %s) error, %v", config.DataId, err)
			continue
		}

		if mcpServerBasicConfig == "" {
			mcpServerLog.Infof("get empty mcp server version config (dataId: %s)", config.DataId)
			continue
		}

		mcpServer := BasicMcpServerInfo{}
		err = json.Unmarshal([]byte(mcpServerBasicConfig), &mcpServer)
		if err != nil {
			mcpServerLog.Errorf("Parse mcp server version config error %v", err)
			continue
		}

		if !isMcpServerShouldBeDiscoveryForGateway(mcpServer) {
			mcpServerLog.Infof("mcp server %s don't need to be discovered for gateway, skip it", mcpServerBasicConfig)
			continue
		}

		result = append(result, mcpServer)
	}
	return result, nil
}

func isMcpServerShouldBeDiscoveryForGateway(info BasicMcpServerInfo) bool {
	return "mcp-sse" == info.FrontProtocol || "mcp-streamable" == info.FrontProtocol
}

// ListenToMcpServer Listen to mcp server config and backend service
func (n *NacosRegistryClient) ListenToMcpServer(id string, listener McpServerListener) error {
	versionConfigId := fmt.Sprintf("%s-mcp-versions.json", id)
	serverVersionConfig, err := n.configClient.GetConfig(vo.ConfigParam{
		Group:  McpServerVersionGroup,
		DataId: versionConfigId,
	})
	if err != nil {
		mcpServerLog.Errorf("Get mcp server(id: %s) version config error, %v", id, err)
	} else {
		mcpServerLog.Infof("Get mcp server(id: %s) version config success, config is:\n %v", id, serverVersionConfig)
	}

	versionConfigCallBack := func(namespace string, group string, dataId string, content string) {
		mcpServerLog.Infof("Call back to mcp server %s", id)
		info := VersionsMcpServerInfo{}
		err = json.Unmarshal([]byte(content), &info)
		if err != nil {
			mcpServerLog.Errorf("Parse mcp server (id: %s) version config callback error, %v", id, err)
			return
		}
		latestVersion := info.LatestPublishedVersion

		ctx := n.servers[id]
		if ctx.versionedMcpServerInfo == nil {
			ctx.versionedMcpServerInfo = &VersionedMcpServerInfo{}
		}
		ctx.versionedMcpServerInfo.serverInfo = &info.BasicMcpServerInfo

		// first time the version is empty so it will trigger the change finally.
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

	// trigger callback manually
	versionConfigCallBack(n.namespaceId, McpServerVersionGroup, versionConfigId, serverVersionConfig)
	// Listen after get config to avoid multi-callback on same version
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
		// Only listen to the last version of the server, so we should exist and cancel it first
		if data, exist := ctx.configsMap[configsKey]; exist {
			err := n.cancelListenToConfig(data)
			if err != nil {
				mcpServerLog.Errorf("cancel listen to old config %v error %v", dataId, err)
			}
		}

		configListenerWrap, err := n.ListenToConfig(ctx, dataId, group)
		if err != nil {
			mcpServerLog.Errorf("listen to config %v error %v", dataId, err)
			continue
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
		Credentials: map[string]interface{}{},
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
			var credData interface{}
			if err := json.Unmarshal([]byte(data.data), &credData); err != nil {
				mcpServerLog.Errorf("parse credential %v error %v", credentialId, err)
				// keep origin data if data is not an object
				result.Credentials[credentialId] = data.data
			} else {
				result.Credentials[credentialId] = credData
			}
		}
	}

	result.ServiceInfo = ctx.serviceInfo
	return result
}

func (n *NacosRegistryClient) replaceTemplateAndExactConfigsItems(ctx *ServerContext, config *ConfigListenerWrap) map[string]*ConfigListenerWrap {
	var result map[string]*ConfigListenerWrap
	compile := regexp.MustCompile("\\$\\{nacos\\.([a-zA-Z0-9-_:\\\\.]+/[a-zA-Z0-9-_:\\\\.]+)}")
	allConfigs := compile.FindAllString(config.data, -1)
	allConfigsMap := map[string]string{}
	for _, config := range allConfigs {
		allConfigsMap[config] = config
	}

	newContent := config.data
	for _, data := range allConfigsMap {
		dataIdAndGroup := strings.ReplaceAll(data, "${nacos.", "")
		dataIdAndGroup = dataIdAndGroup[0 : len(dataIdAndGroup)-1]
		dataIdAndGroupArray := strings.Split(dataIdAndGroup, "/")
		dataId := strings.TrimSpace(dataIdAndGroupArray[0])
		group := strings.TrimSpace(dataIdAndGroupArray[1])
		configWrap, err := n.ListenToConfig(ctx, dataId, group)
		if err != nil {
			mcpServerLog.Errorf("exact configs %v from content error %v", dataId, err)
			continue
		}
		result[CredentialPrefix+configWrap.group+"_"+configWrap.dataId] = configWrap
		newContent = strings.Replace(newContent, data, ".config.credentials."+group+"_"+dataId, -1)
	}

	config.data = newContent
	return result
}

func (n *NacosRegistryClient) resetNacosTemplateConfigs(ctx *ServerContext, config *ConfigListenerWrap) {
	newCredentials := n.replaceTemplateAndExactConfigsItems(ctx, config)

	// cancel all old config listener
	for key, wrap := range ctx.configsMap {
		if strings.HasPrefix(key, CredentialPrefix) {
			if _, ok := newCredentials[key]; !ok {
				err := n.cancelListenToConfig(wrap)
				if err != nil {
					mcpServerLog.Errorf("cancel listen to old credential listener error %v", err)
					continue
				}
			}
		}
	}

	for _, data := range newCredentials {
		ctx.configsMap[CredentialPrefix+data.group+"_"+data.dataId] = data
	}
}

func (n *NacosRegistryClient) refreshServiceListenerIfNeeded(ctx *ServerContext, serverConfig string) {
	var serverInfo ServerSpecInfo
	err := json.Unmarshal([]byte(serverConfig), &serverInfo)
	if err != nil {
		mcpServerLog.Errorf("parse server config error %v", err)
		return
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
				SubscribeCallback: ctx.namingCallBack,
			})
			if err != nil {
				mcpServerLog.Errorf("unsubscribe service error:%v, groupName:%s, serviceName:%s", err, ctx.serviceInfo.GroupName, ctx.serviceInfo.Name)
			}
		}

		service, err := n.namingClient.GetService(vo.GetServiceParam{
			GroupName:   ref.GroupName,
			ServiceName: ref.ServiceName,
		})

		if err != nil {
			mcpServerLog.Errorf("get service error:%v, groupName:%s, serviceName:%s", err, ref.GroupName, ref.ServiceName)
			return
		}

		ctx.serviceInfo = &service

		if ctx.namingCallBack == nil {
			ctx.namingCallBack = func(services []model.Instance, err error) {
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
			SubscribeCallback: ctx.namingCallBack,
		})
		if err != nil {
			mcpServerLog.Errorf("subscribe service error:%v, groupName:%s, serviceName:%s", err, ctx.serviceInfo.GroupName, ctx.serviceInfo.Name)
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
		} else if group == McpServerSpecGroup {
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
		return nil, err
	}

	wrap.listener = configListener
	wrap.data = config
	if group == McpToolSpecGroup {
		n.resetNacosTemplateConfigs(ctx, &wrap)
	} else if group == McpServerSpecGroup {
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
	if server, exist := n.servers[id]; exist && server != nil {
		defer delete(n.servers, id)

		for _, wrap := range server.configsMap {
			if wrap != nil {
				err := n.configClient.CancelListenConfig(vo.ConfigParam{
					DataId:   wrap.dataId,
					Group:    wrap.group,
					OnChange: wrap.listener,
				})

				if err != nil {
					mcpServerLog.Errorf("cancel listen config error:%v, dataId:%s, group:%s", err, wrap.dataId, wrap.group)
					continue
				}
			}
		}

		if server.serviceInfo != nil {
			err := n.namingClient.Unsubscribe(&vo.SubscribeParam{
				GroupName:         server.serviceInfo.GroupName,
				ServiceName:       server.serviceInfo.Name,
				SubscribeCallback: server.namingCallBack,
			})
			if err != nil {
				mcpServerLog.Errorf("unsubscribe service error:%v, groupName:%s, serviceName:%s", err, server.serviceInfo.GroupName, server.serviceInfo.Name)
				return err
			}
		}
	}
	return nil
}

func (n *NacosRegistryClient) CloseClient() {
	n.namingClient.CloseClient()
	n.configClient.CloseClient()
}
