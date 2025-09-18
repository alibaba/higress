# Higress RAG MCP Server

这是一个 Model Context Protocol (MCP) 服务器，提供知识管理和检索功能。

该 MCP 服务器提供以下工具：

## MCP Tools

### 知识管理 (Default Knowledge Base)
- `create_knowledge_from_text` - 从 Text 创建知识 (p1)
- `create_knowledge_from_url` - 从 URL 创建知识
- `list_knowledge` - 列出知识 (p1)
- `get_knowledge` - 获取知识详情 (p1)
- `delete_knowledge` - 删除知识 (p1)

### 块管理
- `list_chunks` - 列出知识块 
- `delete_chunk` - 删除知识块 

### 会话管理
- `create_session` - 创建聊天会话 (p1)
- `get_session` - 获取会话详情 (p1)
- `list_sessions` - 列出会话 (p1)
- `delete_session` - 删除会话 (p1)

### 搜索 
- `search` - 搜索 (p1)

### 聊天功能
- `chat` - 发送聊天消息 (p1)


## 配置说明

### 配置结构

```yaml
rag:
  # RAG系统基础配置
  splitter:
    type: "recursive"  # 递归分块器
    chunk_size: 1000
    chunk_overlap: 200

embedding:
  provider: "openai"  # openai, dashscope
  api_key: "your-embedding-api-key"
  base_url: "https://api.openai.com/v1"  # 可选
  model: "text-embedding-ada-002"  # 嵌入模型

vectordb:
  provider: "milvus"  # milvus, qdrant, chroma
  host: "localhost"
  port: 19530
  database: "default"
  collection: "test_collection"
  username: ""  # 可选
  password: ""  # 可选


```

### 支持的提供商

#### Embedding 提供商
- **OpenAI**: text-embedding-ada-002, text-embedding-3-small, text-embedding-3-large
- **DashScope**: text-embedding-v1, text-embedding-v2

#### Vector Database 提供商
- **Milvus**: 开源向量数据库


## Test Dataset 
- MultiHop-RAG(https://github.com/yixuantt/MultiHop-RAG) (p1)
- DomainRAG (https://github.com/ShootingWong/DomainRAG)

# test

## docker on mac
配置 Docker Desktop 镜像加速器
打开 Docker Desktop → Settings（设置） → Docker Engine。
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
点击 Apply & Restart。

## milvus install on docker
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


Start Milvus
sudo docker compose up -d

Creating milvus-etcd  ... done
Creating milvus-minio ... done
Creating milvus-standalone ... done

docker run -p 8000:3000 -e MILVUS_URL=http://192.168.31.72:19530  zilliz/attu:v2.6

Open your browser and navigate to http://localhost:8000
```




