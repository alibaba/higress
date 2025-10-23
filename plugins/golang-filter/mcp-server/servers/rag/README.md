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


## Python 代码样例

这里提供一个 基于 langchain 代码样例，用于生成测试数据集。

```python
#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Milvus向量数据库文档处理系统
功能：
1. 使用langchain `UnstructuredFileLoader` 加载文本文件成 Document
2. 使用 langchain `RecursiveTextSplitter` 对 Document 进行 chunk 分割
3. 对每个 chunk 调用 OpenAI 兼容的 embedding
4. 将每个 chunk 写入 milvus vectordb
"""

import os
import json
import logging
import uuid
import time
from typing import List, Dict, Any, Optional
from pathlib import Path

# LangChain imports
from langchain.document_loaders import UnstructuredFileLoader
from langchain.text_splitter import RecursiveCharacterTextSplitter
from langchain.schema import Document

# OpenAI client import
from openai import OpenAI

# Milvus imports
from pymilvus import (
    connections,
    Collection,
    CollectionSchema,
    FieldSchema,
    DataType,
    utility
)

# 配置日志
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class MilvusDocumentProcessor:
    """Milvus文档处理器"""
    
    def __init__(
        self,
        milvus_host: str = "localhost",
        milvus_port: str = "19530",
        user_name: str = "",
        password: str = "",
        db_name: str = "default",
        collection_name: str = "test_rag",
        embedding_model: str = "text-embedding-ada-002",
        openai_api_key: Optional[str] = None,
        openai_api_base: Optional[str] = None,
        chunk_size: int = 1000,
        chunk_overlap: int = 100,
        embedding_dim: int = 1536
    ):
        """
        初始化Milvus文档处理器
        
        Args:
            milvus_host: Milvus服务器地址
            milvus_port: Milvus服务器端口
            collection_name: 集合名称
            embedding_model: 嵌入模型名称
            openai_api_key: OpenAI API密钥
            openai_api_base: OpenAI API基础URL（用于兼容其他服务）
            chunk_size: 文本分割大小
            chunk_overlap: 文本分割重叠大小
        """
        self.milvus_host = milvus_host
        self.milvus_port = milvus_port
        self.user_name = user_name
        self.password = password
        self.db_name = db_name
        self.collection_name = collection_name
        self.chunk_size = chunk_size
        self.chunk_overlap = chunk_overlap
        self.embedding_dim = embedding_dim  
      
        
        # 初始化文本分割器
        self.text_splitter = RecursiveCharacterTextSplitter(
            chunk_size=chunk_size,
            chunk_overlap=chunk_overlap,
            length_function=len,
            separators=["\n\n", "\n", " ", ""]
        )
        
        # 初始化嵌入模型
        self.openai_client = OpenAI(
            api_key=openai_api_key or os.getenv("DASHSCOPE_API_KEY"),
            base_url=openai_api_base or "https://dashscope.aliyuncs.com/compatible-mode/v1"
        )
        
        self.embedding_model = embedding_model
        
        # Milvus连接和集合
        self.collection = None
        
    def connect_milvus(self):
        """连接到Milvus数据库"""
        try:
            connections.connect(
                alias="default",
                host=self.milvus_host,
                port=self.milvus_port,
                user=self.user_name,
                password=self.password,
                db_name=self.db_name
            )
            logger.info(f"成功连接到Milvus: {self.milvus_host}:{self.milvus_port} 数据库: {self.db_name}")
            return True
        except Exception as e:
            logger.error(f"连接Milvus失败: {e}")
            return False
    
    def create_collection(self):
        """
        创建Milvus集合
        
        Args:
            dimension: 向量维度，默认1536（OpenAI text-embedding-ada-002的维度）
        """
        try:
            # 检查集合是否已存在
            if utility.has_collection(self.collection_name):
                logger.info(f"集合 {self.collection_name} 已存在")
                self.collection = Collection(self.collection_name)
                return True
            
            # 定义字段
            fields = [
                FieldSchema(name="id", dtype=DataType.VARCHAR, is_primary=True, auto_id=False, max_length=256),
                FieldSchema(name="content", dtype=DataType.VARCHAR, max_length=8192),
                FieldSchema(name="vector", dtype=DataType.FLOAT_VECTOR, dim=self.embedding_dim),
                FieldSchema(name="metadata", dtype=DataType.JSON),
                FieldSchema(name="created_at", dtype=DataType.INT64)
            ]
            
            # 创建集合schema
            schema = CollectionSchema(
                fields=fields,
                description="文档chunk向量存储"
            )
            
            # 创建集合
            self.collection = Collection(
                name=self.collection_name,
                schema=schema
            )
            
            # 创建索引
            index_params = {
                "metric_type": "IP",  # 内积
                "index_type": "HNSW",
                "params": {"M": 8, "efConstruction": 64}
            }
            
            self.collection.create_index(
                field_name="vector",
                index_params=index_params
            )
            
            logger.info(f"成功创建集合: {self.collection_name}")
            return True
            
        except Exception as e:
            logger.error(f"创建集合失败: {e}")
            return False
    
    def load_document(self, file_path: str) -> List[Document]:
        """
        使用UnstructuredFileLoader加载文档
        Args:
            file_path: 文件路径
            
        Returns:
            Document列表
        """
        try:
            logger.info(f"加载文档: {file_path}")
            # 检查文件是否存在
            if not os.path.exists(file_path):
                raise FileNotFoundError(f"文件不存在: {file_path}")
            
            # 使用UnstructuredFileLoader加载文档
            loader = UnstructuredFileLoader(file_path)
            documents = loader.load()
            
            logger.info(f"成功加载 {len(documents)} 个文档")
            return documents
            
        except Exception as e:
            logger.error(f"加载文档失败: {e}")
            return []
    
    def split_documents(self, documents: List[Document]) -> List[Document]:
        """
        使用RecursiveTextSplitter分割文档
        
        Args:
            documents: 文档列表
            
        Returns:
            分割后的文档chunk列表
        """
        try:
            logger.info(f"开始分割 {len(documents)} 个文档")
            
            chunks = self.text_splitter.split_documents(documents)
            
            logger.info(f"文档分割完成，共生成 {len(chunks)} 个chunk")
            return chunks
            
        except Exception as e:
            logger.error(f"文档分割失败: {e}")
            return []
    
    def generate_embeddings(self, texts: List[str]) -> List[List[float]]:
        """
        生成文本嵌入向量
        
        Args:
            texts: 文本列表
            
        Returns:
            嵌入向量列表
        """
        try:
            logger.info(f"开始生成 {len(texts)} 个文本的嵌入向量")
            
            # 使用 OpenAI 客户端生成嵌入向量
            response = self.openai_client.embeddings.create(
                model=self.embedding_model,
                input=texts,
                dimensions=self.embedding_dim,
                encoding_format="float"
            )
            
            # 提取嵌入向量
            embeddings = [data.embedding for data in response.data]
            
            logger.info(f"嵌入向量生成完成，向量维度: {len(embeddings[0]) if embeddings else 0}")
            return embeddings
            
        except Exception as e:
            logger.error(f"生成嵌入向量失败: {e}")
            return []
    
    def insert_chunks_to_milvus(self, chunks: List[Document]) -> bool:
        """
        将文档chunk插入到Milvus
        
        Args:
            chunks: 文档chunk列表
            
        Returns:
            是否成功
        """
        try:
            if not chunks:
                logger.warning("没有chunk需要插入")
                return True
            
            logger.info(f"开始插入 {len(chunks)} 个chunk到Milvus")
            
            # 准备数据
            ids = [str(uuid.uuid4()) for _ in range(len(chunks))]
            texts = [chunk.page_content for chunk in chunks]
            metadatas = [json.dumps(chunk.metadata, ensure_ascii=False) for chunk in chunks]
            created_ats = [int(time.time()) for _ in range(len(chunks))]
            
            # 生成嵌入向量
            embeddings = self.generate_embeddings(texts)
            
            if not embeddings:
                logger.error("生成嵌入向量失败")
                return False
            
            # 准备插入数据
            data = [
                ids,
                texts,
                embeddings,
                metadatas,
                created_ats
            ]
            
            # 插入数据
            mr = self.collection.insert(data)
            
            # 刷新集合以确保数据持久化
            self.collection.flush()
            
            logger.info(f"成功插入 {len(chunks)} 个chunk，插入ID范围: {mr.primary_keys[0]} - {mr.primary_keys[-1]}")
            return True
            
        except Exception as e:
            logger.error(f"插入chunk到Milvus失败: {e}")
            return False
    
    def process_file(self, file_path: str) -> bool:
        """
        处理单个文件的完整流程
        
        Args:
            file_path: 文件路径
            
        Returns:
            是否成功
        """
        try:
            logger.info(f"开始处理文件: {file_path}")
            
            # 1. 加载文档
            documents = self.load_document(file_path)
            if not documents:
                return False
            
            # 2. 分割文档
            chunks = self.split_documents(documents)
            if not chunks:
                return False
            
            # 3. 插入到Milvus
            success = self.insert_chunks_to_milvus(chunks)
            
            if success:
                logger.info(f"文件处理完成: {file_path}")
            else:
                logger.error(f"文件处理失败: {file_path}")
            
            return success
            
        except Exception as e:
            logger.error(f"处理文件失败: {e}")
            return False
    
    def process_directory(self, directory_path: str, file_extensions: List[str] = None) -> Dict[str, bool]:
        """
        处理目录中的所有文件
        
        Args:
            directory_path: 目录路径
            file_extensions: 支持的文件扩展名列表，默认为常见文本文件
            
        Returns:
            文件处理结果字典
        """
        if file_extensions is None:
            file_extensions = ['.txt', '.md']
        
        results = {}
        
        try:
            directory = Path(directory_path)
            if not directory.exists():
                logger.error(f"目录不存在: {directory_path}")
                return results
            
            # 遍历目录中的文件
            for file_path in directory.rglob('*'):
                if file_path.is_file() and file_path.suffix.lower() in file_extensions:
                    logger.info(f"处理文件: {file_path}")
                    results[str(file_path)] = self.process_file(str(file_path))
            
            # 统计结果
            success_count = sum(1 for success in results.values() if success)
            total_count = len(results)
            
            logger.info(f"目录处理完成: {success_count}/{total_count} 个文件成功处理")
            
        except Exception as e:
            logger.error(f"处理目录失败: {e}")
        
        return results
    
    def search_similar(self, query: str, top_k: int = 5) -> List[Dict[str, Any]]:
        """
        搜索相似文档
        
        Args:
            query: 查询文本
            top_k: 返回结果数量
            
        Returns:
            搜索结果列表
        """
        try:
            # 生成查询向量
            query_embeddings = self.generate_embeddings([query])
            if not query_embeddings:
                logger.error("生成查询向量失败")
                return []
            query_embedding = query_embeddings[0]
            
            # 加载集合
            self.collection.load()
            # 搜索参数
            search_params = {
                "metric_type": "IP",
                "params": {"efSearch": 64}
            }
            
            # 执行搜索
            results = self.collection.search(
                data=[query_embedding],
                anns_field="vector",
                param=search_params,
                limit=top_k,
                output_fields=["id", "content", "metadata"]
            )
            
            # 格式化结果
            formatted_results = []
            for hit in results[0]:
                formatted_results.append({
                    "id": hit.id,
                    "score": hit.score,
                    "content": hit.entity.get("content"),
                    "metadata": json.loads(hit.entity.get("metadata", "{}"))
                })
            
            return formatted_results
            
        except Exception as e:
            logger.error(f"搜索失败: {e}")
            return []
    
    def get_collection_stats(self) -> Dict[str, Any]:
        """
        获取集合统计信息
        
        Returns:
            集合统计信息字典
        """
        try:
            if not self.collection:
                logger.warning("集合未初始化")
                return {}
            
            # 加载集合
            self.collection.load()
            
            # 获取集合信息
            stats = {
                "collection_name": self.collection_name,
                "num_entities": self.collection.num_entities,
                "description": self.collection.description,
                "schema": {
                    "fields": [
                        {
                            "name": field.name,
                            "type": str(field.dtype),
                            "is_primary": field.is_primary,
                            "auto_id": field.auto_id
                        }
                        for field in self.collection.schema.fields
                    ]
                }
            }
            
            # 获取索引信息
            try:
                indexes = self.collection.indexes
                stats["indexes"] = [
                    {
                        "field_name": index.field_name,
                        "index_name": index.index_name,
                        "params": index.params
                    }
                    for index in indexes
                ]
            except Exception as e:
                logger.warning(f"获取索引信息失败: {e}")
                stats["indexes"] = []
            
            logger.info(f"成功获取集合 {self.collection_name} 的统计信息")
            return stats
            
        except Exception as e:
            logger.error(f"获取集合统计信息失败: {e}")
            return {}


def main():
    """主函数 - 示例用法"""
    # 配置参数
    config = {
        "milvus_host": "localhost",
        "milvus_port": "19530",
        "user_name": "",
        "password": "",
        "db_name": "default",
        "collection_name": "test_rag",
        "embedding_model": "text-embedding-v4",
        "openai_api_key": "sk-xxxxxx",
        "openai_api_base": "https://dashscope.aliyuncs.com/compatible-mode/v1",  # 可选，用于兼容其他服务
        "chunk_size": 500,
        "chunk_overlap": 50,
        "embedding_dim": 1024,
    }
    # 创建处理器
    processor = MilvusDocumentProcessor(**config)
    # 连接Milvus
    if not processor.connect_milvus():
        logger.error("无法连接到Milvus，退出程序")
        return
    
    # 创建集合
    if not processor.create_collection():
        logger.error("无法创建集合，退出程序")
        return
    
    # 示例：处理单个文件
    file_path = "path/to/your/documents/a.txt"
    processor.process_file(file_path)
    
    # 示例：处理目录
    # directory_path = "path/to/your/documents"
    # results = processor.process_directory(directory_path)
    
    # 示例：搜索
    query = "人工智能"
    search_results = processor.search_similar(query, top_k=5)
    for result in search_results:
        print(f"ID: {result['id']}")
        print(f"Score: {result['score']:.4f}")
        print(f"Content: {result['content'][:100]}...")
        print(f"MetaData: {result['metadata']}")
        print("-" * 50)
    
    # 获取统计信息
    stats = processor.get_collection_stats()
    print("集合统计信息:")
    for key, value in stats.items():
        print(f"  {key}: {value}")


if __name__ == "__main__":
    main()

```

python 参考 requirements.txt 如下：

```
# Milvus向量数据库文档处理系统依赖包
# 基于milvus.py文件生成
# LangChain相关依赖
langchain>=0.3.27
langchain-community>=0.3.31
unstructured[all-docs]
openai>=1.14.3

# Milvus向量数据库
pymilvus>=2.6.2

# 基础依赖
numpy>=1.24.0
pandas>=2.0.0

# 文档处理相关
python-magic>=0.4.27
python-magic-bin>=0.4.14  # Windows用户需要
filetype>=1.2.0

# 网络请求
requests>=2.31.0
urllib3>=2.0.0

# 日志和配置
pyyaml>=6.0
python-dotenv>=1.0.0

# 可选：如果需要处理特定文件格式
# docx2txt>=0.8  # Word文档
# pdfplumber>=0.9.0  # PDF文档
# python-pptx>=0.6.21  # PowerPoint文档
# openpyxl>=3.1.0  # Excel文档
# markdown>=3.5.0  # Markdown文档

```

