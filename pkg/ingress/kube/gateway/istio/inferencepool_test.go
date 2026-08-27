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

package istio

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	inferencev1 "sigs.k8s.io/gateway-api-inference-extension/api/v1"

	"istio.io/istio/pilot/pkg/features"
	"istio.io/istio/pkg/config/constants"
	"istio.io/istio/pkg/config/gateway/kube"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/test"
	"istio.io/istio/pkg/test/util/assert"
	"istio.io/istio/pkg/util/sets"
)

func TestReconcileInferencePool(t *testing.T) {
	test.SetForTest(t, &features.EnableGatewayAPIInferenceExtension, true)
	pool := &inferencev1.InferencePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pool",
			Namespace: "default",
		},
		Spec: inferencev1.InferencePoolSpec{
			TargetPorts: []inferencev1.Port{
				{
					Number: inferencev1.PortNumber(8080),
				},
				{
					Number: inferencev1.PortNumber(8081),
				},
				{
					Number: inferencev1.PortNumber(8082),
				},
			},
			Selector: inferencev1.LabelSelector{
				MatchLabels: map[inferencev1.LabelKey]inferencev1.LabelValue{
					"app": "test",
				},
			},
			EndpointPickerRef: inferencev1.EndpointPickerRef{
				Name: "dummy",
				Port: &inferencev1.Port{
					Number: inferencev1.PortNumber(5421),
				},
			},
		},
	}
	controller := setupController(t,
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		NewGateway("test-gw", InNamespace(DefaultTestNS), WithGatewayClass("istio")),
		NewHTTPRoute("test-route", InNamespace(DefaultTestNS),
			WithParentRefAndStatus("test-gw", DefaultTestNS, IstioController),
			WithBackendRef("test-pool", DefaultTestNS),
		),
		pool,
	)

	dumpOnFailure(t, krt.GlobalDebugHandler)

	// Verify the service was created
	var service *corev1.Service
	var err error
	assert.EventuallyEqual(t, func() bool {
		svcName := "test-pool-ip-" + generateHash("test-pool", hashSize)
		service, err = controller.client.Kube().CoreV1().Services("default").Get(t.Context(), svcName, metav1.GetOptions{})
		if err != nil {
			t.Logf("Service %s not found yet: %v", svcName, err)
			return false
		}
		return service != nil
	}, true)

	assert.Equal(t, service.ObjectMeta.Labels[constants.InternalServiceSemantics], constants.ServiceSemanticsInferencePool)
	assert.Equal(t, service.ObjectMeta.Labels[InferencePoolRefLabel], pool.Name)
	assert.Equal(t, service.OwnerReferences[0].Name, pool.Name)
	assert.Equal(t, len(service.Spec.Ports), 3)
	for i, servicePort := range service.Spec.Ports {
		assert.Equal(t, servicePort.Name, fmt.Sprintf("http-%d", i))
		assert.Equal(t, servicePort.Port, int32(54321+i))
		assert.Equal(t, servicePort.TargetPort.IntVal, int32(8080+i))
	}
}

func TestInferencePoolEndpointPickerRefModes(t *testing.T) {
	builtinGroup := inferencev1.Group(kube.BuiltinInferenceEndpointPickerGroup)
	coreGroup := inferencev1.Group("")
	tests := []struct {
		name      string
		ref       inferencev1.EndpointPickerRef
		wantMode  kube.InferencePoolEndpointPickerMode
		wantError bool
	}{
		{name: "default core Service", ref: inferencev1.EndpointPickerRef{Name: "epp"}, wantMode: kube.InferencePoolEndpointPickerModeExternal},
		{name: "explicit core Service", ref: inferencev1.EndpointPickerRef{Group: &coreGroup, Kind: "Service", Name: "epp"}, wantMode: kube.InferencePoolEndpointPickerModeExternal},
		{name: "well-known BuiltIn", ref: inferencev1.EndpointPickerRef{Group: &builtinGroup, Kind: "WasmPlugin", Name: "ai-endpoint-picker"}, wantMode: kube.InferencePoolEndpointPickerModeBuiltin},
		{name: "near-match group", ref: inferencev1.EndpointPickerRef{Group: &coreGroup, Kind: "WasmPlugin", Name: "ai-endpoint-picker"}, wantMode: kube.InferencePoolEndpointPickerModeExternal, wantError: true},
		{name: "near-match kind", ref: inferencev1.EndpointPickerRef{Group: &builtinGroup, Kind: "Service", Name: "ai-endpoint-picker"}, wantMode: kube.InferencePoolEndpointPickerModeExternal, wantError: true},
		{name: "near-match name", ref: inferencev1.EndpointPickerRef{Group: &builtinGroup, Kind: "WasmPlugin", Name: "custom"}, wantMode: kube.InferencePoolEndpointPickerModeExternal, wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, message := inferencePoolEndpointPickerMode(tt.ref)
			assert.Equal(t, mode, tt.wantMode)
			assert.Equal(t, message != "", tt.wantError)
			if tt.wantError {
				pool := &inferencev1.InferencePool{Spec: inferencev1.InferencePoolSpec{EndpointPickerRef: tt.ref}}
				if got := createInferencePoolObject(pool, sets.New[types.NamespacedName]()); got != nil {
					t.Fatalf("unsupported ref created an InferencePool object: %+v", got)
				}
				resolved := calculateResolvedRefsStatus(pool, nil)
				assert.Equal(t, resolved.status, metav1.ConditionFalse)
				assert.Equal(t, resolved.reason, string(inferencev1.InferencePoolReasonInvalidExtensionRef))
			}
		})
	}
}

func TestBuiltinInferencePoolDoesNotResolveWasmOrUseServiceOptions(t *testing.T) {
	builtinGroup := inferencev1.Group(kube.BuiltinInferenceEndpointPickerGroup)
	pool := &inferencev1.InferencePool{
		ObjectMeta: metav1.ObjectMeta{Name: "pool", Namespace: "default"},
		Spec: inferencev1.InferencePoolSpec{EndpointPickerRef: inferencev1.EndpointPickerRef{
			Group: &builtinGroup, Kind: "WasmPlugin", Name: "ai-endpoint-picker",
			Port: &inferencev1.Port{Number: 9002}, FailureMode: inferencev1.EndpointPickerFailOpen,
		}},
	}
	object := createInferencePoolObject(pool, sets.New(types.NamespacedName{Name: "gateway", Namespace: "default"}))
	if object == nil || object.extRef.mode != kube.InferencePoolEndpointPickerModeBuiltin ||
		object.extRef.name != "" || object.extRef.port != 0 || object.extRef.failureMode != "" {
		t.Fatalf("unexpected BuiltIn collection result: %+v", object)
	}
	resolved := calculateResolvedRefsStatus(pool, nil)
	assert.Equal(t, resolved.status, metav1.ConditionTrue)
	assert.Equal(t, resolved.reason, string(inferencev1.InferencePoolReasonResolvedRefs))

	service := translateShadowServiceToService(map[string]string{
		InferencePoolExtensionRefSvc:         "stale-epp",
		InferencePoolExtensionRefPort:        "9002",
		InferencePoolExtensionRefFailureMode: string(inferencev1.EndpointPickerFailOpen),
	}, shadowServiceInfo{key: types.NamespacedName{Name: "pool-ip", Namespace: "default"}, poolName: "pool"}, object.extRef)
	assert.Equal(t, service.Labels[constants.InferencePoolEndpointPickerModeLabel], string(kube.InferencePoolEndpointPickerModeBuiltin))
	for _, label := range []string{InferencePoolExtensionRefSvc, InferencePoolExtensionRefPort, InferencePoolExtensionRefFailureMode} {
		if _, found := service.Labels[label]; found {
			t.Fatalf("BuiltIn shadow Service retained External-only label %q: %v", label, service.Labels)
		}
	}
}

func TestInvalidInferencePoolDeletesOnlyManagedShadowService(t *testing.T) {
	test.SetForTest(t, &features.EnableGatewayAPIInferenceExtension, true)
	builtinGroup := inferencev1.Group(kube.BuiltinInferenceEndpointPickerGroup)
	invalidGroup := inferencev1.Group(kube.BuiltinInferenceEndpointPickerGroup)
	for _, tt := range []struct {
		name string
		ref  inferencev1.EndpointPickerRef
	}{
		{name: "External to invalid", ref: inferencev1.EndpointPickerRef{Name: "epp", Port: &inferencev1.Port{Number: 9002}}},
		{name: "BuiltIn to invalid", ref: inferencev1.EndpointPickerRef{Group: &builtinGroup, Kind: "WasmPlugin", Name: "ai-endpoint-picker"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pool := &inferencev1.InferencePool{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pool", Namespace: DefaultTestNS},
				Spec: inferencev1.InferencePoolSpec{
					Selector:          inferencev1.LabelSelector{MatchLabels: map[inferencev1.LabelKey]inferencev1.LabelValue{"app": "test"}},
					TargetPorts:       []inferencev1.Port{{Number: 8080}},
					EndpointPickerRef: tt.ref,
				},
			}
			controller := setupController(t,
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: DefaultTestNS}},
				NewGateway("test-gw", InNamespace(DefaultTestNS), WithGatewayClass("istio")),
				NewHTTPRoute("test-route", InNamespace(DefaultTestNS),
					WithParentRefAndStatus("test-gw", DefaultTestNS, IstioController),
					WithBackendRef(pool.Name, DefaultTestNS)),
				pool,
			)
			serviceName, _ := InferencePoolServiceName(pool.Name)
			assert.EventuallyEqual(t, func() bool {
				_, err := controller.client.Kube().CoreV1().Services(DefaultTestNS).Get(t.Context(), serviceName, metav1.GetOptions{})
				return err == nil
			}, true)

			current, err := controller.client.GatewayAPIInference().InferenceV1().InferencePools(DefaultTestNS).Get(t.Context(), pool.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			current.Spec.EndpointPickerRef = inferencev1.EndpointPickerRef{Group: &invalidGroup, Kind: "WasmPlugin", Name: "near-match"}
			if _, err = controller.client.GatewayAPIInference().InferenceV1().InferencePools(DefaultTestNS).Update(t.Context(), current, metav1.UpdateOptions{}); err != nil {
				t.Fatal(err)
			}
			assert.EventuallyEqual(t, func() bool {
				_, err := controller.client.Kube().CoreV1().Services(DefaultTestNS).Get(t.Context(), serviceName, metav1.GetOptions{})
				return apierrors.IsNotFound(err)
			}, true)
		})
	}

	managed := translateShadowServiceToService(nil,
		shadowServiceInfo{key: types.NamespacedName{Name: "pool-ip", Namespace: "default"}, poolName: "pool"},
		extRefInfo{mode: kube.InferencePoolEndpointPickerModeBuiltin})
	if !isManagedShadowServiceForPool(managed, "pool") {
		t.Fatal("controller-created shadow Service was not recognized")
	}
	userManaged := managed.DeepCopy()
	userManaged.OwnerReferences = nil
	if isManagedShadowServiceForPool(userManaged, "pool") {
		t.Fatal("user-managed Service without the InferencePool owner must never be deleted")
	}
}

func invalidInferencePoolAndManagedShadowService(t *testing.T) (*inferencev1.InferencePool, *corev1.Service) {
	t.Helper()
	invalidGroup := inferencev1.Group(kube.BuiltinInferenceEndpointPickerGroup)
	pool := &inferencev1.InferencePool{
		ObjectMeta: metav1.ObjectMeta{Name: "invalid-pool", Namespace: DefaultTestNS, UID: types.UID("pool-uid")},
		Spec: inferencev1.InferencePoolSpec{
			Selector:          inferencev1.LabelSelector{MatchLabels: map[inferencev1.LabelKey]inferencev1.LabelValue{"app": "test"}},
			TargetPorts:       []inferencev1.Port{{Number: 8080}},
			EndpointPickerRef: inferencev1.EndpointPickerRef{Group: &invalidGroup, Kind: "WasmPlugin", Name: "near-match"},
		},
	}
	serviceName, err := InferencePoolServiceName(pool.Name)
	if err != nil {
		t.Fatal(err)
	}
	service := translateShadowServiceToService(nil, shadowServiceInfo{
		key:      types.NamespacedName{Name: serviceName, Namespace: pool.Namespace},
		poolName: pool.Name,
		poolUID:  pool.UID,
	}, extRefInfo{mode: kube.InferencePoolEndpointPickerModeBuiltin})
	return pool, service
}

func invalidInferencePoolControllerObjects(pool *inferencev1.InferencePool) []runtime.Object {
	return []runtime.Object{
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: DefaultTestNS}},
		NewGateway("test-gw", InNamespace(DefaultTestNS), WithGatewayClass("istio")),
		NewHTTPRoute("test-route", InNamespace(DefaultTestNS),
			WithParentRefAndStatus("test-gw", DefaultTestNS, IstioController),
			WithBackendRef(pool.Name, DefaultTestNS)),
		pool,
	}
}

func TestInvalidInferencePoolCleansPreexistingManagedShadowService(t *testing.T) {
	test.SetForTest(t, &features.EnableGatewayAPIInferenceExtension, true)
	pool, service := invalidInferencePoolAndManagedShadowService(t)
	objects := append(invalidInferencePoolControllerObjects(pool), service)
	controller := setupController(t, objects...)

	assert.EventuallyEqual(t, func() bool {
		_, err := controller.client.Kube().CoreV1().Services(service.Namespace).Get(t.Context(), service.Name, metav1.GetOptions{})
		return apierrors.IsNotFound(err)
	}, true)
}

func TestInvalidInferencePoolCleansLateManagedShadowService(t *testing.T) {
	test.SetForTest(t, &features.EnableGatewayAPIInferenceExtension, true)
	pool, service := invalidInferencePoolAndManagedShadowService(t)
	controller := setupController(t, invalidInferencePoolControllerObjects(pool)...)
	if _, err := controller.client.Kube().CoreV1().Services(service.Namespace).Create(t.Context(), service, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	assert.EventuallyEqual(t, func() bool {
		_, err := controller.client.Kube().CoreV1().Services(service.Namespace).Get(t.Context(), service.Name, metav1.GetOptions{})
		return apierrors.IsNotFound(err)
	}, true)
}
