package provider

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/higress-group/wasm-go/pkg/log"
)

// SmartKeyExtractor 智能关键信息提取器
// 根据内容类型提取不同的关键信息
type SmartKeyExtractor struct {
	codeSignatureRegex *regexp.Regexp
	coordinateRegex    *regexp.Regexp
	filePathRegex      *regexp.Regexp
	urlRegex           *regexp.Regexp
	numberRegex        *regexp.Regexp
}

// NewSmartKeyExtractor 创建智能提取器
func NewSmartKeyExtractor() *SmartKeyExtractor {
	return &SmartKeyExtractor{
		// 代码签名：函数定义、类定义等
		codeSignatureRegex: regexp.MustCompile(`(?:func|function|def|class|struct|interface)\s+\w+\s*(?:\(|{|:|\<)`),

		// 坐标系统：(x,y)、[row,col]等
		coordinateRegex: regexp.MustCompile(`[(\[](-?\d+)\s*,\s*(-?\d+)[)\]]`),

		// 文件路径
		filePathRegex: regexp.MustCompile(`(?:^|[\s\n])(/[^\s\n]+|\./[^\s\n]+|[A-Z]:\\[^\s\n]+)`),

		// URL
		urlRegex: regexp.MustCompile(`https?://[^\s\n]+`),

		// 数字
		numberRegex: regexp.MustCompile(`\b\d+\b`),
	}
}

// ExtractSmartKeyInfo 智能提取关键信息
// 根据内容类型选择不同的提取策略
func (ske *SmartKeyExtractor) ExtractSmartKeyInfo(content string, contentType ContentType) string {
	switch contentType {
	case ContentTypeMaze:
		return ske.extractMazeKeyInfo(content)
	case ContentTypeCode:
		return ske.extractCodeKeyInfo(content)
	case ContentTypeJSON:
		return ske.extractJSONKeyInfo(content)
	case ContentTypeStructuredData:
		return ske.extractStructuredDataKeyInfo(content)
	default:
		return ske.extractGenericKeyInfo(content)
	}
}

// extractMazeKeyInfo 提取迷宫关键信息
// 保留：起点、终点、网格大小、坐标系统
func (ske *SmartKeyExtractor) extractMazeKeyInfo(content string) string {
	var keyInfo []string

	lines := strings.Split(content, "\n")
	width, height := 0, len(lines)

	// 提取迷宫尺寸
	for _, line := range lines {
		if len(line) > width {
			width = len(line)
		}
	}
	if width > 0 && height > 0 {
		keyInfo = append(keyInfo, "Size: "+string(rune(width))+"x"+string(rune(height)))
	}

	// 提取起点位置
	for i, line := range lines {
		if idx := strings.Index(line, "S"); idx >= 0 {
			keyInfo = append(keyInfo, "Start: ("+string(rune(idx))+","+string(rune(i))+")")
			break
		}
	}

	// 提取终点位置
	for i, line := range lines {
		if idx := strings.Index(line, "E"); idx >= 0 {
			keyInfo = append(keyInfo, "Exit: ("+string(rune(idx))+","+string(rune(i))+")")
			break
		}
	}

	// 提取迷宫复杂度指标
	wallCount := strings.Count(content, "#")
	pathCount := strings.Count(content, " ")
	if wallCount > 0 && pathCount > 0 {
		totalCells := wallCount + pathCount
		wallPercentage := (wallCount * 100) / totalCells
		keyInfo = append(keyInfo, "Complexity: "+string(rune(wallPercentage))+"%")
	}

	log.Debugf("[SmartKeyExtractor] Maze key info extracted: %v", keyInfo)
	return strings.Join(keyInfo, "; ")
}

// extractCodeKeyInfo 提取代码关键信息
// 保留：函数签名、类定义、导入语句、关键逻辑
func (ske *SmartKeyExtractor) extractCodeKeyInfo(content string) string {
	var keyInfo []string

	lines := strings.Split(content, "\n")

	// 提取函数和类定义
	signatures := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if ske.codeSignatureRegex.MatchString(trimmed) {
			// 取第一 50 个字符作为签名
			if len(trimmed) > 50 {
				signatures = append(signatures, trimmed[:50]+"...")
			} else {
				signatures = append(signatures, trimmed)
			}
		}
	}

	if len(signatures) > 0 {
		// 最多保留 3 个签名
		maxSigs := 3
		if len(signatures) > maxSigs {
			signatures = signatures[:maxSigs]
		}
		keyInfo = append(keyInfo, "Signatures: "+strings.Join(signatures, " | "))
	}

	// 提取导入语句
	imports := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") || 
		   strings.HasPrefix(trimmed, "from ") ||
		   strings.HasPrefix(trimmed, "require(") ||
		   strings.HasPrefix(trimmed, "package ") {
			if len(trimmed) > 60 {
				imports = append(imports, trimmed[:60]+"...")
			} else {
				imports = append(imports, trimmed)
			}
		}
	}

	if len(imports) > 0 {
		keyInfo = append(keyInfo, "Imports: "+strings.Join(imports[:min(2, len(imports))], " | "))
	}

	// 提取代码统计信息
	lineCount := len(lines)
	commentCount := 0
	codeCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			commentCount++
		} else if len(trimmed) > 0 {
			codeCount++
		}
	}

	keyInfo = append(keyInfo, "Stats: "+string(rune(lineCount))+" lines, "+string(rune(codeCount))+" code")

	log.Debugf("[SmartKeyExtractor] Code key info extracted: %v", keyInfo)
	return strings.Join(keyInfo, "; ")
}

// extractJSONKeyInfo 提取 JSON 关键信息
// 保留：顶级键名、数据结构、统计信息
func (ske *SmartKeyExtractor) extractJSONKeyInfo(content string) string {
	var keyInfo []string

	var data interface{}
	err := json.Unmarshal([]byte(content), &data)
	if err != nil {
		log.Warnf("[SmartKeyExtractor] Failed to parse JSON: %v", err)
		return ""
	}

	// 提取顶级键名
	if m, ok := data.(map[string]interface{}); ok {
		keys := []string{}
		for key := range m {
			keys = append(keys, key)
		}

		// 最多保留 5 个键名
		maxKeys := 5
		if len(keys) > maxKeys {
			keys = keys[:maxKeys]
		}

		if len(keys) > 0 {
			keyInfo = append(keyInfo, "Keys: "+strings.Join(keys, ", "))
		}

		// 统计数据结构
		keyInfo = append(keyInfo, "Structure: "+ske.analyzeJSONStructure(data))
	}

	// 记录原始数据大小和估计的 token 数
	tokens := calculateTokensDeepSeekFromString(content)
	keyInfo = append(keyInfo, "Tokens: "+string(rune(tokens)))

	log.Debugf("[SmartKeyExtractor] JSON key info extracted: %v", keyInfo)
	return strings.Join(keyInfo, "; ")
}

// analyzeJSONStructure 分析 JSON 结构
func (ske *SmartKeyExtractor) analyzeJSONStructure(data interface{}) string {
	switch v := data.(type) {
	case map[string]interface{}:
		return "Object{" + string(rune(len(v))) + "}"
	case []interface{}:
		return "Array[" + string(rune(len(v))) + "]"
	case string:
		return "String"
	case float64:
		return "Number"
	case bool:
		return "Boolean"
	case nil:
		return "Null"
	default:
		return "Unknown"
	}
}

// extractStructuredDataKeyInfo 提取结构化数据关键信息
// 保留：配置键名、值类型、配置项数量
func (ske *SmartKeyExtractor) extractStructuredDataKeyInfo(content string) string {
	var keyInfo []string

	lines := strings.Split(content, "\n")

	// 提取配置键名
	keys := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// 尝试分解键值对
		if idx := strings.Index(trimmed, ":"); idx > 0 {
			key := strings.TrimSpace(trimmed[:idx])
			if len(key) > 0 && len(key) < 100 {
				keys = append(keys, key)
			}
		} else if idx := strings.Index(trimmed, "="); idx > 0 {
			key := strings.TrimSpace(trimmed[:idx])
			if len(key) > 0 && len(key) < 100 {
				keys = append(keys, key)
			}
		}
	}

	// 最多保留 5 个键名
	maxKeys := 5
	if len(keys) > maxKeys {
		keys = keys[:maxKeys]
	}

	if len(keys) > 0 {
		keyInfo = append(keyInfo, "Keys: "+strings.Join(keys, ", "))
	}

	// 统计配置项数量
	itemCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") {
			itemCount++
		}
	}

	keyInfo = append(keyInfo, "Items: "+string(rune(itemCount)))

	log.Debugf("[SmartKeyExtractor] Structured data key info extracted: %v", keyInfo)
	return strings.Join(keyInfo, "; ")
}

// extractGenericKeyInfo 提取通用关键信息
// 保留：文件路径、URL、重要数字、执行状态
func (ske *SmartKeyExtractor) extractGenericKeyInfo(content string) string {
	var keyInfo []string

	// 提取文件路径
	filePaths := ske.filePathRegex.FindAllString(content, 3)
	if len(filePaths) > 0 {
		uniquePaths := uniqueStringsFiltered(filePaths)
		keyInfo = append(keyInfo, "Files: "+strings.Join(uniquePaths, ", "))
	}

	// 提取 URL
	urls := ske.urlRegex.FindAllString(content, 2)
	if len(urls) > 0 {
		keyInfo = append(keyInfo, "URLs: "+strings.Join(urls, ", "))
	}

	// 提取执行状态
	if strings.Contains(strings.ToLower(content), "success") ||
		strings.Contains(strings.ToLower(content), "ok") ||
		strings.Contains(strings.ToLower(content), "completed") {
		keyInfo = append(keyInfo, "Status: Success")
	}
	if strings.Contains(strings.ToLower(content), "error") ||
		strings.Contains(strings.ToLower(content), "failed") ||
		strings.Contains(strings.ToLower(content), "exception") {
		keyInfo = append(keyInfo, "Status: Error")
	}

	// 提取数字
	numbers := ske.numberRegex.FindAllString(content, 3)
	if len(numbers) > 0 {
		keyInfo = append(keyInfo, "Numbers: "+strings.Join(numbers, ", "))
	}

	log.Debugf("[SmartKeyExtractor] Generic key info extracted: %v", keyInfo)
	return strings.Join(keyInfo, "; ")
}

// uniqueStringsFiltered 去重并过滤字符串
func uniqueStringsFiltered(strs []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, s := range strs {
		trimmed := strings.TrimSpace(s)
		if len(trimmed) > 0 && !seen[trimmed] {
			seen[trimmed] = true
			result = append(result, trimmed)
		}
	}

	return result
}

// min 返回两个整数的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

