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
	"testing"

	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestConfig(t *testing.T) {
	json := gjson.Result{Type: gjson.JSON, Raw: `{"serviceSource": "k8s","serviceName": "opa","servicePort": 8181,"namespace": "example1","policy": "example1","timeout": "5s"}`}
	config := &OpaConfig{}
	assert.NoError(t, parseConfig(json, config, log.Log{}))
	assert.Equal(t, config.policy, "example1")
	assert.Equal(t, config.timeout, uint32(5000))
	assert.NotNil(t, config.client)

	type tt struct {
		raw    string
		result bool
	}

	tests := []tt{
		{raw: `{}`, result: false},
		{raw: `{"policy": "example1","timeout": "5s"}`, result: false},
		{raw: `{"serviceSource": "route","host": "example.com","policy": "example1","timeout": "5s"}`, result: true},
		{raw: `{"serviceSource": "nacos","serviceName": "opa","servicePort": 8181,"policy": "example1","timeout": "5s"}`, result: false},
		{raw: `{"serviceSource": "nacos","serviceName": "opa","servicePort": 8181,"namespace": "example1","policy": "example1","timeout": "5s"}`, result: true},
	}

	for _, test := range tests {
		json = gjson.Result{Type: gjson.JSON, Raw: test.raw}
		assert.Equal(t, parseConfig(json, config, log.Log{}) == nil, test.result)
	}
}
