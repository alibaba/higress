package mcpserver

import (
	"reflect"
	"sync"
	"testing"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
	higressmcpserver "github.com/alibaba/higress/pkg/ingress/kube/mcpserver"
	provider "github.com/alibaba/higress/registry"
	"github.com/alibaba/higress/registry/memory"
	"github.com/stretchr/testify/mock"
	wrappers "google.golang.org/protobuf/types/known/wrapperspb"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/gvk"
)

type mockWatcher struct {
	watcher
	mock.Mock
}

func newTestWatcher(cache memory.Cache, opts ...WatcherOption) mockWatcher {
	w := &watcher{
		watchingConfig: make(map[string]bool),
		RegistryType:   "mcpserver",
		Status:         provider.UnHealthy,
		cache:          cache,
		mutex:          &sync.Mutex{},
		stop:           make(chan struct{}),
	}

	w.NacosRefreshInterval = int64(DefaultRefreshInterval)

	for _, opt := range opts {
		opt(w)
	}

	if w.NacosNamespace == "" {
		w.NacosNamespace = w.NacosNamespaceId
	}

	return mockWatcher{watcher: *w, Mock: mock.Mock{}}
}

func testCallback(msc *McpServerConfig) memory.Cache {
	registryConfig := &apiv1.RegistryConfig{
		Type:                   string(provider.Nacos),
		Name:                   "mse-nacos-public",
		Domain:                 "",
		Port:                   8848,
		NacosAddressServer:     "",
		NacosAccessKey:         "ak",
		NacosSecretKey:         "sk",
		NacosNamespaceId:       "",
		NacosNamespace:         "public",
		NacosGroups:            []string{"dev"},
		NacosRefreshInterval:   0,
		EnableMCPServer:        wrappers.Bool(true),
		McpServerExportDomains: []string{"mcp-export-domain"},
		McpServerBaseUrl:       "/mcp-servers/",
		EnableScopeMcpServers:  wrappers.Bool(true),
		AllowMcpServers:        []string{"mcp-server-1", "mcp-server-2"},
		Metadata: map[string]*apiv1.InnerMap{
			"routeName": &apiv1.InnerMap{
				InnerMap: map[string]string{"mcp-server-1": "mcp-route-1", "mcp-server-2": "mcp-route-2"},
			},
		},
	}
	localCache := memory.NewCache()

	testWatcher := newTestWatcher(localCache,
		WithType(registryConfig.Type),
		WithName(registryConfig.Name),
		WithNacosAddressServer(registryConfig.NacosAddressServer),
		WithDomain(registryConfig.Domain),
		WithPort(registryConfig.Port),
		WithNacosNamespaceId(registryConfig.NacosNamespaceId),
		WithNacosNamespace(registryConfig.NacosNamespace),
		WithNacosGroups(registryConfig.NacosGroups),
		WithNacosAccessKey(registryConfig.NacosAccessKey),
		WithNacosSecretKey(registryConfig.NacosSecretKey),
		WithNacosRefreshInterval(registryConfig.NacosRefreshInterval),
		WithMcpExportDomains(registryConfig.McpServerExportDomains),
		WithMcpBaseUrl(registryConfig.McpServerBaseUrl),
		WithEnableMcpServer(registryConfig.EnableMCPServer))

	callback := testWatcher.mcpServerListener("mock-data-id")
	callback(msc)
	return localCache
}

func Test_Watcher(t *testing.T) {
	dataId := "mock-data-id"

	testCase := []struct {
		name       string
		msc        *McpServerConfig
		dataId     string
		wantConfig map[string]map[string]*config.Config
	}{
		{
			name:   "test_watcher",
			dataId: dataId,
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			localCache := testCallback(tc.msc)
			se := localCache.GetAllConfigs(gvk.ServiceEntry)
			wantSe := tc.wantConfig[gvk.ServiceEntry.String()][tc.dataId]
			if !reflect.DeepEqual(se, wantSe) {
				t.Errorf("se is not equal, want %v, got %v", wantSe, se)
			}

			vs := localCache.GetAllConfigs(gvk.VirtualService)
			wantVs := tc.wantConfig[gvk.VirtualService.String()][tc.dataId]
			if !reflect.DeepEqual(vs, wantVs) {
				t.Errorf("vs is not equal, want %v, got %v", wantVs, vs)
			}

			dr := localCache.GetAllConfigs(gvk.DestinationRule)
			wantDr := tc.wantConfig[gvk.DestinationRule.String()][tc.dataId]
			if !reflect.DeepEqual(dr, wantDr) {
				t.Errorf("dr is not equal, want %v, got %v", wantDr, dr)
			}

			wasm := localCache.GetAllConfigs(gvk.WasmPlugin)
			wantWasm := tc.wantConfig[gvk.WasmPlugin.String()][tc.dataId]
			if !reflect.DeepEqual(wasm, wantWasm) {
				t.Errorf("wasm is not equal, want %v, got %v", wantWasm, wasm)
			}

			mcpServer := localCache.GetAllConfigs(higressmcpserver.GvkMcpServer)
			wantServer := tc.wantConfig[higressmcpserver.GvkMcpServer.String()][tc.dataId]
			if !reflect.DeepEqual(mcpServer, wantServer) {
				t.Errorf("mcpserver is not equal, want %v, got %v", wantServer, mcpServer)
			}
		})
	}
}
