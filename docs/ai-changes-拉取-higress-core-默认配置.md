# AI 变更记录：拉取 Higress Core 默认配置

## 变更内容

- 基于本地 `helm/core/values.yaml`（Higress Core Chart 2.2.1）创建 `helm/core/default-2.2.1.yaml` 默认配置副本。
- 两个文件的 SHA-256 校验值一致，未修改任何配置项。

## 验证

- 执行 `helm lint helm/core --values helm/core/default-2.2.1.yaml`，通过。

## 影响

- 仅新增本地配置副本，不会应用至 Kubernetes 集群。
