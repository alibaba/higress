//go:build karmada

package karmada

import (
	context "context"

	policyv1alpha1 "github.com/karmada-io/api/policy/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KarmadaSync propagates Higress resources across member clusters via Karmada.
// This implementation requires building with the "karmada" tag.
//
// Example build: CGO_ENABLED=0 go build -tags karmada ./...

type KarmadaSync struct {
	Client client.Client
}

func NewKarmadaSync(c client.Client) *KarmadaSync {
	return &KarmadaSync{Client: c}
}

// SyncConfigMap ensures the provided ConfigMap is distributed to all clusters
// according to a ClusterPropagationPolicy.
func (s *KarmadaSync) SyncConfigMap(cm *corev1.ConfigMap) error {
	if s == nil || s.Client == nil || cm == nil {
		return nil
	}
	// Create or update a ClusterPropagationPolicy to propagate the target ConfigMap.
	cpp := &policyv1alpha1.ClusterPropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "higress-configmaps",
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: corev1.SchemeGroupVersion.String(),
					Kind:       "ConfigMap",
					Name:       cm.Name,
					Namespace:  cm.Namespace,
				},
			},
			Placement: &policyv1alpha1.Placement{ // all clusters by default
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{},
			},
		},
	}
	ctx := context.TODO()
	var existing policyv1alpha1.ClusterPropagationPolicy
	if err := s.Client.Get(ctx, client.ObjectKey{Name: cpp.Name}, &existing); err == nil {
		existing.Spec = cpp.Spec
		return s.Client.Update(ctx, &existing)
	}
	return s.Client.Create(ctx, cpp)
}

// SyncCRD propagates custom resources by label/selector through a CPP.
func (s *KarmadaSync) SyncCRD(groupVersionKind, labelSelector string) error {
	if s == nil || s.Client == nil {
		return nil
	}
	cpp := &policyv1alpha1.ClusterPropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "higress-crds",
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: groupVersionKind, // e.g. apiextensions.k8s.io/v1
					Kind:       "CustomResourceDefinition",
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app.kubernetes.io/part-of": "higress"},
					},
				},
			},
			Placement: &policyv1alpha1.Placement{ClusterAffinity: &policyv1alpha1.ClusterAffinity{}},
		},
	}
	ctx := context.TODO()
	var existing policyv1alpha1.ClusterPropagationPolicy
	if err := s.Client.Get(ctx, client.ObjectKey{Name: cpp.Name}, &existing); err == nil {
		existing.Spec = cpp.Spec
		return s.Client.Update(ctx, &existing)
	}
	return s.Client.Create(ctx, cpp)
}