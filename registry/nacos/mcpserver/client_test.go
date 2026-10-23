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
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/stretchr/testify/assert"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type MockedNacosConfigClient struct {
	configs           map[string]interface{}
	configListenerMap map[string][]func(string, string, string, string)
}

func (m MockedNacosConfigClient) GetConfig(param vo.ConfigParam) (string, error) {
	if result, exist := m.configs[param.DataId+"$$"+param.Group]; exist {
		config, ok := result.(string)
		if ok {
			return config, nil
		}

		err, ok := result.(error)
		if ok {
			return "", err
		}

		return "", fmt.Errorf("unknown config type")
	}
	return "", nil
}

func (m MockedNacosConfigClient) PublishConfig(param vo.ConfigParam) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockedNacosConfigClient) DeleteConfig(param vo.ConfigParam) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockedNacosConfigClient) ListenConfig(params vo.ConfigParam) (err error) {
	if _, ok := m.configListenerMap[params.Group]; !ok {
		m.configListenerMap[params.Group] = []func(string, string, string, string){}
	}
	m.configListenerMap[params.DataId+"$$"+params.Group] = append(m.configListenerMap[params.DataId+"$$"+params.Group], params.OnChange)
	return nil
}

func (m MockedNacosConfigClient) CancelListenConfig(params vo.ConfigParam) (err error) {
	delete(m.configListenerMap, params.DataId+"$$"+params.Group)
	return nil
}

func (m MockedNacosConfigClient) SearchConfig(param vo.SearchConfigParam) (*model.ConfigPage, error) {
	dataIdRegex := strings.Replace(param.DataId, "*", ".*", -1)
	groupRegex := strings.Replace(param.Group, "*", ".*", -1)
	result := []model.ConfigItem{}

	for key, value := range m.configs {
		dataIdAndGroup := strings.Split(key, "$$")
		dataId := dataIdAndGroup[0]
		group := dataIdAndGroup[1]
		if regexp.MustCompile(dataIdRegex).MatchString(dataId) && regexp.MustCompile(groupRegex).MatchString(group) {
			result = append(result, model.ConfigItem{
				DataId:  dataId,
				Group:   group,
				Content: value.(string),
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].DataId < result[j].DataId
	})

	offset := param.PageSize * (param.PageNo - 1)
	size := param.PageSize
	if offset+param.PageSize > len(result) {
		size = len(result) - offset
	}
	finalResult := result[offset : offset+size]
	return &model.ConfigPage{
		TotalCount:     len(result),
		PageNumber:     param.PageNo,
		PagesAvailable: len(result)/param.PageSize + 1,
		PageItems:      finalResult,
	}, nil
}

func (m MockedNacosConfigClient) CloseClient() {
	//TODO implement me
	panic("implement me")
}

type MockedNacosNamingClient struct {
	listenerMap map[string][]func(services []model.Instance, err error)
}

func (m MockedNacosNamingClient) RegisterInstance(param vo.RegisterInstanceParam) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockedNacosNamingClient) BatchRegisterInstance(param vo.BatchRegisterInstanceParam) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockedNacosNamingClient) DeregisterInstance(param vo.DeregisterInstanceParam) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockedNacosNamingClient) UpdateInstance(param vo.UpdateInstanceParam) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockedNacosNamingClient) GetService(param vo.GetServiceParam) (model.Service, error) {
	return model.Service{
		Name:      param.ServiceName,
		GroupName: param.GroupName,
		Hosts: []model.Instance{
			{

				Ip:   "127.0.0.1",
				Port: 8080,
			},
		},
	}, nil
}

func (m MockedNacosNamingClient) SelectAllInstances(param vo.SelectAllInstancesParam) ([]model.Instance, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockedNacosNamingClient) SelectInstances(param vo.SelectInstancesParam) ([]model.Instance, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockedNacosNamingClient) SelectOneHealthyInstance(param vo.SelectOneHealthInstanceParam) (*model.Instance, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockedNacosNamingClient) Subscribe(param *vo.SubscribeParam) error {
	if m.listenerMap[param.ServiceName+"$$"+param.GroupName] == nil {
		m.listenerMap[param.ServiceName+"$$"+param.GroupName] = []func([]model.Instance, error){}
	}
	m.listenerMap[param.ServiceName+"$$"+param.GroupName] = append(m.listenerMap[param.ServiceName+"$$"+param.GroupName], param.SubscribeCallback)
	return nil
}

func (m MockedNacosNamingClient) Unsubscribe(param *vo.SubscribeParam) error {
	return nil
}

func (m MockedNacosNamingClient) GetAllServicesInfo(param vo.GetAllServiceInfoParam) (model.ServiceList, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockedNacosNamingClient) ServerHealthy() bool {
	//TODO implement me
	panic("implement me")
}

func (m MockedNacosNamingClient) CloseClient() {
	//TODO implement me
	panic("implement me")
}

func TestNacosRegistryClient_ListMcpServer(t *testing.T) {

	// test list multi pages
	mockedConfigs := map[string]interface{}{}
	for i := 0; i < 151; i++ {
		mockedConfigs[fmt.Sprintf("%d-mcp-versions.json$$mcp-server-versions", i)] = fmt.Sprintf("{\"id\":\"%d\",\"name\":\"test\",\"protocol\":\"http\",\"frontProtocol\":\"mcp-sse\",\"description\":\"test\",\"enabled\":true,\"capabilities\":[\"TOOL\"],\"latestPublishedVersion\":\"1.0.2\",\"versionDetails\":[{\"version\":\"1.0.0\",\"release_date\":\"2025-06-09T05:41:16Z\",\"is_latest\":false},{\"version\":\"1.0.1\",\"release_date\":\"2025-06-09T05:41:37Z\",\"is_latest\":false},{\"version\":\"1.0.2\",\"release_date\":\"2025-06-09T05:42:46Z\",\"is_latest\":true}]}", i)
	}

	client := NacosRegistryClient{
		configClient: MockedNacosConfigClient{configs: mockedConfigs},
	}

	server, err := client.ListMcpServer()
	if err != nil {
		panic(err)
	}
	assert.Equal(t, 151, len(server))

	serverMap := map[string]string{}
	for _, info := range server {
		if _, ok := serverMap[info.Id]; ok {
			panic("server exist " + info.Id)
		}
		serverMap[info.Id] = info.Id
	}

	// test local server should not be list
	mockedConfigs["65-mcp-versions.json$$mcp-server-versions"] = "{\"id\":\"52df06fe-5433-4154-b8e2-3fbb33ca5a33\",\"name\":\"test\",\"protocol\":\"http\",\"frontProtocol\":\"stdio\",\"description\":\"test\",\"enabled\":true,\"capabilities\":[\"TOOL\"],\"latestPublishedVersion\":\"1.0.2\",\"versionDetails\":[{\"version\":\"1.0.0\",\"release_date\":\"2025-06-09T05:41:16Z\",\"is_latest\":false},{\"version\":\"1.0.1\",\"release_date\":\"2025-06-09T05:41:37Z\",\"is_latest\":false},{\"version\":\"1.0.2\",\"release_date\":\"2025-06-09T05:42:46Z\",\"is_latest\":true}]}"
	servers, err := client.ListMcpServer()
	if err != nil {
		panic(err)
	}
	assert.Equal(t, 150, len(servers))

	// test broken config should not be list
	mockedConfigs["65-mcp-versions.json$$mcp-server-versions"] = "{"
	servers, err = client.ListMcpServer()
	if err != nil {
		panic(err)
	}
	assert.Equal(t, 150, len(servers))
}

func TestNacosRegistryClient_ListenToMcpServer(t *testing.T) {
	configClient := MockedNacosConfigClient{
		configs: map[string]interface{}{
			"a4768d16-8263-48ea-8994-e003a2c80271-mcp-versions.json$$mcp-server-versions": "{\"id\":\"a4768d16-8263-48ea-8994-e003a2c80271\",\"name\":\"explore\",\"protocol\":\"https\",\"frontProtocol\":\"mcp-sse\",\"description\":\"explore\",\"enabled\":true,\"capabilities\":[\"TOOL\"],\"latestPublishedVersion\":\"1.0.12\",\"versionDetails\":[{\"version\":\"1.0.0\",\"release_date\":\"2025-06-05T10:11:40Z\",\"is_latest\":false},{\"version\":\"1.0.1\",\"release_date\":\"2025-06-05T10:12:59Z\",\"is_latest\":false},{\"version\":\"1.0.2\",\"release_date\":\"2025-06-05T10:21:28Z\",\"is_latest\":false},{\"version\":\"1.0.3\",\"release_date\":\"2025-06-05T10:21:39Z\",\"is_latest\":false},{\"version\":\"1.0.4\",\"release_date\":\"2025-06-05T10:25:04Z\",\"is_latest\":false},{\"version\":\"1.0.6\",\"release_date\":\"2025-06-05T10:25:24Z\",\"is_latest\":false},{\"version\":\"1.0.8\",\"release_date\":\"2025-06-05T10:27:38Z\",\"is_latest\":false},{\"version\":\"1.0.9\",\"release_date\":\"2025-06-05T10:32:13Z\",\"is_latest\":false},{\"version\":\"1.0.10\",\"release_date\":\"2025-06-05T10:32:28Z\",\"is_latest\":false},{\"version\":\"1.0.11\",\"release_date\":\"2025-06-05T11:04:09Z\",\"is_latest\":true},{\"version\":\"1.0.12\"}]}",
			"a4768d16-8263-48ea-8994-e003a2c80271-1.0.12-mcp-server.json$$mcp-server":     "{\"id\":\"a4768d16-8263-48ea-8994-e003a2c80271\",\"name\":\"explore\",\"protocol\":\"https\",\"frontProtocol\":\"mcp-sse\",\"description\":\"explore\",\"versionDetail\":{\"version\":\"1.0.12\"},\"remoteServerConfig\":{\"serviceRef\":{\"namespaceId\":\"public\",\"groupName\":\"DEFAULT_GROUP\",\"serviceName\":\"explore\"},\"exportPath\":\"\"},\"enabled\":true,\"capabilities\":[\"TOOL\"],\"toolsDescriptionRef\":\"a4768d16-8263-48ea-8994-e003a2c80271-1.0.12-mcp-tools.json\"}",
			"a4768d16-8263-48ea-8994-e003a2c80271-1.0.12-mcp-tools.json$$mcp-tools":       "{\"tools\":[{\"name\":\"explore\",\"description\":\"explore\",\"inputSchema\":{\"type\":\"object\",\"properties\":{\"tags\":{\"description\":\"tags\",\"type\":\"string\"}}}}],\"toolsMeta\":{\"explore\":{\"enabled\":true,\"templates\":{\"json-go-template\":{\"requestTemplate\":{\"method\":\"GET\",\"url\":\"/v0/explore?key={{ ${nacos.test/test}.key }}\",\"argsToUrlParam\":true}}}}}}",
			"test$$test":   "{\n    \"key\": \"secret_key\"\n}",
			"test1$$test1": "{\n    \"key\": \"secret_key_1\"\n}",
			"test3$$test3": "{\n    \"key\": \"secret_key_3\"\n}",
			"a4768d16-8263-48ea-8994-e003a2c80271-1.0.13-mcp-server.json$$mcp-server": "{\"id\":\"a4768d16-8263-48ea-8994-e003a2c80271\",\"name\":\"explore\",\"protocol\":\"https\",\"frontProtocol\":\"mcp-sse\",\"description\":\"explore\",\"versionDetail\":{\"version\":\"1.0.13\"},\"remoteServerConfig\":{\"serviceRef\":{\"namespaceId\":\"public\",\"groupName\":\"DEFAULT_GROUP\",\"serviceName\":\"explore\"},\"exportPath\":\"\"},\"enabled\":true,\"capabilities\":[\"TOOL\"],\"toolsDescriptionRef\":\"a4768d16-8263-48ea-8994-e003a2c80271-1.0.13-mcp-tools.json\"}",
			"a4768d16-8263-48ea-8994-e003a2c80271-1.0.13-mcp-tools.json$$mcp-tools":   "{\"tools\":[{\"name\":\"explore\",\"description\":\"explore\",\"inputSchema\":{\"type\":\"object\",\"properties\":{\"tags\":{\"description\":\"tags\",\"type\":\"string\"}}}}],\"toolsMeta\":{\"explore\":{\"enabled\":true,\"templates\":{\"json-go-template\":{\"requestTemplate\":{\"method\":\"GET\",\"url\":\"/v0/explore?key={{ ${nacos.test3/test3}.key }}\",\"argsToUrlParam\":true}}}}}}",
		},
		configListenerMap: map[string][]func(string, string, string, string){},
	}

	namingClient := MockedNacosNamingClient{
		listenerMap: map[string][]func(services []model.Instance, err error){},
	}
	client := NacosRegistryClient{
		configClient: configClient,
		namingClient: namingClient,
		servers:      map[string]*ServerContext{},
	}

	server, err := client.ListMcpServer()
	if err != nil {
		panic(err)
	}

	assert.Equal(t, 1, len(server))

	var newConfig *McpServerConfig
	err = client.ListenToMcpServer("a4768d16-8263-48ea-8994-e003a2c80271", func(info *McpServerConfig) {
		newConfig = info
	})

	if err != nil {
		panic(err)
	}

	assert.Equal(t, "{\"id\":\"a4768d16-8263-48ea-8994-e003a2c80271\",\"name\":\"explore\",\"protocol\":\"https\",\"frontProtocol\":\"mcp-sse\",\"description\":\"explore\",\"versionDetail\":{\"version\":\"1.0.12\"},\"remoteServerConfig\":{\"serviceRef\":{\"namespaceId\":\"public\",\"groupName\":\"DEFAULT_GROUP\",\"serviceName\":\"explore\"},\"exportPath\":\"\"},\"enabled\":true,\"capabilities\":[\"TOOL\"],\"toolsDescriptionRef\":\"a4768d16-8263-48ea-8994-e003a2c80271-1.0.12-mcp-tools.json\"}", newConfig.ServerSpecConfig)
	assert.Equal(t, "{\"tools\":[{\"name\":\"explore\",\"description\":\"explore\",\"inputSchema\":{\"type\":\"object\",\"properties\":{\"tags\":{\"description\":\"tags\",\"type\":\"string\"}}}}],\"toolsMeta\":{\"explore\":{\"enabled\":true,\"templates\":{\"json-go-template\":{\"requestTemplate\":{\"method\":\"GET\",\"url\":\"/v0/explore?key={{ .config.credentials.test_test.key }}\",\"argsToUrlParam\":true}}}}}}", newConfig.ToolsSpecConfig)
	assert.Equal(t, 1, len(newConfig.Credentials))
	assert.Equal(t, map[string]interface{}{"key": "secret_key"}, newConfig.Credentials["test_test"])

	// change the tool nacos template
	listener := configClient.configListenerMap["a4768d16-8263-48ea-8994-e003a2c80271-1.0.12-mcp-tools.json$$mcp-tools"][0]
	listener("public", "mcp-tools", "a4768d16-8263-48ea-8994-e003a2c80271-1.0.12-mcp-tools.json", "{\"tools\":[{\"name\":\"explore\",\"description\":\"explore\",\"inputSchema\":{\"type\":\"object\",\"properties\":{\"tags\":{\"description\":\"tags\",\"type\":\"string\"}}}}],\"toolsMeta\":{\"explore\":{\"enabled\":true,\"templates\":{\"json-go-template\":{\"requestTemplate\":{\"method\":\"GET\",\"url\":\"/v0/explore?key={{ ${nacos.test1/test1}.key }}\",\"argsToUrlParam\":true}}}}}}")
	assert.Equal(t, "{\"tools\":[{\"name\":\"explore\",\"description\":\"explore\",\"inputSchema\":{\"type\":\"object\",\"properties\":{\"tags\":{\"description\":\"tags\",\"type\":\"string\"}}}}],\"toolsMeta\":{\"explore\":{\"enabled\":true,\"templates\":{\"json-go-template\":{\"requestTemplate\":{\"method\":\"GET\",\"url\":\"/v0/explore?key={{ .config.credentials.test1_test1.key }}\",\"argsToUrlParam\":true}}}}}}", newConfig.ToolsSpecConfig)
	assert.Equal(t, 1, len(newConfig.Credentials))
	assert.Equal(t, map[string]interface{}{"key": "secret_key_1"}, newConfig.Credentials["test1_test1"])

	// change backend service
	serviceListener := configClient.configListenerMap["a4768d16-8263-48ea-8994-e003a2c80271-1.0.12-mcp-server.json$$mcp-server"][0]
	serviceListener("public", "mcp-server", "a4768d16-8263-48ea-8994-e003a2c80271-1.0.12-mcp-server.json", "{\"id\":\"a4768d16-8263-48ea-8994-e003a2c80271\",\"name\":\"explore\",\"protocol\":\"https\",\"frontProtocol\":\"mcp-sse\",\"description\":\"explore\",\"versionDetail\":{\"version\":\"1.0.12\"},\"remoteServerConfig\":{\"serviceRef\":{\"namespaceId\":\"public\",\"groupName\":\"DEFAULT_GROUP\",\"serviceName\":\"explore-new\"},\"exportPath\":\"\"},\"enabled\":true,\"capabilities\":[\"TOOL\"],\"toolsDescriptionRef\":\"a4768d16-8263-48ea-8994-e003a2c80271-1.0.12-mcp-tools.json\"}")

	// publish new version mcp server
	versionListener := configClient.configListenerMap["a4768d16-8263-48ea-8994-e003a2c80271-mcp-versions.json$$mcp-server-versions"][0]
	versionListener("public", "mc-server-versions", "a4768d16-8263-48ea-8994-e003a2c80271-mcp-versions.json", "{\"id\":\"a4768d16-8263-48ea-8994-e003a2c80271\",\"name\":\"explore\",\"protocol\":\"https\",\"frontProtocol\":\"mcp-sse\",\"description\":\"explore\",\"enabled\":true,\"capabilities\":[\"TOOL\"],\"latestPublishedVersion\":\"1.0.13\",\"versionDetails\":[{\"version\":\"1.0.0\",\"release_date\":\"2025-06-05T10:11:40Z\",\"is_latest\":false},{\"version\":\"1.0.1\",\"release_date\":\"2025-06-05T10:12:59Z\",\"is_latest\":false},{\"version\":\"1.0.2\",\"release_date\":\"2025-06-05T10:21:28Z\",\"is_latest\":false},{\"version\":\"1.0.3\",\"release_date\":\"2025-06-05T10:21:39Z\",\"is_latest\":false},{\"version\":\"1.0.4\",\"release_date\":\"2025-06-05T10:25:04Z\",\"is_latest\":false},{\"version\":\"1.0.6\",\"release_date\":\"2025-06-05T10:25:24Z\",\"is_latest\":false},{\"version\":\"1.0.8\",\"release_date\":\"2025-06-05T10:27:38Z\",\"is_latest\":false},{\"version\":\"1.0.9\",\"release_date\":\"2025-06-05T10:32:13Z\",\"is_latest\":false},{\"version\":\"1.0.10\",\"release_date\":\"2025-06-05T10:32:28Z\",\"is_latest\":false},{\"version\":\"1.0.11\",\"release_date\":\"2025-06-05T11:04:09Z\",\"is_latest\":true},{\"version\":\"1.0.12\"}]}")

	assert.Equal(t, "{\"id\":\"a4768d16-8263-48ea-8994-e003a2c80271\",\"name\":\"explore\",\"protocol\":\"https\",\"frontProtocol\":\"mcp-sse\",\"description\":\"explore\",\"versionDetail\":{\"version\":\"1.0.13\"},\"remoteServerConfig\":{\"serviceRef\":{\"namespaceId\":\"public\",\"groupName\":\"DEFAULT_GROUP\",\"serviceName\":\"explore\"},\"exportPath\":\"\"},\"enabled\":true,\"capabilities\":[\"TOOL\"],\"toolsDescriptionRef\":\"a4768d16-8263-48ea-8994-e003a2c80271-1.0.13-mcp-tools.json\"}", newConfig.ServerSpecConfig)
	assert.Equal(t, "{\"tools\":[{\"name\":\"explore\",\"description\":\"explore\",\"inputSchema\":{\"type\":\"object\",\"properties\":{\"tags\":{\"description\":\"tags\",\"type\":\"string\"}}}}],\"toolsMeta\":{\"explore\":{\"enabled\":true,\"templates\":{\"json-go-template\":{\"requestTemplate\":{\"method\":\"GET\",\"url\":\"/v0/explore?key={{ .config.credentials.test3_test3.key }}\",\"argsToUrlParam\":true}}}}}}", newConfig.ToolsSpecConfig)
	assert.Equal(t, 1, len(newConfig.Credentials))
	assert.Equal(t, map[string]interface{}{"key": "secret_key_3"}, newConfig.Credentials["test3_test3"])
}
