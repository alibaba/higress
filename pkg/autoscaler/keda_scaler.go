//go:build keda

package autoscaler

import (
	context "context"
	"fmt"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KedaScaler struct {
	Client      client.Client
	Namespace   string
	Deployment  string
}

func NewKedaScaler(c client.Client, namespace, deployment string) *KedaScaler {
	return &KedaScaler{Client: c, Namespace: namespace, Deployment: deployment}
}

// ScaleByLLMMetrics creates/updates a ScaledObject using a Prometheus trigger on a given metric.
func (s *KedaScaler) ScaleByLLMMetrics(metricName string, targetValue int) error {
	if s == nil || s.Client == nil || metricName == "" {
		return nil
	}
	ctx := context.TODO()
	so := &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "higress-" + s.Deployment + "-scaledobject",
			Namespace: s.Namespace,
		},
		Spec: kedav1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &kedav1alpha1.ScaleTarget{
				Kind: "Deployment",
				Name: s.Deployment,
			},
			Triggers: []kedav1alpha1.ScaleTriggers{
				{
					Type: "prometheus",
					Metadata: map[string]string{
						"serverAddress": "http://prometheus-server.monitoring.svc.cluster.local",
						"metricName":    metricName,
						"threshold":     fmt.Sprintf("%d", targetValue),
					},
				},
			},
		},
	}
	var existing kedav1alpha1.ScaledObject
	if err := s.Client.Get(ctx, client.ObjectKey{Name: so.Name, Namespace: s.Namespace}, &existing); err == nil {
		existing.Spec = so.Spec
		return s.Client.Update(ctx, &existing)
	}
	return s.Client.Create(ctx, so)
}