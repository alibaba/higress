# Autoscaling for AI Workloads (KEDA)

Enable KEDA-driven autoscaling from Prometheus metrics:

```bash
helm upgrade --install higress ./helm/core \
  --set keda.enabled=true \
  --set keda.metric.name=higress_ai_token_usage_total \
  --set keda.metric.threshold=100
```

At runtime, the ai-proxy plugin emits metrics which can be scraped by Prometheus and used by KEDA ScaledObject.

To manage ScaledObjects programmatically, build with `-tags keda` and use `pkg/autoscaler.NewKedaScaler`.