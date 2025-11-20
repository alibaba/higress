# Tool Search MCP Server 设计文档

## 插件目的与使用场景

Tool Search插件是一个基于Milvus向量数据库的MCP（Model Context Protocol）服务器，用于提供语义化工具搜索功能。它解决了传统基于关键字匹配的工具搜索局限性，允许用户通过自然语言查询来搜索最相关的工具。

主要使用场景包括：
- 在大量工具中快速找到与用户需求最匹配的工具
- 提供更智能的工具推荐功能
- 支持基于语义理解的工具发现

## 核心功能设计

### 语义化工具搜索
- 利用向量嵌入技术将工具描述转换为向量表示
- 使用Milvus向量数据库进行高效的相似度搜索
- 支持按相关性排序返回最匹配的工具

### 工具数据管理
- 从Milvus数据库中检索工具信息
- 支持获取所有可用工具的列表
- 工具信息包括名称、描述和元数据

### MCP协议兼容
- 实现标准的MCP协议接口
- 提供tools/list和tools/call接口
- 支持与其他MCP组件无缝集成

## 配置参数说明

### 向量数据库配置 (vector)
| 参数 | 类型 | 必填 | 默认值 | 描述 |
|------|------|------|--------|------|
| type | string | 是 | - | 向量数据库类型，目前仅支持"milvus" |
| host | string | 是 | - | Milvus服务器主机地址 |
| port | integer | 是 | - | Milvus服务器端口 |
| database | string | 否 | "default" | Milvus数据库名称 |
| tableName | string | 否 | "apig_mcp_tools" | Milvus集合名称 |
| username | string | 否 | - | Milvus用户名 |
| password | string | 否 | - | Milvus密码 |
| maxTools | integer | 否 | 1000 | 获取工具列表时的最大工具数量限制 |

### 嵌入模型配置 (embedding)
| 参数 | 类型 | 必填 | 默认值 | 描述 |
|------|------|------|--------|------|
| apiKey | string | 是 | - | 嵌入模型API密钥 |
| baseURL | string | 否 | "https://dashscope.aliyuncs.com/compatible-mode/v1" | 嵌入模型API基础URL |
| model | string | 否 | "text-embedding-v4" | 嵌入模型名称 |
| dimensions | integer | 否 | 1024 | 嵌入向量维度 |

## 技术选型与依赖

### 核心技术
- **Go语言**: 作为主要开发语言，符合Higress项目技术栈
- **Milvus向量数据库**: 用于存储和检索工具向量信息
- **OpenAI兼容API**: 用于生成文本向量嵌入
- **gRPC**: Milvus数据库通信协议

### 主要依赖库
- `github.com/milvus-io/milvus-sdk-go/v2`: Milvus Go SDK
- `github.com/openai/openai-go/v2`: OpenAI兼容客户端
- `github.com/mark3labs/mcp-go`: MCP协议Go实现
- `github.com/alibaba/higress/plugins/golang-filter`: Higress Go插件框架

## 边界条件与限制

### 适用范围
- 适用于需要语义化工具搜索的场景
- 支持大规模工具集合的高效检索
- 可与其他MCP服务集成使用

### 局限性
- 依赖于向量数据库的性能和可用性
- 嵌入模型的质量直接影响搜索准确性
- 需要预先为工具生成向量表示
- 当前仅支持Milvus作为向量数据库

### 性能考虑
- 向量搜索性能受Milvus配置和硬件资源影响
- 嵌入模型调用可能成为性能瓶颈
- 需要合理设置topK参数以平衡准确性和性能
- 通过maxTools参数限制获取工具列表的数量，避免大量数据传输

## 测试策略

### 集成测试
- 验证与Milvus数据库的连接和操作
- 测试完整的MCP协议交互流程
- 验证嵌入模型API调用
- 配置好后运行server_test.go

### 环境要求
- 需要运行Milvus数据库实例
- 需要有效的嵌入模型API密钥（可以从阿里云控制台中获取https://bailian.console.aliyun.com/?tab=model#/api-key，确保要开启embedding模型）
- 测试数据集（工具的元数据信息）需要预先导入到Milvus中


## 设计思路
1. 主要参考基于阿里云adb的工具路由的实现
2. 参考rag mcp server中连接milvus的处理，复用其中的VectorDB
3. 遵循go-filter中mcp-server实现的规范