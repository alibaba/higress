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
	"reflect"
	"testing"
)

func TestAuthzParse(t *testing.T) {
	authz := authz{}
	inputCases := []struct {
		input  map[string]string
		expect *AuthzConfig
	}{
		{
			input: map[string]string{
				buildHigressAnnotationKey(authzTypeAnn): defaultAuthzType,
			},
			expect: nil,
		},
		{
			input: map[string]string{
				buildHigressAnnotationKey(authzTypeAnn): defaultAuthzType,
				buildHigressAnnotationKey(serviceAnn):   "ext-authz-service",
			},
			expect: &AuthzConfig{
				AuthzType: "ext-authz",
				ExtAuthz: &ExtAuthzConfig{
					AuthzProto: GRPC,
					AuthzService: &ServiceConfig{
						ServiceName: "ext-authz-service",
						ServicePort: 80,
					},
					RbacPolicyId: "default-ingress-test-ext-authz-policy",
				},
			},
		},
		{
			input: map[string]string{
				buildHigressAnnotationKey(authzTypeAnn):    defaultAuthzType,
				buildHigressAnnotationKey(serviceAnn):      "ext-authz-service",
				buildHigressAnnotationKey(protoAnn):        "grpc",
				buildHigressAnnotationKey(servicePortAnn):  "9000",
				buildHigressAnnotationKey(rbacPolicyIdAnn): "test-policy",
			},
			expect: &AuthzConfig{
				AuthzType: "ext-authz",
				ExtAuthz: &ExtAuthzConfig{
					AuthzProto: GRPC,
					AuthzService: &ServiceConfig{
						ServiceName: "ext-authz-service",
						ServicePort: 9000,
					},
					RbacPolicyId: "test-policy",
				},
			},
		},
		{
			input: map[string]string{
				buildHigressAnnotationKey(authzTypeAnn): defaultAuthzType,
				buildHigressAnnotationKey(serviceAnn):   "ext-authz-service",
				buildHigressAnnotationKey(protoAnn):     "http",
			},
			expect: &AuthzConfig{
				AuthzType: "ext-authz",
				ExtAuthz: &ExtAuthzConfig{
					AuthzProto: HTTP,
					AuthzService: &ServiceConfig{
						ServiceName: "ext-authz-service",
						ServicePort: 80,
					},
					RbacPolicyId: "default-ingress-test-ext-authz-policy",
				},
			},
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			config := &Ingress{
				Meta: Meta{
					Namespace: "default",
					Name:      "ingress-test",
					ClusterId: "cluster",
				},
			}

			_ = authz.Parse(inputCase.input, config, nil)
			if !reflect.DeepEqual(inputCase.expect, config.Authz) {
				t.Fatal("Should be equal")
			}
		})
	}
}
