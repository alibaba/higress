// Copyright (c) 2023 Alibaba Group Holding Ltd.
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

package common_test

import (
	"testing"

	"key-auth/common"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestParseConsumersConfig(t *testing.T) {
	assert := assert.New(t)

	testcases := []struct {
		description string
		input       struct {
			gjson gjson.Result
		}
		expect struct {
			consumers *common.Consumers
		}
	}{
		{
			description: "convert Consumers",
			input: struct {
				gjson gjson.Result
			}{
				gjson: gjson.Result{
					Type: gjson.JSON,
					Raw:  "",
				},
			},
			expect: struct {
				consumers *common.Consumers
			}{
				consumers: &common.Consumers{
					Consumers: []common.Consumer{},
				},
			},
		},
	}
	for _, testcase := range testcases {
		consumers := &common.Consumers{}
		err := common.ParseConsumersConfig(testcase.input.gjson, consumers, wrapper.Log{})
		assert.Nil(err)
		assert.Equal(testcase.expect.consumers, consumers)
	}
}
