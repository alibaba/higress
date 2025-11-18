package observability_test

import (
	"testing"

	"github.com/alibaba/higress/pkg/observability"
)

func TestAIMetricsRecorders(t *testing.T) {
	m := observability.InitAIMetrics()
	m.RecordTokenUsage(42)
	m.ObserveLatencySeconds(0.123)
}