# Higress RAG MCP Server

这是一个 Model Context Protocol (MCP) 服务器，提供知识管理和检索功能。

## MCP 工具说明

Higress RAG MCP Server 提供以下工具，根据配置不同，可用工具也会有所差异：

| 工具名称 | 功能描述 | 依赖配置 | 必选/可选 |
|---------|---------|---------|----------|
| `create-chunks-from-text` | 将文本内容分块并存储到向量数据库，用于知识库构建 | embedding, vectordb | **必选** |
| `list-chunks` | 列出已存储的知识块，用于知识库管理 | vectordb | **必选** |
| `delete-chunk` | 删除指定的知识块，用于知识库维护 | vectordb | **必选** |
| `search` | 基于语义相似度搜索知识库中的内容 | embedding, vectordb | **必选** |
| `chat` | 基于检索增强生成(RAG)回答用户问题，结合知识库内容生成回答 | embedding, vectordb, llm | **可选** |

### 工具与配置的关系

- **基础功能**（知识管理、搜索）：只需配置 `embedding` 和 `vectordb`
- **高级功能**（聊天问答）：需额外配置 `llm`

具体关系如下：
- 未配置 `llm` 时，`chat` 工具将不可用
- 所有工具都依赖 `embedding` 和 `vectordb` 配置
- `rag` 配置用于调整分块和检索参数，影响所有工具的行为

## 典型使用场景

### 最小工具集场景（无LLM配置）

适用于仅需要知识库管理和检索的场景，不需要生成式回答。

**可用工具**：`create-chunks-from-text`、`list-chunks`、`delete-chunk`、`search`

**典型用例**：
1. 构建企业文档库，仅需检索相关文档片段
2. 数据索引系统，通过语义搜索快速定位信息
3. 内容管理系统，管理和检索结构化/非结构化内容

**示例流程**：
```
1. 使用 create-chunks-from-text 导入文档
2. 使用 search 检索相关内容
3. 使用 list-chunks 和 delete-chunk 管理知识库
```

### 完整工具集场景（含LLM配置）

适用于需要智能问答和内容生成的高级场景。

**可用工具**：`create-chunks-from-text`、`list-chunks`、`delete-chunk`、`search`、`chat`

**典型用例**：
1. 智能客服系统，基于企业知识库回答用户问题
2. 文档助手，帮助用户理解和分析复杂文档
3. 专业领域问答系统，如法律、金融、技术支持等

**示例流程**：
```
1. 使用 create-chunks-from-text 导入专业领域文档
2. 用户通过 chat 工具提问
3. 系统使用 search 检索相关知识
4. LLM 结合检索结果生成回答
5. 管理员使用 list-chunks 和 delete-chunk 维护知识库
```

## 配置说明

### 配置结构

| 名称                         | 数据类型 | 填写要求 | 默认值 | 描述 |
|----------------------------|----------|-----------|---------|--------|
| **rag**                    | object | 必填 | - | RAG系统基础配置 |
| rag.splitter.provider      | string | 必填 | recursive | 分块器类型：recursive或nosplitter |
| rag.splitter.chunk_size    | integer | 可选 | 500 | 块大小 |
| rag.splitter.chunk_overlap | integer | 可选 | 50 | 块重叠大小 |
| rag.top_k                  | integer | 可选 | 10 | 搜索返回的知识块数量 |
| rag.threshold              | float | 可选 | 0.5 | 搜索阈值 |
| **llm**                    | object | 可选 | - | LLM配置（不配置则无chat功能） |
| llm.provider               | string | 可选 | openai | LLM提供商 |
| llm.api_key                | string | 可选 | - | LLM API密钥 |
| llm.base_url               | string | 可选 |  | LLM API基础URL |
| llm.model                  | string | 可选 | gpt-4o | LLM模型名称 |
| llm.max_tokens             | integer | 可选 | 2048 | 最大令牌数 |
| llm.temperature            | float | 可选 | 0.5 | 温度参数 |
| **embedding**              | object | 必填 | - | 嵌入配置（所有工具必需） |
| embedding.provider         | string | 必填 | openai | 嵌入提供商：支持openai协议的任意供应商 |
| embedding.api_key          | string | 必填 | - | 嵌入API密钥 |
| embedding.base_url         | string | 可选 |  | 嵌入API基础URL |
| embedding.model            | string | 必填 | text-embedding-ada-002 | 嵌入模型名称 |
| embedding.dimensions       | integer | 可选 | 1536 | 嵌入维度 |
| **vectordb**               | object | 必填 | - | 向量数据库配置（所有工具必需） |
| vectordb.provider          | string | 必填 | milvus | 向量数据库提供商 |
| vectordb.host              | string | 必填 | localhost | 数据库主机地址 |
| vectordb.port              | integer | 必填 | 19530 | 数据库端口 |
| vectordb.database          | string | 必填 | default | 数据库名称 |
| vectordb.collection        | string | 必填 | test_collection | 集合名称 |
| vectordb.username          | string | 可选 | - | 数据库用户名 |
| vectordb.password          | string | 可选 | - | 数据库密码 |
| **vectordb.mapping**       | object | 可选 | - | 字段映射配置 |
| vectordb.mapping.fields    | array | 可选 | - | 字段映射列表 |
| vectordb.mapping.fields[].standard_name | string | 必填 | - | 标准字段名称（如 id, content, vector 等） |
| vectordb.mapping.fields[].raw_name | string | 必填 | - | 原始字段名称（数据库中的实际字段名） |
| vectordb.mapping.fields[].properties | object | 可选 | - | 字段属性（如 auto_id, max_length 等） |
| vectordb.mapping.index     | object | 可选 | - | 索引配置 |
| vectordb.mapping.index.index_type | string | 必填 | - | 索引类型（如 FLAT, IVF_FLAT, HNSW 等） |
| vectordb.mapping.index.params | object | 可选 | - | 索引参数（根据索引类型不同而异） |
| vectordb.mapping.search    | object | 可选 | - | 搜索配置 |
| vectordb.mapping.search.metric_type | string | 可选 | L2 | 度量类型（如 L2, IP, COSINE 等） |
| vectordb.mapping.search.params | object | 可选 | - | 搜索参数（如 nprobe, ef_search 等）


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
            provider: openai
            base_url: https://dashscope.aliyuncs.com/compatible-mode/v1
            api_key: sk-xxx
            model: text-embedding-v4
            dimensions: 1536
          vectordb:
            provider: milvus
            host: localhost
            port: 19530
            database: default
            collection: test_rag
            mapping:
              fields:
              - standard_name: id
                raw_name: id
                properties:
                  auto_id: false
                  max_length: 256
              - standard_name: content
                raw_name: content
                properties:
                  max_length: 8192
              - standard_name: vector
                raw_name: vector
              - standard_name: metadata
                raw_name: metadata
              - standard_name: created_at
                raw_name: created_at
              index:
                index_type: HNSW
                params:
                  M: 4
                  efConstruction: 32
              search:
                metric_type: IP
                params:
                  ef: 32

```
### 支持的提供商
#### Embedding
- **OpenAI 兼容**

#### Vector Database
- **Milvus**

#### LLM 
- **OpenAI 兼容**

## 如何测试数据集的效果

测试数据集的效果分两步，第一步导入数据集语料，第二步测试Chat效果。

### 导入数据集语料

使用 `RAGClient.CreateChunkFromText` 工具导入数据集语料，比如数据集语料格式为 JSON，每个 JSON 对象包含 `body`、`title` 和 `url` 等字段。样例代码如下：

```golang
func TestRAGClient_LoadChunks(t *testing.T) {
	t.Logf("TestRAGClient_LoadChunks")
	ragClient, err := getRAGClient()
	if err != nil {
		t.Errorf("getRAGClient() error = %v", err)
		return
	}
	// load json output/corpus.json and then call ragclient CreateChunkFromText to insert chunks
	file, err := os.Open("/dataset/corpus.json")
	if err != nil {
		t.Errorf("LoadData() error = %v", err)
		return
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var data []struct {
		Body  string `json:"body"`
		Title string `json:"title"`
		Url   string `json:"url"`
	}
	if err := decoder.Decode(&data); err != nil {
		t.Errorf("LoadData() error = %v", err)
		return
	}

	for _, item := range data {
		t.Logf("LoadData() url = %s", item.Url)
		t.Logf("LoadData() title = %s", item.Title)
		t.Logf("LoadData() len body = %d", len(item.Body))
		chunks, err := ragClient.CreateChunkFromText(item.Body, item.Title)
		if err != nil {
			t.Errorf("LoadData() error = %v", err)
			continue
		} else {
			t.Logf("LoadData() chunks len = %d", len(chunks))
		}
	}
	t.Logf("TestRAGClient_LoadChunks done")
}
```

### 测试Chat效果

使用 `RAGClient.Chat` 工具测试 Chat 效果。样例代码如下：

```golang
func TestRAGClient_Chat(t *testing.T) {
	ragClient, err := getRAGClient()
	if err != nil {
		t.Errorf("getRAGClient() error = %v", err)
		return
	}
	query := "Which online betting platform provides a welcome bonus of up to $1000 in bonus bets for new customers' first losses, runs NBA betting promotions, and is anticipated to extend the same sign-up offer to new users in Vermont, as reported by both CBSSports.com and Sporting News?"
	resp, err := ragClient.Chat(query)
	if err != nil {
		t.Errorf("Chat() error = %v", err)
		return
	}
	if resp == "" {
		t.Errorf("Chat() resp = %s, want not empty", resp)
		return
	}
	t.Logf("Chat() resp = %s", resp)
}
```

## Milvus 安装

### Docker 配置
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

### 安装 milvus

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

### 安装 attu

Attu 是 Milvus 的可视化管理工具，用于查看和管理 Milvus 中的数据。

```
docker run -p 8000:3000 -e MILVUS_URL=http://<本机 IP>:19530  zilliz/attu:v2.6
Open your browser and navigate to http://localhost:8000
```




