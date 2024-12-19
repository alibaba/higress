package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertHttpHeadersToStruct(t *testing.T) {
	cases := []struct {
		name        string
		httpHeaders map[string][]string
		expected    [][2]string
	}{
		{
			name:        "empty header",
			httpHeaders: map[string][]string{},
			expected:    [][2]string{},
		},
		{
			name: "headers with content type",
			httpHeaders: map[string][]string{
				"Content-Type": []string{"application/json", "application/xml"},
			},
			expected: [][2]string{
				{"Content-Type", "application/json"},
				{"Content-Type", "application/xml"},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := convertHttpHeadersToStruct(c.httpHeaders)
			require.Equal(t, c.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	cases := []struct {
		name     string
		array    []int
		value    int
		expected bool
	}{
		{
			name:     "multi value, contains",
			array:    []int{404, 304},
			value:    404,
			expected: true,
		},
		{
			name:     "multi value, no contains",
			array:    []int{404, 304},
			value:    200,
			expected: false,
		},
		{
			name:     "one value contains",
			array:    []int{200},
			value:    200,
			expected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := contains(c.array, c.value)
			require.Equal(t, c.expected, result)
		})
	}
}
