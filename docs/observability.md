# Observability for AI Workloads

Higress can export AI-specific metrics and traces.

## Metrics (Prometheus)
- Counter: `higress_ai_token_usage_total`
- Histogram: `higress_ai_model_latency_milliseconds`

Enable Prometheus scraping with Helm:

```bash
helm upgrade --install higress ./helm/core \
  --set observability.prometheus.enabled=true
```

## Tracing (OpenTelemetry)
Enable the built-in OpenTelemetry Collector via Helm:

```bash
helm upgrade --install higress ./helm/core \
  --set observability.otelCollector.enabled=true
```

Programmatically, initialize tracing via `pkg/observability.SetupTracing`.