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

package tests

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/alibaba/higress/test/e2e/conformance/utils/http"
	"github.com/alibaba/higress/test/e2e/conformance/utils/roundtripper"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
)

func init() {
	Register(RustWasmPluginsSseTiming)
}

const SseResponsePattern = `: server-timing: higress;dur=\d+\n` +
	`: this is a test stream\n\n` +
	`: server-timing: higress;dur=\d+\n` +
	`data: some text\n` +
	`data: another message\n` +
	`data: with two lines\n\n` +
	`: server-timing: higress;dur=\d+\n` +
	`event: userconnect\n` +
	`data: \{"username": "bobby", "time": "02:33:48"\}\n\n` +
	`: server-timing: higress;dur=\d+\n` +
	`event: usermessage\n` +
	`data: \{"username": "bobby", "time": "02:34:11", "text": "Hi everyone."\}\n\n` +
	`: server-timing: higress;dur=\d+\n` +
	`event: userdisconnect\n` +
	`data: \{"username": "bobby", "time": "02:34:23"\}\n\n` +
	`: server-timing: higress;dur=\d+\n` +
	`event: usermessage\n` +
	`data: \{"username": "sean", "time": "02:34:36", "text": "Bye, bobby."\}\n\n`

const SseEosResponsePattern = `: server-timing: higress;dur=\d+\n` +
	`: this is a test stream\n\n` +
	`: server-timing: higress;dur=\d+\n` +
	`data: some text\n` +
	`data: another message\n` +
	`data: with two lines\n\n` +
	`: server-timing: higress;dur=\d+\n` +
	`event: userconnect\n` +
	`data: \{"username": "bobby", "time": "02:33:48"\}\n\n` +
	`: server-timing: higress;dur=\d+\n` +
	`event: usermessage\n` +
	`data: \{"username": "bobby", "time": "02:34:11", "text": "Hi everyone."\}\n\n` +
	`: server-timing: higress;dur=\d+\n` +
	`event: userdisconnect\n` +
	`data: \{"username": "bobby", "time": "02:34:23"\}\n\n` +
	`: server-timing: higress;dur=\d+\n` +
	`event: usermessage\n` +
	`data: \{"username": "sean", "time": "02:34:36", "text": "Bye, bobby."\}`

var RustWasmPluginsSseTiming = suite.ConformanceTest{
	ShortName:   "RustWasmPluginsSseTiming",
	Description: "The Ingress in the higress-conformance-infra namespace test the rust sse-timing wasmplugins.",
	Manifests:   []string{"tests/rust-wasm-sse-timing.yaml"},
	Features:    []suite.SupportedFeature{suite.WASMRustConformanceFeature},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "sse.openai.com",
						Path:             "/",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeEventStream,
						Assert:      RegexpAssert(SseResponsePattern),
					},
					ExpectedResponseNoRequest: true,
				},
			},
			{
				Meta: http.AssertionMeta{
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "sse-eos.openai.com",
						Path:             "/",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeEventStream,
						Assert:      RegexpAssert(SseEosResponsePattern),
					},
					ExpectedResponseNoRequest: true,
				},
			},
			{
				Meta: http.AssertionMeta{
					CompareTarget: http.CompareTargetResponse,
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Host:             "json.openai.com",
						Path:             "/",
						UnfollowRedirect: true,
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode:  200,
						ContentType: http.ContentTypeApplicationJson,
						Body:        []byte("{\"foo\":\"bar\"}"),
					},
					ExpectedResponseNoRequest: true,
				},
			},
		}
		t.Run("WasmPlugins sse-timing", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
			}
		})
	},
}

func RegexpAssert(p string) func(*roundtripper.CapturedResponse) error {
	pattern := regexp.MustCompile(p)
	return func(cr *roundtripper.CapturedResponse) error {
		b := cr.Body
		if !pattern.Match(b) {
			return fmt.Errorf("expected response pattern to be %s, got %s", p, string(cr.Body))
		}
		return nil
	}
}
