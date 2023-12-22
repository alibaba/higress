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

package config

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/alibaba/higress/pkg/cmd/hgctl/kubernetes"
	"github.com/stretchr/testify/assert"
)

var _ kubernetes.PortForwarder = &fakePortForwarder{}

type fakePortForwarder struct {
	responseBody []byte
	localPort    int
	l            net.Listener
	mux          *http.ServeMux
	stopCh       chan struct{}
}

func newFakePortForwarder(b []byte) (kubernetes.PortForwarder, error) {
	p, err := kubernetes.LocalAvailablePort("localhost")
	if err != nil {
		return nil, err
	}

	fw := &fakePortForwarder{
		responseBody: b,
		localPort:    p,
		mux:          http.NewServeMux(),
		stopCh:       make(chan struct{}),
	}
	fw.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fw.responseBody)
	})

	return fw, nil
}

func (fw *fakePortForwarder) WaitForStop() {
	<-fw.stopCh
}

func (fw *fakePortForwarder) Start() error {
	l, err := net.Listen("tcp", fw.Address())
	if err != nil {
		return err
	}
	fw.l = l

	go func() {
		if err := http.Serve(l, fw.mux); err != nil {
			log.Fatal(err)
		}
	}()

	return nil
}

func (fw *fakePortForwarder) Stop() {}

func (fw *fakePortForwarder) Address() string {
	return fmt.Sprintf("localhost:%d", fw.localPort)
}

func TestExtractAllConfigDump(t *testing.T) {
	input, err := readInputConfig("in.all.json")
	assert.NoError(t, err)
	fw, err := newFakePortForwarder(input)
	assert.NoError(t, err)
	err = fw.Start()
	assert.NoError(t, err)

	cases := []struct {
		output       string
		expected     string
		resourceType string
	}{
		{
			output:   "json",
			expected: "out.all.json",
		},
		{
			output:   "yaml",
			expected: "out.all.yaml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.output, func(t *testing.T) {
			configDump, err := fetchGatewayConfig(fw, true)
			assert.NoError(t, err)
			data, err := getXDSResource(AllEnvoyConfigType, configDump)
			assert.NoError(t, err)
			got, err := formatGatewayConfig(data, tc.output)
			assert.NoError(t, err)
			out, err := readOutputConfig(tc.expected)
			assert.NoError(t, err)
			if tc.output == "yaml" {
				assert.YAMLEq(t, string(out), string(got))
			} else {
				assert.JSONEq(t, string(out), string(got))
			}
		})
	}

	fw.Stop()
}

func TestExtractSubResourcesConfigDump(t *testing.T) {
	input, err := readInputConfig("in.all.json")
	assert.NoError(t, err)
	fw, err := newFakePortForwarder(input)
	assert.NoError(t, err)
	err = fw.Start()
	assert.NoError(t, err)

	cases := []struct {
		output       string
		expected     string
		resourceType EnvoyConfigType
	}{
		{
			output:       "json",
			resourceType: BootstrapEnvoyConfigType,
			expected:     "out.bootstrap.json",
		},
		{
			output:       "yaml",
			resourceType: BootstrapEnvoyConfigType,
			expected:     "out.bootstrap.yaml",
		}, {
			output:       "json",
			resourceType: ClusterEnvoyConfigType,
			expected:     "out.cluster.json",
		},
		{
			output:       "yaml",
			resourceType: ClusterEnvoyConfigType,
			expected:     "out.cluster.yaml",
		}, {
			output:       "json",
			resourceType: ListenerEnvoyConfigType,
			expected:     "out.listener.json",
		},
		{
			output:       "yaml",
			resourceType: ListenerEnvoyConfigType,
			expected:     "out.listener.yaml",
		}, {
			output:       "json",
			resourceType: RouteEnvoyConfigType,
			expected:     "out.route.json",
		},
		{
			output:       "yaml",
			resourceType: RouteEnvoyConfigType,
			expected:     "out.route.yaml",
		},
		{
			output:       "json",
			resourceType: EndpointEnvoyConfigType,
			expected:     "out.endpoints.json",
		},
		{
			output:       "yaml",
			resourceType: EndpointEnvoyConfigType,
			expected:     "out.endpoints.yaml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.output, func(t *testing.T) {
			configDump, err := fetchGatewayConfig(fw, false)
			assert.NoError(t, err)
			resource, err := getXDSResource(tc.resourceType, configDump)
			assert.NoError(t, err)
			got, err := formatGatewayConfig(resource, tc.output)
			assert.NoError(t, err)
			out, err := readOutputConfig(tc.expected)
			assert.NoError(t, err)
			if tc.output == "yaml" {
				assert.YAMLEq(t, string(out), string(got))
			} else {
				assert.JSONEq(t, string(out), string(got))
			}
		})
	}

	fw.Stop()
}

func readInputConfig(filename string) ([]byte, error) {
	b, err := os.ReadFile(path.Join("testdata", "config", "input", filename))
	if err != nil {
		return nil, err
	}
	return b, nil
}

func readOutputConfig(filename string) ([]byte, error) {
	b, err := os.ReadFile(path.Join("testdata", "config", "output", filename))
	if err != nil {
		return nil, err
	}
	return b, nil
}
