package provider

import (
	"github.com/higress-group/wasm-go/pkg/log"
)

// LayeredCompressionStrategy 分层压缩策略
// 根据数据优先级应用不同的压缩策略
type LayeredCompressionStrategy struct {
	thresholdMgr   *ThresholdManager
	keyExtractor   *SmartKeyExtractor
	analyzer       *ContentAnalyzer
}

// NewLayeredCompressionStrategy 创建分层压缩策略
func NewLayeredCompressionStrategy() *LayeredCompressionStrategy {
	return &LayeredCompressionStrategy{
		thresholdMgr: NewThresholdManager(),
		keyExtractor: NewSmartKeyExtractor(),
		analyzer:     NewContentAnalyzer(),
	}
}

// CompressionDecision 压缩决策结果
type CompressionDecision struct {
	// 是否应该压缩
	ShouldCompress bool

	// 数据优先级层级
	DataLayer DataLayer

	// 压缩策略
	Strategy CompressionStrategy

	// 推荐的关键信息
	KeyInfo string

	// 压缩理由
	Reason string

	// Token 阈值
	TokenThreshold int

	// 字节阈值
	BytesThreshold int
}

// DecideCompression 决策是否压缩以及如何压缩
// 综合考虑：内容类型、数据优先级、配置策略
func (lcs *LayeredCompressionStrategy) DecideCompression(
	content string,
	useTokenBased bool,
	agentMode string,
	preserveKeyInfo bool) *CompressionDecision {

	decision := &CompressionDecision{
		ShouldCompress: false,
		DataLayer:      DataLayerNormal,
		Strategy:       CompressionStrategyNormal,
		KeyInfo:        "",
		Reason:         "",
	}

	if len(content) == 0 {
		decision.Reason = "Empty content"
		return decision
	}

	// 1. 分析内容类型
	contentType, confidence := lcs.analyzer.AnalyzeContent(content)
	log.Debugf("[LayeredCompression] Content type: %v (confidence: %d%%)", 
		lcs.contentTypeString(contentType), confidence)

	// 2. 获取数据优先级层级
	dataLayer := lcs.thresholdMgr.AnalyzeDataLayer(content)
	decision.DataLayer = dataLayer

	// 3. 根据 Agent 模式调整策略
	strategy := lcs.thresholdMgr.analyzer.GetCompressionStrategy(contentType)
	if agentMode == "conservative" {
		// Agent 保守模式：提升策略的保守程度
		if strategy == CompressionStrategyAggressive {
			strategy = CompressionStrategyNormal
		} else if strategy == CompressionStrategyNormal {
			strategy = CompressionStrategyConservative
		}
	}
	decision.Strategy = strategy

	// 4. 获取压缩阈值
	shouldCompress, tokenThreshold, bytesThreshold, finalStrategy := 
		lcs.thresholdMgr.GetCompressionThreshold(content, useTokenBased)

	decision.TokenThreshold = tokenThreshold
	decision.BytesThreshold = bytesThreshold
	decision.Strategy = finalStrategy

	// 5. 根据数据优先级调整压缩决策
	switch dataLayer {
	case DataLayerCritical:
		// 关键数据：不压缩
		decision.ShouldCompress = false
		decision.Reason = "Critical data: compression disabled"
		decision.Strategy = CompressionStrategyNone

	case DataLayerImportant:
		// 重要数据：只在特别大时压缩
		if useTokenBased {
			tokens := calculateTokensDeepSeekFromString(content)
			decision.ShouldCompress = tokens > tokenThreshold*2
			decision.Reason = "Important data: conservative compression"
		} else {
			decision.ShouldCompress = len(content) > bytesThreshold*2
			decision.Reason = "Important data: conservative compression"
		}

	case DataLayerNormal:
		// 普通数据：正常压缩
		decision.ShouldCompress = shouldCompress
		decision.Reason = "Normal data: standard compression"

	case DataLayerLow:
		// 低优先级数据：积极压缩
		decision.ShouldCompress = true
		decision.Reason = "Low priority data: aggressive compression"
		decision.Strategy = CompressionStrategyAggressive
	}

	// 6. 如果要压缩，提取关键信息
	if decision.ShouldCompress && preserveKeyInfo {
		keyInfo := lcs.keyExtractor.ExtractSmartKeyInfo(content, contentType)
		if len(keyInfo) > 0 {
			decision.KeyInfo = keyInfo
		}
	}

	// 7. 记录决策
	log.Infof("[LayeredCompression] Decision: shouldCompress=%v, layer=%v, strategy=%v, reason=%s",
		decision.ShouldCompress, lcs.dataLayerString(dataLayer), 
		strategy, decision.Reason)

	return decision
}

// ApplyLayeredCompression 应用分层压缩
// 返回压缩后的内容或原始内容
func (lcs *LayeredCompressionStrategy) ApplyLayeredCompression(
	content string,
	contentType ContentType,
	decision *CompressionDecision) string {

	if !decision.ShouldCompress {
		return content
	}

	// 构建压缩后的内容
	var compressed string

	switch decision.Strategy {
	case CompressionStrategyNone:
		// 不压缩
		compressed = content

	case CompressionStrategyConservative:
		// 保守压缩：保留更多信息
		compressed = lcs.compressConservative(content, contentType, decision.KeyInfo)

	case CompressionStrategyNormal:
		// 标准压缩
		compressed = lcs.compressNormal(content, contentType, decision.KeyInfo)

	case CompressionStrategyAggressive:
		// 积极压缩：只保留摘要
		compressed = lcs.compressAggressive(content, contentType, decision.KeyInfo)

	default:
		compressed = content
	}

	log.Debugf("[LayeredCompression] Compressed: %d -> %d bytes (%.1f%%)",
		len(content), len(compressed), float64(len(compressed))/float64(len(content))*100)

	return compressed
}

// compressConservative 保守压缩：保留原始内容的大部分
func (lcs *LayeredCompressionStrategy) compressConservative(
	content string,
	contentType ContentType,
	keyInfo string) string {

	// 保守压缩：只删除冗余行
	lines := breakIntoLines(content)
	var compressed []string

	for i, line := range lines {
		// 保留不重复的行
		isDuplicate := false
		if i > 0 && line == lines[i-1] {
			isDuplicate = true
		}

		if !isDuplicate && len(line) > 0 {
			compressed = append(compressed, line)
		}
	}

	result := joinLines(compressed)

	// 如果有关键信息，加入摘要
	if len(keyInfo) > 0 {
		result = "[Key Info: " + keyInfo + "]\n" + result
	}

	return result
}

// compressNormal 标准压缩
func (lcs *LayeredCompressionStrategy) compressNormal(
	content string,
	contentType ContentType,
	keyInfo string) string {

	// 标准压缩：删除冗余信息和简化格式
	lines := breakIntoLines(content)
	var compressed []string
	var lastLine string

	for _, line := range lines {
		trimmed := trimSpaceCompact(line)

		// 跳过空行和重复行
		if len(trimmed) == 0 || trimmed == lastLine {
			continue
		}

		// 跳过纯注释行
		if isCommentLine(trimmed) {
			continue
		}

		compressed = append(compressed, trimmed)
		lastLine = trimmed
	}

	result := joinLines(compressed)

	// 如果有关键信息，加入摘要
	if len(keyInfo) > 0 {
		result = "[Key Info: " + keyInfo + "]\n" + result
	}

	return result
}

// compressAggressive 积极压缩：只保留摘要
func (lcs *LayeredCompressionStrategy) compressAggressive(
	content string,
	contentType ContentType,
	keyInfo string) string {

	// 积极压缩：生成摘要，只保留核心信息
	var summary string

	switch contentType {
	case ContentTypeMaze:
		summary = "Maze data (structure preserved via key info)"
	case ContentTypeCode:
		summary = "Code content (structure preserved via key info)"
	case ContentTypeJSON:
		summary = "JSON data (structure preserved via key info)"
	case ContentTypeStructuredData:
		summary = "Structured data configuration"
	default:
		// 提取前几行作为摘要
		lines := breakIntoLines(content)
		maxLines := 3
		if len(lines) > maxLines {
			lines = lines[:maxLines]
		}
		summary = joinLines(lines) + "\n[...content compressed...]"
	}

	// 组合关键信息和摘要
	if len(keyInfo) > 0 {
		summary = "[Key Info: " + keyInfo + "]\n" + summary
	}

	return summary
}

// breakIntoLines 将内容分成行
func breakIntoLines(content string) []string {
	return splitByNewline(content)
}

// joinLines 将行合并成字符串
func joinLines(lines []string) string {
	result := ""
	for _, line := range lines {
		if len(result) > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}

// trimSpaceCompact 紧凑地修剪空格
func trimSpaceCompact(line string) string {
	// 保留原始内容，只是修剪前后空格
	return trimSpace(line)
}

// isCommentLine 检查是否为注释行
func isCommentLine(line string) bool {
	return len(line) > 0 && 
		(line[0:1] == "#" || 
		 (len(line) > 1 && line[0:2] == "//") ||
		 (len(line) > 1 && line[0:2] == "--"))
}

// contentTypeString 返回内容类型的字符串表示
func (lcs *LayeredCompressionStrategy) contentTypeString(ct ContentType) string {
	switch ct {
	case ContentTypeMaze:
		return "Maze"
	case ContentTypeCode:
		return "Code"
	case ContentTypeJSON:
		return "JSON"
	case ContentTypeStructuredData:
		return "StructuredData"
	case ContentTypeText:
		return "Text"
	default:
		return "Unknown"
	}
}

// dataLayerString 返回数据优先级层级的字符串表示
func (lcs *LayeredCompressionStrategy) dataLayerString(dl DataLayer) string {
	switch dl {
	case DataLayerCritical:
		return "Critical"
	case DataLayerImportant:
		return "Important"
	case DataLayerNormal:
		return "Normal"
	case DataLayerLow:
		return "Low"
	default:
		return "Unknown"
	}
}

