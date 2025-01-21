package expr

import (
	"strings"

	regexp "github.com/wasilibs/go-re2"
)

const (
	ModeWhitelist = "whitelist"
	ModeBlacklist = "blacklist"
)

type MatchRules struct {
	Mode     string
	RuleList []Rule
}

type Rule struct {
	Domain string
	Path   Matcher
}

func MatchRulesDefaults() MatchRules {
	return MatchRules{
		Mode:     ModeWhitelist,
		RuleList: []Rule{},
	}
}

// IsAllowedByMode checks if the given domain and path are allowed based on the configuration mode.
func (config *MatchRules) IsAllowedByMode(domain, path string) bool {
	switch config.Mode {
	case ModeWhitelist:
		for _, rule := range config.RuleList {
			if rule.matchDomainAndPath(domain, path) {
				return true
			}
		}
		return false
	case ModeBlacklist:
		for _, rule := range config.RuleList {
			if rule.matchDomainAndPath(domain, path) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// matchDomainAndPath checks if the given domain and path match the rule.
// If rule.Domain is empty, it only checks rule.Path.
// If rule.Path is empty, it only checks rule.Domain.
// If both are empty, it returns false.
func (rule *Rule) matchDomainAndPath(domain, path string) bool {
	if rule.Domain == "" && rule.Path == nil {
		return false
	}
	domainMatch := rule.Domain == "" || matchDomain(domain, rule.Domain)
	pathMatch := rule.Path == nil || rule.Path.Match(path)
	return domainMatch && pathMatch
}

// matchDomain checks if the given domain matches the pattern.
func matchDomain(domain string, pattern string) bool {
	// Convert wildcard pattern to regex pattern
	regexPattern := convertWildcardToRegex(pattern)
	matched, _ := regexp.MatchString(regexPattern, domain)
	return matched
}

// convertWildcardToRegex converts a wildcard pattern to a regex pattern.
func convertWildcardToRegex(pattern string) string {
	pattern = regexp.QuoteMeta(pattern)
	pattern = "^" + strings.ReplaceAll(pattern, "\\*", ".*") + "$"
	return pattern
}
