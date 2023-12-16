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

import (
	"fmt"
	"regexp"
)

// RegexValidation is the validation for regex.
// validation for regex type, just string
// check if param is match regex
type RegexValidation struct {
	Required  bool
	Type      string
	MinLength int
	MaxLength int
	Pattern   string
}

func (r RegexValidation) Validation(schema map[string]interface{}, paramName string) error {
	switch r.Type {
	case "string":
		// check request header or body
		if _, ok := schema[paramName]; !ok {
			// param not exist, but required
			if r.Required {
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
		if len(paramValue.(string)) < r.MinLength || len(paramValue.(string)) > r.MaxLength {
			return fmt.Errorf("param %s length is not in range", paramName)
		}
		// check regex
		if !matchRegex(paramValue.(string), r.Pattern) {
			return fmt.Errorf("param %s is not match regex", paramName)
		}
		return nil
	default:
		return fmt.Errorf("param %s is not string", paramName)
	}
}

func matchRegex(s string, regex string) bool {
	matched, err := regexp.Match(regex, []byte(s))
	if err != nil {
		return false
	}
	return matched
}
