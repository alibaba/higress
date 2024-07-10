package expr

import (
	"errors"
	"github.com/tidwall/gjson"
	"regexp"
	"strings"
)

const (
	matchPatternExact    string = "exact"
	matchPatternPrefix   string = "prefix"
	matchPatternSuffix   string = "suffix"
	matchPatternContains string = "contains"
	matchPatternRegex    string = "regex"
)

type Matcher interface {
	Match(s string) bool
	IgnoreCase() bool
}

type stringPrefixMatcher struct {
	target     string
	ignoreCase bool
}

func (m *stringPrefixMatcher) Match(s string) bool {
	return strings.HasPrefix(s, m.target)
}

func (m *stringPrefixMatcher) IgnoreCase() bool {
	return m.ignoreCase
}

type stringSuffixMatcher struct {
	target     string
	ignoreCase bool
}

func (m *stringSuffixMatcher) Match(s string) bool {
	return strings.HasSuffix(s, m.target)
}

func (m *stringSuffixMatcher) IgnoreCase() bool {
	return m.ignoreCase
}

type stringRegexMatcher struct {
	regex *regexp.Regexp
}

func (m *stringRegexMatcher) Match(s string) bool {
	return m.regex.MatchString(s)
}

func (m *stringRegexMatcher) IgnoreCase() bool {
	return false
}

type stringContainsMatcher struct {
	target     string
	ignoreCase bool
}

func (m *stringContainsMatcher) Match(s string) bool {
	return strings.Contains(s, m.target)
}

func (m *stringContainsMatcher) IgnoreCase() bool {
	return m.ignoreCase
}

type stringExactMatcher struct {
	target     string
	ignoreCase bool
}

func (m *stringExactMatcher) Match(s string) bool {
	return s == m.target
}

func (m *stringExactMatcher) IgnoreCase() bool {
	return m.ignoreCase
}

type repeatedStringMatcher struct {
	matchers       []Matcher
	needIgnoreCase bool
}

func (rsm *repeatedStringMatcher) Match(s string) bool {
	var ls string
	if rsm.needIgnoreCase {
		// the repeated string matcher will share one case-insensitive input
		ls = strings.ToLower(s)
	}
	for _, m := range rsm.matchers {
		input := s
		if m.IgnoreCase() {
			input = ls
		}
		if m.Match(input) {
			return true
		}
	}
	return false
}

func (rsm *repeatedStringMatcher) IgnoreCase() bool {
	return rsm.needIgnoreCase
}

func buildRepeatedStringMatcher(matchers []gjson.Result, ignoreCase bool) (Matcher, error) {
	builtMatchers := make([]Matcher, len(matchers))
	for i, item := range matchers {
		var matcher Matcher

		exactResult := item.Get(matchPatternExact)
		if exactResult.Exists() && exactResult.String() != "" {
			target := exactResult.String()
			if ignoreCase {
				target = strings.ToLower(target)
			}
			matcher = &stringExactMatcher{target: target, ignoreCase: ignoreCase}
		}

		prefixResult := item.Get(matchPatternPrefix)
		if prefixResult.Exists() && prefixResult.String() != "" {
			target := prefixResult.String()
			if ignoreCase {
				target = strings.ToLower(target)
			}
			matcher = &stringPrefixMatcher{target: target, ignoreCase: ignoreCase}
		}

		suffixResult := item.Get(matchPatternSuffix)
		if suffixResult.Exists() && suffixResult.String() != "" {
			target := suffixResult.String()
			if ignoreCase {
				target = strings.ToLower(target)
			}
			matcher = &stringSuffixMatcher{target: target, ignoreCase: ignoreCase}
		}

		containsResult := item.Get(matchPatternContains)
		if containsResult.Exists() && containsResult.String() != "" {
			target := containsResult.String()
			if ignoreCase {
				target = strings.ToLower(target)
			}
			matcher = &stringContainsMatcher{target: target, ignoreCase: ignoreCase}
		}

		regexResult := item.Get(matchPatternRegex)
		if regexResult.Exists() && regexResult.String() != "" {
			target := regexResult.String()
			if ignoreCase && !strings.HasPrefix(target, "(?i)") {
				target = "(?i)" + target
			}
			re, err := regexp.Compile(target)
			if err != nil {
				return nil, err
			}
			matcher = &stringRegexMatcher{regex: re}
		}

		if matcher == nil {
			return nil, errors.New("unknown string matcher type")
		}

		builtMatchers[i] = matcher
	}

	return &repeatedStringMatcher{
		matchers:       builtMatchers,
		needIgnoreCase: ignoreCase,
	}, nil
}

func BuildRepeatedStringMatcherIgnoreCase(matchers []gjson.Result) (Matcher, error) {
	return buildRepeatedStringMatcher(matchers, true)
}
