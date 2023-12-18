// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation

import "fmt"

// ArrayValidation is the validation for array.
// validation for array type, including string, integer, float
// check if array length is greater than MinLength
// check if array element is in range
// check if array element is unique
type ArrayValidation struct {
	Type      string      `json:"type"`
	Required  bool        `json:"required"`
	MinLength int         `json:"minLength"`
	Unique    bool        `json:"uniqueItems"`
	MinValue  interface{} `json:"minItems"`
	MaxValue  interface{} `json:"maxItems"`
}

func (a ArrayValidation) Validation(schema map[string]interface{}, paramName string) error {
	switch a.Type {
	case "string":
		// check MinValue and MaxValue
		if _, ok := a.MinValue.(string); !ok {
			return fmt.Errorf("minValue is not string")
		}
		if _, ok := a.MaxValue.(string); !ok {
			return fmt.Errorf("maxValue is not string")
		}
		// check request header or body
		if _, ok := schema[paramName]; !ok {
			// param not exist, but required
			if a.Required {
				return fmt.Errorf("param %s is required", paramName)
			}
			// param not exist, but not required, return nil
			return nil
		}
		paramValue := schema[paramName]
		// param exist, but not []string
		if _, ok := paramValue.([]string); !ok {
			return fmt.Errorf("param %s is not a string array", paramName)
		}
		// check MinLength
		if len(paramValue.([]string)) < a.MinLength {
			return fmt.Errorf("param %s array length should be greater than %d", paramName, a.MinLength)
		}
		// check if in range
		for _, v := range paramValue.([]string) {
			if v < a.MinValue.(string) || v > a.MaxValue.(string) {
				return fmt.Errorf("param %s is not in range", paramName)
			}
		}
		// check if unique
		if a.Unique {
			set := make(map[string]bool)
			for _, v := range paramValue.([]string) {
				if _, ok := set[v]; ok {
					return fmt.Errorf("param %s is not unique", paramName)
				}
				set[v] = true
			}
		}
		return nil
	case "integer":
		// check MinValue and MaxValue
		if _, ok := a.MinValue.(int); !ok {
			return fmt.Errorf("minValue is not integer")
		}
		if _, ok := a.MaxValue.(int); !ok {
			return fmt.Errorf("maxValue is not integer")
		}
		// check request header or body
		if _, ok := schema[paramName]; !ok {
			// param not exist, but required
			if a.Required {
				return fmt.Errorf("param %s is required", paramName)
			}
			// param not exist, but not required, return nil
			return nil
		}
		paramValue := schema[paramName]
		// param exist, but not []int
		if _, ok := paramValue.([]int); !ok {
			return fmt.Errorf("param %s is not an integer array", paramName)
		}
		// check MinLength
		if len(paramValue.([]int)) < a.MinLength {
			return fmt.Errorf("param %s array length should be greater than %d", paramName, a.MinLength)
		}
		// check if in range
		for _, v := range paramValue.([]int) {
			if v < a.MinValue.(int) || v > a.MaxValue.(int) {
				return fmt.Errorf("param %s is not in range", paramName)
			}
		}
		// check if unique
		if a.Unique {
			set := make(map[int]bool)
			for _, v := range paramValue.([]int) {
				if _, ok := set[v]; ok {
					return fmt.Errorf("param %s is not unique", paramName)
				}
				set[v] = true
			}
		}
		return nil
	case "float":
		// check MinValue and MaxValue
		if _, ok := a.MinValue.(float64); !ok {
			return fmt.Errorf("minValue is not float")
		}
		if _, ok := a.MaxValue.(float64); !ok {
			return fmt.Errorf("maxValue is not float")
		}
		// check request header or body
		if _, ok := schema[paramName]; !ok {
			// param not exist, but required
			if a.Required {
				return fmt.Errorf("param %s is required", paramName)
			}
			// param not exist, but not required, return nil
			return nil
		}
		paramValue := schema[paramName]
		// param exist, but not []float64
		if _, ok := paramValue.([]float64); !ok {
			return fmt.Errorf("param %s is not a float array", paramName)
		}
		// check MinLength
		if len(paramValue.([]float64)) < a.MinLength {
			return fmt.Errorf("param %s array length should be greater than %d", paramName, a.MinLength)
		}
		// check if in range
		for _, v := range paramValue.([]float64) {
			if v < a.MinValue.(float64) || v > a.MaxValue.(float64) {
				return fmt.Errorf("param %s is not in range", paramName)
			}
		}
		// check if unique
		if a.Unique {
			set := make(map[float64]bool)
			for _, v := range paramValue.([]float64) {
				if _, ok := set[v]; ok {
					return fmt.Errorf("param %s is not unique", paramName)
				}
				set[v] = true
			}
		}
		return nil
	default:
		return fmt.Errorf("array type %s is not supported", a.Type)
	}
}
