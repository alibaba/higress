package provider

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ContentType 定义内容类型
type ContentType int

const (
	// ContentTypeUnknown 未知类型
	ContentTypeUnknown ContentType = iota
	// ContentTypeMaze 迷宫数据
	ContentTypeMaze
	// ContentTypeCode 代码内容
	ContentTypeCode
	// ContentTypeJSON JSON 数据
	ContentTypeJSON
	// ContentTypeStructuredData 结构化数据（配置、YAML等）
	ContentTypeStructuredData
	// ContentTypeText 普通文本
	ContentTypeText
)

// ContentAnalyzer 内容分析器
// 负责识别内容类型并返回相应的处理策略
type ContentAnalyzer struct {
	// 正则表达式缓存
	mazePattern        *regexp.Regexp
	jsonPattern        *regexp.Regexp
	codePatterns       []*regexp.Regexp
	structuredPatterns []*regexp.Regexp
}

// NewContentAnalyzer 创建内容分析器
func NewContentAnalyzer() *ContentAnalyzer {
	return &ContentAnalyzer{
		// 迷宫模式：包含 # 和空格的网格
		mazePattern: regexp.MustCompile(`(?m)^[#\s]{3,}$`),

		// JSON 模式
		jsonPattern: regexp.MustCompile(`^\s*[\{\[]`),

		// 代码模式
		codePatterns: []*regexp.Regexp{
			regexp.MustCompile(`\bfunc\s+\w+\s*\(`),             // Go 函数
			regexp.MustCompile(`\bfunction\s+\w+\s*\(`),         // JS 函数
			regexp.MustCompile(`\bdef\s+\w+\s*\(`),              // Python 函数
			regexp.MustCompile(`\b(class|struct|interface)\s+`), // 类定义
			regexp.MustCompile(`\b(if|for|while|switch)\s*\(`),  // 控制流
			regexp.MustCompile(`\bimport\s+`),                   // 导入语句
		},

		// 结构化数据模式
		structuredPatterns: []*regexp.Regexp{
			regexp.MustCompile(`^\s*\w+:\s*`), // YAML 键值对
			regexp.MustCompile(`\[^\]]+\]=`),  // INI 配置
		},
	}
}

// AnalyzeContent 分析内容类型
// 返回: 内容类型、置信度(0-100)
func (ca *ContentAnalyzer) AnalyzeContent(content string) (ContentType, int) {
	if len(content) == 0 {
		return ContentTypeText, 0
	}

	// 1. 尝试识别 JSON
	if ca.isJSON(content) {
		return ContentTypeJSON, 95
	}

	// 2. 尝试识别迷宫
	if ca.isMaze(content) {
		return ContentTypeMaze, 90
	}

	// 3. 尝试识别代码
	if ca.isCode(content) {
		return ContentTypeCode, 85
	}

	// 4. 尝试识别结构化数据
	if ca.isStructuredData(content) {
		return ContentTypeStructuredData, 80
	}

	// 默认为普通文本
	return ContentTypeText, 50
}

// isJSON 检测是否为 JSON 内容
func (ca *ContentAnalyzer) isJSON(content string) bool {
	// 尝试解析 JSON
	var data interface{}
	err := json.Unmarshal([]byte(content), &data)
	if err == nil {
		return true
	}

	// 检查是否以 { 或 [ 开头
	trimmed := strings.TrimSpace(content)
	return ca.jsonPattern.MatchString(trimmed)
}

// isMaze 检测是否为迷宫数据
// 迷宫特征：
// - 包含 # 和空格
// - 多行
// - 行长度相近（网格结构）
// - 包含 S（起点）和 E（终点）可选
func (ca *ContentAnalyzer) isMaze(content string) bool {
	lines := strings.Split(content, "\n")

	// 必须至少 3 行
	if len(lines) < 3 {
		return false
	}

	// 统计包含迷宫字符的行
	mazeLines := 0
	totalWidth := 0
	hasStartOrEnd := false

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// 检查是否只包含迷宫相关字符
		if ca.mazePattern.MatchString(line) {
			mazeLines++
			totalWidth += len(line)

			// 检查是否包含 S 或 E
			if strings.ContainsAny(line, "SE") {
				hasStartOrEnd = true
			}
		}
	}

	// 如果超过 70% 的行都是迷宫格式，判定为迷宫
	if mazeLines > 0 && float64(mazeLines)/float64(len(lines)) > 0.7 {
		return true
	}

	// 如果包含 S 和空格与 # 的混合，也认为是迷宫
	if hasStartOrEnd && strings.Contains(content, "#") && strings.Contains(content, " ") {
		return true
	}

	return false
}

// isCode 检测是否为代码内容
func (ca *ContentAnalyzer) isCode(content string) bool {
	// 检查是否匹配任何代码模式
	matchCount := 0
	for _, pattern := range ca.codePatterns {
		if pattern.MatchString(content) {
			matchCount++
		}
	}

	// 如果匹配 2 个或以上的代码模式，判定为代码
	return matchCount >= 2
}

// isStructuredData 检测是否为结构化数据
func (ca *ContentAnalyzer) isStructuredData(content string) bool {
	lines := strings.Split(content, "\n")

	// 检查是否为 YAML/INI 格式
	structuredLines := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			continue
		}

		for _, pattern := range ca.structuredPatterns {
			if pattern.MatchString(trimmed) {
				structuredLines++
				break
			}
		}
	}

	// 如果超过 50% 的行都是结构化格式，判定为结构化数据
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}

	if nonEmptyLines > 0 && structuredLines > nonEmptyLines/2 {
		return true
	}

	return false
}

// GetCompressionStrategy 获取内容的压缩策略
func (ca *ContentAnalyzer) GetCompressionStrategy(contentType ContentType) CompressionStrategy {
	switch contentType {
	case ContentTypeMaze:
		return CompressionStrategyNone // 迷宫禁用压缩
	case ContentTypeCode:
		return CompressionStrategyConservative // 代码保守压缩
	case ContentTypeJSON:
		return CompressionStrategyAggressive // JSON 积极压缩
	case ContentTypeStructuredData:
		return CompressionStrategyConservative // 结构化数据保守压缩
	default:
		return CompressionStrategyNormal // 普通文本正常压缩
	}
}

// CompressionStrategy 定义压缩策略
type CompressionStrategy int

const (
	// CompressionStrategyNone 禁用压缩
	CompressionStrategyNone CompressionStrategy = iota
	// CompressionStrategyConservative 保守压缩（阈值提高 2-3 倍）
	CompressionStrategyConservative
	// CompressionStrategyNormal 正常压缩
	CompressionStrategyNormal
	// CompressionStrategyAggressive 积极压缩（阈值降低）
	CompressionStrategyAggressive
)

// StringContent 返回策略的字符串表示
func (cs CompressionStrategy) String() string {
	switch cs {
	case CompressionStrategyNone:
		return "none"
	case CompressionStrategyConservative:
		return "conservative"
	case CompressionStrategyNormal:
		return "normal"
	case CompressionStrategyAggressive:
		return "aggressive"
	default:
		return "unknown"
	}
}
