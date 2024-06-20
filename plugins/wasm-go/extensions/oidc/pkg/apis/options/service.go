package options

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

// Cookie contains configuration options relating to Service configuration
type Service struct {
	ServiceSource string `mapstructure:"service_source"`
	ServiceName   string `mapstructure:"service_name"`
	ServicePort   int64  `mapstructure:"service_port"`
	ServiceHost   string `mapstructure:"service_host"`
	ServiceDomain string `mapstructure:"service_domain"`
}

func (s *Service) NewService() (wrapper.HttpClient, error) {
	if s.ServiceName == "" || s.ServicePort == 0 {
		return nil, errors.New("invalid service config")
	}
	switch s.ServiceSource {
	case "ip":
		Client := wrapper.NewClusterClient(&wrapper.StaticIpCluster{
			ServiceName: s.ServiceName,
			Host:        s.ServiceHost,
			Port:        s.ServicePort,
		})
		return Client, nil
	case "dns":
		if s.ServiceDomain == "" {
			return nil, errors.New("missing service_domain in config")
		}
		Client := wrapper.NewClusterClient(&wrapper.DnsCluster{
			ServiceName: s.ServiceName,
			Port:        s.ServicePort,
			Domain:      s.ServiceDomain,
		})
		return Client, nil
	default:
		return nil, errors.New("unknown service source: " + s.ServiceSource)
	}
}
