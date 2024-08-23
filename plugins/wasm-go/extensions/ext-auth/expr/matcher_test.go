package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestStringMatcher(t *testing.T) {
	tests := []struct {
		name       string
		cfg        string
		matched    []string
		mismatched []string
	}{
		{
			name:       "exact",
			cfg:        `{"exact": "foo"}`,
			matched:    []string{"foo"},
			mismatched: []string{"fo", "fooo"},
		},
		{
			name:    "exact, ignore_case",
			cfg:     `{"exact": "foo", "ignore_case": true}`,
			matched: []string{"Foo", "foo"},
		},
		{
			name:       "prefix",
			cfg:        `{"prefix": "/p"}`,
			matched:    []string{"/p", "/pa"},
			mismatched: []string{"/P"},
		},
		{
			name:       "prefix, ignore_case",
			cfg:        `{"prefix": "/p", "ignore_case": true}`,
			matched:    []string{"/P", "/p", "/pa", "/Pa"},
			mismatched: []string{"/"},
		},
		{
			name:       "suffix",
			cfg:        `{"suffix": "foo"}`,
			matched:    []string{"foo", "0foo"},
			mismatched: []string{"fo", "fooo", "aFoo"},
		},
		{
			name:       "suffix, ignore_case",
			cfg:        `{"suffix": "foo", "ignore_case": true}`,
			matched:    []string{"aFoo", "foo"},
			mismatched: []string{"fo", "fooo"},
		},
		{
			name:       "contains",
			cfg:        `{"contains": "foo"}`,
			matched:    []string{"foo", "0foo", "fooo"},
			mismatched: []string{"fo", "aFoo"},
		},
		{
			name:       "contains, ignore_case",
			cfg:        `{"contains": "foo", "ignore_case": true}`,
			matched:    []string{"aFoo", "foo", "FoO"},
			mismatched: []string{"fo"},
		},
		{
			name:       "regex",
			cfg:        `{"regex": "fo{2}"}`,
			matched:    []string{"foo", "0foo", "fooo"},
			mismatched: []string{"aFoo", "fo"},
		},
		{
			name:       "regex, ignore_case",
			cfg:        `{"regex": "fo{2}", "ignore_case": true}`,
			matched:    []string{"foo", "0foo", "fooo", "aFoo"},
			mismatched: []string{"fo"},
		},
		{
			name:       "regex, ignore_case & case insensitive specified in regex",
			cfg:        `{"regex": "(?i)fo{2}", "ignore_case": true}`,
			matched:    []string{"foo", "0foo", "fooo", "aFoo"},
			mismatched: []string{"fo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			built, _ := BuildStringMatcher(gjson.Parse(tt.cfg))
			for _, s := range tt.matched {
				assert.True(t, built.Match(s))
			}
			for _, s := range tt.mismatched {
				assert.False(t, built.Match(s))
			}
		})
	}
}

func TestBuildRepeatedStringMatcherIgnoreCase(t *testing.T) {
	cfgs := []string{
		`{"exact":"foo"}`,
		`{"prefix":"pre"}`,
		`{"regex":"^Cache"}`,
	}
	matched := []string{"Foo", "foO", "foo", "PreA", "cache-control", "Cache-Control"}
	mismatched := []string{"afoo", "fo"}
	ms := []gjson.Result{}
	for _, cfg := range cfgs {
		ms = append(ms, gjson.Parse(cfg))
	}
	built, _ := BuildRepeatedStringMatcherIgnoreCase(ms)
	for _, s := range matched {
		assert.True(t, built.Match(s))
	}
	for _, s := range mismatched {
		assert.False(t, built.Match(s))
	}
}

func TestPassOutRegexCompileErr(t *testing.T) {
	cfg := `{"regex":"(?!)aa"}`
	_, err := BuildRepeatedStringMatcher([]gjson.Result{gjson.Parse(cfg)})
	assert.NotNil(t, err)
}
