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
}

type stringPrefixMatcher struct {
	target string
}

func (m *stringPrefixMatcher) Match(s string) bool {
	return strings.HasPrefix(s, m.target)
}

type stringSuffixMatcher struct {
	target string
}

func (m *stringSuffixMatcher) Match(s string) bool {
	return strings.HasSuffix(s, m.target)
}

type stringRegexMatcher struct {
	regex *regexp.Regexp
}

func (m *stringRegexMatcher) Match(s string) bool {
	return m.regex.MatchString(s)
}

type stringContainsMatcher struct {
	target string
}

func (m *stringContainsMatcher) Match(s string) bool {
	return strings.Contains(s, m.target)
}

type stringExactMatcher struct {
	target string
}

func (m *stringExactMatcher) Match(s string) bool {
	return s == m.target
}

type repeatedStringMatcher struct {
	matchers []Matcher
}

func (rsm *repeatedStringMatcher) Match(s string) bool {
	for _, m := range rsm.matchers {
		if m.Match(s) {
			return true
		}
	}
	return false
}

func BuildRepeatedStringMatcher(matchers []gjson.Result) (Matcher, error) {
	builtMatchers := make([]Matcher, len(matchers))
	for i, item := range matchers {
		var matcher Matcher

		exactResult := item.Get(matchPatternExact)
		if exactResult.Exists() && exactResult.String() != "" {
			target := exactResult.String()
			matcher = &stringExactMatcher{target: target}
		}

		prefixResult := item.Get(matchPatternPrefix)
		if prefixResult.Exists() && prefixResult.String() != "" {
			target := prefixResult.String()
			matcher = &stringPrefixMatcher{target: target}
		}

		suffixResult := item.Get(matchPatternSuffix)
		if suffixResult.Exists() && suffixResult.String() != "" {
			target := suffixResult.String()
			matcher = &stringSuffixMatcher{target: target}
		}

		containsResult := item.Get(matchPatternContains)
		if containsResult.Exists() && containsResult.String() != "" {
			target := containsResult.String()
			matcher = &stringContainsMatcher{target: target}
		}

		regexResult := item.Get(matchPatternRegex)
		if regexResult.Exists() && regexResult.String() != "" {
			target := regexResult.String()
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
		matchers: builtMatchers,
	}, nil
}
