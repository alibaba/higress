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
	a2aPluginName          = "a2a-protocol"
	a2aPluginVersion       = "1.0.0-alpha"
	canonicalAgentCardPath = "/.well-known/agent-card.json"
)

type a2aRoutePublication struct {
	name     string
	route    map[string]interface{}
	instance map[string]interface{}
}

type a2aPublication struct {
	service map[string]interface{}
	routes  []a2aRoutePublication
	plugin  map[string]interface{}
}

// PublishA2A creates the service and route before attaching the protocol
// plugin to that route. It does not create or persist Agent task state.
func PublishA2A(client *Client, name, rawURL, externalBaseURL string) error {
	publication, err := buildA2APublication(name, rawURL, externalBaseURL)
	if err != nil {
		return err
	}
	if response, err := client.post("/v1/service-sources", publication.service); err != nil {
		return fmt.Errorf("publish A2A service (%s): %w", boundedResponse(response), err)
	}
	for _, route := range publication.routes {
		if response, err := client.post("/v1/routes", route.route); err != nil {
			return fmt.Errorf("publish A2A route %s (%s): %w", route.name, boundedResponse(response), err)
		}
	}
	if response, err := client.post("/v1/wasm-plugins", publication.plugin); err != nil {
		var httpErr *HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != 409 {
			return fmt.Errorf("publish A2A WasmPlugin (%s): %w", boundedResponse(response), err)
		}
	}
	for _, route := range publication.routes {
		instancePath := fmt.Sprintf("/v1/routes/%s/plugin-instances/%s", url.PathEscape(route.name), a2aPluginName)
		if response, err := client.put(instancePath, route.instance); err != nil {
			return fmt.Errorf("attach A2A WasmPlugin to %s (%s): %w", route.name, boundedResponse(response), err)
		}
	}
	return nil
}

func buildA2APublication(name, rawURL, externalBaseURL string) (*a2aPublication, error) {
	if name == "" {
		return nil, fmt.Errorf("A2A agent name is required")
	}
	upstream, err := url.Parse(rawURL)
	if err != nil || upstream.Hostname() == "" || (upstream.Scheme != "http" && upstream.Scheme != "https") {
		return nil, fmt.Errorf("invalid A2A agent URL %q", rawURL)
	}
	external, err := url.Parse(externalBaseURL)
	if err != nil || externalBaseURL == "" || external.Hostname() == "" || external.Scheme != "https" || external.User != nil || external.Fragment != "" {
		return nil, fmt.Errorf("a public HTTPS external A2A base URL is required")
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
	serviceDomain := upstream.Hostname()
	servicePort := interface{}(port)
	if net.ParseIP(upstream.Hostname()) != nil {
		serviceType = "static"
		serviceDomain = net.JoinHostPort(upstream.Hostname(), port)
		servicePort = 80
	}
	serviceName := "agent-" + name
	targetService := serviceName + "." + serviceType
	routeName := name + "-route"
	discoveryRouteName := name + "-discovery-route"
	matchPath := path.Clean("/" + strings.TrimPrefix(upstream.EscapedPath(), "/"))
	if matchPath == "." {
		matchPath = "/"
	}
	service := map[string]interface{}{
		"domain": serviceDomain, "type": serviceType, "port": servicePort,
		"name": serviceName, "proxyName": "", "domainForEdit": serviceDomain, "protocol": upstream.Scheme,
	}
	route := map[string]interface{}{
		"name":       routeName,
		"path":       map[string]interface{}{"matchType": "PRE", "matchValue": matchPath, "caseSensitive": true},
		"authConfig": map[string]interface{}{"enabled": false},
		"services":   []map[string]interface{}{{"name": targetService}},
	}
	discoveryRoute := map[string]interface{}{
		"name":       discoveryRouteName,
		"path":       map[string]interface{}{"matchType": "EQUAL", "matchValue": canonicalAgentCardPath, "caseSensitive": true},
		"authConfig": map[string]interface{}{"enabled": false},
		"services":   []map[string]interface{}{{"name": targetService}},
	}
	plugin := map[string]interface{}{
		"name": a2aPluginName, "pluginVersion": "1.0.0", "category": "protocol",
		"title": "A2A Protocol", "description": "Bounded A2A 1.0 JSON-RPC and SSE metadata extraction",
		"builtIn":         false,
		"imageRepository": "oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/a2a-protocol",
		"imageVersion":    a2aPluginVersion, "imagePullPolicy": "IfNotPresent", "phase": "AUTHN", "priority": 300,
	}
	configurations := map[string]interface{}{
		"protocolVersion": "1.0", "mode": "enforce",
		"legacy03":      map[string]interface{}{"enabled": false},
		"agent":         map[string]interface{}{"id": name, "externalBaseURL": strings.TrimRight(externalBaseURL, "/")},
		"jsonrpc":       map[string]interface{}{"maxRequestBytes": 4194304, "maxSSEEventBytes": 262144},
		"authorization": map[string]interface{}{"exposeInternalHeaders": true},
	}
	instance := func(target string) map[string]interface{} {
		return map[string]interface{}{
			"targets":        map[string]interface{}{"ROUTE": target},
			"pluginName":     a2aPluginName,
			"pluginVersion":  "1.0.0",
			"enabled":        true,
			"internal":       false,
			"configurations": configurations,
		}
	}
	return &a2aPublication{
		service: service,
		plugin:  plugin,
		routes: []a2aRoutePublication{
			{name: routeName, route: route, instance: instance(routeName)},
			{name: discoveryRouteName, route: discoveryRoute, instance: instance(discoveryRouteName)},
		},
	}, nil
}

func boundedResponse(response []byte) string {
	const max = 512
	if len(response) > max {
		response = response[:max]
	}
	return strings.TrimSpace(string(response))
}
