//go:build volcano

package volcano

import (
	context "context"

	batchv1alpha1 "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VolcanoScheduler struct {
	Client    client.Client
	Namespace string
}

func NewVolcanoScheduler(c client.Client, namespace string) *VolcanoScheduler {
	return &VolcanoScheduler{Client: c, Namespace: namespace}
}

func (s *VolcanoScheduler) ScheduleBatchJob(job *batchv1alpha1.Job) error {
	if s == nil || s.Client == nil || job == nil {
		return nil
	}
	job.ObjectMeta = metav1.ObjectMeta{
		Name:      job.Name,
		Namespace: s.Namespace,
	}
	return s.Client.Create(context.TODO(), job)
}