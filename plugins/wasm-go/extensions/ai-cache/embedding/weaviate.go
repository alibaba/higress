package embedding

// import (
// 	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
// )

// const (
// 	weaviateURL = "172.17.0.1:8081"
// )

// type weaviateProviderInitializer struct {
// }

// func (d *weaviateProviderInitializer) ValidateConfig(config ProviderConfig) error {
// 	return nil
// }

// func (d *weaviateProviderInitializer) CreateProvider(config ProviderConfig) (Provider, error) {
// 	return &DSProvider{
// 		config: config,
// 		client: wrapper.NewClusterClient(wrapper.DnsCluster{
// 			ServiceName: config.ServiceName,
// 			Port:        dashScopePort,
// 			Domain:      dashScopeDomain,
// 		}),
// 	}, nil
// }
