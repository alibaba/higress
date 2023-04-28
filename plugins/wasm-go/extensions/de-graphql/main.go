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

package main

import (
	"errors"
	"fmt"
	"net/http"

	"de-graphql/config"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"de-graphql",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
	)
}

func parseConfig(json gjson.Result, config *config.DeGraphQLConfig, log wrapper.Log) error {
	log.Debug("parseConfig()")
	gql := json.Get("gql").String()
	endpoint := json.Get("endpoint").String()
	timeout := json.Get("timeout").Int()
	log.Debugf("gql:%s endpoint:%s timeout:%d", gql, endpoint, timeout)
	err := config.SetGql(gql)
	if err != nil {
		return err
	}
	err = config.SetEndpoint(endpoint)
	if err != nil {
		return err
	}
	config.SetTimeout(uint32(timeout))
	serviceSource := json.Get("serviceSource").String()
	serviceName := json.Get("serviceName").String()
	servicePort := json.Get("servicePort").Int()
	log.Debugf("serviceSource:%s serviceName:%s servicePort:%d", serviceSource, serviceName, servicePort)
	if serviceName == "" || servicePort == 0 {
		return errors.New("invalid service config")
	}
	switch serviceSource {
	case "k8s":
		namespace := json.Get("namespace").String()
		config.SetClient(wrapper.NewClusterClient(wrapper.K8sCluster{
			ServiceName: serviceName,
			Namespace:   namespace,
			Port:        servicePort,
		}))
		return nil
	case "nacos":
		namespace := json.Get("namespace").String()
		config.SetClient(wrapper.NewClusterClient(wrapper.NacosCluster{
			ServiceName: serviceName,
			NamespaceID: namespace,
			Port:        servicePort,
		}))
		return nil
	case "ip":
		config.SetClient(wrapper.NewClusterClient(wrapper.StaticIpCluster{
			ServiceName: serviceName,
			Port:        servicePort,
		}))
		return nil
	case "dns":
		domain := json.Get("domain").String()
		config.SetClient(wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: serviceName,
			Port:        servicePort,
			Domain:      domain,
		}))
		return nil
	default:
		return errors.New("unknown service source: " + serviceSource)
	}

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config config.DeGraphQLConfig, log wrapper.Log) types.Action {
	log.Debug("onHttpRequestHeaders()")
	log.Debugf("schema:%s host:%s path:%s", ctx.Scheme(), ctx.Host(), ctx.Path())
	requestUrl, _ := proxywasm.GetHttpRequestHeader(":path")
	method, _ := proxywasm.GetHttpRequestHeader(":method")
	log.Debugf("method:%s, request:%s", method, requestUrl)
	if err := proxywasm.RemoveHttpRequestHeader("content-length"); err != nil {
		log.Debug("can not reset content-length")
	}
	replaceBody, err := config.ParseGqlFromUrl(requestUrl)
	if err != nil {
		log.Warnf("failed to parse request url %s : %v", requestUrl, err)
	}
	log.Debugf("replace body:%s", replaceBody)

	// Pass headers to upstream cluster
	headers, _ := proxywasm.GetHttpRequestHeaders()
	for i := len(headers) - 1; i >= 0; i-- {
		key := headers[i][0]
		if key == ":method" || key == ":path" || key == ":authority" {
			headers = append(headers[:i], headers[i+1:]...)
		}
	}
	// Add header Content-Type: application/json
	headers = append(headers, [2]string{"Content-Type", "application/json"})
	// Call upstream graphql endpoint
	config.GetClient().Post(config.GetEndpoint(), headers, []byte(replaceBody),
		func(statusCode int, responseHeaders http.Header, responseBody []byte) {
			// Pass response headers and body to client
			headers := make([][2]string, 0, len(responseHeaders)+3)
			for headK, headV := range responseHeaders {
				headers = append(headers, [2]string{headK, headV[0]})
			}
			// Add debug headers
			headers = append(headers, [2]string{"x-degraphql-endpoint", config.GetEndpoint()})
			headers = append(headers, [2]string{"x-degraphql-timeout", fmt.Sprintf("%d", config.GetTimeout())})
			headers = append(headers, [2]string{"x-degraphql-version", config.GetVersion()})
			proxywasm.SendHttpResponse(uint32(statusCode), headers, responseBody, -1)
			return
		}, config.GetTimeout())

	return types.ActionPause
}

func onHttpRequestBody(ctx wrapper.HttpContext, config config.DeGraphQLConfig, body []byte, log wrapper.Log) types.Action {
	log.Debug("onHttpRequestBody()")
	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config config.DeGraphQLConfig, log wrapper.Log) types.Action {
	log.Debug("onHttpResponseHeaders()")
	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config config.DeGraphQLConfig, body []byte, log wrapper.Log) types.Action {
	log.Debug("onHttpResponseBody()")
	return types.ActionContinue
}
