/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package names

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/networking/pkg/apis/networking/v1alpha1"
	"knative.dev/pkg/kmeta"
)

func TestNamer(t *testing.T) {
	tests := []struct {
		name    string
		ingress *v1alpha1.Ingress
		f       func(kmeta.Accessor) string
		want    string
	}{{
		name: "IngressVirtualService",
		ingress: &v1alpha1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
		f:    IngressVirtualService,
		want: "foo-ingress",
	}, {
		name: "IngressVirtualService",
		ingress: &v1alpha1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "ns1",
			},
		},
		f:    IngressVirtualService,
		want: "foo-ingress",
	}, {
		name: "MeshVirtualService",
		ingress: &v1alpha1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		},
		f:    MeshVirtualService,
		want: "foo-mesh",
	}, {
		name: "MeshVirtualService",
		ingress: &v1alpha1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "ns2",
			},
		},
		f:    MeshVirtualService,
		want: "foo-mesh",
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.f(test.ingress)
			if got != test.want {
				t.Errorf("%s() = %v, wanted %v", test.name, got, test.want)
			}
		})
	}
}
