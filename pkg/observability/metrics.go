package observability

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsOnce sync.Once
	aiMetrics   *AIMetrics
)

// AIMetrics holds Prometheus metrics for AI usage.
type AIMetrics struct {
	TokenUsage   prometheus.Counter
	ModelLatency prometheus.Histogram
}

// InitAIMetrics initializes the global AI metrics registry.
func InitAIMetrics() *AIMetrics {
	metricsOnce.Do(func() {
		aiMetrics = &AIMetrics{
			TokenUsage: prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: "higress",
				Subsystem: "ai",
				Name:      "token_usage_total",
				Help:      "Total number of LLM tokens processed by Higress.",
			}),
			ModelLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
				Namespace: "higress",
				Subsystem: "ai",
				Name:      "model_latency_seconds",
				Help:      "Latency of LLM requests handled by Higress.",
				Buckets:   prometheus.DefBuckets,
			}),
		}
		prometheus.MustRegister(aiMetrics.TokenUsage, aiMetrics.ModelLatency)
	})
	return aiMetrics
}

func Metrics() *AIMetrics { return InitAIMetrics() }

func (m *AIMetrics) RecordTokenUsage(tokens int) {
	if m == nil {
		return
	}
	m.TokenUsage.Add(float64(tokens))
}

func (m *AIMetrics) ObserveLatencySeconds(seconds float64) {
	if m == nil {
		return
	}
	m.ModelLatency.Observe(seconds)
}