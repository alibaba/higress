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

package config

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	. "github.com/alibaba/higress/pkg/ingress/log"
	"istio.io/istio/pkg/config"
)

// TemplateProcessor handles template substitution in configs
type TemplateProcessor struct {
	// getValue is a function that retrieves values by type, namespace, name and key
	getValue  func(valueType, namespace, name, key string) (string, error)
	namespace string
}

// NewTemplateProcessor creates a new TemplateProcessor with the given value getter function
func NewTemplateProcessor(getValue func(valueType, namespace, name, key string) (string, error), namespace string) *TemplateProcessor {
	return &TemplateProcessor{
		getValue:  getValue,
		namespace: namespace,
	}
}

// ProcessConfig processes a config and substitutes any template variables
func (p *TemplateProcessor) ProcessConfig(cfg *config.Config) error {
	// Convert spec to JSON string to process substitutions
	jsonBytes, err := json.Marshal(cfg.Spec)
	if err != nil {
		return fmt.Errorf("failed to marshal config spec: %v", err)
	}

	configStr := string(jsonBytes)
	// Find all value references in format:
	// ${type/name.key} or ${type.namespace/name.key}
	valueRegex := regexp.MustCompile(`\$\{([^./}]+)(?:\.([^/]+))?/([^.}]+)\.([^}]+)\}`)
	matches := valueRegex.FindAllStringSubmatch(configStr, -1)
	// If there are no value references, return immediately
	if len(matches) == 0 {
		return nil
	}
	IngressLog.Infof("start to handle name:%s found %d variabes", cfg.Meta.Name, len(matches))
	for _, match := range matches {
		valueType := match[1]
		var namespace, name, key string
		if match[2] != "" {
			// Format: ${type.namespace/name.key}
			namespace = match[2]
		} else {
			// Format: ${type/name.key} - use default namespace
			namespace = p.namespace
		}
		name = match[3]
		key = match[4]

		// Get value using the provided getter function
		value, err := p.getValue(valueType, namespace, name, key)
		if err != nil {
			return fmt.Errorf("failed to get %s value for %s/%s.%s: %v", valueType, namespace, name, key, err)
		}

		// Replace placeholder with actual value
		configStr = strings.Replace(configStr, match[0], value, 1)
	}
	// Unmarshal back to config spec
	if err := json.Unmarshal([]byte(configStr), &cfg.Spec); err != nil {
		return fmt.Errorf("failed to unmarshal substituted config: %v", err)
	}
	return nil
}
