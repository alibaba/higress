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
	"fmt"
	"testing"

	"github.com/alibaba/higress/pkg/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type Assertion struct {
	Path                     string
	ExceptContainEnvoyConfig map[string]interface{}
	TargetNamespace          string
}

func AssertEnvoyConfig(t *testing.T, expected Assertion) error {
	options := config.NewDefaultGetEnvoyConfigOptions()
	options.PodNamespace = expected.TargetNamespace

	out, err := config.GetEnvoyConfig(options)
	if err != nil {
		return err
	}
	allEnvoyConfig := string(out)

	result := gjson.Get(allEnvoyConfig, expected.Path)
	actualValue := result.Value()
	if actualValue == nil {
		if expected.ExceptContainEnvoyConfig == nil {
			return nil
		} else {
			return fmt.Errorf("Key '%s' not found in actual config", expected.Path)
		}
	}

	result.ForEach(func(key, value gjson.Result) bool {
		err = compareValues(value.Value(), expected.ExceptContainEnvoyConfig)
		require.NoError(t, err)
		return true
	})
	return nil
}

func compareValues(actual interface{}, expected map[string]interface{}) error {
	for key, expectedValue := range expected {
		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			return fmt.Errorf("Expected type map[string]interface{} for key '%s'", key)
		}
		actualValue := actualMap[key]
		if actualValue == nil {
			return fmt.Errorf("Key '%s' not found in actual config", key)
		}
		switch v := expectedValue.(type) {
		case map[string]interface{}:
			err := compareValues(actualValue, v)
			if err != nil {
				return err
			}
		default:
			if actualValue != expectedValue {
				return fmt.Errorf("Value mismatch for key '%s'. Expected '%v', but got '%v'", key, expectedValue, actualValue)
			}
		}
	}
	return nil
}
