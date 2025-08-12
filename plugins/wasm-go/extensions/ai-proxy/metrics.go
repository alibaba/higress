package main

import (
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/wasm-go/pkg/wrapper"
)

const (
	metricTokenUsage       = "higress_ai_token_usage_total"
	metricLatencyMilliP50  = "higress_ai_model_latency_ms_p50"
	metricLatencyMilliP95  = "higress_ai_model_latency_ms_p95"
	metricLatencyMilliP999 = "higress_ai_model_latency_ms_p999"
	ctxStartTimeKey        = "ai_proxy_start_time"
)

var (
	counterTokenUsage proxywasm.MetricCounter
	latP50            proxywasm.MetricCounter
	latP95            proxywasm.MetricCounter
	latP999           proxywasm.MetricCounter
	metricsInit       = false
)

func ensureMetrics() {
	if metricsInit {
		return
	}
	counterTokenUsage = proxywasm.DefineCounterMetric(metricTokenUsage)
	latP50 = proxywasm.DefineCounterMetric(metricLatencyMilliP50)
	latP95 = proxywasm.DefineCounterMetric(metricLatencyMilliP95)
	latP999 = proxywasm.DefineCounterMetric(metricLatencyMilliP999)
	metricsInit = true
}

func metricsOnRequestStart(ctx wrapper.HttpContext) {
	ensureMetrics()
	ctx.SetContext(ctxStartTimeKey, time.Now().UnixMilli())
}

func metricsOnResponse(ctx wrapper.HttpContext) {
	ensureMetrics()
	if v := ctx.GetContext(ctxStartTimeKey); v != nil {
		if startMs, ok := v.(int64); ok {
			elapsed := time.Now().UnixMilli() - startMs
			// naive percentile-like buckets
			switch {
			case elapsed <= 100:
				latP50.Add(1)
			case elapsed <= 1000:
				latP95.Add(1)
			default:
				latP999.Add(1)
			}
		}
	}
}

func metricsAddTokens(tokens int) {
	ensureMetrics()
	if tokens > 0 {
		counterTokenUsage.Add(uint64(tokens))
	}
}