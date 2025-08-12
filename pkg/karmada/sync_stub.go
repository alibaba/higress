//go:build !karmada

package karmada

import corev1 "k8s.io/api/core/v1"

// KarmadaSync is a no-op stub when built without the "karmada" build tag.
type KarmadaSync struct{}

func NewKarmadaSync(_ any) *KarmadaSync { return &KarmadaSync{} }

func (s *KarmadaSync) SyncConfigMap(_ *corev1.ConfigMap) error { return nil }

func (s *KarmadaSync) SyncCRD(_ string, _ string) error { return nil }