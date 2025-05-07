package common

import (
	"regexp"
	"strings"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

// RuleType defines the type of matching rule
type RuleType string

// UpstreamType defines the type of matching rule
type UpstreamType string

const (
	ExactMatch    RuleType = "exact"
	PrefixMatch   RuleType = "prefix"
	SuffixMatch   RuleType = "suffix"
	ContainsMatch RuleType = "contains"
	RegexMatch    RuleType = "regex"

	RestUpstream       UpstreamType = "rest"
	SSEUpstream        UpstreamType = "sse"
	StreamableUpstream UpstreamType = "streamable"
)

// MatchRule defines the structure for a matching rule
type MatchRule struct {
	MatchRuleDomain   string       `json:"match_rule_domain"`   // Domain pattern, supports wildcards
	MatchRulePath     string       `json:"match_rule_path"`     // Path pattern to match
	MatchRuleType     RuleType     `json:"match_rule_type"`     // Type of match rule
	UpstreamType      UpstreamType `json:"upstream_type"`       // Type of upstream(s) matched by the rule
	EnablePathRewrite bool         `json:"enable_path_rewrite"` // Enable request path rewrite for matched routes
	PathRewritePrefix string       `json:"path_rewrite_prefix"` // Prefix the request path would be rewritten to.
}

// ParseMatchList parses the match list from the config
func ParseMatchList(matchListConfig []interface{}) []MatchRule {
	matchList := make([]MatchRule, 0)
	for _, item := range matchListConfig {
		if ruleMap, ok := item.(map[string]interface{}); ok {
			rule := MatchRule{}
			if domain, ok := ruleMap["match_rule_domain"].(string); ok {
				rule.MatchRuleDomain = domain
			}
			if path, ok := ruleMap["match_rule_path"].(string); ok {
				rule.MatchRulePath = path
			}
			if ruleType, ok := ruleMap["match_rule_type"].(string); ok {
				rule.MatchRuleType = RuleType(ruleType)
			}
			if upstreamType, ok := ruleMap["upstream_type"].(string); ok {
				rule.UpstreamType = UpstreamType(upstreamType)
			}
			if len(rule.UpstreamType) == 0 {
				rule.UpstreamType = RestUpstream
			} else {
				switch rule.UpstreamType {
				case RestUpstream:
				case SSEUpstream:
				case StreamableUpstream:
					break
				default:
					api.LogWarnf("Unknown upstream type: %s", rule.UpstreamType)
				}
			}
			if enablePathRewrite, ok := ruleMap["enable_path_rewrite"].(bool); ok {
				rule.EnablePathRewrite = enablePathRewrite
			}
			if pathRewritePrefix, ok := ruleMap["path_rewrite_prefix"].(string); ok {
				rule.PathRewritePrefix = pathRewritePrefix
			}
			if rule.EnablePathRewrite {
				if rule.UpstreamType != SSEUpstream {
					api.LogWarnf("Path rewrite is only supported for SSE upstream type")
				} else if rule.PathRewritePrefix == "" {
					rule.PathRewritePrefix = "/"
				}
			}
			matchList = append(matchList, rule)
		}
	}
	return matchList
}

// convertWildcardToRegex converts wildcard pattern to regex pattern
func convertWildcardToRegex(pattern string) string {
	pattern = regexp.QuoteMeta(pattern)
	pattern = "^" + strings.ReplaceAll(pattern, "\\*", ".*") + "$"
	return pattern
}

// matchPattern checks if the target matches the pattern based on rule type
func matchPattern(pattern string, target string, ruleType RuleType) bool {
	if pattern == "" {
		return true
	}

	switch ruleType {
	case ExactMatch:
		return pattern == target
	case PrefixMatch:
		return strings.HasPrefix(target, pattern)
	case SuffixMatch:
		return strings.HasSuffix(target, pattern)
	case ContainsMatch:
		return strings.Contains(target, pattern)
	case RegexMatch:
		matched, err := regexp.MatchString(pattern, target)
		if err != nil {
			return false
		}
		return matched
	default:
		return false
	}
}

// matchDomain checks if the domain matches the pattern
func matchDomain(domain string, pattern string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	// Convert wildcard pattern to regex pattern
	regexPattern := convertWildcardToRegex(pattern)
	matched, _ := regexp.MatchString(regexPattern, domain)
	return matched
}

// matchDomainAndPath checks if both domain and path match the rule
func matchDomainAndPath(domain, path string, rule MatchRule) bool {
	return matchDomain(domain, rule.MatchRuleDomain) &&
		matchPattern(rule.MatchRulePath, path, rule.MatchRuleType)
}

// IsMatch checks if the request matches any rule in the rule list
// Returns true if no rules are specified
func IsMatch(rules []MatchRule, host, path string) (bool, MatchRule) {
	if len(rules) == 0 {
		return true, MatchRule{}
	}

	for _, rule := range rules {
		if matchDomainAndPath(host, path, rule) {
			return true, rule
		}
	}
	return false, MatchRule{}
}

// MatchDomainList checks if the domain matches any of the domains in the list
func MatchDomainList(domain string, domainList []string) bool {
	for _, d := range domainList {
		if matchDomain(domain, d) {
			return true
		}
	}
	return false
}
