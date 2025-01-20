package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAllowedByMode(t *testing.T) {
	tests := []struct {
		name     string
		config   MatchRules
		domain   string
		path     string
		expected bool
	}{
		{
			name: "Whitelist mode, rule matches",
			config: MatchRules{
				Mode: ModeWhitelist,
				RuleList: []Rule{
					{
						Domain: "example.com",
						Path: func() Matcher {
							pathMatcher, err := newStringExactMatcher("/foo", true)
							if err != nil {
								t.Fatalf("Failed to create Matcher: %v", err)
							}
							return pathMatcher
						}(),
					},
				},
			},
			domain:   "example.com",
			path:     "/foo",
			expected: true,
		},
		{
			name: "Whitelist mode, rule does not match",
			config: MatchRules{
				Mode: ModeWhitelist,
				RuleList: []Rule{
					{
						Domain: "example.com",
						Path: func() Matcher {
							pathMatcher, err := newStringExactMatcher("/foo", true)
							if err != nil {
								t.Fatalf("Failed to create Matcher: %v", err)
							}
							return pathMatcher
						}(),
					},
				},
			},
			domain:   "example.com",
			path:     "/bar",
			expected: false,
		},
		{
			name: "Blacklist mode, rule matches",
			config: MatchRules{
				Mode: ModeBlacklist,
				RuleList: []Rule{
					{
						Domain: "example.com",
						Path: func() Matcher {
							pathMatcher, err := newStringExactMatcher("/foo", true)
							if err != nil {
								t.Fatalf("Failed to create Matcher: %v", err)
							}
							return pathMatcher
						}(),
					},
				},
			},
			domain:   "example.com",
			path:     "/foo",
			expected: false,
		},
		{
			name: "Blacklist mode, rule does not match",
			config: MatchRules{
				Mode: ModeBlacklist,
				RuleList: []Rule{
					{
						Domain: "example.com",
						Path: func() Matcher {
							pathMatcher, err := newStringExactMatcher("/foo", true)
							if err != nil {
								t.Fatalf("Failed to create Matcher: %v", err)
							}
							return pathMatcher
						}(),
					},
				},
			},
			domain:   "example.com",
			path:     "/bar",
			expected: true,
		},
		{
			name: "Domain matches, Path is empty",
			config: MatchRules{
				Mode: ModeWhitelist,
				RuleList: []Rule{
					{Domain: "example.com", Path: nil},
				},
			},
			domain:   "example.com",
			path:     "/foo",
			expected: true,
		},
		{
			name: "Domain is empty, Path matches",
			config: MatchRules{
				Mode: ModeWhitelist,
				RuleList: []Rule{
					{
						Domain: "",
						Path: func() Matcher {
							pathMatcher, err := newStringExactMatcher("/foo", true)
							if err != nil {
								t.Fatalf("Failed to create Matcher: %v", err)
							}
							return pathMatcher
						}(),
					},
				},
			},
			domain:   "example.com",
			path:     "/foo",
			expected: true,
		},
		{
			name: "Both Domain and Path are empty",
			config: MatchRules{
				Mode: ModeWhitelist,
				RuleList: []Rule{
					{Domain: "", Path: nil},
				},
			},
			domain:   "example.com",
			path:     "/foo",
			expected: false,
		},
		{
			name: "Invalid mode",
			config: MatchRules{
				Mode: "invalid",
				RuleList: []Rule{
					{
						Domain: "example.com",
						Path: func() Matcher {
							pathMatcher, err := newStringExactMatcher("/foo", true)
							if err != nil {
								t.Fatalf("Failed to create Matcher: %v", err)
							}
							return pathMatcher
						}(),
					},
				},
			},
			domain:   "example.com",
			path:     "/foo",
			expected: false,
		},
		{
			name: "Whitelist mode, generic domain matches",
			config: MatchRules{
				Mode: ModeWhitelist,
				RuleList: []Rule{
					{
						Domain: "*.example.com",
						Path: func() Matcher {
							pathMatcher, err := newStringExactMatcher("/foo", true)
							if err != nil {
								t.Fatalf("Failed to create Matcher: %v", err)
							}
							return pathMatcher
						}(),
					},
				},
			},
			domain:   "sub.example.com",
			path:     "/foo",
			expected: true,
		},
		{
			name: "Whitelist mode, generic domain does not match",
			config: MatchRules{
				Mode: ModeWhitelist,
				RuleList: []Rule{
					{
						Domain: "*.example.com",
						Path: func() Matcher {
							pathMatcher, err := newStringExactMatcher("/foo", true)
							if err != nil {
								t.Fatalf("Failed to create Matcher: %v", err)
							}
							return pathMatcher
						}(),
					},
				},
			},
			domain:   "example.com",
			path:     "/foo",
			expected: false,
		},
		{
			name: "Blacklist mode, generic domain matches",
			config: MatchRules{
				Mode: ModeBlacklist,
				RuleList: []Rule{
					{
						Domain: "*.example.com",
						Path: func() Matcher {
							pathMatcher, err := newStringExactMatcher("/foo", true)
							if err != nil {
								t.Fatalf("Failed to create Matcher: %v", err)
							}
							return pathMatcher
						}(),
					},
				},
			},
			domain:   "sub.example.com",
			path:     "/foo",
			expected: false,
		},
		{
			name: "Blacklist mode, generic domain does not match",
			config: MatchRules{
				Mode: ModeBlacklist,
				RuleList: []Rule{
					{
						Domain: "*.example.com",
						Path: func() Matcher {
							pathMatcher, err := newStringExactMatcher("/foo", true)
							if err != nil {
								t.Fatalf("Failed to create Matcher: %v", err)
							}
							return pathMatcher
						}(),
					},
				},
			},
			domain:   "example.com",
			path:     "/foo",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsAllowedByMode(tt.domain, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
