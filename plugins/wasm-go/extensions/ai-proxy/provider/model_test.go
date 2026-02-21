package provider

import (
	"reflect"
	"testing"
)

func TestCleanFunctionParameters(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "remove $schema at root level",
			input: map[string]interface{}{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"type":    "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city and state",
					},
				},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city and state",
					},
				},
			},
		},
		{
			name: "remove multiple unsupported fields",
			input: map[string]interface{}{
				"$schema":     "http://json-schema.org/draft-07/schema#",
				"$id":         "https://example.com/schema",
				"$comment":    "This is a comment",
				"definitions": map[string]interface{}{},
				"type":        "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
		{
			name: "nested $schema in properties",
			input: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"nested": map[string]interface{}{
						"$schema": "should be removed",
						"type":    "object",
						"properties": map[string]interface{}{
							"field": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"nested": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"field": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
			},
		},
		{
			name: "array with map elements",
			input: map[string]interface{}{
				"type": "array",
				"items": []interface{}{
					map[string]interface{}{
						"$schema": "should be removed",
						"type":    "string",
					},
					map[string]interface{}{
						"type": "number",
					},
				},
			},
			expected: map[string]interface{}{
				"type": "array",
				"items": []interface{}{
					map[string]interface{}{
						"type": "string",
					},
					map[string]interface{}{
						"type": "number",
					},
				},
			},
		},
		{
			name: "preserve valid fields",
			input: map[string]interface{}{
				"type":        "object",
				"description": "A valid description",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city",
						"enum":        []interface{}{"NYC", "LA", "SF"},
					},
				},
				"required": []interface{}{"location"},
			},
			expected: map[string]interface{}{
				"type":        "object",
				"description": "A valid description",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city",
						"enum":        []interface{}{"NYC", "LA", "SF"},
					},
				},
				"required": []interface{}{"location"},
			},
		},
		{
			name: "remove $defs field",
			input: map[string]interface{}{
				"$defs": map[string]interface{}{
					"Address": map[string]interface{}{
						"type": "object",
					},
				},
				"type": "object",
			},
			expected: map[string]interface{}{
				"type": "object",
			},
		},
		{
			name: "remove ref field without dollar sign",
			input: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"options": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"ref":  "QuestionOption",
							"type": "object",
							"properties": map[string]interface{}{
								"label": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"options": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"label": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "real world question tool schema",
			input: map[string]interface{}{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type":    "object",
				"properties": map[string]interface{}{
					"questions": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"options": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"ref":  "QuestionOption",
										"type": "object",
										"properties": map[string]interface{}{
											"label": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"questions": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"options": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"label": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanFunctionParameters(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("cleanFunctionParameters() = %v, want %v", result, tt.expected)
			}
		})
	}
}
