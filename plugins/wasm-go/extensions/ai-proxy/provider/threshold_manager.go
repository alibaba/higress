package provider

import (
	"github.com/higress-group/wasm-go/pkg/log"
)

// ThresholdConfig 定义不同内容类型的压缩阈值配置
type ThresholdConfig struct {
	// 关键数据层（Critical）：完全不压缩
	// 用于：迷宫地图、完整代码逻辑、关键结构等
	CriticalTokenThreshold int
	CriticalBytesThreshold int

	// 重要数据层（Important）：保守压缩
	// 用于：代码片段、配置文件、结构化数据等
	ImportantTokenThreshold int
	ImportantBytesThreshold int

	// 普通数据层（Normal）：标准压缩
	// 用于：日志信息、一般文本、简单工具输出等
	NormalTokenThreshold int
	NormalBytesThreshold int

	// 低优先级数据（Low）：积极压缩
	// 用于：JSON 数据、重复信息等
	LowTokenThreshold int
	LowBytesThreshold int
}

// DefaultThresholdConfig 返回默认阈值配置
// 基于 DeepSeek 的 Token 计算标准
func DefaultThresholdConfig() ThresholdConfig {
	return ThresholdConfig{
		// 关键层：100KB（约 30000 tokens）
		CriticalTokenThreshold: 30000,
		CriticalBytesThreshold: 102400,

		// 重要层：5KB（约 1500 tokens）
		ImportantTokenThreshold: 1500,
		ImportantBytesThreshold: 5120,

		// 普通层：1KB（约 300 tokens）
		NormalTokenThreshold: 300,
		NormalBytesThreshold: 1024,

		// 低优先级层：300B（约 100 tokens）
		LowTokenThreshold: 100,
		LowBytesThreshold: 300,
	}
}

// ThresholdManager 动态阈值管理器
type ThresholdManager struct {
	config   ThresholdConfig
	analyzer *ContentAnalyzer
}

// NewThresholdManager 创建阈值管理器
func NewThresholdManager() *ThresholdManager {
	return &ThresholdManager{
		config:   DefaultThresholdConfig(),
		analyzer: NewContentAnalyzer(),
	}
}

// GetCompressionThreshold 获取压缩阈值
// 根据内容类型和大小返回相应的阈值
// 返回：(shouldCompress, tokenThreshold, bytesThreshold, strategy)
func (tm *ThresholdManager) GetCompressionThreshold(content string, useTokenBased bool) (bool, int, int, CompressionStrategy) {
	if len(content) == 0 {
		return false, 0, 0, CompressionStrategyNone
	}

	// 分析内容类型
	contentType, confidence := tm.analyzer.AnalyzeContent(content)
	strategy := tm.analyzer.GetCompressionStrategy(contentType)

	// 记录分析结果
	log.Debugf("[ThresholdManager] Content analyzed: type=%v, confidence=%d%%, strategy=%v",
		tm.contentTypeString(contentType), confidence, strategy)

	// 根据策略返回是否压缩
	switch strategy {
	case CompressionStrategyNone:
		// 禁用压缩：阈值设置为最大值
		return false, tm.config.CriticalTokenThreshold, tm.config.CriticalBytesThreshold, strategy

	case CompressionStrategyConservative:
		// 保守压缩：使用重要层阈值
		tokenThreshold := tm.config.ImportantTokenThreshold
		bytesThreshold := tm.config.ImportantBytesThreshold

		if useTokenBased {
			tokens := calculateTokensDeepSeekFromString(content)
			return tokens > tokenThreshold, tokenThreshold, bytesThreshold, strategy
		}
		return len(content) > bytesThreshold, tokenThreshold, bytesThreshold, strategy

	case CompressionStrategyNormal:
		// 正常压缩：使用普通层阈值
		tokenThreshold := tm.config.NormalTokenThreshold
		bytesThreshold := tm.config.NormalBytesThreshold

		if useTokenBased {
			tokens := calculateTokensDeepSeekFromString(content)
			return tokens > tokenThreshold, tokenThreshold, bytesThreshold, strategy
		}
		return len(content) > bytesThreshold, tokenThreshold, bytesThreshold, strategy

	case CompressionStrategyAggressive:
		// 积极压缩：使用低优先级层阈值
		tokenThreshold := tm.config.LowTokenThreshold
		bytesThreshold := tm.config.LowBytesThreshold

		if useTokenBased {
			tokens := calculateTokensDeepSeekFromString(content)
			return tokens > tokenThreshold, tokenThreshold, bytesThreshold, strategy
		}
		return len(content) > bytesThreshold, tokenThreshold, bytesThreshold, strategy

	default:
		// 默认使用普通层阈值
		tokenThreshold := tm.config.NormalTokenThreshold
		bytesThreshold := tm.config.NormalBytesThreshold

		if useTokenBased {
			tokens := calculateTokensDeepSeekFromString(content)
			return tokens > tokenThreshold, tokenThreshold, bytesThreshold, CompressionStrategyNormal
		}
		return len(content) > bytesThreshold, tokenThreshold, bytesThreshold, CompressionStrategyNormal
	}
}

// ShouldCompressContent 判断是否应该压缩内容
// 综合考虑内容类型、大小和配置
func (tm *ThresholdManager) ShouldCompressContent(content string, useTokenBased bool) bool {
	shouldCompress, _, _, _ := tm.GetCompressionThreshold(content, useTokenBased)
	return shouldCompress
}

// GetStrategy 获取内容的压缩策略
func (tm *ThresholdManager) GetStrategy(content string) CompressionStrategy {
	_, _ = tm.analyzer.AnalyzeContent(content)
	contentType, _ := tm.analyzer.AnalyzeContent(content)
	return tm.analyzer.GetCompressionStrategy(contentType)
}

// UpdateThresholds 更新阈值配置
func (tm *ThresholdManager) UpdateThresholds(config ThresholdConfig) {
	tm.config = config
	log.Infof("[ThresholdManager] Thresholds updated: %+v", config)
}

// contentTypeString 返回内容类型的字符串表示
func (tm *ThresholdManager) contentTypeString(ct ContentType) string {
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

// ExtractDataLayer 提取数据的优先级层级
// 用于分层压缩决策
type DataLayer int

const (
	// DataLayerCritical 关键层：完全不压缩
	DataLayerCritical DataLayer = iota
	// DataLayerImportant 重要层：保守压缩
	DataLayerImportant
	// DataLayerNormal 普通层：标准压缩
	DataLayerNormal
	// DataLayerLow 低优先级层：积极压缩
	DataLayerLow
)

// AnalyzeDataLayer 分析数据的优先级层级
// 根据内容中的关键词判断数据优先级
func (tm *ThresholdManager) AnalyzeDataLayer(content string) DataLayer {
	contentType, confidence := tm.analyzer.AnalyzeContent(content)

	// 高置信度的特殊类型应提升优先级
	if confidence >= 90 {
		switch contentType {
		case ContentTypeMaze, ContentTypeCode:
			return DataLayerCritical // 迷宫和代码：关键层
		case ContentTypeStructuredData:
			return DataLayerImportant // 结构化数据：重要层
		}
	}

	// 根据内容中的关键词判断优先级
	if tm.containsCriticalKeywords(content) {
		return DataLayerCritical
	}

	if tm.containsImportantKeywords(content) {
		return DataLayerImportant
	}

	if tm.containsLowKeywords(content) {
		return DataLayerLow
	}

	// 默认为普通层
	return DataLayerNormal
}

// containsCriticalKeywords 检查是否包含关键词
func (tm *ThresholdManager) containsCriticalKeywords(content string) bool {
	criticalKeywords := []string{
		"error", "ERROR", "failed", "FAILED",
		"exception", "panic", "fatal",
		"permission denied", "access denied",
		"critical", "CRITICAL",
	}

	for _, keyword := range criticalKeywords {
		if len(content) > 5000 && !containsKeyword(content, keyword) {
			continue
		}
		if containsKeyword(content, keyword) {
			return true
		}
	}

	return false
}

// containsImportantKeywords 检查是否包含重要关键词
func (tm *ThresholdManager) containsImportantKeywords(content string) bool {
	importantKeywords := []string{
		"success", "SUCCESS", "completed", "COMPLETED",
		"info", "INFO", "warning", "WARNING",
		"status", "result", "output",
	}

	for _, keyword := range importantKeywords {
		if containsKeyword(content, keyword) {
			return true
		}
	}

	return false
}

// containsLowKeywords 检查是否包含低优先级关键词
func (tm *ThresholdManager) containsLowKeywords(content string) bool {
	lowKeywords := []string{
		"debug", "DEBUG", "trace", "TRACE",
		"empty", "null", "none", "N/A",
		"redundant", "duplicate",
	}

	for _, keyword := range lowKeywords {
		if containsKeyword(content, keyword) {
			return true
		}
	}

	return false
}

// containsKeyword 检查字符串是否包含关键词
func containsKeyword(content, keyword string) bool {
	return len(content) > 0 && len(keyword) > 0 &&
		(content == keyword ||
			len(content) > len(keyword) &&
				(content[:len(keyword)] == keyword ||
					content[len(content)-len(keyword):] == keyword ||
					indexOfKeyword(content, keyword) >= 0))
}

// indexOfKeyword 查找关键词的位置（简化版）
func indexOfKeyword(content, keyword string) int {
	for i := 0; i <= len(content)-len(keyword); i++ {
		match := true
		for j := 0; j < len(keyword); j++ {
			if content[i+j] != keyword[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
