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
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/alibaba/higress/test/e2e/conformance/utils/http"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func init() {
	Register(HTTPRouteMirrorTargetService)
}

var HTTPRouteMirrorTargetService = suite.ConformanceTest{
	ShortName:   "HTTPRouteMirrorTargetService",
	Description: "The Ingress in the higress-conformance-infra namespace mirror request to target service",
	Features:    []suite.SupportedFeature{suite.HTTPConformanceFeature},
	Manifests:   []string{"tests/httproute-mirror-target-service.yaml"},
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		testcases := []http.Assertion{
			{
				Meta: http.AssertionMeta{
					TargetBackend:   "infra-backend-v1",
					TargetNamespace: "higress-conformance-infra",
				},
				Request: http.AssertionRequest{
					ActualRequest: http.Request{
						Path: "/mirror",
					},
				},
				Response: http.AssertionResponse{
					ExpectedResponse: http.Response{
						StatusCode: 200,
					},
				},
			},
		}

		t.Run("HTTPRoute mirror request to target service", func(t *testing.T) {
			for _, testcase := range testcases {
				http.MakeRequestAndExpectEventuallyConsistentResponse(t, suite.RoundTripper, suite.TimeoutConfig, suite.GatewayAddress, testcase)
				//check mirror's logs for request
				cfg, err := config.GetConfig()
				if err != nil {
					t.Fatalf("[httproute-mirror] get config failed.")
					return
				}
				clientSet, err := kubernetes.NewForConfig(cfg)
				if err != nil {
					t.Fatalf("[httproute-mirror] init clientset failed.")
					return
				}
				pods, err := clientSet.CoreV1().Pods("higress-conformance-infra").List(context.Background(), meta_v1.ListOptions{
					LabelSelector: meta_v1.FormatLabelSelector(&meta_v1.LabelSelector{MatchLabels: map[string]string{"app": "infra-backend-mirror"}}),
				})
				if err != nil || len(pods.Items) == 0 {
					t.Fatalf("[httproute-mirror] get pods by label of [\"app\": \"infra-backend-mirror\"] failed.")
					return
				}
				req := clientSet.CoreV1().Pods("higress-conformance-infra").GetLogs(pods.Items[0].Name, &v1.PodLogOptions{
					Container: "infra-backend-mirror",
					SinceTime: &meta_v1.Time{Time: time.Now().Add(-time.Second * 10)},
				})
				podLogs, err := req.Stream(context.Background())
				defer podLogs.Close()
				if err != nil {
					t.Fatalf("[httproute-mirror] init pod logs stream failed.")
					return
				}

				podBuf := new(bytes.Buffer)
				_, err = io.Copy(podBuf, podLogs)
				if err != nil {
					t.Fatalf("[httproute-mirror] read pod logs stream failed.")
					return
				}
				if !strings.Contains(podBuf.String(), "Echoing back request made to /mirror") {
					t.Fatalf("[httproute-mirror] mirror pod hasn't received any mirror requests in logs.")
					return
				}
			}
		})
	},
}
