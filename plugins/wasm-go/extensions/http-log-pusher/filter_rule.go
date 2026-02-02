package main

import (
	"regexp"
	"strings"
)

const (
	ModeWhitelist = "whitelist"
	ModeBlacklist = "blacklist"
)

// FilterRules 日志过滤规则
type FilterRules struct {
	Mode     string       // whitelist 或 blacklist
	RuleList []FilterRule // 过滤规则列表
}

// FilterRule 单条过滤规则
type FilterRule struct {
	Domain string   // 域名匹配，支持通配符 *.example.com
	Method []string // HTTP 方法列表，为空表示匹配所有方法
	Path   Matcher  // 路径匹配器
}

// FilterRulesDefaults 返回默认的过滤规则配置
func FilterRulesDefaults() FilterRules {
	return FilterRules{
		Mode:     ModeWhitelist,
		RuleList: []FilterRule{},
	}
}

// ShouldLog 判断是否应该记录日志
// 白名单模式：匹配规则则记录日志
// 黑名单模式：匹配规则则不记录日志
func (config *FilterRules) ShouldLog(domain, method, path string) bool {
	switch config.Mode {
	case ModeWhitelist:
		// 白名单为空，记录所有日志
		if len(config.RuleList) == 0 {
			return true
		}
		// 匹配任一规则则记录
		for _, rule := range config.RuleList {
			if rule.matchesAllConditions(domain, method, path) {
				return true
			}
		}
		return false
	case ModeBlacklist:
		// 黑名单为空，记录所有日志
		if len(config.RuleList) == 0 {
			return true
		}
		// 匹配任一规则则不记录
		for _, rule := range config.RuleList {
			if rule.matchesAllConditions(domain, method, path) {
				return false
			}
		}
		return true
	default:
		// 未知模式，默认记录所有日志
		return true
	}
}

// matchesAllConditions 检查域名、方法和路径是否都匹配规则
func (rule *FilterRule) matchesAllConditions(domain, method, path string) bool {
	// 如果所有条件都为空，返回 false
	if rule.Domain == "" && rule.Path == nil && len(rule.Method) == 0 {
		return false
	}

	// 检查域名和路径匹配
	domainMatch := rule.Domain == "" || matchDomain(domain, rule.Domain)
	pathMatch := rule.Path == nil || rule.Path.Match(path)

	// 检查 HTTP 方法匹配：如果未指定方法，则匹配所有方法
	methodMatch := len(rule.Method) == 0 || containsString(rule.Method, method)

	return domainMatch && pathMatch && methodMatch
}

// matchDomain 检查域名是否匹配模式（支持通配符）
func matchDomain(domain string, pattern string) bool {
	// 将通配符模式转换为正则表达式
	regexPattern := convertWildcardToRegex(pattern)
	matched, _ := regexp.MatchString(regexPattern, domain)
	return matched
}

// convertWildcardToRegex 将通配符模式转换为正则表达式
func convertWildcardToRegex(pattern string) string {
	pattern = regexp.QuoteMeta(pattern)
	pattern = "^" + strings.ReplaceAll(pattern, "\\*", ".*") + "$"
	return pattern
}

// containsString 检查字符串数组是否包含指定字符串
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, str) {
			return true
		}
	}
	return false
}
