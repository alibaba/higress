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

package hgctl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/alibaba/higress/pkg/cmd/hgctl/kubernetes"
	"github.com/alibaba/higress/pkg/cmd/options"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

var (
	output       string
	podName      string
	podNamespace string
)

const (
	defaultProxyAdminPort = 15000
	containerName         = "envoy"
)

func retrieveConfigDump(args []string, includeEds bool) ([]byte, error) {
	if len(args) != 0 {
		podName = args[0]
	}

	if podNamespace == "" {
		return nil, fmt.Errorf("pod namespace is required")
	}

	if podName == "" || len(args) == 0 {
		c, err := kubernetes.NewCLIClient(options.DefaultConfigFlags.ToRawKubeConfigLoader())
		if err != nil {
			return nil, fmt.Errorf("failed to build kubernetes client: %w", err)
		}
		podList, err := c.PodsForSelector(podNamespace, "app=higress-gateway")
		if err != nil {
			return nil, err
		}
		if len(podList.Items) == 0 {
			return nil, fmt.Errorf("higress gateway pod is not existed in namespace %s", podNamespace)
		}

		podName = podList.Items[0].GetName()
	}

	fw, err := portForwarder(types.NamespacedName{
		Namespace: podNamespace,
		Name:      podName,
	})
	if err != nil {
		return nil, err
	}
	if err := fw.Start(); err != nil {
		return nil, err
	}
	defer fw.Stop()

	configDump, err := fetchGatewayConfig(fw, includeEds)
	if err != nil {
		return nil, err
	}

	return configDump, nil
}

func portForwarder(nn types.NamespacedName) (kubernetes.PortForwarder, error) {
	c, err := kubernetes.NewCLIClient(options.DefaultConfigFlags.ToRawKubeConfigLoader())
	if err != nil {
		return nil, fmt.Errorf("build CLI client fail: %w", err)
	}

	pod, err := c.Pod(nn)
	if err != nil {
		return nil, fmt.Errorf("get pod %s fail: %w", nn, err)
	}
	if pod.Status.Phase != "Running" {
		return nil, fmt.Errorf("pod %s is not running", nn)
	}

	fw, err := kubernetes.NewLocalPortForwarder(c, nn, 0, defaultProxyAdminPort, bindAddress)
	if err != nil {
		return nil, err
	}

	return fw, nil
}

func formatGatewayConfig(configDump any, output string) ([]byte, error) {
	out, err := json.MarshalIndent(configDump, "", "  ")
	if err != nil {
		return nil, err
	}

	if output == "yaml" {
		out, err = yaml.JSONToYAML(out)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func fetchGatewayConfig(fw kubernetes.PortForwarder, includeEds bool) ([]byte, error) {
	out, err := configDumpRequest(fw.Address(), includeEds)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func configDumpRequest(address string, includeEds bool) ([]byte, error) {
	url := fmt.Sprintf("http://%s/config_dump", address)
	if includeEds {
		url = fmt.Sprintf("%s?include_eds", url)
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return io.ReadAll(resp.Body)
}
