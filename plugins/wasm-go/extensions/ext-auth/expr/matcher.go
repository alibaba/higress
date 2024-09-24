package expr

import (
	"errors"
	"strings"

	"github.com/tidwall/gjson"
	regexp "github.com/wasilibs/go-re2"
)

const (
	matchPatternExact    string = "exact"
	matchPatternPrefix   string = "prefix"
	matchPatternSuffix   string = "suffix"
	matchPatternContains string = "contains"
	matchPatternRegex    string = "regex"

	matchIgnoreCase string = "ignore_case"
)

type Matcher interface {
	Match(s string) bool
}

type stringExactMatcher struct {
	target     string
	ignoreCase bool
}

func (m *stringExactMatcher) Match(s string) bool {
	if m.ignoreCase {
		return strings.ToLower(s) == m.target
	}
	return s == m.target
}

type stringPrefixMatcher struct {
	target     string
	ignoreCase bool
}

func (m *stringPrefixMatcher) Match(s string) bool {
	if m.ignoreCase {
		return strings.HasPrefix(strings.ToLower(s), m.target)
	}
	return strings.HasPrefix(s, m.target)
}

type stringSuffixMatcher struct {
	target     string
	ignoreCase bool
}

func (m *stringSuffixMatcher) Match(s string) bool {
	if m.ignoreCase {
		return strings.HasSuffix(strings.ToLower(s), m.target)
	}
	return strings.HasSuffix(s, m.target)
}

type stringContainsMatcher struct {
	target     string
	ignoreCase bool
}

func (m *stringContainsMatcher) Match(s string) bool {
	if m.ignoreCase {
		return strings.Contains(strings.ToLower(s), m.target)
	}
	return strings.Contains(s, m.target)
}

type stringRegexMatcher struct {
	regex *regexp.Regexp
}

func (m *stringRegexMatcher) Match(s string) bool {
	return m.regex.MatchString(s)
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

func buildRepeatedStringMatcher(matchers []gjson.Result, allIgnoreCase bool) (Matcher, error) {
	builtMatchers := make([]Matcher, len(matchers))

	createMatcher := func(json gjson.Result, targetKey string, ignoreCase bool, matcherType MatcherConstructor) (Matcher, error) {
		result := json.Get(targetKey)
		if result.Exists() && result.String() != "" {
			target := result.String()
			return matcherType(target, ignoreCase)
		}
		return nil, nil
	}

	for i, item := range matchers {
		var matcher Matcher
		var err error

		// If allIgnoreCase is true, it takes precedence over any user configuration,
		// forcing case-insensitive matching regardless of individual item settings.
		ignoreCase := allIgnoreCase
		if !allIgnoreCase {
			ignoreCaseResult := item.Get(matchIgnoreCase)
			if ignoreCaseResult.Exists() && ignoreCaseResult.Bool() {
				ignoreCase = true
			}
		}

		for _, matcherType := range []struct {
			key     string
			creator MatcherConstructor
		}{
			{matchPatternExact, newStringExactMatcher},
			{matchPatternPrefix, newStringPrefixMatcher},
			{matchPatternSuffix, newStringSuffixMatcher},
			{matchPatternContains, newStringContainsMatcher},
			{matchPatternRegex, newStringRegexMatcher},
		} {
			if matcher, err = createMatcher(item, matcherType.key, ignoreCase, matcherType.creator); err != nil {
				return nil, err
			}
			if matcher != nil {
				break
			}
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

type MatcherConstructor func(string, bool) (Matcher, error)

func newStringExactMatcher(target string, ignoreCase bool) (Matcher, error) {
	if ignoreCase {
		target = strings.ToLower(target)
	}
	return &stringExactMatcher{target: target, ignoreCase: ignoreCase}, nil
}

func newStringPrefixMatcher(target string, ignoreCase bool) (Matcher, error) {
	if ignoreCase {
		target = strings.ToLower(target)
	}
	return &stringPrefixMatcher{target: target, ignoreCase: ignoreCase}, nil
}

func newStringSuffixMatcher(target string, ignoreCase bool) (Matcher, error) {
	if ignoreCase {
		target = strings.ToLower(target)
	}
	return &stringSuffixMatcher{target: target, ignoreCase: ignoreCase}, nil
}

func newStringContainsMatcher(target string, ignoreCase bool) (Matcher, error) {
	if ignoreCase {
		target = strings.ToLower(target)
	}
	return &stringContainsMatcher{target: target, ignoreCase: ignoreCase}, nil
}

func newStringRegexMatcher(target string, ignoreCase bool) (Matcher, error) {
	if ignoreCase && !strings.HasPrefix(target, "(?i)") {
		target = "(?i)" + target
	}
	re, err := regexp.Compile(target)
	if err != nil {
		return nil, err
	}
	return &stringRegexMatcher{regex: re}, nil
}

func BuildRepeatedStringMatcherIgnoreCase(matchers []gjson.Result) (Matcher, error) {
	return buildRepeatedStringMatcher(matchers, true)
}

func BuildRepeatedStringMatcher(matchers []gjson.Result) (Matcher, error) {
	return buildRepeatedStringMatcher(matchers, false)
}

func BuildStringMatcher(matcher gjson.Result) (Matcher, error) {
	return BuildRepeatedStringMatcher([]gjson.Result{matcher})
}
