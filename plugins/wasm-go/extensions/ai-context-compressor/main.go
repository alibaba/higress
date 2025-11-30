package main

import (
	"encoding/json"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {}

// ContextCompressorConfig 配置结构体
type ContextCompressorConfig struct {
	Method       string  `json:"method"`
	Rate         float64 `json:"rate"`
	Model        string  `json:"model"`
	MinTokens    int     `json:"minTokens"`
	UseTokenizer bool    `json:"useTokenizer"`
}

// SentenceInfo 句子信息
type SentenceInfo struct {
	Text   string  `json:"text"`
	Tokens int     `json:"tokens"`
	Score  float64 `json:"score"`
	Index  int     `json:"index"`
}

// parseConfig 解析配置
func parseConfig(json gjson.Result) (interface{}, error) {
	config := &ContextCompressorConfig{
		Method:       json.Get("method").String(),
		Rate:         json.Get("rate").Float(),
		Model:        json.Get("model").String(),
		MinTokens:    int(json.Get("minTokens").Int()),
		UseTokenizer: json.Get("useTokenizer").Bool(),
	}

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

// onHttpRequestBody 处理请求体
func onHttpRequestBody(ctx interface{}, config interface{}, body []byte) types.Action {
	cfg := config.(*ContextCompressorConfig)

	// 解析请求体
	request := make(map[string]interface{})
	if err := json.Unmarshal(body, &request); err != nil {
		proxywasm.LogCriticalf("Failed to unmarshal request body: %v", err)
		return types.ActionContinue
	}

	// 获取messages字段
	messages, ok := request["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		proxywasm.LogDebug("No messages found in request")
		return types.ActionContinue
	}

	// 查找最后一个用户消息作为query
	var query string
	for i := len(messages) - 1; i >= 0; i-- {
		if msg, ok := messages[i].(map[string]interface{}); ok {
			if role, ok := msg["role"].(string); ok && role == "user" {
				if content, ok := msg["content"].(string); ok {
					query = content
					break
				}
			}
		}
	}

	if query == "" {
		proxywasm.LogDebug("No user query found")
		return types.ActionContinue
	}

	// 查找tool或function角色的消息作为context
	var contextMsgIndex int = -1
	var context string
	for i := len(messages) - 1; i >= 0; i-- {
		if msg, ok := messages[i].(map[string]interface{}); ok {
			if role, ok := msg["role"].(string); ok && (role == "tool" || role == "function") {
				if content, ok := msg["content"].(string); ok {
					contextMsgIndex = i
					context = content
					break
				}
			}
		}
	}

	if contextMsgIndex == -1 || context == "" {
		proxywasm.LogDebug("No tool/function message found for compression")
		return types.ActionContinue
	}

	// 执行压缩
	compressedContext := compressContext(query, context, cfg)

	// 更新消息内容
	if msg, ok := messages[contextMsgIndex].(map[string]interface{}); ok {
		msg["content"] = compressedContext
		messages[contextMsgIndex] = msg
		request["messages"] = messages
	}

	// 重新序列化请求体
	modifiedBody, err := json.Marshal(request)
	if err != nil {
		proxywasm.LogCriticalf("Failed to marshal modified request: %v", err)
		return types.ActionContinue
	}

	// 替换请求体
	proxywasm.ReplaceHttpRequestBody(modifiedBody)
	proxywasm.LogInfof("Context compressed, original length: %d, compressed length: %d", len(context), len(compressedContext))

	return types.ActionContinue
}

// compressContext 压缩上下文
func compressContext(query, context string, config *ContextCompressorConfig) string {
	switch config.Method {
	case "bm25_extract":
		return bm25Extract(query, context, config.Rate)
	case "llmlingua":
		return simpleCompress(context, config.Rate)
	case "token_based":
		return tokenBasedCompress(query, context, config)
	default:
		return simpleCompress(context, config.Rate)
	}
}

// simpleCompress 简单压缩方法 - 保留前rate比例的内容
func simpleCompress(context string, rate float64) string {
	contentLen := len(context)
	targetLen := int(float64(contentLen) * rate)

	if targetLen >= contentLen {
		return context
	}

	// 简单截取前targetLen个字符
	return context[:targetLen]
}

// bm25Extract BM25提取方法
func bm25Extract(query, context string, rate float64) string {
	// 将上下文切割为句子
	rawSentences := cutSent(context)

	// 清理句子
	var sentences []string
	for _, rawSentence := range rawSentences {
		rawSentence = strings.TrimSpace(rawSentence)
		if rawSentence != "" {
			sentences = append(sentences, rawSentence)
		}
	}

	if len(sentences) == 0 {
		return context
	}

	// 计算每个句子与query的简单相关性得分（这里使用简化实现）
	scores := make([]float64, len(sentences))
	for i, sentence := range sentences {
		scores[i] = calculateSimpleScore(query, sentence)
	}

	// 创建索引排序数组
	type scoreIndex struct {
		score float64
		index int
	}

	scoreIndices := make([]scoreIndex, len(scores))
	for i, score := range scores {
		scoreIndices[i] = scoreIndex{score: score, index: i}
	}

	// 按得分降序排序
	sort.Slice(scoreIndices, func(i, j int) bool {
		return scoreIndices[i].score > scoreIndices[j].score
	})

	// 按原句子相对顺序拼接分数高的句子，直到长度超过原长度的rate比例
	preLen := len(context)
	nowLen := 0
	selectedIndices := make([]int, 0)

	for i, si := range scoreIndices {
		sentenceLen := len(sentences[si.index])
		nowLen += sentenceLen
		selectedIndices = append(selectedIndices, si.index)

		if nowLen >= int(float64(preLen)*rate) {
			// 只取到当前索引
			selectedIndices = selectedIndices[:i+1]
			break
		}
	}

	// 按原句子顺序排序
	sort.Ints(selectedIndices)

	// 构建新上下文
	var newContext strings.Builder
	for _, idx := range selectedIndices {
		newContext.WriteString(sentences[idx])
	}

	return newContext.String()
}

// tokenBasedCompress 基于token的智能压缩（方案一）
func tokenBasedCompress(query, context string, config *ContextCompressorConfig) string {
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

// calculateSimpleScore 计算简单相关性得分
func calculateSimpleScore(query, sentence string) float64 {
	// 简化实现：计算query中的词在sentence中出现的次数
	queryWords := strings.Fields(strings.ToLower(query))
	sentenceWords := strings.Fields(strings.ToLower(sentence))

	score := 0.0
	for _, qWord := range queryWords {
		for _, sWord := range sentenceWords {
			if strings.Contains(sWord, qWord) || strings.Contains(qWord, sWord) {
				score += 1.0
			}
		}
	}

	// 长度归一化
	if len(sentenceWords) > 0 {
		score = score / float64(len(sentenceWords))
	}

	return score
}

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

// cutSent 将文本切割为句子
func cutSent(text string) []string {
	// 简化实现：按句号、问号、感叹号分割
	separators := []string{".", "?", "!"}

	// 替换所有分隔符为统一的分隔符
	unifiedText := text
	for _, sep := range separators {
		unifiedText = strings.ReplaceAll(unifiedText, sep, "|SENT_SEP|")
	}

	// 按分隔符分割
	parts := strings.Split(unifiedText, "|SENT_SEP|")

	// 过滤空字符串并清理空格
	var sentences []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			sentences = append(sentences, trimmed)
		}
	}

	return sentences
}

// init 初始化插件
func init() {
	// 注册插件处理函数
	proxywasm.SetVMContext(&vmContext{})
}

// vmContext VM上下文
type vmContext struct {
	types.DefaultVMContext
}

// NewPluginContext 创建插件上下文
func (*vmContext) NewPluginContext(contextID uint32) types.PluginContext {
	return &pluginContext{}
}

// pluginContext 插件上下文
type pluginContext struct {
	types.DefaultPluginContext
	contextID uint32
	config    *ContextCompressorConfig
}

// OnPluginStart 插件启动时调用
func (p *pluginContext) OnPluginStart(pluginConfigurationSize int) types.OnPluginStartStatus {
	data, err := proxywasm.GetPluginConfiguration()
	if err != nil {
		proxywasm.LogCriticalf("error reading plugin configuration: %v", err)
		return types.OnPluginStartStatusFailed
	}

	if len(data) == 0 {
		proxywasm.LogCritical("no configuration provided")
		return types.OnPluginStartStatusFailed
	}

	jsonData := gjson.ParseBytes(data)
	config, err := parseConfig(jsonData)
	if err != nil {
		proxywasm.LogCriticalf("error parsing configuration: %v", err)
		return types.OnPluginStartStatusFailed
	}

	p.config = config.(*ContextCompressorConfig)
	return types.OnPluginStartStatusOK
}

// NewHttpContext 创建HTTP上下文
func (p *pluginContext) NewHttpContext(contextID uint32) types.HttpContext {
	return &httpContext{contextID: contextID, config: p.config}
}

// httpContext HTTP上下文
type httpContext struct {
	types.DefaultHttpContext
	contextID uint32
	config    *ContextCompressorConfig
}

// OnHttpRequestBody 处理请求体
func (h *httpContext) OnHttpRequestBody(bodySize int, endOfStream bool) types.Action {
	if !endOfStream {
		return types.ActionPause
	}

	body, err := proxywasm.GetHttpRequestBody(0, bodySize)
	if err != nil {
		proxywasm.LogCriticalf("failed to get request body: %v", err)
		return types.ActionContinue
	}

	return onHttpRequestBody(h, h.config, body)
}
