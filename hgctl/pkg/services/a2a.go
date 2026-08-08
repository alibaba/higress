// Copyright (c) 2026 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"path"
	"strings"
)

const (
	a2aPluginName    = "a2a-protocol"
	a2aPluginVersion = "1.0.0-alpha"
)

// PublishA2A creates the service and route before attaching the protocol
// plugin to that route. It does not create or persist Agent task state.
func PublishA2A(client *Client, name, rawURL string) error {
	serviceBody, routeBody, pluginBody, instanceBody, routeName, err := buildA2APublication(name, rawURL)
	if err != nil {
		return err
	}
	if response, err := client.post("/v1/service-sources", serviceBody); err != nil {
		return fmt.Errorf("publish A2A service (%s): %w", boundedResponse(response), err)
	}
	if response, err := client.post("/v1/routes", routeBody); err != nil {
		return fmt.Errorf("publish A2A route (%s): %w", boundedResponse(response), err)
	}
	if response, err := client.post("/v1/wasm-plugins", pluginBody); err != nil {
		var httpErr *HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != 409 {
			return fmt.Errorf("publish A2A WasmPlugin (%s): %w", boundedResponse(response), err)
		}
	}
	instancePath := fmt.Sprintf("/v1/routes/%s/plugin-instances/%s", url.PathEscape(routeName), a2aPluginName)
	if response, err := client.put(instancePath, instanceBody); err != nil {
		return fmt.Errorf("attach A2A WasmPlugin (%s): %w", boundedResponse(response), err)
	}
	return nil
}

func buildA2APublication(name, rawURL string) (service, route, plugin, instance map[string]interface{}, routeName string, err error) {
	if name == "" {
		return nil, nil, nil, nil, "", fmt.Errorf("A2A agent name is required")
	}
	upstream, err := url.Parse(rawURL)
	if err != nil || upstream.Hostname() == "" || (upstream.Scheme != "http" && upstream.Scheme != "https") {
		return nil, nil, nil, nil, "", fmt.Errorf("invalid A2A agent URL %q", rawURL)
	}
	port := upstream.Port()
	if port == "" {
		if upstream.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	serviceType := "dns"
	if net.ParseIP(upstream.Hostname()) != nil {
		serviceType = "static"
	}
	serviceName := "agent-" + name
	targetService := serviceName + "." + serviceType
	routeName = name + "-route"
	matchPath := path.Clean("/" + strings.TrimPrefix(upstream.EscapedPath(), "/"))
	if matchPath == "." {
		matchPath = "/"
	}
	service = map[string]interface{}{
		"domain": upstream.Hostname(), "type": serviceType, "port": port,
		"name": serviceName, "proxyName": "", "domainForEdit": upstream.Hostname(), "protocol": upstream.Scheme,
	}
	route = map[string]interface{}{
		"name":       routeName,
		"path":       map[string]interface{}{"matchType": "PRE", "matchValue": matchPath, "caseSensitive": true},
		"authConfig": map[string]interface{}{"enabled": false},
		"services":   []map[string]interface{}{{"name": targetService}},
	}
	plugin = map[string]interface{}{
		"name": a2aPluginName, "pluginVersion": "1.0.0", "category": "protocol",
		"title": "A2A Protocol", "description": "Bounded A2A 1.0 JSON-RPC and SSE metadata extraction",
		"builtIn":         false,
		"imageRepository": "oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/a2a-protocol",
		"imageVersion":    a2aPluginVersion, "imagePullPolicy": "IfNotPresent", "phase": "AUTHN", "priority": 300,
	}
	instance = map[string]interface{}{
		"targets":    map[string]interface{}{"ROUTE": routeName},
		"pluginName": a2aPluginName, "pluginVersion": "1.0.0", "enabled": true, "internal": false,
		"configurations": map[string]interface{}{
			"protocolVersion": "1.0", "mode": "enforce",
			"legacy03":      map[string]interface{}{"enabled": false},
			"agent":         map[string]interface{}{"id": name},
			"jsonrpc":       map[string]interface{}{"maxRequestBytes": 4194304, "maxSSEEventBytes": 262144},
			"authorization": map[string]interface{}{"exposeInternalHeaders": true},
		},
	}
	return service, route, plugin, instance, routeName, nil
}

func boundedResponse(response []byte) string {
	const max = 512
	if len(response) > max {
		response = response[:max]
	}
	return strings.TrimSpace(string(response))
}
