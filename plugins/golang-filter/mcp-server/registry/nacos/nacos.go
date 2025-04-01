package nacos

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/registry"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type NacosMcpRegsitry struct {
	serviceMatcher           map[string]string
	configClient             config_client.IConfigClient
	namingClient             naming_client.INamingClient
	toolsDescription         map[string]*registry.ToolDescription
	toolsRpcContext          map[string]*registry.RpcContext
	toolChangeEventListeners []registry.ToolChangeEventListener
	currentServiceSet        map[string]bool
}

const DEFAULT_SERVICE_LIST_MAX_PGSIZXE = 10000
const MCP_TOOL_SUBFIX = "-mcp-tools.json"

func (n *NacosMcpRegsitry) ListToolsDesciption() []*registry.ToolDescription {
	if n.toolsDescription == nil {
		n.refreshToolsList()
	}

	result := []*registry.ToolDescription{}
	for _, tool := range n.toolsDescription {
		result = append(result, tool)
	}
	return result
}

func (n *NacosMcpRegsitry) GetToolRpcContext(toolName string) (*registry.RpcContext, bool) {
	tool, ok := n.toolsRpcContext[toolName]
	return tool, ok
}

func (n *NacosMcpRegsitry) RegisterToolChangeEventListener(listener registry.ToolChangeEventListener) {
	n.toolChangeEventListeners = append(n.toolChangeEventListeners, listener)
}

func (n *NacosMcpRegsitry) refreshToolsList() bool {
	changed := false
	for group, serviceMatcher := range n.serviceMatcher {
		if n.refreshToolsListForGroup(group, serviceMatcher) {
			changed = true
		}
	}
	return changed
}

func (n *NacosMcpRegsitry) refreshToolsListForGroup(group string, serviceMatcher string) bool {
	services, err := n.namingClient.GetAllServicesInfo(vo.GetAllServiceInfoParam{
		GroupName: group,
		PageNo:    1,
		PageSize:  DEFAULT_SERVICE_LIST_MAX_PGSIZXE,
	})

	if err != nil {
		api.LogError(fmt.Sprintf("Get service list error when refresh tools list for group %s, error %s", group, err))
		return false
	}

	changed := false
	serviceList := services.Doms
	pattern, err := regexp.Compile(serviceMatcher)
	if err != nil {
		api.LogErrorf("Match service error for patter %s", serviceMatcher)
		return false
	}

	currentServiceList := map[string]bool{}

	for _, service := range serviceList {
		if !pattern.MatchString(service) {
			continue
		}

		formatServiceName := getFormatServiceName(group, service)
		if _, ok := n.currentServiceSet[formatServiceName]; !ok {
			changed = true
			n.refreshToolsListForService(group, service)
			n.listenToService(group, service)
		}

		currentServiceList[formatServiceName] = true
	}

	serviceShouldBeDeleted := []string{}
	for serviceName, _ := range n.currentServiceSet {
		if !strings.HasPrefix(serviceName, group) {
			continue
		}

		if _, ok := currentServiceList[serviceName]; !ok {
			serviceShouldBeDeleted = append(serviceShouldBeDeleted, serviceName)
			changed = true
			toolsShouldBeDeleted := []string{}
			for toolName, _ := range n.toolsDescription {
				if strings.HasPrefix(toolName, serviceName) {
					toolsShouldBeDeleted = append(toolsShouldBeDeleted, toolName)
				}
			}

			for _, toolName := range toolsShouldBeDeleted {
				delete(n.toolsDescription, toolName)
				delete(n.toolsRpcContext, toolName)
			}
		}
	}

	for _, service := range serviceShouldBeDeleted {
		delete(n.currentServiceSet, service)
	}

	return changed
}

func getFormatServiceName(group string, service string) string {
	return fmt.Sprintf("%s_%s", group, service)
}

func (n *NacosMcpRegsitry) refreshToolsListForServiceWithContent(group string, service string, newConfig *string, instances *[]model.Instance) {

	if newConfig == nil {
		dataId := makeToolsConfigId(service)
		content, err := n.configClient.GetConfig(vo.ConfigParam{
			DataId: dataId,
			Group:  group,
		})

		if err != nil {
			api.LogError(fmt.Sprintf("Get tools config for sercice %s:%s error %s", group, service, err))
			return
		}

		newConfig = &content
	}

	if instances == nil {
		instancesFromNacos, err := n.namingClient.SelectInstances(vo.SelectInstancesParam{
			ServiceName: service,
			GroupName:   group,
			HealthyOnly: true,
		})

		if err != nil {
			api.LogError(fmt.Sprintf("List instance for sercice %s:%s error %s", group, service, err))
			return
		}

		instances = &instancesFromNacos
	}

	var applicationDescription registry.McpApplicationDescription
	err := json.Unmarshal([]byte(*newConfig), &applicationDescription)
	if err != nil {
		api.LogError(fmt.Sprintf("Parse tools config for sercice %s:%s error, config is %s, error is %s", group, service, *newConfig, err))
		return
	}

	wrappedInstances := []registry.Instance{}
	for _, instance := range *instances {
		wrappedInstance := registry.Instance{
			Host: instance.Ip,
			Port: instance.Port,
			Meta: instance.Metadata,
		}
		wrappedInstances = append(wrappedInstances, wrappedInstance)
	}

	if n.toolsDescription == nil {
		n.toolsDescription = map[string]*registry.ToolDescription{}
	}

	if n.toolsRpcContext == nil {
		n.toolsRpcContext = map[string]*registry.RpcContext{}
	}

	for _, tool := range applicationDescription.ToolsDescription {
		meta := applicationDescription.ToolsMeta[tool.Name]

		var cred *registry.CredentialInfo
		credentialRef := meta.CredentialRef
		if credentialRef != nil {
			cred = n.GetCredential(*credentialRef, group)
		}

		context := registry.RpcContext{
			ToolMeta:   meta,
			Instances:  &wrappedInstances,
			Protocol:   applicationDescription.Protocol,
			Credential: cred,
		}

		tool.Name = makeToolName(group, service, tool.Name)
		n.toolsDescription[tool.Name] = tool
		n.toolsRpcContext[tool.Name] = &context
	}
	n.currentServiceSet[getFormatServiceName(group, service)] = true
}

func (n *NacosMcpRegsitry) GetCredential(name string, group string) *registry.CredentialInfo {
	dataId := makeCredentialDataId(name)
	content, err := n.configClient.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})

	if err != nil {
		api.LogError(fmt.Sprintf("Get credentials for %s:%s error %s", group, dataId, err))
		return nil
	}

	var credential registry.CredentialInfo
	err = json.Unmarshal([]byte(content), &credential)
	if err != nil {
		api.LogError(fmt.Sprintf("Parse credentials for %s:%s error %s", group, dataId, err))
		return nil
	}

	return &credential
}

func (n *NacosMcpRegsitry) refreshToolsListForService(group string, service string) {
	n.refreshToolsListForServiceWithContent(group, service, nil, nil)
}

func (n *NacosMcpRegsitry) listenToService(group string, service string) {

	// config changed, tools description may be changed
	err := n.configClient.ListenConfig(vo.ConfigParam{
		DataId: makeToolsConfigId(service),
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			n.refreshToolsListForServiceWithContent(group, service, &data, nil)
			for _, listener := range n.toolChangeEventListeners {
				listener.OnToolChanged(n)
			}
		},
	})

	if err != nil {
		api.LogError(fmt.Sprintf("Listen to service's tool config error %s", err))
	}

	err = n.namingClient.Subscribe(&vo.SubscribeParam{
		ServiceName: service,
		GroupName:   group,
		SubscribeCallback: func(services []model.Instance, err error) {
			n.refreshToolsListForServiceWithContent(group, service, nil, &services)
			for _, listener := range n.toolChangeEventListeners {
				listener.OnToolChanged(n)
			}
		},
	})
	if err != nil {
		api.LogError(fmt.Sprintf("Listen to service's tool instance list error %s", err))
	}
}

func makeToolName(group string, service string, toolName string) string {
	return fmt.Sprintf("%s_%s_%s", group, service, toolName)
}

func makeToolsConfigId(service string) string {
	return service + MCP_TOOL_SUBFIX
}

func makeCredentialDataId(credentialName string) string {
	return credentialName
}
