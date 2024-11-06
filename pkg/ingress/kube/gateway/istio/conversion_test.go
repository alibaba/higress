// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Updated based on Istio codebase by Higress

package istio

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"istio.io/istio/pilot/pkg/config/kube/crd"
	credentials "istio.io/istio/pilot/pkg/credentials/kube"
	"istio.io/istio/pilot/pkg/features"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/model/kstatus"
	"istio.io/istio/pilot/pkg/networking/core/v1alpha3"
	"istio.io/istio/pilot/test/util"
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/config"
	crdvalidation "istio.io/istio/pkg/config/crd"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/test"
	"istio.io/istio/pkg/test/util/assert"
	"istio.io/istio/pkg/util/sets"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8s "sigs.k8s.io/gateway-api/apis/v1alpha2"
	"sigs.k8s.io/yaml"
)

// Start - Updated by Higress
var ports = []corev1.ServicePort{
	{
		Name:     "http",
		Port:     80,
		Protocol: "HTTP",
	},
	{
		Name:     "tcp",
		Port:     34000,
		Protocol: "TCP",
	},
}

var defaultGatewaySelector = map[string]string{
	"higress": "higress-system-higress-gateway",
}

var services = []corev1.Service{
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "higress-gateway",
			Namespace: "higress-system",
		},
		Spec: corev1.ServiceSpec{
			Ports:       ports,
			ExternalIPs: []string{"1.2.3.4"},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example.com",
			Namespace: "higress-system",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin-apple",
			Namespace: "apple",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin-banana",
			Namespace: "banana",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin-second",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin-wildcard",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin-other",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "echo",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin",
			Namespace: "cert",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-svc",
			Namespace: "service",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "google.com",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc2",
			Namespace: "allowed-1",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc2",
			Namespace: "allowed-2",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc1",
			Namespace: "allowed-1",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc3",
			Namespace: "allowed-2",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc4",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin",
			Namespace: "group-namespace1",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin",
			Namespace: "group-namespace2",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin-zero",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin",
			Namespace: "higress-system",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin-mirror",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin-foo",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin-alt",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "higress-controller",
			Namespace: "higress-system",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "echo",
			Namespace: "higress-system",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin-bad",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: ports,
		},
	},
}

var endpoints = []corev1.Endpoints{
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "higress-gateway",
			Namespace: "higress-system",
		},
		Subsets: []corev1.EndpointSubset{
			{
				Ports: []corev1.EndpointPort{
					{
						Port: 8080,
					},
				},
			},
		},
	},
}

// End - Updated by Higress

var (
	// https://github.com/kubernetes/kubernetes/blob/v1.25.4/staging/src/k8s.io/kubectl/pkg/cmd/create/create_secret_tls_test.go#L31
	rsaCertPEM = `-----BEGIN CERTIFICATE-----
MIIB0zCCAX2gAwIBAgIJAI/M7BYjwB+uMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTIwOTEyMjE1MjAyWhcNMTUwOTEyMjE1MjAyWjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBANLJ
hPHhITqQbPklG3ibCVxwGMRfp/v4XqhfdQHdcVfHap6NQ5Wok/4xIA+ui35/MmNa
rtNuC+BdZ1tMuVCPFZcCAwEAAaNQME4wHQYDVR0OBBYEFJvKs8RfJaXTH08W+SGv
zQyKn0H8MB8GA1UdIwQYMBaAFJvKs8RfJaXTH08W+SGvzQyKn0H8MAwGA1UdEwQF
MAMBAf8wDQYJKoZIhvcNAQEFBQADQQBJlffJHybjDGxRMqaRmDhX0+6v02TUKZsW
r5QuVbpQhH6u+0UgcW0jp9QwpxoPTLTWGXEWBBBurxFwiCBhkQ+V
-----END CERTIFICATE-----
`
	rsaKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIBOwIBAAJBANLJhPHhITqQbPklG3ibCVxwGMRfp/v4XqhfdQHdcVfHap6NQ5Wo
k/4xIA+ui35/MmNartNuC+BdZ1tMuVCPFZcCAwEAAQJAEJ2N+zsR0Xn8/Q6twa4G
6OB1M1WO+k+ztnX/1SvNeWu8D6GImtupLTYgjZcHufykj09jiHmjHx8u8ZZB/o1N
MQIhAPW+eyZo7ay3lMz1V01WVjNKK9QSn1MJlb06h/LuYv9FAiEA25WPedKgVyCW
SmUwbPw8fnTcpqDWE3yTO3vKcebqMSsCIBF3UmVue8YU3jybC3NxuXq3wNm34R8T
xVLHwDXh/6NJAiEAl2oHGGLz64BuAfjKrqwz7qMYr9HCLIe/YsoWq/olzScCIQDi
D2lWusoe2/nEqfDVVWGWlyJ7yOmqaVm/iNUN9B2N2g==
-----END RSA PRIVATE KEY-----
`

	secrets = []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-cert-http",
				Namespace: "higress-system",
			},
			Data: map[string][]byte{
				"tls.crt": []byte(rsaCertPEM),
				"tls.key": []byte(rsaKeyPEM),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cert",
				Namespace: "cert",
			},
			Data: map[string][]byte{
				"tls.crt": []byte(rsaCertPEM),
				"tls.key": []byte(rsaKeyPEM),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "malformed",
				Namespace: "higress-system",
			},
			Data: map[string][]byte{
				// nolint: lll
				// https://github.com/kubernetes-sigs/gateway-api/blob/d7f71d6b7df7e929ae299948973a693980afc183/conformance/tests/gateway-invalid-tls-certificateref.yaml#L87-L90
				// this certificate is invalid because contains an invalid pem (base64 of "Hello world"),
				// and the certificate and the key are identical
				"tls.crt": []byte("SGVsbG8gd29ybGQK"),
				"tls.key": []byte("SGVsbG8gd29ybGQK"),
			},
		},
	}
)

func init() {
	features.EnableAlphaGatewayAPI = true
	features.EnableAmbientControllers = true
	// Recompute with ambient enabled
	classInfos = getClassInfos()
	builtinClasses = getBuiltinClasses()
}

func TestConvertResources(t *testing.T) {
	validator := crdvalidation.NewIstioValidator(t)

	// Start - Updated by Higress
	client := kube.NewFakeClient()
	for _, svc := range services {
		if _, err := client.Kube().CoreV1().Services(svc.Namespace).Create(context.TODO(), &svc, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, endpoint := range endpoints {
		if _, err := client.Kube().CoreV1().Endpoints(endpoint.Namespace).Create(context.TODO(), &endpoint, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	// End - Updated by Higress

	cases := []struct {
		name string
	}{
		{"http"},
		{"tcp"},
		{"tls"},
		{"mismatch"},
		{"weighted"},
		{"zero"},
		{"invalid"},
		// 目前仅支持 type 为 Hostname 和 ServiceImport
		//{"multi-gateway"},
		{"delegated"},
		{"route-binding"},
		{"reference-policy-tls"},
		{"reference-policy-service"},
		//{"serviceentry"},
		{"alias"},
		//{"mcs"},
		{"route-precedence"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			input := readConfig(t, fmt.Sprintf("testdata/%s.yaml", tt.name), validator)
			cg := v1alpha3.NewConfigGenTest(t, v1alpha3.TestOptions{})
			kr := splitInput(t, input)
			kr.Context = NewGatewayContext(cg.PushContext(), client, "domain.suffix", "")
			output := convertResources(kr)
			output.AllowedReferences = AllowedReferences{} // Not tested here
			output.ReferencedNamespaceKeys = nil           // Not tested here
			output.ResourceReferences = nil                // Not tested here

			// sort virtual services to make the order deterministic
			sort.Slice(output.VirtualService, func(i, j int) bool {
				return output.VirtualService[i].Namespace+"/"+output.VirtualService[i].Name < output.VirtualService[j].Namespace+"/"+output.VirtualService[j].Name
			})
			goldenFile := fmt.Sprintf("testdata/%s.yaml.golden", tt.name)
			res := append(output.Gateway, output.VirtualService...)
			util.CompareContent(t, marshalYaml(t, res), goldenFile)
			golden := splitOutput(readConfig(t, goldenFile, validator))

			// sort virtual services to make the order deterministic
			sort.Slice(golden.VirtualService, func(i, j int) bool {
				return golden.VirtualService[i].Namespace+"/"+golden.VirtualService[i].Name < golden.VirtualService[j].Namespace+"/"+golden.VirtualService[j].Name
			})

			assert.Equal(t, golden, output)

			outputStatus := getStatus(t, kr.GatewayClass, kr.Gateway, kr.HTTPRoute, kr.TLSRoute, kr.TCPRoute)
			goldenStatusFile := fmt.Sprintf("testdata/%s.status.yaml.golden", tt.name)
			if util.Refresh() {
				if err := os.WriteFile(goldenStatusFile, outputStatus, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			goldenStatus, err := os.ReadFile(goldenStatusFile)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(string(goldenStatus), string(outputStatus)); diff != "" {
				t.Fatalf("Diff:\n%s", diff)
			}
		})
	}
}

func TestReferencePolicy(t *testing.T) {
	validator := crdvalidation.NewIstioValidator(t)
	type res struct {
		name, namespace string
		allowed         bool
	}
	cases := []struct {
		name         string
		config       string
		expectations []res
	}{
		{
			name: "simple",
			config: `apiVersion: gateway.networking.k8s.io/v1alpha2
kind: ReferenceGrant
metadata:
  name: allow-gateways-to-ref-secrets
  namespace: default
spec:
  from:
  - group: gateway.networking.k8s.io
    kind: Gateway
    namespace: higress-system
  to:
  - group: ""
    kind: Secret
`,
			expectations: []res{
				// allow cross namespace
				{"kubernetes-gateway://default/wildcard-example-com-cert", "higress-system", true},
				// denied same namespace. We do not implicitly allow (in this code - higher level code does)
				{"kubernetes-gateway://default/wildcard-example-com-cert", "default", false},
				// denied namespace
				{"kubernetes-gateway://default/wildcard-example-com-cert", "bad", false},
			},
		},
		{
			name: "multiple in one",
			config: `apiVersion: gateway.networking.k8s.io/v1alpha2
kind: ReferenceGrant
metadata:
  name: allow-gateways-to-ref-secrets
  namespace: default
spec:
  from:
  - group: gateway.networking.k8s.io
    kind: Gateway
    namespace: ns-1
  - group: gateway.networking.k8s.io
    kind: Gateway
    namespace: ns-2
  to:
  - group: ""
    kind: Secret
`,
			expectations: []res{
				{"kubernetes-gateway://default/wildcard-example-com-cert", "ns-1", true},
				{"kubernetes-gateway://default/wildcard-example-com-cert", "ns-2", true},
				{"kubernetes-gateway://default/wildcard-example-com-cert", "bad", false},
			},
		},
		{
			name: "multiple",
			config: `apiVersion: gateway.networking.k8s.io/v1alpha2
kind: ReferenceGrant
metadata:
  name: ns1
  namespace: default
spec:
  from:
  - group: gateway.networking.k8s.io
    kind: Gateway
    namespace: ns-1
  to:
  - group: ""
    kind: Secret
---
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: ReferenceGrant
metadata:
  name: ns2
  namespace: default
spec:
  from:
  - group: gateway.networking.k8s.io
    kind: Gateway
    namespace: ns-2
  to:
  - group: ""
    kind: Secret
`,
			expectations: []res{
				{"kubernetes-gateway://default/wildcard-example-com-cert", "ns-1", true},
				{"kubernetes-gateway://default/wildcard-example-com-cert", "ns-2", true},
				{"kubernetes-gateway://default/wildcard-example-com-cert", "bad", false},
			},
		},
		{
			name: "same namespace",
			config: `apiVersion: gateway.networking.k8s.io/v1alpha2
kind: ReferenceGrant
metadata:
  name: allow-gateways-to-ref-secrets
  namespace: default
spec:
  from:
  - group: gateway.networking.k8s.io
    kind: Gateway
    namespace: default
  to:
  - group: ""
    kind: Secret
`,
			expectations: []res{
				{"kubernetes-gateway://default/wildcard-example-com-cert", "higress-system", false},
				{"kubernetes-gateway://default/wildcard-example-com-cert", "default", true},
				{"kubernetes-gateway://default/wildcard-example-com-cert", "bad", false},
			},
		},
		{
			name: "same name",
			config: `apiVersion: gateway.networking.k8s.io/v1alpha2
kind: ReferenceGrant
metadata:
  name: allow-gateways-to-ref-secrets
  namespace: default
spec:
  from:
  - group: gateway.networking.k8s.io
    kind: Gateway
    namespace: default
  to:
  - group: ""
    kind: Secret
    name: public
`,
			expectations: []res{
				{"kubernetes-gateway://default/public", "higress-system", false},
				{"kubernetes-gateway://default/public", "default", true},
				{"kubernetes-gateway://default/private", "default", false},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			input := readConfigString(t, tt.config, validator)
			cg := v1alpha3.NewConfigGenTest(t, v1alpha3.TestOptions{})
			kr := splitInput(t, input)
			kr.Context = NewGatewayContext(cg.PushContext(), nil, "", "")
			output := convertResources(kr)
			c := &Controller{
				state: output,
			}
			for _, sc := range tt.expectations {
				t.Run(fmt.Sprintf("%v/%v", sc.name, sc.namespace), func(t *testing.T) {
					got := c.SecretAllowed(sc.name, sc.namespace)
					if got != sc.allowed {
						t.Fatalf("expected allowed=%v, got allowed=%v", sc.allowed, got)
					}
				})
			}
		})
	}
}

func getStatus(t test.Failer, acfgs ...[]config.Config) []byte {
	cfgs := []config.Config{}
	for _, cl := range acfgs {
		cfgs = append(cfgs, cl...)
	}
	for i, c := range cfgs {
		c = c.DeepCopy()
		c.Spec = nil
		c.Labels = nil
		c.Annotations = nil
		if c.Status.(*kstatus.WrappedStatus) != nil {
			c.Status = c.Status.(*kstatus.WrappedStatus).Status
		}
		cfgs[i] = c
	}
	return timestampRegex.ReplaceAll(marshalYaml(t, cfgs), []byte("lastTransitionTime: fake"))
}

var timestampRegex = regexp.MustCompile(`lastTransitionTime:.*`)

func splitOutput(configs []config.Config) IstioResources {
	out := IstioResources{
		Gateway:        []config.Config{},
		VirtualService: []config.Config{},
	}
	for _, c := range configs {
		c.Domain = "domain.suffix"
		switch c.GroupVersionKind {
		case gvk.Gateway:
			out.Gateway = append(out.Gateway, c)
		case gvk.VirtualService:
			out.VirtualService = append(out.VirtualService, c)
		}
	}
	return out
}

func splitInput(t test.Failer, configs []config.Config) GatewayResources {
	out := GatewayResources{DefaultGatewaySelector: defaultGatewaySelector}
	namespaces := sets.New[string]()
	for _, c := range configs {
		namespaces.Insert(c.Namespace)
		switch c.GroupVersionKind {
		case gvk.GatewayClass:
			out.GatewayClass = append(out.GatewayClass, c)
		case gvk.KubernetesGateway:
			out.Gateway = append(out.Gateway, c)
		case gvk.HTTPRoute:
			out.HTTPRoute = append(out.HTTPRoute, c)
		case gvk.TCPRoute:
			out.TCPRoute = append(out.TCPRoute, c)
		case gvk.TLSRoute:
			out.TLSRoute = append(out.TLSRoute, c)
		case gvk.ReferenceGrant:
			out.ReferenceGrant = append(out.ReferenceGrant, c)
		}
	}
	out.Namespaces = map[string]*corev1.Namespace{}
	for ns := range namespaces {
		out.Namespaces[ns] = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
				Labels: map[string]string{
					"istio.io/test-name-part": strings.Split(ns, "-")[0],
				},
			},
		}
	}

	client := kube.NewFakeClient(secrets...)
	out.Credentials = credentials.NewCredentialsController(client)
	client.RunAndWait(test.NewStop(t))

	out.Domain = "domain.suffix"
	return out
}

func readConfig(t testing.TB, filename string, validator *crdvalidation.Validator) []config.Config {
	t.Helper()

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read input yaml file: %v", err)
	}
	return readConfigString(t, string(data), validator)
}

func readConfigString(t testing.TB, data string, validator *crdvalidation.Validator) []config.Config {
	if err := validator.ValidateCustomResourceYAML(data); err != nil {
		t.Error(err)
	}
	c, _, err := crd.ParseInputs(data)
	if err != nil {
		t.Fatalf("failed to parse CRD: %v", err)
	}
	return insertDefaults(c)
}

// insertDefaults sets default values that would be present when reading from Kubernetes but not from
// files
func insertDefaults(cfgs []config.Config) []config.Config {
	res := make([]config.Config, 0, len(cfgs))
	for _, c := range cfgs {
		switch c.GroupVersionKind {
		case gvk.GatewayClass:
			c.Status = kstatus.Wrap(&k8s.GatewayClassStatus{})
		case gvk.KubernetesGateway:
			c.Status = kstatus.Wrap(&k8s.GatewayStatus{})
		case gvk.HTTPRoute:
			c.Status = kstatus.Wrap(&k8s.HTTPRouteStatus{})
		case gvk.TCPRoute:
			c.Status = kstatus.Wrap(&k8s.TCPRouteStatus{})
		case gvk.TLSRoute:
			c.Status = kstatus.Wrap(&k8s.TLSRouteStatus{})
		}
		res = append(res, c)
	}
	return res
}

// Print as YAML
func marshalYaml(t test.Failer, cl []config.Config) []byte {
	t.Helper()
	result := []byte{}
	separator := []byte("---\n")
	for _, config := range cl {
		obj, err := crd.ConvertConfig(config)
		if err != nil {
			t.Fatalf("Could not decode %v: %v", config.Name, err)
		}
		bytes, err := yaml.Marshal(obj)
		if err != nil {
			t.Fatalf("Could not convert %v to YAML: %v", config, err)
		}
		result = append(result, bytes...)
		result = append(result, separator...)
	}
	return result
}

func TestHumanReadableJoin(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{[]string{"a"}, "a"},
		{[]string{"a", "b"}, "a and b"},
		{[]string{"a", "b", "c"}, "a, b, and c"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.input, "_"), func(t *testing.T) {
			if got := humanReadableJoin(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkBuildHTTPVirtualServices(b *testing.B) {
	ports := []*model.Port{
		{
			Name:     "http",
			Port:     80,
			Protocol: "HTTP",
		},
		{
			Name:     "tcp",
			Port:     34000,
			Protocol: "TCP",
		},
	}
	ingressSvc := &model.Service{
		Attributes: model.ServiceAttributes{
			Name:      "higress-gateway",
			Namespace: "higress-system",
			ClusterExternalAddresses: &model.AddressMap{
				Addresses: map[cluster.ID][]string{
					"Kubernetes": {"1.2.3.4"},
				},
			},
		},
		Ports:    ports,
		Hostname: "higress-gateway.higress-system.svc.domain.suffix",
	}
	altIngressSvc := &model.Service{
		Attributes: model.ServiceAttributes{
			Namespace: "higress-system",
		},
		Ports:    ports,
		Hostname: "example.com",
	}
	cg := v1alpha3.NewConfigGenTest(b, v1alpha3.TestOptions{
		Services: []*model.Service{ingressSvc, altIngressSvc},
		Instances: []*model.ServiceInstance{
			{Service: ingressSvc, ServicePort: ingressSvc.Ports[0], Endpoint: &model.IstioEndpoint{EndpointPort: 8080}},
			{Service: ingressSvc, ServicePort: ingressSvc.Ports[1], Endpoint: &model.IstioEndpoint{}},
			{Service: altIngressSvc, ServicePort: altIngressSvc.Ports[0], Endpoint: &model.IstioEndpoint{}},
			{Service: altIngressSvc, ServicePort: altIngressSvc.Ports[1], Endpoint: &model.IstioEndpoint{}},
		},
	})

	validator := crdvalidation.NewIstioValidator(b)
	input := readConfig(b, "testdata/benchmark-httproute.yaml", validator)
	kr := splitInput(b, input)
	kr.Context = NewGatewayContext(cg.PushContext(), nil, "", "")
	ctx := configContext{
		GatewayResources:  kr,
		AllowedReferences: convertReferencePolicies(kr),
	}
	_, gwMap, _ := convertGateways(ctx)
	ctx.GatewayReferences = gwMap

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		// for gateway routes, build one VS per gateway+host
		gatewayRoutes := make(map[string]map[string]*config.Config)
		// for mesh routes, build one VS per namespace+host
		meshRoutes := make(map[string]map[string]*config.Config)
		for _, obj := range kr.HTTPRoute {
			buildHTTPVirtualServices(ctx, obj, gatewayRoutes, meshRoutes)
		}
	}
}

// Start - Updated by Higress
func TestExtractGatewayServices(t *testing.T) {
	tests := []struct {
		name              string
		r                 GatewayResources
		kgw               *k8s.GatewaySpec
		obj               config.Config
		gatewayServices   []string
		useDefaultService bool
		err               *ConfigError
	}{
		{
			name: "default gateway",
			r:    GatewayResources{Domain: "cluster.local", DefaultGatewaySelector: defaultGatewaySelector},
			kgw: &k8s.GatewaySpec{
				GatewayClassName: "higress",
			},
			obj: config.Config{
				Meta: config.Meta{
					Name:      "foo",
					Namespace: "default",
				},
			},
			gatewayServices:   []string{"higress-gateway.higress-system.svc.cluster.local"},
			useDefaultService: true,
		},
		{
			name: "default gateway with name overridden",
			r:    GatewayResources{Domain: "cluster.local", DefaultGatewaySelector: defaultGatewaySelector},
			kgw: &k8s.GatewaySpec{
				GatewayClassName: "higress",
			},
			obj: config.Config{
				Meta: config.Meta{
					Name:      "foo",
					Namespace: "default",
					Annotations: map[string]string{
						gatewayNameOverride: "bar",
					},
				},
			},
			gatewayServices: []string{"bar.default.svc.cluster.local"},
		},
		{
			name: "unmanaged gateway with only hostname address",
			r:    GatewayResources{Domain: "domain", DefaultGatewaySelector: defaultGatewaySelector},
			kgw: &k8s.GatewaySpec{
				GatewayClassName: "higress",
				Addresses: []k8s.GatewayAddress{
					{
						Type: func() *k8s.AddressType {
							t := k8s.HostnameAddressType
							return &t
						}(),
						Value: "example.com",
					},
				},
			},
			obj: config.Config{
				Meta: config.Meta{
					Name:      "foo",
					Namespace: "default",
				},
			},
			gatewayServices: []string{"example.com"},
		},
		{
			name: "unmanaged gateway with other address types",
			r:    GatewayResources{Domain: "domain", DefaultGatewaySelector: defaultGatewaySelector},
			kgw: &k8s.GatewaySpec{
				GatewayClassName: "higress",
				Addresses: []k8s.GatewayAddress{
					{
						Value: "abc",
					},
					{
						Type: func() *k8s.AddressType {
							t := k8s.HostnameAddressType
							return &t
						}(),
						Value: "example.com",
					},
					{
						Type: func() *k8s.AddressType {
							t := k8s.IPAddressType
							return &t
						}(),
						Value: "1.2.3.4",
					},
				},
			},
			obj: config.Config{
				Meta: config.Meta{
					Name:      "foo",
					Namespace: "default",
				},
			},
			gatewayServices: []string{"abc.default.svc.domain", "example.com"},
			err: &ConfigError{
				Reason:  InvalidAddress,
				Message: "only Hostname is supported, ignoring [1.2.3.4]",
			},
		},
		{
			name: "unmanaged gateway with empty type address",
			r:    GatewayResources{Domain: "domain", DefaultGatewaySelector: defaultGatewaySelector},
			kgw: &k8s.GatewaySpec{
				GatewayClassName: "higress",
				Addresses: []k8s.GatewayAddress{
					{
						Value: "abc",
					},
				},
			},
			obj: config.Config{
				Meta: config.Meta{
					Name:      "foo",
					Namespace: "default",
				},
			},
			gatewayServices: []string{"abc.default.svc.domain"},
		},
		{
			name: "unmanaged gateway with no hostname address",
			r:    GatewayResources{Domain: "domain", DefaultGatewaySelector: defaultGatewaySelector},
			kgw: &k8s.GatewaySpec{
				GatewayClassName: "higress",
				Addresses: []k8s.GatewayAddress{
					{
						Type: func() *k8s.AddressType {
							t := k8s.IPAddressType
							return &t
						}(),
						Value: "1.2.3.4",
					},
				},
			},
			obj: config.Config{
				Meta: config.Meta{
					Name:      "foo",
					Namespace: "default",
				},
			},
			gatewayServices:   []string{"higress-gateway.higress-system.svc.cluster.local"},
			useDefaultService: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gatewayServices, useDefaultService, err := extractGatewayServices(tt.r, tt.kgw, tt.obj)
			assert.Equal(t, gatewayServices, tt.gatewayServices)
			assert.Equal(t, useDefaultService, tt.useDefaultService)
			assert.Equal(t, err, tt.err)
		})
	}
}

// End - Updated by Higress
