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
	"net/http"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
)

type OpaConfig struct {
	policy  string
	timeout uint32

	// the result json path, which must be a boolean value
	resultPath string
	// whether execute on request headers
	skipHeader bool
	// whether execute on request body
	skipBody bool

	// for some cases, we need to send custom deny message
	denyCodePath          string
	denyMappingMessages   map[string]string
	denyMessageContenType string

	// opa not 200
	no200Message    string
	no200Code       uint32
	no200ContenType string

	// after authz, allow add extra headers by result path
	// eg: {"result.user_id": "x-user-real-id"}
	// get result.user-realid from opa response, and add to request header x-user-realid
	extratHeaders map[string]string

	client wrapper.HttpClient
}

const (
	defaultResultPath = "result"
)

func Client(json gjson.Result) (wrapper.HttpClient, error) {
	serviceSource := strings.TrimSpace(json.Get("serviceSource").String())
	serviceName := strings.TrimSpace(json.Get("serviceName").String())
	servicePort := json.Get("servicePort").Int()

	host := strings.TrimSpace(json.Get("host").String())
	if host == "" {
		if serviceName == "" || servicePort == 0 {
			return nil, errors.New("invalid service config")
		}
	}

	var namespace string
	if serviceSource == "k8s" || serviceSource == "nacos" {
		if namespace = strings.TrimSpace(json.Get("namespace").String()); namespace == "" {
			return nil, errors.New("namespace not allow empty")
		}
	}

	switch serviceSource {
	case "k8s":
		return wrapper.NewClusterClient(wrapper.K8sCluster{
			ServiceName: serviceName,
			Namespace:   namespace,
			Port:        servicePort,
		}), nil
	case "nacos":
		return wrapper.NewClusterClient(wrapper.NacosCluster{
			ServiceName: serviceName,
			NamespaceID: namespace,
			Port:        servicePort,
		}), nil
	case "ip":
		return wrapper.NewClusterClient(wrapper.StaticIpCluster{
			ServiceName: serviceName,
			Host:        host,
			Port:        servicePort,
		}), nil
	case "dns":
		return wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: serviceName,
			Port:        servicePort,
			Domain:      json.Get("domain").String(),
		}), nil
	case "route":
		return wrapper.NewClusterClient(wrapper.RouteCluster{
			Host: host,
		}), nil
	}
	return nil, errors.New("unknown service source: " + serviceSource)
}

func (config OpaConfig) rspCall(statusCode int, _ http.Header, responseBody []byte) {
	if statusCode != http.StatusOK {
		proxywasm.LogWarnf("opa policy failed , status code %d, responseBody %s", statusCode, responseBody)
		if config.no200Message != "" {
			proxywasm.SendHttpResponseWithDetail(
				config.no200Code,
				"opa.status_ne_200",
				contentType(config.no200ContenType),
				[]byte(config.no200Message),
				-1,
			)
			return
		} else {
			proxywasm.SendHttpResponseWithDetail(uint32(statusCode), "opa.status_ne_200", nil, []byte("opa state not is 200"), -1)
		}
		return
	}

	ok := gjson.GetBytes(responseBody, config.resultPath).Bool()
	if !ok {
		proxywasm.LogDebugf("opa policy failed , raw opa response %s", responseBody)
		if config.denyCodePath != "" {
			denyCode := gjson.GetBytes(responseBody, config.denyCodePath).String()
			denyMessage := config.denyMappingMessages[denyCode]
			if denyMessage == "" {
				denyMessage = "opa server not allowed"
			}
			proxywasm.SendHttpResponseWithDetail(
				http.StatusUnauthorized,
				"opa.server_not_allowed",
				contentType(config.denyMessageContenType),
				[]byte(denyMessage),
				-1,
			)
		} else {
			proxywasm.SendHttpResponseWithDetail(http.StatusUnauthorized, "opa.server_not_allowed", nil, []byte("opa server not allowed"), -1)
		}
		return
	}
	if len(config.extratHeaders) > 0 {
		for k, v := range config.extratHeaders {
			rv := gjson.GetBytes(responseBody, k).String()
			if rv != "" {
				proxywasm.LogDebugf("opa add header %s: %s", v, rv)
				proxywasm.AddHttpRequestHeader(v, rv)
			}
		}
	}
	proxywasm.ResumeHttpRequest()
}

func contentType(ct string) [][2]string {
	return [][2]string{{"Content-Type", ct}}
}
