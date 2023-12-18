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

// StringLengthValidation is the validation for string length.
// validation for string length type, just string
// check if param length is in range
type StringLengthValidation struct {
	Required  bool   `json:"required"`
	Type      string `json:"type"`
	MinLength int    `json:"minLength"`
	MaxLength int    `json:"maxLength"`
}

func (s StringLengthValidation) Validation(schema map[string]interface{}, paramName string) error {
	switch s.Type {
	case "string":
		// check request header or body
		if _, ok := schema[paramName]; !ok {
			// param not exist, but required
			if s.Required {
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
		// check if in range
		if len(paramValue.(string)) < s.MinLength || len(paramValue.(string)) > s.MaxLength {
			return fmt.Errorf("param %s length is not in range", paramName)
		}
		return nil
	default:
		return fmt.Errorf("param %s is not string", paramName)
	}
}
