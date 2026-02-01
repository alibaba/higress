# Token Details 支持

基于Higress PR #3424，Agent Session Monitor现在支持更细粒度的token统计。

## 新增字段

### 1. reasoning_tokens
- **说明**: 推理过程产生的token数（OpenAI o1等推理模型）
- **来源**: 从`output_token_details.reasoning_tokens`提取
- **计费**: reasoning tokens通常与output tokens相同计费标准
- **示例模型**: o1, o1-mini, DeepSeek-R1

### 2. cached_tokens
- **说明**: 从缓存中命中的token数（Prompt Caching）
- **来源**: 从`input_token_details.cached_tokens`提取
- **计费**: cached tokens通常比regular input便宜50-90%
- **使用场景**: 重复的system prompt、常用对话模板

### 3. input_token_details
- **说明**: 完整的输入token详情对象（JSON格式）
- **内容**: 包含cached_tokens等详细信息
- **示例**: `{"cached_tokens": 80}`

### 4. output_token_details
- **说明**: 完整的输出token详情对象（JSON格式）
- **内容**: 包含reasoning_tokens等详细信息
- **示例**: `{"reasoning_tokens": 500}`

## 日志格式

### 带有token details的ai_log示例

```json
{
  "ai_log": "{
    \"session_id\": \"agent:main:discord:123\",
    \"model\": \"gpt-4o\",
    \"input_token\": 150,
    \"output_token\": 100,
    \"reasoning_tokens\": 0,
    \"cached_tokens\": 120,
    \"input_token_details\": \"{\\\"cached_tokens\\\":120}\",
    \"output_token_details\": \"{}\",
    \"messages\": [...],
    \"question\": \"...\",
    \"answer\": \"...\"
  }"
}
```

## 成本计算

### GPT-4o（带Prompt Caching）

```
Input: 150 tokens
  - Cached: 120 tokens @ $0.00125/M = $0.00000015
  - Regular: 30 tokens @ $0.0025/M = $0.000000075
Output: 100 tokens @ $0.01/M = $0.00000100
─────────────────────────────────────────
Total: $0.000001225 USD
```

### OpenAI o1（带Reasoning）

```
Input: 100 tokens @ $0.015/M = $0.00000150
Output: 80 tokens @ $0.06/M = $0.00000480
Reasoning: 500 tokens @ $0.06/M = $0.00003000
─────────────────────────────────────────
Total: $0.00003630 USD
```

## CLI显示

### Session详情

```
📈 Token Statistics:
   Input:             650 tokens
   Cached:            400 tokens (from cache)
   Total Input:       650 tokens
   Output:            450 tokens
   ────────────────────────
   Total:            1100 tokens

💰 Estimated Cost: $0.00000563 USD

📝 Conversation Rounds:

  Round 1 @ 2026-02-01T10:00:00Z
    Tokens: 150 in → 100 out
    📦 120 cached
    📊 Input Token Details: {'cached_tokens': 120}
    ...
```

### 按模型统计

```
Model                Sessions   Input           Output          Cost (USD)
────────────────────────────────────────────────────────────────────────
gpt-4o               1                   650           450  $  0.000004
o1                    1                   100            80  $  0.000036
────────────────────────────────────────────────────────────────────────
TOTAL                 2                   750           530  $  0.000040
```

## Web界面

### 总览页面
显示所有session的统计，包括新的token类型。

### Session详情页
每轮对话显示：
- Token统计（包含cached/reasoning badge）
- Token Details（JSON格式）
- 完整对话历史

示例显示：
```
Round 1 @ 2026-02-01T10:00:00Z
  150 in → 100 out 📦 120 cached
  ...
  📊 Token Details:
    - Input: {'cached_tokens': 120}
```

## 配置Higress

要在Higress中记录token details，需要在ai-statistics插件配置中添加：

```yaml
attributes:
  # 记录推理token（o1等模型）
  - key: reasoning_tokens
    apply_to_log: true
  
  # 记录缓存token（prompt caching）
  - key: cached_tokens
    apply_to_log: true
  
  # 记录完整token详情
  - key: input_token_details
    apply_to_log: true
  
  - key: output_token_details
    apply_to_log: true
```

## 优势

### 成本优化
- **缓存命中率追踪**: 了解prompt caching的效果
- **缓存vs非缓存对比**: 分析缓存带来的成本节省

### 性能分析
- **Reasoning开销**: 评估推理模型的实际成本
- **Token效率**: 分析不同模型的token使用效率

### 使用统计
- **细粒度统计**: 区分regular/cached/reasoning tokens
- **趋势分析**: 追踪不同token类型的使用趋势

## 定价表

| 模型 | Input ($/M) | Output ($/M) | Cached ($/M) | Reasoning ($/M) |
|------|-------------|--------------|--------------|----------------|
| GPT-4o | 0.0025 | 0.01 | 0.00125 | - |
| o1 | 0.015 | 0.06 | 0.0075 | 0.06 |
| o1-mini | 0.003 | 0.012 | 0.0015 | 0.012 |
| Claude | 0.015 | 0.075 | 0.0015 (90% off) | - |
| Qwen | 0.0002 | 0.0006 | 0.0001 | - |
| DeepSeek-R1 | 0.004 | 0.012 | 0.002 | 0.002 |

## 测试

运行新功能测试：

```bash
cd example
bash demo_v2.sh
```

这会解析包含token details的日志，并显示：
- cached tokens统计（gpt-4o）
- reasoning tokens统计（o1）
- 优化后的成本计算

## 向后兼容

- ✅ 不包含token details的旧日志仍然可以正常解析
- ✅ 新字段默认为0
- ✅ 成本计算自动适配（无cached/reasoning时按旧逻辑）
