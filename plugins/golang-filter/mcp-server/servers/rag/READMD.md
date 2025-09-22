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

| 名称                         | 数据类型 | 填写要求 | 默认值 | 描述 |
|----------------------------|----------|-----------|---------|--------|
| **rag**                    | object | 必填 | - | RAG系统基础配置 |
| rag.splitter.provider      | string | 必填 | recursive | 分块器类型：recursive或nosplitter |
| rag.splitter.chunk_size    | integer | 可选 | 500 | 块大小 |
| rag.splitter.chunk_overlap | integer | 可选 | 50 | 块重叠大小 |
| rag.top_k                  | integer | 可选 | 10 | 搜索返回的知识块数量 |
| rag.threshold              | float | 可选 | 0.5 | 搜索阈值 |
| **llm**                    | object | 可选 | - | LLM配置 |
| llm.provider               | string | 可选 | openai | LLM提供商 |
| llm.api_key                | string | 可选 | - | LLM API密钥 |
| llm.base_url               | string | 可选 |  | LLM API基础URL |
| llm.model                  | string | 可选 | gpt-4o | LLM模型名称 |
| llm.max_tokens             | integer | 可选 | 2048 | 最大令牌数 |
| llm.temperature            | float | 可选 | 0.5 | 温度参数 |
| **embedding**              | object | 必填 | - | 嵌入配置 |
| embedding.provider         | string | 必填 | dashscope | 嵌入提供商：openai或dashscope |
| embedding.api_key          | string | 必填 | - | 嵌入API密钥 |
| embedding.base_url         | string | 可选 |  | 嵌入API基础URL |
| embedding.model            | string | 必填 | text-embedding-v4 | 嵌入模型名称 |
| **vectordb**               | object | 必填 | - | 向量数据库配置 |
| vectordb.provider          | string | 必填 | milvus | 向量数据库提供商 |
| vectordb.host              | string | 必填 | localhost | 数据库主机地址 |
| vectordb.port              | integer | 必填 | 19530 | 数据库端口 |
| vectordb.database          | string | 必填 | default | 数据库名称 |
| vectordb.collection        | string | 必填 | test_collection | 集合名称 |
| vectordb.username          | stri选ng | 可选 | - | 数据库用户名 |
| vectordb.password          | string | 可选 | - | 数据库密码 |


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
            host: <milvus IP>
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




