# 功能说明

`chatgpt-azure-director` 将 OpenAI v1 协议的接口转换为 Azure OpenAI 协议。
为 OpenAI 开发的 ChatGPT 客户端应用不必修改客户端代码即可将 ChatGPT 服务商或实例从 OpenAI 切换到 Azure.

此外, 还支持了虚拟 api-key, 开发者可以自行构造虚拟的 api-key, 分发给其他用户, 可以基于这个虚拟 api-key 对用户进行审计、计费等。
请求经过 `chatgpt-azure-director` 时，插件将虚拟 api-key 替换成真实的 Azure api-key。

## 目前支持的转换的接口

|                         |                                                           |
|-------------------------|-----------------------------------------------------------|
| `/v1/models`            | "/openai/models"                                          |
| `^/v1/models/{model}`   | `/openai/models/{model}`                                  |
| `^/v1/completions`      | `/openai/developments/{DevelopmentName}/completions`      |
| `^/v1/chat/completions` | `/openai/developments/{DevelopmentName}/chat/completions` |

# 配置字段

| 名称              | 数据类型              | 填写要求        | 默认值                               | 描述                        |
|-----------------|-------------------|-------------|-----------------------------------|---------------------------|
| allowedKeys     | []string 字符串类型的数组 | 必填, 长度至少为 1 | -                                 | 虚拟 api-key 列表, 管理员构造和二次分发 |
| apiKey          | string            | 必填          | -                                 | Azure OpenAI 服务 api-key   |
| apiVersion      | string            | 选填          | 2023-03-15-preview                | Azure OpenAI API 版本       |
| developmentName | string            | 必填          | -                                 | Azure 上部署的实例名称            |
| scheme          | string            | 选填          | https                             | Azure 上部署的实例的 http scheme |
| azureHost       | string            | 选填          | {deploymentName}.openai.azure.com | Azure 上部署的实例的域名           |

# 配置示例

见本目下 [envoy.yaml](envoy.yaml) 文件。
