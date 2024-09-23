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
	"github.com/alibaba/higress/pkg/ingress/kube/util"
	"github.com/golang/protobuf/proto"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"
	"reflect"
	"testing"
)

func TestParseMirror(t *testing.T) {
	testCases := []struct {
		input  []map[string]string
		expect *MirrorConfig
	}{
		{},
		{
			input: []map[string]string{
				{buildHigressAnnotationKey(mirrorTargetService): "test/app"},
				{buildNginxAnnotationKey(mirrorTargetService): "test/app"},
			},
			expect: &MirrorConfig{
				ServiceInfo: util.ServiceInfo{
					NamespacedName: model.NamespacedName{
						Namespace: "test",
						Name:      "app",
					},
					Port: 80,
				},
			},
		},
		{
			input: []map[string]string{
				{buildHigressAnnotationKey(mirrorTargetService): "test/app:8080"},
				{buildNginxAnnotationKey(mirrorTargetService): "test/app:8080"},
			},
			expect: &MirrorConfig{
				ServiceInfo: util.ServiceInfo{
					NamespacedName: model.NamespacedName{
						Namespace: "test",
						Name:      "app",
					},
					Port: 8080,
				},
			},
		},
		{
			input: []map[string]string{
				{buildHigressAnnotationKey(mirrorTargetService): "test/app:hi"},
				{buildNginxAnnotationKey(mirrorTargetService): "test/app:hi"},
			},
			expect: &MirrorConfig{
				ServiceInfo: util.ServiceInfo{
					NamespacedName: model.NamespacedName{
						Namespace: "test",
						Name:      "app",
					},
					Port: 80,
				},
			},
		},
		{
			input: []map[string]string{
				{buildHigressAnnotationKey(mirrorTargetService): "test/app"},
				{buildNginxAnnotationKey(mirrorTargetService): "test/app"},
			},
			expect: &MirrorConfig{
				ServiceInfo: util.ServiceInfo{
					NamespacedName: model.NamespacedName{
						Namespace: "test",
						Name:      "app",
					},
					Port: 80,
				},
			},
		},
	}

	mirror := mirror{}

	for _, testCase := range testCases {
		t.Run("", func(t *testing.T) {
			config := &Ingress{
				Meta: Meta{
					Namespace: "test",
					ClusterId: "cluster",
				},
			}
			globalContext, cancel := initGlobalContextForService()
			defer cancel()

			for _, in := range testCase.input {
				_ = mirror.Parse(in, config, globalContext)
				if !reflect.DeepEqual(testCase.expect, config.Mirror) {
					t.Log("expect:", *testCase.expect)
					t.Log("actual:", *config.Mirror)
					t.Fatal("Should be equal")
				}
			}
		})
	}
}

func TestMirror_ApplyRoute(t *testing.T) {
	testCases := []struct {
		config *Ingress
		input  *networking.HTTPRoute
		expect *networking.HTTPRoute
	}{
		{
			config: &Ingress{},
			input:  &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{},
		},
		{
			config: &Ingress{
				Mirror: &MirrorConfig{
					ServiceInfo: util.ServiceInfo{
						NamespacedName: model.NamespacedName{
							Namespace: "default",
							Name:      "test",
						},
						Port: 8080,
					},
				},
			},
			input: &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{
				Mirror: &networking.Destination{
					Host: "test.default.svc.cluster.local",
					Port: &networking.PortSelector{
						Number: 8080,
					},
				},
			},
		},
	}

	mirror := mirror{}
	for _, testCase := range testCases {
		t.Run("", func(t *testing.T) {
			mirror.ApplyRoute(testCase.input, testCase.config)
			if !proto.Equal(testCase.input, testCase.expect) {
				t.Fatal("Must be equal.")
			}
		})
	}
}
