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

package installer

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/alibaba/higress/hgctl/pkg/helm"
	hgctlkubernetes "github.com/alibaba/higress/hgctl/pkg/kubernetes"
	"github.com/alibaba/higress/hgctl/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// fakeCLIClient is a minimal kubernetes.CLIClient stub. Only
// KubernetesInterface is implemented because ConfigmapProfileStore.Save never
// calls any other CLIClient method.
type fakeCLIClient struct {
	hgctlkubernetes.CLIClient
	kube kubernetes.Interface
}

func (f *fakeCLIClient) KubernetesInterface() kubernetes.Interface {
	return f.kube
}

func newTestProfile() *helm.Profile {
	return &helm.Profile{
		Profile:        "test",
		HigressVersion: "2.1.0-test",
		Global: helm.ProfileGlobal{
			Install:      helm.InstallK8s,
			Namespace:    "higress-system",
			IngressClass: "higress",
		},
	}
}

func newConfigmapProfileStore(t *testing.T, objects ...runtime.Object) (ProfileStore, *fake.Clientset) {
	t.Helper()
	clientset := fake.NewSimpleClientset(objects...)
	store, err := NewConfigmapProfileStore(&fakeCLIClient{kube: clientset})
	if err != nil {
		t.Fatalf("NewConfigmapProfileStore() error = %v, want nil", err)
	}
	return store, clientset
}

func actionsForVerb(clientset *fake.Clientset, verb string) []k8stesting.Action {
	matched := make([]k8stesting.Action, 0)
	for _, action := range clientset.Actions() {
		if action.GetVerb() == verb {
			matched = append(matched, action)
		}
	}
	return matched
}

func wantProfileAnnotation(t *testing.T, profile *helm.Profile) string {
	t.Helper()
	bytes, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("json.Marshal(profile) error = %v, want nil", err)
	}
	return string(bytes)
}

func TestConfigmapProfileStoreSaveCreatesAbsentConfigmap(t *testing.T) {
	store, clientset := newConfigmapProfileStore(t)
	profile := newTestProfile()

	name, err := store.Save(profile)
	if err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}
	wantName := fmt.Sprintf("%s/%s", profile.Global.Namespace, ProfileConfigmapName)
	if name != wantName {
		t.Errorf("Save() name = %q, want %q", name, wantName)
	}

	creates := actionsForVerb(clientset, "create")
	if len(creates) != 1 {
		t.Fatalf("Save() performed %d create actions, want exactly 1 (all actions: %v)", len(creates), clientset.Actions())
	}
	if updates := actionsForVerb(clientset, "update"); len(updates) != 0 {
		t.Errorf("Save() performed %d update actions, want 0 for an absent ConfigMap", len(updates))
	}

	createAction, ok := creates[0].(k8stesting.CreateAction)
	if !ok {
		t.Fatalf("create action has type %T, want testing.CreateAction", creates[0])
	}
	created, ok := createAction.GetObject().(*corev1.ConfigMap)
	if !ok {
		t.Fatalf("created object has type %T, want *corev1.ConfigMap", createAction.GetObject())
	}
	if created.Name != ProfileConfigmapName {
		t.Errorf("created ConfigMap name = %q, want %q", created.Name, ProfileConfigmapName)
	}
	if created.Namespace != profile.Global.Namespace {
		t.Errorf("created ConfigMap namespace = %q, want %q", created.Namespace, profile.Global.Namespace)
	}
	if wantData := util.ToYAML(profile); created.Data[ProfileConfigmapKey] != wantData {
		t.Errorf("created ConfigMap Data[%q] = %q, want %q", ProfileConfigmapKey, created.Data[ProfileConfigmapKey], wantData)
	}
	if wantAnnotation := wantProfileAnnotation(t, profile); created.Annotations[ProfileConfigmapAnnotation] != wantAnnotation {
		t.Errorf("created ConfigMap Annotations[%q] = %q, want %q", ProfileConfigmapAnnotation, created.Annotations[ProfileConfigmapAnnotation], wantAnnotation)
	}
}

func TestConfigmapProfileStoreSaveUpdatesExistingConfigmapWithResourceVersion(t *testing.T) {
	const existingResourceVersion = "12345"
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "higress-system",
			Name:            ProfileConfigmapName,
			ResourceVersion: existingResourceVersion,
		},
	}
	store, clientset := newConfigmapProfileStore(t, existing)

	// The pinned k8s.io/client-go v0.34.1 fake clientset neither validates nor
	// bumps resourceVersion on update, so the object handed to Update is
	// captured with a reactor to assert the resourceVersion contract directly.
	var updated *corev1.ConfigMap
	clientset.PrependReactor("update", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateAction, ok := action.(k8stesting.UpdateAction)
		if !ok {
			t.Fatalf("update action has type %T, want testing.UpdateAction", action)
		}
		configmap, ok := updateAction.GetObject().(*corev1.ConfigMap)
		if !ok {
			t.Fatalf("updated object has type %T, want *corev1.ConfigMap", updateAction.GetObject())
		}
		updated = configmap
		return false, nil, nil
	})

	profile := newTestProfile()
	name, err := store.Save(profile)
	if err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}
	wantName := fmt.Sprintf("%s/%s", profile.Global.Namespace, ProfileConfigmapName)
	if name != wantName {
		t.Errorf("Save() name = %q, want %q", name, wantName)
	}

	if creates := actionsForVerb(clientset, "create"); len(creates) != 0 {
		t.Errorf("Save() performed %d create actions, want 0 for an existing ConfigMap", len(creates))
	}
	if updated == nil {
		t.Fatal("Save() did not update the existing ConfigMap, want exactly one update")
	}
	if updated.ResourceVersion != existingResourceVersion {
		t.Errorf("updated ConfigMap resourceVersion = %q, want existing resourceVersion %q", updated.ResourceVersion, existingResourceVersion)
	}
	if updated.Name != ProfileConfigmapName {
		t.Errorf("updated ConfigMap name = %q, want %q", updated.Name, ProfileConfigmapName)
	}
	if updated.Namespace != profile.Global.Namespace {
		t.Errorf("updated ConfigMap namespace = %q, want %q", updated.Namespace, profile.Global.Namespace)
	}
	if wantData := util.ToYAML(profile); updated.Data[ProfileConfigmapKey] != wantData {
		t.Errorf("updated ConfigMap Data[%q] = %q, want %q", ProfileConfigmapKey, updated.Data[ProfileConfigmapKey], wantData)
	}
	if wantAnnotation := wantProfileAnnotation(t, profile); updated.Annotations[ProfileConfigmapAnnotation] != wantAnnotation {
		t.Errorf("updated ConfigMap Annotations[%q] = %q, want %q", ProfileConfigmapAnnotation, updated.Annotations[ProfileConfigmapAnnotation], wantAnnotation)
	}
}

func TestConfigmapProfileStoreSaveReturnsGetErrorWithoutWrite(t *testing.T) {
	store, clientset := newConfigmapProfileStore(t)
	getErr := apierrors.NewInternalError(fmt.Errorf("boom"))
	clientset.PrependReactor("get", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, getErr
	})

	name, err := store.Save(newTestProfile())
	if err == nil {
		t.Fatal("Save() error = nil, want the Get error to be returned")
	}
	if err != getErr {
		t.Errorf("Save() error = %v, want the Get error %v returned unchanged", err, getErr)
	}
	if name != "" {
		t.Errorf("Save() name = %q, want empty on error", name)
	}
	if creates := actionsForVerb(clientset, "create"); len(creates) != 0 {
		t.Errorf("Save() performed %d create actions after a non-NotFound Get error, want 0", len(creates))
	}
	if updates := actionsForVerb(clientset, "update"); len(updates) != 0 {
		t.Errorf("Save() performed %d update actions after a non-NotFound Get error, want 0", len(updates))
	}
}

func TestConfigmapProfileStoreSavePropagatesUpdateConflictWithoutRetry(t *testing.T) {
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "higress-system",
			Name:            ProfileConfigmapName,
			ResourceVersion: "12345",
		},
	}
	store, clientset := newConfigmapProfileStore(t, existing)

	updateCalls := 0
	conflictErr := apierrors.NewConflict(schema.GroupResource{Resource: "configmaps"}, ProfileConfigmapName, fmt.Errorf("conflict"))
	clientset.PrependReactor("update", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateCalls++
		return true, nil, conflictErr
	})

	name, err := store.Save(newTestProfile())
	if err == nil {
		t.Fatal("Save() error = nil, want the Update Conflict to be returned")
	}
	if err != conflictErr {
		t.Errorf("Save() error = %v, want the Conflict error %v returned unchanged", err, conflictErr)
	}
	if !apierrors.IsConflict(err) {
		t.Errorf("Save() error = %v, want an error for which IsConflict reports true", err)
	}
	if name != "" {
		t.Errorf("Save() name = %q, want empty on error", name)
	}
	if updateCalls != 1 {
		t.Errorf("Save() attempted %d update calls, want exactly 1 (no Conflict retry)", updateCalls)
	}
}
