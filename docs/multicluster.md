# Multi-Cluster with Karmada

This guide shows how to enable multi-cluster configuration sync for Higress using Karmada.

## Prerequisites
- A running Karmada control plane and at least one member cluster. See `https://karmada.io/docs`.
- Higress installed via Helm.

## Enable Helm options

Set the following values:

```bash
helm upgrade --install higress ./helm/core \
  --set karmada.enabled=true
```

This installs a ClusterPropagationPolicy that propagates the Higress ConfigMap to all clusters.

## Programmatic usage

Build Higress with Karmada integration to use the `pkg/karmada` package:

```bash
CGO_ENABLED=0 go build -tags karmada ./...
```

Then use `karmada.NewKarmadaSync(client)` and call `SyncConfigMap` / `SyncCRD`.