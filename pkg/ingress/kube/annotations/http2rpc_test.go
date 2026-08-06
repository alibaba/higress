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

package annotations

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHttp2RpcParse(t *testing.T) {
	parser := http2rpc{}

	testCases := []struct {
		input  Annotations
		expect *Http2RpcConfig
	}{
		{
			input:  Annotations{},
			expect: nil,
		},
		{
			input: Annotations{
				buildHigressAnnotationKey(rpcDestinationName): "",
			},
			expect: nil,
		},
		{
			input: Annotations{
				buildHigressAnnotationKey(rpcDestinationName): "http-dubbo-alibaba-nacos-example-DemoService",
			},
			expect: &Http2RpcConfig{
				Name: "http-dubbo-alibaba-nacos-example-DemoService",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run("", func(t *testing.T) {
			config := &Ingress{}
			_ = parser.Parse(testCase.input, config, nil)
			if diff := cmp.Diff(config.Http2Rpc, testCase.expect); diff != "" {
				t.Fatalf("TestHttp2RpcParse() mismatch: (-want +got)\n%s", diff)
			}
		})
	}
}
