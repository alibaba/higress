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
	"reflect"
	"testing"
	"time"

	"github.com/alibaba/higress/pkg/config"
	"github.com/tidwall/gjson"
	"k8s.io/apimachinery/pkg/util/wait"
)

type CheckType string

const (
	// CheckTypeMatch checks if the actual value matches the expected value.
	CheckTypeMatch CheckType = "match"
	// CheckTypeExist checks if the actual value exists.
	CheckTypeExist CheckType = "exist"
	// CheckTypeNotExist checks if the actual value does not exist.
	CheckTypeNotExist CheckType = "notexist"
)

// Assertion defines the assertion to be made on the Envoy config.
// TODO: It can support localization judgment so that this configuration check function will be more universal.
// TODO: Can be used for general e2e tests, rather than just envoy filter scenarios.
type Assertion struct {
	// Path is the path of gjson to the value to be asserted.
	Path string
	// CheckType is the type of assertion to be made.
	CheckType CheckType
	// ExpectEnvoyConfig is the expected value of the Envoy config.
	ExpectEnvoyConfig map[string]interface{}
	// TargetNamespace is the namespace of the Envoy pod.
	TargetNamespace string
}

// AssertEnvoyConfig asserts the Envoy config.
func AssertEnvoyConfig(t *testing.T, expected Assertion) error {
	options := config.NewDefaultGetEnvoyConfigOptions()
	options.PodNamespace = expected.TargetNamespace

	switch expected.CheckType {
	case CheckTypeMatch:
		tryNumber := 1
		return wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
			t.Logf("🔍 Checking Envoy config for the %d time, type: %s", tryNumber, expected.CheckType)
			tryNumber++
			out, err := config.GetEnvoyConfig(options)
			if err != nil {
				return false, nil
			}
			allEnvoyConfig := string(out)
			err = assertEnvoyConfigMatch(t, allEnvoyConfig, expected)
			if err != nil {
				return false, nil
			}
			return true, nil
		})
	case CheckTypeExist:
		tryNumber := 1
		return wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
			t.Logf("🔍 Checking Envoy config for the %d time, type: %s", tryNumber, expected.CheckType)
			tryNumber++
			out, err := config.GetEnvoyConfig(options)
			if err != nil {
				return false, nil
			}
			allEnvoyConfig := string(out)
			err = assertEnvoyConfigExist(t, allEnvoyConfig, expected)
			if err != nil {
				return false, nil
			}
			return true, nil
		})
	case CheckTypeNotExist:
		tryNumber := 1
		return wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
			t.Logf("🔍 Checking Envoy config for the %d time, type: %s", tryNumber, expected.CheckType)
			tryNumber++
			out, err := config.GetEnvoyConfig(options)
			if err != nil {
				return false, nil
			}
			allEnvoyConfig := string(out)
			err = assertEnvoyConfigNotExist(t, allEnvoyConfig, expected)
			if err != nil {
				return false, nil
			}
			return true, nil
		})
	default:
		return fmt.Errorf("Unknown check type '%s'", expected.CheckType)
	}
}

// assertEnvoyConfigNotExist asserts the Envoy config does not exist.
func assertEnvoyConfigNotExist(t *testing.T, envoyConfig string, expected Assertion) error {
	result := gjson.Get(envoyConfig, expected.Path).Value()
	if result == nil {
		return nil
	}
	if !findMustNotExist(t, result, expected.ExpectEnvoyConfig) {
		return fmt.Errorf("the expected value %s exists in path '%s'", expected.ExpectEnvoyConfig, expected.Path)
	}
	return nil
}

// assertEnvoyConfigExist asserts the Envoy config exists.
func assertEnvoyConfigExist(t *testing.T, envoyConfig string, expected Assertion) error {
	result := gjson.Get(envoyConfig, expected.Path).Value()
	if result == nil {
		return fmt.Errorf("failed to get value from path '%s'", expected.Path)
	}
	if !findMustExist(t, result, expected.ExpectEnvoyConfig) {
		return fmt.Errorf("the expected value %s does not exist in path '%s'", expected.ExpectEnvoyConfig, expected.Path)
	}
	return nil
}

// assertEnvoyConfigMatch asserts the Envoy config matches the expected value.
func assertEnvoyConfigMatch(t *testing.T, envoyConfig string, expected Assertion) error {
	result := gjson.Get(envoyConfig, expected.Path).Value()
	if result == nil {
		return fmt.Errorf("failed to get value from path '%s'", expected.Path)
	}
	if !match(t, result, expected.ExpectEnvoyConfig) {
		return fmt.Errorf("failed to match value from path '%s'", expected.Path)
	}
	t.Logf("✅ Matched value %s in path '%s'", expected.ExpectEnvoyConfig, expected.Path)
	return nil
}

// match
// 1. interface{} is a slice: if one of the slice elements matches, the assertion passes
// Notice: can recursively find slices
// 2. interface{} is a map: if all the map elements match, the assertion passes
// 3. interface{} is a field: if the field matches, the assertion passes
func match(t *testing.T, actual interface{}, expected map[string]interface{}) bool {
	reflectValue := reflect.ValueOf(actual)
	kind := reflectValue.Kind()
	switch kind {
	case reflect.Slice:
		actualValueSlice := actual.([]interface{})
		for _, v := range actualValueSlice {
			if match(t, v, expected) {
				return true
			}
		}
		return false
	case reflect.Map:
		actualValueMap := actual.(map[string]interface{})
		for key, expectValue := range expected {
			actualValue, ok := actualValueMap[key]
			if !ok {
				return false
			}
			if !reflect.DeepEqual(actualValue, expectValue) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(actual, expected)
	}
}

// findMustExist finds the value of the given path in the given Envoy config.
func findMustExist(t *testing.T, actual interface{}, expected map[string]interface{}) bool {
	for key, expectValue := range expected {
		// If the key does not exist, the assertion fails.
		t.Logf("🔍 Finding key %s", key)
		findKey(actual, key, expectValue)
		if !flag {
			t.Logf("❌ Not found key %s", key)
			return false
		}
		flag = false
		t.Logf("✅ Found key %s", key)
	}
	return true
}

// findMustNotExist finds the value of the given path in the given Envoy config.
func findMustNotExist(t *testing.T, actual interface{}, expected map[string]interface{}) bool {
	for key, expectValue := range expected {
		// If the key exists, the assertion fails.
		t.Logf("🔍 Finding key %s", key)
		findKey(actual, key, expectValue)
		if flag {
			t.Logf("❌ Found key %s", key)
			flag = false
			return false
		}
		t.Logf("✅ Not found key %s", key)
	}
	return true
}

var flag = false

// findKey finds the value of the given key in the given Envoy config.
func findKey(actual interface{}, key string, expectValue interface{}) {
	if flag {
		return
	}
	reflectValue := reflect.ValueOf(actual)
	kind := reflectValue.Kind()
	switch kind {
	case reflect.Slice:
		actualValueSlice := actual.([]interface{})
		for _, v := range actualValueSlice {
			findKey(v, key, expectValue)
		}
	case reflect.Map:
		actualValueMap := actual.(map[string]interface{})
		for actualKey, actualValue := range actualValueMap {
			if actualKey == key && reflect.DeepEqual(convertType(actualValue, expectValue), expectValue) {
				flag = true
			}
			findKey(actualValue, key, expectValue)
		}
	default:
		if reflectValue.String() == key && reflect.DeepEqual(convertType(actual, expectValue), expectValue) {
			flag = true
		}
	}
}

func convertType(value interface{}, targetType interface{}) interface{} {
	targetTypeValue := reflect.ValueOf(targetType)
	targetTypeKind := targetTypeValue.Kind()

	switch targetTypeKind {
	case reflect.Int:
		switch value.(type) {
		case int:
			return value
		case float64:
			return int(value.(float64))
		}
	case reflect.Float64:
		switch value.(type) {
		case int:
			return float64(value.(int))
		case float64:
			return value
		}
	}

	return value
}
