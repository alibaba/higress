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

	"istio.io/istio/pkg/config"
	v1 "k8s.io/api/core/v1"
)

// TemplateProcessor handles template substitution in configs
type TemplateProcessor struct {
	// getSecret is a function that retrieves secrets by name and namespace
	getSecret func(namespace, name string) (*v1.Secret, error)
	namespace string
}

// NewTemplateProcessor creates a new TemplateProcessor with the given secret getter function
func NewTemplateProcessor(getSecret func(namespace, name string) (*v1.Secret, error), namespace string) *TemplateProcessor {
	return &TemplateProcessor{
		getSecret: getSecret,
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

	// Find all secret references in format:
	// ${secret/name.key.field} or ${secret.namespace/name.key.field}
	secretRegex := regexp.MustCompile(`\$\{secret(?:\.([^/]+))?/([^.}]+)\.([^}]+)\}`)
	matches := secretRegex.FindAllStringSubmatch(configStr, -1)

	// If there are no secret references, return immediately
	if len(matches) == 0 {
		return nil
	}

	for _, match := range matches {
		var secretNamespace, secretName, secretKey string
		if match[1] != "" {
			// Format: ${secret.namespace/name.field}
			secretNamespace = match[1]
		} else {
			// Format: ${secret/name.field} - use higress-system namespace
			secretNamespace = p.namespace
		}
		secretName = match[2]
		secretKey = match[3]

		// Get secret using the provided getter function
		secret, err := p.getSecret(secretNamespace, secretName)
		if err != nil {
			return fmt.Errorf("failed to get secret %s/%s: %v", secretNamespace, secretName, err)
		}

		// Get value from secret
		data, exists := secret.Data[secretKey]
		if !exists {
			return fmt.Errorf("key %s not found in secret %s/%s", secretKey, secretNamespace, secretName)
		}
		secretValue := string(data)
		// Replace placeholder with actual value
		configStr = strings.Replace(configStr, match[0], secretValue, 1)
	}

	// Unmarshal back to config spec
	if err := json.Unmarshal([]byte(configStr), &cfg.Spec); err != nil {
		return fmt.Errorf("failed to unmarshal substituted config: %v", err)
	}
	return nil
}
