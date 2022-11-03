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

package ingressv1

import (
	"testing"

	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/higress/ingress/kube/common"
)

func TestShouldProcessIngressUpdate(t *testing.T) {
	c := controller{
		options: common.Options{
			IngressClass: "mse",
		},
		ingresses: make(map[string]*v1.Ingress),
	}

	ingressClass := "mse"

	ingress1 := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-1",
		},
		Spec: v1.IngressSpec{
			IngressClassName: &ingressClass,
			Rules: []v1.IngressRule{
				{
					Host: "test.com",
					IngressRuleValue: v1.IngressRuleValue{
						HTTP: &v1.HTTPIngressRuleValue{
							Paths: []v1.HTTPIngressPath{
								{
									Path: "/test",
								},
							},
						},
					},
				},
			},
		},
	}

	should, _ := c.shouldProcessIngressUpdate(ingress1)
	if !should {
		t.Fatal("should be true")
	}

	ingress2 := *ingress1
	should, _ = c.shouldProcessIngressUpdate(&ingress2)
	if should {
		t.Fatal("should be false")
	}

	ingress3 := *ingress1
	ingress3.Annotations = map[string]string{
		"test": "true",
	}
	should, _ = c.shouldProcessIngressUpdate(&ingress3)
	if !should {
		t.Fatal("should be true")
	}
}
