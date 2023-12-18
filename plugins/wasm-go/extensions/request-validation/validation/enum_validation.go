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

// EnumValidation is the validation for enum.
// validation for enum type, including string, integer, float
// check if param is in enum
type EnumValidation struct {
	Required bool          `json:"required"`
	Type     string        `json:"type"`
	Enum     []interface{} `json:"enum"`
}

func (e EnumValidation) Validation(schema map[string]interface{}, paramName string) error {
	switch e.Type {
	case "string":
		// check enum
		enum := make([]string, len(e.Enum))
		for i, v := range e.Enum {
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("enum value is not string")
			}
			enum[i] = s
		}
		// check request header or body
		if _, ok := schema[paramName]; !ok {
			// param not exist, but required
			if e.Required {
				return fmt.Errorf("param %s is required", paramName)
			}
			// param not exist, but not required, return nil
			return nil
		}
		paramValue := schema[paramName]
		// param exist, but not string
		if _, ok := paramValue.(string); !ok {
			return fmt.Errorf("param %s is not string", paramName)
		}
		// param exist, but not in enum
		for _, v := range enum {
			if paramValue.(string) == v {
				return nil
			}
		}
		return fmt.Errorf("param %s is not in enum", paramName)
	case "integer":
		// check enum
		enum := make([]int, len(e.Enum))
		for i, v := range e.Enum {
			s, ok := v.(int)
			if !ok {
				return fmt.Errorf("enum value is not integer")
			}
			enum[i] = s
		}
		// check request header or body
		if _, ok := schema[paramName]; !ok {
			// param not exist, but required
			if e.Required {
				return fmt.Errorf("param %s is required", paramName)
			}
			// param not exist, but not required, return nil
			return nil
		}
		paramValue := schema[paramName]
		// param exist, but not integer
		if _, ok := paramValue.(int); !ok {
			return fmt.Errorf("param %s is not integer", paramName)
		}
		// param exist, but not in enum
		for _, v := range enum {
			if paramValue.(int) == v {
				return nil
			}
		}
		return fmt.Errorf("param %s is not in enum", paramName)
	case "float":
		// check enum
		enum := make([]float64, len(e.Enum))
		for i, v := range e.Enum {
			s, ok := v.(float64)
			if !ok {
				return fmt.Errorf("enum value is not float")
			}
			enum[i] = s
		}
		// check request header or body
		if _, ok := schema[paramName]; !ok {
			// param not exist, but required
			if e.Required {
				return fmt.Errorf("param %s is required", paramName)
			}
			// param not exist, but not required, return nil
			return nil
		}
		paramValue := schema[paramName]
		// param exist, but not float
		if _, ok := paramValue.(float64); !ok {
			return fmt.Errorf("param %s is not float", paramName)
		}
		// param exist, but not in enum
		for _, v := range enum {
			if paramValue.(float64) == v {
				return nil
			}
		}
		return fmt.Errorf("param %s is not in enum", paramName)
	default:
		return fmt.Errorf("enum type %s is not supported", e.Type)
	}
}
