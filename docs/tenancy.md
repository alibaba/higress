# Multi-Tenancy

Higress can isolate resources by Kubernetes namespace. Use `pkg/tenancy.TenantManager` to validate and check namespace access.

RBAC: ensure each tenant has access only to their namespace resources.

Future work: integrate filters in the gateway reconcilers to enforce tenant scoping.