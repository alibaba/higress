# Volcano Integration for Batch AI Workloads

Enable the Volcano sample job with Helm:

```bash
helm upgrade --install higress ./helm/core \
  --set volcano.enabled=true
```

To programmatically submit jobs from code, build with `-tags volcano` and use `pkg/volcano.NewVolcanoScheduler`.