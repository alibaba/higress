package main

import (
	"testing"
)

func Test_prefixMatchCode(t *testing.T) {
	rules := map[string]*CustomResponseRule{
		"x01": {},
		"2x3": {},
		"45x": {},
		"6xx": {},
		"x7x": {},
		"xx8": {},
	}

	tests := []struct {
		code      string
		expectHit bool
	}{
		{"101", true},  // 匹配x01
		{"201", true},  // 匹配x01
		{"111", false}, // 不匹配
		{"203", true},  // 匹配2x3
		{"213", true},  // 匹配2x3
		{"450", true},  // 匹配45x
		{"451", true},  // 匹配45x
		{"600", true},  // 匹配6xx
		{"611", true},  // 匹配6xx
		{"612", true},  // 匹配6xx
		{"171", true},  // 匹配x7x
		{"161", false}, // 不匹配
		{"228", true},  // 匹配xx8
		{"229", false}, // 不匹配
		{"123", false}, // 不匹配
	}

	for _, tt := range tests {
		_, found := fuzzyMatchCode(rules, tt.code)
		if found != tt.expectHit {
			t.Errorf("code:%s expect:%v got:%v", tt.code, tt.expectHit, found)
		}
	}
}

func TestIsValidPrefixString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasError bool
	}{
		{"x1x", "x1x", false},
		{"X2X", "x2x", false},
		{"xx1", "xx1", false},
		{"x12", "x12", false},
		{"1x2", "1x2", false},
		{"12x", "12x", false},
		{"123", "", true},  // 缺少x
		{"xxx", "", true},  // 缺少数字
		{"xYx", "", true},  // 非法字符
		{"x1", "", true},   // 长度不足
		{"x123", "", true}, // 长度超限
	}

	for _, tt := range tests {
		result, err := isValidFuzzyMatchString(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("%q: expected error but got none", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("%q: unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("%q: expected %q, got %q", tt.input, tt.expected, result)
			}
		}
	}
}
