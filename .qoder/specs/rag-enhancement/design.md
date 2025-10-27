# Higress AI网关RAG增强方案 - 技术设计文档

## 概述

本设计方案针对Higress AI网关编程挑战赛RAG增强方向，基于现有的`plugins/golang-filter/mcp-server/servers/rag`模块，实现检索前优化、多路混合检索、检索后处理和纠正性检索(CRAG)等先进RAG技术，提升检索准确性和响应质量。

**设计目标：**
- 在现有MCP架构基础上实现RAG技术增强
- 保持代码架构一致性和向后兼容性
- 实现CRAG纠正性检索核心机制
- 提供可配置的多路检索策略
- 确保方案可在比赛期间完整实现

**置信度评估：高置信度**
- 需求明确，技术路径成熟
- 基于现有稳定代码架构扩展
- 核心算法有充分理论支撑
- 实现复杂度适中，风险可控

## 技术架构设计

### 整体架构

```
现有RAG架构
├── MCP Server Layer (保持)
├── RAG Client (扩展)
├── Providers (扩展)
│   ├── Embedding Provider (保持)
│   ├── LLM Provider (保持)
│   ├── VectorDB Provider (保持)
│   └── BM25 Provider (新增)
└── Configuration (扩展)

新增RAG增强模块
├── Query Enhancement (查询增强)
│   ├── Query Rewriter (查询重写)
│   ├── Query Decomposer (查询分解)
│   └── Intent Classifier (意图识别)
├── Hybrid Retrieval (混合检索)
│   ├── Semantic Search (语义搜索)
│   ├── BM25 Search (关键词搜索)
│   └── Fusion Ranker (融合排序)
├── Post Retrieval (检索后处理)
│   ├── Reranker (重排序器)
│   ├── Context Compressor (上下文压缩)
│   └── Relevance Filter (相关性过滤)
└── CRAG Module (纠正性检索)
    ├── Retrieval Evaluator (检索评估)
    ├── Knowledge Refiner (知识优化)
    └── Fallback Strategy (回退策略)
```

### 核心模块设计

#### 1. 查询增强模块 (Query Enhancement)

**功能职责：**
- 查询重写：处理模糊查询、同义词替换
- 查询分解：将复杂查询拆分为子查询
- 意图识别：识别查询类型和意图

**技术实现：**
- 基于规则和ML的混合方法
- 支持中英文查询处理
- 集成外部NLP服务

#### 2. 混合检索模块 (Hybrid Retrieval)

**功能职责：**
- 语义搜索：现有向量搜索能力
- 关键词搜索：新增BM25全文检索
- 融合排序：RRF(Reciprocal Rank Fusion)算法

**技术实现：**
- BM25使用内存实现或集成Elasticsearch
- 支持权重配置的融合策略
- 并发执行提升性能

#### 3. 检索后处理模块 (Post Retrieval)

**功能职责：**
- 重排序：基于交叉编码器的精确排序
- 上下文压缩：移除冗余信息
- 相关性过滤：基于阈值的结果筛选

**技术实现：**
- 轻量级重排序模型
- 智能文本摘要算法
- 可配置的过滤策略

#### 4. CRAG纠正性检索模块

**功能职责：**
- 检索评估：评估检索结果质量
- 知识优化：优化和补充检索内容
- 回退策略：处理检索失败场景

**技术实现：**
- 基于相似度分布的质量评估
- 自适应阈值调整机制
- 多层次回退策略

## 数据流设计

### 增强检索流程

```
用户查询 → 查询增强 → 混合检索 → CRAG评估 → 检索后处理 → LLM生成
    ↓           ↓          ↓         ↓         ↓         ↓
原始查询    增强查询    检索结果   质量评估   精排结果   最终答案
    ↓           ↓          ↓         ↓         ↓
意图识别    查询分解    并发搜索   纠错机制   上下文压缩
查询重写    子查询集    结果融合   补充检索   相关性过滤
```

### 核心处理逻辑

1. **查询预处理阶段**
   - 查询清洗和标准化
   - 意图识别和分类
   - 查询重写和扩展
   - 复杂查询分解

2. **混合检索阶段**
   - 并发执行语义搜索和BM25搜索
   - 应用RRF融合算法
   - 初步结果合并和去重

3. **CRAG评估阶段**
   - 计算检索置信度分数
   - 判断是否需要补充检索
   - 执行纠正性检索策略

4. **后处理阶段**
   - 重排序优化结果顺序
   - 上下文压缩减少冗余
   - 相关性过滤确保质量

## 组件设计

### RAG Client 扩展

```go
type EnhancedRAGClient struct {
    *RAGClient                    // 继承现有功能
    queryEnhancer   *QueryEnhancer
    hybridRetriever *HybridRetriever
    postProcessor   *PostProcessor
    cragModule      *CRAGModule
    config          *EnhancedConfig
}
```

### 查询增强器

```go
type QueryEnhancer struct {
    rewriter     QueryRewriter
    decomposer   QueryDecomposer
    classifier   IntentClassifier
}

type EnhancedQuery struct {
    Original     string
    Rewritten    []string
    Subqueries   []string
    Intent       QueryIntent
    Keywords     []string
}
```

### 混合检索器

```go
type HybridRetriever struct {
    vectorSearch VectorSearcher
    bm25Search   BM25Searcher
    fusionRanker FusionRanker
}

type HybridResult struct {
    VectorResults []SearchResult
    BM25Results   []SearchResult
    FusedResults  []SearchResult
    Confidence    float64
}
```

### CRAG模块

```go
type CRAGModule struct {
    evaluator    RetrievalEvaluator
    refiner      KnowledgeRefiner
    fallback     FallbackStrategy
}

type CRAGDecision struct {
    Action       CRAGAction // CORRECT, INCORRECT, AMBIGUOUS
    Confidence   float64
    NeedRefine   bool
    FallbackType FallbackType
}
```

## 算法实现细节

### 1. 查询重写算法

**同义词扩展：**
- 基于预构建的同义词词典
- 支持领域特定术语映射
- 上下文感知的同义词选择

**查询补全：**
- 基于历史查询模式
- 利用词嵌入相似性
- 自动纠错机制

### 2. RRF融合算法

```go
func RRFFusion(results [][]SearchResult, k float64) []SearchResult {
    // RRF Score = Σ(1/(k + rank_i))
    // 其中 k 是平滑参数，通常为60
    scoreMap := make(map[string]float64)
    for _, resultList := range results {
        for rank, result := range resultList {
            scoreMap[result.ID] += 1.0 / (k + float64(rank+1))
        }
    }
    // 按分数排序返回
}
```

### 3. CRAG评估机制

**质量评估指标：**
- 检索结果间的相似性分布
- 查询与结果的语义匹配度
- 结果的多样性指标

**决策逻辑：**
```go
func (c *CRAGModule) Evaluate(query string, results []SearchResult) CRAGDecision {
    confidence := c.calculateConfidence(query, results)
    diversity := c.calculateDiversity(results)
    
    if confidence > c.config.HighThreshold {
        return CRAGDecision{Action: CORRECT, Confidence: confidence}
    } else if confidence < c.config.LowThreshold {
        return CRAGDecision{Action: INCORRECT, NeedRefine: true}
    } else {
        return CRAGDecision{Action: AMBIGUOUS, NeedRefine: diversity < c.config.DiversityThreshold}
    }
}
```

### 4. 上下文压缩算法

**重要性评分：**
- TF-IDF权重计算
- 位置权重（开头结尾更重要）
- 与查询的语义相关性

**压缩策略：**
- 保留关键句子
- 移除重复信息
- 智能摘要生成

## API设计

### 新增MCP工具

1. **enhanced-search** - 增强检索工具
```json
{
  "name": "enhanced-search",
  "description": "Enhanced semantic search with query optimization and hybrid retrieval",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query": {"type": "string"},
      "strategy": {"type": "string", "enum": ["semantic", "hybrid", "bm25"]},
      "enable_crag": {"type": "boolean"},
      "top_k": {"type": "integer", "default": 10}
    }
  }
}
```

2. **enhanced-chat** - 增强对话工具
```json
{
  "name": "enhanced-chat",
  "description": "Enhanced RAG chat with query optimization and result refinement",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query": {"type": "string"},
      "context_length": {"type": "integer", "default": 4000},
      "enable_compression": {"type": "boolean", "default": true}
    }
  }
}
```

### 配置扩展

```yaml
rag_enhancement:
  query_enhancement:
    enable_rewrite: true
    enable_decompose: true
    rewrite_strategies: ["synonym", "expansion", "correction"]
    max_subqueries: 3
  
  hybrid_retrieval:
    enable_bm25: true
    fusion_method: "rrf"
    fusion_weights:
      vector: 0.7
      bm25: 0.3
    rrf_k: 60
  
  crag:
    enable: true
    confidence_threshold:
      high: 0.8
      low: 0.3
    diversity_threshold: 0.5
    max_refinement_rounds: 2
  
  post_processing:
    enable_rerank: true
    enable_compression: true
    compression_ratio: 0.7
    relevance_threshold: 0.4

bm25_config:
  provider: "memory"  # or "elasticsearch"
  index_name: "rag_bm25"
  k1: 1.2
  b: 0.75
```

## 性能优化策略

### 并发处理
- 语义搜索和BM25搜索并发执行
- 多子查询并发处理
- 异步重排序处理

### 缓存策略
- 查询重写结果缓存
- 检索结果缓存（基于查询hash）
- 嵌入向量缓存

### 资源管理
- 连接池管理
- 内存使用优化
- 智能批处理

## 错误处理策略

### 回退机制
1. **查询增强失败** → 使用原始查询
2. **BM25检索失败** → 仅使用语义搜索
3. **CRAG评估失败** → 使用默认阈值
4. **重排序失败** → 使用原始排序

### 容错设计
- 各模块独立容错
- 优雅降级处理
- 详细错误日志
- 监控和告警

## 测试策略

### 单元测试
- 各模块独立测试
- 算法正确性验证
- 边界情况测试

### 集成测试
- 端到端流程测试
- 多模块协作测试
- 性能基准测试

### 效果评估
- 检索准确率(Recall@K)
- 检索精确率(Precision@K)
- MRR(Mean Reciprocal Rank)
- NDCG(Normalized Discounted Cumulative Gain)
- 响应时间和吞吐量

## 部署和运维

### 配置管理
- 支持热配置更新
- 环境变量配置
- 配置验证机制

### 监控指标
- 查询处理时间
- 检索成功率
- CRAG决策分布
- 缓存命中率
- 错误率统计

### 扩展性考虑
- 水平扩展支持
- 负载均衡
- 状态分离设计

## 实现风险评估

### 高风险项
- BM25引擎集成复杂度
- CRAG算法调优难度
- 性能优化挑战

### 中风险项
- 配置项复杂度增加
- 多模块协调复杂性
- 错误处理完整性

### 低风险项
- 查询重写实现
- 基础算法实现
- 现有代码扩展

### 风险缓解措施
1. **分阶段实现**：优先实现核心功能
2. **降级方案**：确保向后兼容
3. **充分测试**：覆盖各种场景
4. **文档完善**：便于问题排查

## 创新亮点

1. **完整的CRAG实现**：业界首个在生产级网关中的完整实现
2. **自适应融合策略**：根据查询类型动态调整检索权重
3. **智能上下文压缩**：保持信息完整性的同时减少token消耗
4. **多层回退机制**：确保系统在任何情况下都能提供服务
5. **配置化设计**：支持不同场景的灵活配置

## 总结

本设计方案基于现有稳定的MCP架构，通过模块化的方式实现RAG技术增强，既保证了系统稳定性，又实现了技术创新。方案涵盖了赛题要求的所有核心技术点，具有很强的技术先进性和实用性，能够在比赛期间完整实现并取得优异成绩。