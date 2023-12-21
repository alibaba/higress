/*
Copyright 2022 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package envoy

import (
	"testing"
)

func Test_match(t *testing.T) {
	testCases := []struct {
		name         string
		actual       interface{}
		expected     map[string]interface{}
		expectResult bool
	}{
		{
			name: "case 1",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				map[string]interface{}{
					"foo": "baz",
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 2",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				map[string]interface{}{
					"foo": "baz",
				},
			},
			expected: map[string]interface{}{
				"foo": "bay",
			},
			expectResult: false,
		},
		{
			name: "case 3",
			actual: map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
				"baz": "bay",
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 4",
			actual: map[string]interface{}{
				"foo": "bar",
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 5",
			actual: map[string]interface{}{
				"foo": "bar",
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: false,
		},
		{
			name: "case 6",
			actual: []interface{}{
				[]interface{}{
					map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 7",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					map[string]interface{}{
						"foo": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := match(t, testCase.actual, testCase.expected)
			if result != testCase.expectResult {
				t.Errorf("expected %v, got %v", testCase.expectResult, result)
			}
		})
	}
}

func Test_findMustExist(t *testing.T) {
	testCases := []struct {
		name         string
		actual       interface{}
		expected     map[string]interface{}
		expectResult bool
	}{
		{
			name: "case 1",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 2",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: false,
		},
		{
			name: "case 3",
			actual: map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
				"baz": "bay",
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 4",
			actual: map[string]interface{}{
				"foo": "bar",
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: false,
		},
		{
			name: "case 5",
			actual: []interface{}{
				[]interface{}{
					map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: true,
		},
		{
			name: "case 6",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					map[string]interface{}{
						"foo": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: true,
		},
		{
			name: "case 7",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					map[string]interface{}{
						"test": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo":  "bar",
				"test": "baz",
			},
			expectResult: true,
		},
		{
			name: "case 8",
			actual: []interface{}{
				map[string]interface{}{
					"foo":  "bar",
					"test": "baz",
				},
				[]interface{}{
					map[string]interface{}{
						"foo": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo":  "baz",
				"test": "baz",
			},
			expectResult: true,
		},
		{
			name: "case 9",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					[]interface{}{
						map[string]interface{}{
							"foo": "baz",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: true,
		},
		{
			name: "case 9",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					[]interface{}{
						map[string]interface{}{
							"foo": "baz",
						},
					},
				},
				map[string]interface{}{
					"content": []interface{}{
						"one",
						"two",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
				"content": []interface{}{
					"one",
					"two",
				},
			},
			expectResult: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := findMustExist(t, testCase.actual, testCase.expected)
			if result != testCase.expectResult {
				t.Errorf("expected %v, got %v", testCase.expectResult, result)
			}
		})
	}
}

func Test_findMustNotExist(t *testing.T) {
	testCases := []struct {
		name         string
		actual       interface{}
		expected     map[string]interface{}
		expectResult bool
	}{
		{
			name: "case 1",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: false,
		},
		{
			name: "case 2",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: true,
		},
		{
			name: "case 3",
			actual: map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
				"baz": "bay",
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: false,
		},
		{
			name: "case 4",
			actual: map[string]interface{}{
				"foo": "bar",
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: true,
		},
		{
			name: "case 5",
			actual: []interface{}{
				[]interface{}{
					map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: false,
		},
		{
			name: "case 6",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					map[string]interface{}{
						"foo": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: false,
		},
		{
			name: "case 7",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					map[string]interface{}{
						"test": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "bar",
			},
			expectResult: false,
		},
		{
			name: "case 8",
			actual: []interface{}{
				map[string]interface{}{
					"foo":  "bar",
					"test": "baz",
				},
				[]interface{}{
					map[string]interface{}{
						"foo": "baz",
					},
				},
			},
			expected: map[string]interface{}{
				"foo":  "baz",
				"test": "baz",
			},
			expectResult: false,
		},
		{
			name: "case 9",
			actual: []interface{}{
				map[string]interface{}{
					"foo": "bar",
				},
				[]interface{}{
					[]interface{}{
						map[string]interface{}{
							"foo": "baz",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"foo": "baz",
			},
			expectResult: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := findMustNotExist(t, testCase.actual, testCase.expected)
			if result != testCase.expectResult {
				t.Errorf("expected %v, got %v", testCase.expectResult, result)
			}
		})
	}
}
