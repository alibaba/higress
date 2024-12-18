package main

import (
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
	host := json.Get("host").String()
	servicePort := json.Get("servicePort").Int()
	return wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: host,
		Port: servicePort,
	}), nil
}
