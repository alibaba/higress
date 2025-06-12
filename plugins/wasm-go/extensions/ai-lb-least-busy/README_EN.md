## Introduction

wasm implementation for [gateway-api-inference-extension](https://github.com/kubernetes-sigs/gateway-api-inference-extension/blob/main/README.md)

```mermaid
sequenceDiagram
	participant C as Client
	participant H as Higress
	participant H1 as Host1
	participant H2 as Host2

	loop fetch metrics periodically
		H ->> H1: /metrics
		H1 ->> H: vllm metrics
		H ->> H2: /metrics
		H2 ->> H: vllm metrics
	end

	C ->> H: request
	H ->> H1: select pod according to vllm metrics, bypassing original service load balance policy
	H1 ->> H: response
	H ->> C: response
```

flowchart for pod selection:

![](https://github.com/kubernetes-sigs/gateway-api-inference-extension/blob/main/docs/scheduler-flowchart.png)

## Configuration

| Name                | Type         | Required          | default       | description                                 |
|--------------------|-----------------|------------------|-------------|-------------------------------------|
| `criticalModels`      | []string          | required              |             | critical model names    |

## Configuration Example

```yaml
criticalModels:
- meta-llama/Llama-2-7b-hf
- sql-lora
```