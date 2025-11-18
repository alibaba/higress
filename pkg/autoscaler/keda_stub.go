//go:build !keda

package autoscaler

type KedaScaler struct{}

func NewKedaScaler(_ any, _ string, _ string) *KedaScaler { return &KedaScaler{} }

func (s *KedaScaler) ScaleByLLMMetrics(_ string, _ int) error { return nil }