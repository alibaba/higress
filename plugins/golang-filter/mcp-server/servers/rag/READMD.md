# Higress RAG MCP Server

这是一个 Model Context Protocol (MCP) 服务器，提供知识管理和检索功能。

该 MCP 服务器提供以下工具：

## MCP Tools

### 知识管理 
- `create-chunks-from-text` - 从 Text 创建知识 (p1)

### 块管理
- `list-chunks` - 列出知识块 
- `delete-chunk` - 删除知识块 

### 搜索 
- `search` - 搜索

### 聊天功能
- `chat` - 发送聊天消息

## 配置说明

### 配置结构

```yaml
rag:
  # RAG系统基础配置
  splitter:
    type: "recursive"  # 递归分块器 recursive 和 nosplitter
    chunk_size: 500
    chunk_overlap: 50
  top_k: 5  # 搜索返回的知识块数量
  threshold: 0.5  # 搜索阈值

llm:
  provider: "openai"  # openai
  api_key: "your-llm-api-key"
  base_url: "https://api.openai.com/v1"  # 可选
  model: "gpt-3.5-turbo"  # LLM模型
  max_tokens: 2048  # 最大令牌数
  temperature: 0.5  # 温度参数

embedding:
  provider: "openai"  # openai, dashscope
  api_key: "your-embedding-api-key"
  base_url: "https://api.openai.com/v1"  # 可选
  model: "text-embedding-ada-002"  # 嵌入模型

vectordb:
  provider: "milvus"  # milvus
  host: "localhost"
  port: 19530
  database: "default"
  collection: "test_collection"
  username: ""  # 可选
  password: ""  # 可选

```
### higress-config 配置样例

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: higress-config
  namespace: higress-system
data:
  higress: |
    mcpServer:
      enable: true
      sse_path_suffix: "/sse"
      redis:
        address: "<Redis IP>:6379"
        username: ""
        password: ""
        db: 0
      match_list:
      - path_rewrite_prefix: ""
        upstream_type: ""
        enable_path_rewrite: false
        match_rule_domain: ""
        match_rule_path: "/mcp-servers/rag"
        match_rule_type: "prefix"
      servers:
      - path: "/mcp-servers/rag"
        name: "rag"
        type: "rag"
        config:
          rag:
            splitter:
              provider: recursive
              chunk_size: 500
              chunk_overlap: 50
            top_k: 10
            threshold: 0.5
          llm:
            provider: openai
            api_key: sk-XXX
            base_url: https://openrouter.ai/api/v1
            model: openai/gpt-4o
            temperature: 0.5
            max_tokens: 2048
          embedding:
            provider: dashscope
            api_key: sk-xxx
            model: text-embedding-v4
          vectordb:
            provider: milvus
            host: 192.168.31.72
            port: 19530
            database: default
            collection: test_collection
```

### 支持的提供商
#### Embedding
- **OpenAI**
- **DashScope**

#### Vector Database
- **Milvus**

#### LLM 
- **OpenAI**


## Milvus 安装

### docker 配置
配置 Docker Desktop 镜像加速器
编辑 daemon.json 配置，加上镜像加速器，例如：
```
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://mirror.ccs.tencentyun.com",
    "https://hub-mirror.c.163.com"
  ],
  "dns": ["8.8.8.8", "1.1.1.1"]
}
```


### milvus install on docker
```
v2.6.0
Download the configuration file
wget https://github.com/milvus-io/milvus/releases/download/v2.6.0/milvus-standalone-docker-compose.yml -O docker-compose.yml

v2.4
$ wget https://github.com/milvus-io/milvus/releases/download/v2.4.23/milvus-standalone-docker-compose.yml -O docker-compose.yml

# Start Milvus
$ sudo docker compose up -d

Creating milvus-etcd  ... done
Creating milvus-minio ... done
Creating milvus-standalone ... done
```

### install attu

```
docker run -p 8000:3000 -e MILVUS_URL=http://<本机IP>:19530  zilliz/attu:v2.6
Open your browser and navigate to http://localhost:8000
```




