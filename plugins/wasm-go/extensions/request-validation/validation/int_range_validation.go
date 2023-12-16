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

// IntRangeValidation is the validation for int range.
// validation for int range type, including string, integer, float
// check if param is in range
type IntRangeValidation struct {
	Required bool
	Type     string
	Min      interface{}
	Max      interface{}
}

func (i IntRangeValidation) Validation(schema map[string]interface{}, paramName string) error {
	switch i.Type {
	case "integer":
		// check Min and Max
		if _, ok := i.Min.(int); !ok {
			return fmt.Errorf("min is not int")
		}
		if _, ok := i.Max.(int); !ok {
			return fmt.Errorf("max is not int")
		}
		// check request header or body
		if _, ok := schema[paramName]; !ok {
			// param not exist, but required
			if i.Required {
				return fmt.Errorf("param %s is required", paramName)
			}
			// param not exist, but not required, return nil
			return nil
		}
		paramValue := schema[paramName]
		// param exist, but not int
		if _, ok := paramValue.(int); !ok {
			return fmt.Errorf("param %s is not int", paramName)
		}
		// check if in range
		if paramValue.(int) < i.Min.(int) || paramValue.(int) > i.Max.(int) {
			return fmt.Errorf("param %s is not in range", paramName)
		}
		return nil
	case "float":
		// check Min and Max
		if _, ok := i.Min.(float64); !ok {
			return fmt.Errorf("min is not float")
		}
		if _, ok := i.Max.(float64); !ok {
			return fmt.Errorf("max is not float")
		}
		// check request header or body
		if _, ok := schema[paramName]; !ok {
			// param not exist, but required
			if i.Required {
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
		// check if in range
		if paramValue.(float64) < i.Min.(float64) || paramValue.(float64) > i.Max.(float64) {
			return fmt.Errorf("param %s is not in range", paramName)
		}
		return nil
	case "string":
		// check Min and Max
		if _, ok := i.Min.(string); !ok {
			return fmt.Errorf("min is not string")
		}
		if _, ok := i.Max.(string); !ok {
			return fmt.Errorf("max is not string")
		}
		// check request header or body
		if _, ok := schema[paramName]; !ok {
			// param not exist, but required
			if i.Required {
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
		if paramValue.(string) < i.Min.(string) || paramValue.(string) > i.Max.(string) {
			return fmt.Errorf("param %s is not in range", paramName)
		}
		return nil
	default:
		return fmt.Errorf("type %s is not supported", i.Type)
	}
}
