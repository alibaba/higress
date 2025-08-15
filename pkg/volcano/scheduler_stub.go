//go:build !volcano

package volcano

type VolcanoScheduler struct{}

func NewVolcanoScheduler(_ any, _ string) *VolcanoScheduler { return &VolcanoScheduler{} }

func (s *VolcanoScheduler) ScheduleBatchJob(_ any) error { return nil }