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
	"errors"
	"reflect"
	"testing"

	"github.com/alibaba/higress/hgctl/pkg/helm"
	higresskube "github.com/alibaba/higress/hgctl/pkg/kubernetes"
	"github.com/alibaba/higress/hgctl/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
)

func TestConfigmapProfileStoreSaveCreatesConfigMapWhenAbsent(t *testing.T) {
	profile := testProfile()
	client := fake.NewSimpleClientset()
	store := newTestConfigmapProfileStore(client)

	var created *corev1.ConfigMap
	createCount := 0
	updateCount := 0
	client.PrependReactor("create", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createCount++
		createAction := action.(k8stesting.CreateAction)
		created = createAction.GetObject().(*corev1.ConfigMap).DeepCopy()
		return false, nil, nil
	})
	client.PrependReactor("update", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateCount++
		return false, nil, nil
	})

	name, err := store.Save(profile)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if name != "higress-system/higress-profile" {
		t.Fatalf("Save() name = %q, want %q", name, "higress-system/higress-profile")
	}
	if createCount != 1 {
		t.Fatalf("create count = %d, want 1", createCount)
	}
	if updateCount != 0 {
		t.Fatalf("update count = %d, want 0", updateCount)
	}
	if created == nil {
		t.Fatal("expected create to receive a ConfigMap")
	}
	assertDesiredConfigMap(t, created, profile)
	if created.ResourceVersion != "" {
		t.Fatalf("created ConfigMap resourceVersion = %q, want empty", created.ResourceVersion)
	}
}

func TestConfigmapProfileStoreSaveUpdatesExistingConfigMapWithResourceVersion(t *testing.T) {
	profile := testProfile()
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       profile.Global.Namespace,
			Name:            ProfileConfigmapName,
			ResourceVersion: "rv-123",
		},
	}
	client := fake.NewSimpleClientset(existing)
	store := newTestConfigmapProfileStore(client)

	var updated *corev1.ConfigMap
	updateCount := 0
	client.PrependReactor("update", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateCount++
		updateAction := action.(k8stesting.UpdateAction)
		updated = updateAction.GetObject().(*corev1.ConfigMap).DeepCopy()
		return false, nil, nil
	})

	name, err := store.Save(profile)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if name != "higress-system/higress-profile" {
		t.Fatalf("Save() name = %q, want %q", name, "higress-system/higress-profile")
	}
	if updateCount != 1 {
		t.Fatalf("update count = %d, want 1", updateCount)
	}
	if updated == nil {
		t.Fatal("expected update to receive a ConfigMap")
	}
	assertDesiredConfigMap(t, updated, profile)
	if updated.ResourceVersion != existing.ResourceVersion {
		t.Fatalf("updated ConfigMap resourceVersion = %q, want %q", updated.ResourceVersion, existing.ResourceVersion)
	}
}

func TestConfigmapProfileStoreSaveReturnsGetErrorWithoutWriting(t *testing.T) {
	profile := testProfile()
	client := fake.NewSimpleClientset()
	store := newTestConfigmapProfileStore(client)

	wantErr := errors.New("get failed")
	createCount := 0
	updateCount := 0
	client.PrependReactor("get", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, wantErr
	})
	client.PrependReactor("create", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createCount++
		return false, nil, nil
	})
	client.PrependReactor("update", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateCount++
		return false, nil, nil
	})

	name, err := store.Save(profile)
	if err != wantErr {
		t.Fatalf("Save() error = %v, want %v", err, wantErr)
	}
	if name != "" {
		t.Fatalf("Save() name = %q, want empty", name)
	}
	if createCount != 0 {
		t.Fatalf("create count = %d, want 0", createCount)
	}
	if updateCount != 0 {
		t.Fatalf("update count = %d, want 0", updateCount)
	}
}

func TestConfigmapProfileStoreSaveReturnsUpdateConflictWithoutRetry(t *testing.T) {
	profile := testProfile()
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       profile.Global.Namespace,
			Name:            ProfileConfigmapName,
			ResourceVersion: "rv-123",
		},
	}
	client := fake.NewSimpleClientset(existing)
	store := newTestConfigmapProfileStore(client)

	updateCount := 0
	wantErr := apierrors.NewConflict(schema.GroupResource{Resource: "configmaps"}, ProfileConfigmapName, errors.New("conflict"))
	client.PrependReactor("update", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateCount++
		return true, nil, wantErr
	})

	name, err := store.Save(profile)
	if err != wantErr {
		t.Fatalf("Save() error = %v, want %v", err, wantErr)
	}
	if name != "" {
		t.Fatalf("Save() name = %q, want empty", name)
	}
	if updateCount != 1 {
		t.Fatalf("update count = %d, want 1", updateCount)
	}
}

func testProfile() *helm.Profile {
	return &helm.Profile{
		HigressVersion: "1.0.0",
		Global: helm.ProfileGlobal{
			Install:      helm.InstallK8s,
			Namespace:    "higress-system",
			IngressClass: "higress",
		},
	}
}

func newTestConfigmapProfileStore(client kubernetes.Interface) *ConfigmapProfileStore {
	return &ConfigmapProfileStore{
		kubeCli: &fakeCLIClient{kube: client},
	}
}

func assertDesiredConfigMap(t *testing.T, configmap *corev1.ConfigMap, profile *helm.Profile) {
	t.Helper()

	if configmap.Namespace != profile.Global.Namespace {
		t.Fatalf("ConfigMap namespace = %q, want %q", configmap.Namespace, profile.Global.Namespace)
	}
	if configmap.Name != ProfileConfigmapName {
		t.Fatalf("ConfigMap name = %q, want %q", configmap.Name, ProfileConfigmapName)
	}

	wantData := map[string]string{
		ProfileConfigmapKey: util.ToYAML(profile),
	}
	if !reflect.DeepEqual(configmap.Data, wantData) {
		t.Fatalf("ConfigMap data = %#v, want %#v", configmap.Data, wantData)
	}

	profileJSON, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	wantAnnotations := map[string]string{
		ProfileConfigmapAnnotation: string(profileJSON),
	}
	if !reflect.DeepEqual(configmap.Annotations, wantAnnotations) {
		t.Fatalf("ConfigMap annotations = %#v, want %#v", configmap.Annotations, wantAnnotations)
	}
}

type fakeCLIClient struct {
	kube kubernetes.Interface
}

func (f *fakeCLIClient) RESTConfig() *rest.Config {
	return nil
}

func (f *fakeCLIClient) Pod(types.NamespacedName) (*corev1.Pod, error) {
	return nil, nil
}

func (f *fakeCLIClient) PodsForSelector(string, ...string) (*corev1.PodList, error) {
	return nil, nil
}

func (f *fakeCLIClient) PodExec(types.NamespacedName, string, string) (string, string, error) {
	return "", "", nil
}

func (f *fakeCLIClient) ApplyObject(*unstructured.Unstructured) error {
	return nil
}

func (f *fakeCLIClient) DeleteObject(*unstructured.Unstructured) error {
	return nil
}

func (f *fakeCLIClient) CreateNamespace(string) error {
	return nil
}

func (f *fakeCLIClient) KubernetesInterface() kubernetes.Interface {
	return f.kube
}

var _ higresskube.CLIClient = (*fakeCLIClient)(nil)
