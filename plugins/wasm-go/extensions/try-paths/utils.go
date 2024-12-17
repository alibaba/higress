package main

import (
	"errors"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

func convertHttpHeadersToStruct(responseHeaders http.Header) [][2]string {
	headerStruct := make([][2]string, len(responseHeaders))
	i := 0
	for key, values := range responseHeaders {
		headerStruct[i][0] = key
		headerStruct[i][1] = values[0]
		i++
	}
	return headerStruct
}

func contains(array []int, value int) bool {
	for _, v := range array {
		if v == value {
			return true
		}
	}
	return false
}

func Client(json gjson.Result) (wrapper.HttpClient, error) {
	serviceSource := json.Get("serviceSource").String()
	serviceName := json.Get("serviceName").String()
	host := json.Get("host").String()
	servicePort := json.Get("servicePort").Int()

	switch serviceSource {
	case "k8s":
		namespace := json.Get("namespace").String()
		return wrapper.NewClusterClient(wrapper.K8sCluster{
			ServiceName: serviceName,
			Namespace:   namespace,
			Port:        servicePort,
			Host:        host,
		}), nil
	case "nacos":
		namespace := json.Get("namespace").String()
		return wrapper.NewClusterClient(wrapper.NacosCluster{
			ServiceName: serviceName,
			NamespaceID: namespace,
			Port:        servicePort,
			Host:        host,
		}), nil
	case "ip":
		return wrapper.NewClusterClient(wrapper.StaticIpCluster{
			ServiceName: serviceName,
			Port:        servicePort,
			Host:        host,
		}), nil
	case "dns":
		domain := json.Get("domain").String()
		return wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: serviceName,
			Port:        servicePort,
			Domain:      domain,
		}), nil
	case "fqdn":
		return wrapper.NewClusterClient(wrapper.FQDNCluster{
			FQDN: host,
			Port: servicePort,
		}), nil
	case "route":
		return wrapper.NewClusterClient(wrapper.RouteCluster{
			Host: host,
		}), nil

	default:
		return nil, errors.New("unknown service source: " + serviceSource)
	}
}
