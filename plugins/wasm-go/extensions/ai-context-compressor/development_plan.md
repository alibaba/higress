# AI网关上下文压缩器增强开发方案

## 1. 需求分析

根据用户需求和DeepSeek API文档，我们需要增强现有的上下文压缩器实现，主要改进点包括：

1. 实现更精确的token计算，支持中英文字符的准确统计
2. 集成tokenizer以离线计算文本的token用量
3. 根据实际token数量而非字符数量进行压缩决策
4. 实现方案一（基于token的智能压缩）

## 2. 技术方案设计

### 2.1 Token计算增强

根据DeepSeek API文档提供的换算比例：
- 1个英文字符 ≈ 0.3个token
- 1个中文字符 ≈ 0.6个token

我们将实现一个更精确的token计算器：

```go
// calculateTokens 计算文本的token数量
func calculateTokens(text string) int {
    var tokenCount float64
    for _, r := range text {
        if unicode.Is(unicode.Scripts["Han"], r) {
            // 中文字符
            tokenCount += 0.6
        } else {
            // 英文字符、数字、符号等
            tokenCount += 0.3
        }
    }
    return int(math.Ceil(tokenCount))
}
```

### 2.2 Tokenizer集成

为了更精确地计算token数量，我们将集成tokenizer库：

1. 使用开源tokenizer库（如`samber/go-bpe-tokenizer`）
2. 支持主流模型的tokenizer（GPT系列、LLaMA系列等）
3. 提供离线token计算能力

### 2.3 方案一实现

方案一：基于token的智能压缩

1. 分析上下文内容，识别不同类型的文本段落
2. 根据token数量计算压缩阈值
3. 使用BM25算法结合token权重进行句子重要性评分
4. 按重要性排序并选择token数量符合阈值的句子

## 3. 详细实现方案

### 3.1 核心数据结构

```go
// TokenContextCompressorConfig Token上下文压缩配置
type TokenContextCompressorConfig struct {
    Method string  `json:"method"`          // 压缩方法
    Rate   float64 `json:"rate"`            // 压缩率
    Model  string  `json:"model"`           // 模型类型，用于选择tokenizer
    MinTokens int  `json:"minTokens"`       // 最小token阈值
}

// SentenceInfo 句子信息
type SentenceInfo struct {
    Text      string  `json:"text"`           // 句子文本
    Tokens    int     `json:"tokens"`         // token数量
    Score     float64 `json:"score"`          // BM25得分
    Index     int     `json:"index"`          // 原始索引
}
```

### 3.2 Token计算增强实现

```go
// calculateTokensEnhanced 增强版token计算
func calculateTokensEnhanced(text string) int {
    var tokenCount float64
    for _, r := range text {
        switch {
        case unicode.Is(unicode.Scripts["Han"], r):
            // 中文字符
            tokenCount += 0.6
        case unicode.IsLetter(r) && (r < 128):
            // 英文字母
            tokenCount += 0.3
        case unicode.IsDigit(r):
            // 数字
            tokenCount += 0.3
        case unicode.IsPunct(r):
            // 标点符号
            tokenCount += 0.2
        case unicode.IsSpace(r):
            // 空格
            tokenCount += 0.1
        default:
            // 其他字符
            tokenCount += 0.3
        }
    }
    return int(math.Ceil(tokenCount))
}

// calculateTokensWithTokenizer 使用tokenizer计算token数量
func calculateTokensWithTokenizer(text, model string) (int, error) {
    // 根据模型类型选择对应的tokenizer
    tokenizer, err := getTokenizerForModel(model)
    if err != nil {
        return 0, err
    }
    
    // 使用tokenizer计算token数量
    tokens := tokenizer.Encode(text)
    return len(tokens), nil
}
```

### 3.3 方案一：基于token的智能压缩

```go
// tokenBasedCompress 基于token的智能压缩（方案一）
func tokenBasedCompress(query, context string, config *TokenContextCompressorConfig) string {
    // 1. 将上下文切割为句子
    rawSentences := cutSent(context)
    
    // 2. 清理句子并计算token数量
    var sentences []SentenceInfo
    for i, rawSentence := range rawSentences {
        rawSentence = strings.TrimSpace(rawSentence)
        if rawSentence != "" {
            tokens := calculateTokensEnhanced(rawSentence)
            sentences = append(sentences, SentenceInfo{
                Text:   rawSentence,
                Tokens: tokens,
                Index:  i,
            })
        }
    }
    
    if len(sentences) == 0 {
        return context
    }
    
    // 3. 计算每个句子与query的BM25得分
    scores := make([]float64, len(sentences))
    for i, sentence := range sentences {
        scores[i] = calculateBM25Score(query, sentence.Text)
    }
    
    // 4. 将得分与token数量结合
    for i := range sentences {
        sentences[i].Score = scores[i]
    }
    
    // 5. 按得分降序排序
    sort.Slice(sentences, func(i, j int) bool {
        return sentences[i].Score > sentences[j].Score
    })
    
    // 6. 按原句子相对顺序拼接高分句子，直到token数量达到阈值
    targetTokens := int(float64(calculateTokensEnhanced(context)) * config.Rate)
    if targetTokens < config.MinTokens {
        targetTokens = config.MinTokens
    }
    
    currentTokens := 0
    selectedIndices := make([]int, 0)
    
    for _, sentence := range sentences {
        if currentTokens >= targetTokens {
            break
        }
        currentTokens += sentence.Tokens
        selectedIndices = append(selectedIndices, sentence.Index)
    }
    
    // 7. 按原句子顺序排序并构建新上下文
    sort.Ints(selectedIndices)
    
    var newContext strings.Builder
    for _, idx := range selectedIndices {
        for _, sentence := range sentences {
            if sentence.Index == idx {
                newContext.WriteString(sentence.Text)
                break
            }
        }
    }
    
    return newContext.String()
}
```

### 3.4 BM25算法增强

```go
// calculateBM25Score 增强版BM25得分计算
func calculateBM25Score(query, sentence string) float64 {
    // 分词处理
    queryTerms := tokenizeText(query)
    sentenceTerms := tokenizeText(sentence)
    
    // 计算词频
    termFreq := make(map[string]int)
    for _, term := range sentenceTerms {
        termFreq[term]++
    }
    
    // BM25参数
    k1 := 1.2
    b := 0.75
    avgDocLen := float64(len(sentenceTerms))
    
    // 计算得分
    var score float64
    for _, term := range queryTerms {
        if freq, exists := termFreq[term]; exists {
            // IDF计算（简化版）
            idf := math.Log(float64(len(sentenceTerms)) / float64(freq+1))
            
            // TF计算
            tf := float64(freq)
            
            // BM25公式
            numerator := tf * (k1 + 1)
            denominator := tf + k1*(1-b+b*float64(len(sentenceTerms))/avgDocLen)
            score += idf * numerator / denominator
        }
    }
    
    return score
}

// tokenizeText 简单分词处理
func tokenizeText(text string) []string {
    // 转换为小写
    text = strings.ToLower(text)
    
    // 简单按空格和标点符号分割
    re := regexp.MustCompile(`[\s\p{P}]+`)
    terms := re.Split(text, -1)
    
    // 过滤空字符串
    var result []string
    for _, term := range terms {
        if term != "" {
            result = append(result, term)
        }
    }
    
    return result
}
```

## 4. 集成tokenizer库

### 4.1 依赖添加

在go.mod中添加tokenizer依赖：

```go
require (
    github.com/samber/go-bpe-tokenizer v0.1.0
    // 其他依赖...
)
```

### 4.2 Tokenizer封装

```go
// getTokenizerForModel 根据模型类型获取tokenizer
func getTokenizerForModel(model string) (Tokenizer, error) {
    switch {
    case strings.Contains(model, "gpt"):
        return NewGPTTokenizer()
    case strings.Contains(model, "llama"):
        return NewLLAMATokenizer()
    case strings.Contains(model, "qwen"):
        return NewQwenTokenizer()
    default:
        // 默认使用基础tokenizer
        return NewBaseTokenizer(), nil
    }
}

// Tokenizer tokenizer接口
type Tokenizer interface {
    Encode(text string) []int
    Decode(tokens []int) string
}
```

## 5. 配置增强

### 5.1 新增配置项

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: ai-context-compressor-enhanced
  namespace: higress-system
spec:
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-context-compressor:1.0.0
  phase: AUTHN
  priority: -1
  config:
    method: "token_based"           # 压缩方法：bm25_extract, llmlingua, token_based
    rate: 0.5                       # 压缩率
    model: "gpt-4"                  # 模型类型，用于tokenizer选择
    minTokens: 100                  # 最小token数量
    useTokenizer: true              # 是否使用tokenizer进行精确计算
```

### 5.2 配置解析增强

```go
// parseEnhancedConfig 解析增强配置
func parseEnhancedConfig(json gjson.Result) (interface{}, error) {
    config := &TokenContextCompressorConfig{
        Method:     json.Get("method").String(),
        Rate:       json.Get("rate").Float(),
        Model:      json.Get("model").String(),
        MinTokens:  int(json.Get("minTokens").Int()),
        UseTokenizer: json.Get("useTokenizer").Bool(),
    }
    
    // 设置默认值
    if config.Method == "" {
        config.Method = "token_based"
    }
    
    if config.Rate <= 0 || config.Rate > 1 {
        config.Rate = 0.5
    }
    
    if config.MinTokens <= 0 {
        config.MinTokens = 100
    }
    
    return config, nil
}
```

## 6. 生产环境考虑

### 6.1 性能优化

1. **缓存机制**：对频繁出现的文本进行token计算结果缓存
2. **并发处理**：对大量文本进行并发token计算
3. **资源限制**：设置tokenizer使用的内存和CPU限制

### 6.2 错误处理

1. **降级机制**：tokenizer不可用时降级到字符估算
2. **超时控制**：设置token计算的超时时间
3. **日志记录**：详细记录token计算和压缩过程

### 6.3 监控指标

1. **压缩率统计**：记录压缩前后的token数量对比
2. **处理时间**：记录token计算和压缩处理时间
3. **错误率统计**：记录tokenizer相关错误

## 7. 测试方案

### 7.1 单元测试

```go
func TestTokenBasedCompress(t *testing.T) {
    config := &TokenContextCompressorConfig{
        Method: "token_based",
        Rate:   0.5,
        Model:  "gpt-4",
    }
    
    query := "important information"
    context := "This is some unimportant text. This sentence contains important information. More unimportant text here."
    
    compressed := tokenBasedCompress(query, context, config)
    
    // 验证压缩结果
    if len(compressed) >= len(context) {
        t.Errorf("Expected compressed context to be shorter than original")
    }
    
    // 验证重要信息保留
    if !strings.Contains(compressed, "important information") {
        t.Errorf("Expected important information to be retained")
    }
}
```

### 7.2 集成测试

1. **tokenizer集成测试**：验证不同模型tokenizer的集成
2. **性能测试**：测试大量文本的处理性能
3. **准确性测试**：对比tokenizer计算与估算结果的准确性

## 8. 部署方案

### 8.1 镜像构建

```dockerfile
# 构建阶段
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o main.wasm main.go

# 运行阶段
FROM scratch
COPY --from=builder /app/main.wasm ./main.wasm
```

### 8.2 部署配置

```yaml
apiVersion: extensions.higress.io/v1alpha1
kind: WasmPlugin
metadata:
  name: ai-context-compressor-enhanced
  namespace: higress-system
spec:
  url: oci://higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/ai-context-compressor-enhanced:1.0.0
  phase: AUTHN
  priority: -1
  config:
    method: "token_based"
    rate: 0.5
    model: "gpt-4"
    minTokens: 100
    useTokenizer: true
```

