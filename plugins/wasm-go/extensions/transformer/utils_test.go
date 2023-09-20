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

package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

var jsonData = `
{
  "number-1": 20.5,
  "number-2": 10,
  "boolean": true,
  "string": "hello",
  "dot.in.keys": true,
  "array": ["1", "2", "3", "3", "2", "1"],
  "unique-array": [1, 2, 3],
  "empty-array-1": [],
  "empty-array-2": [],
  "object": {
    "first": "a",
    "second": "b",
    "sub-object": {
      "array": [1, 2, 3],
      "string": "world",
      "dot.in.keys": false
    }
  },
  "object-array": [
    { "name": "zs", "age": 17 },
    { "name": "ls", "age": 18 },
    { "name": "tw", "age": 18}
  ]
}
`

func TestLookup(t *testing.T) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	require.NoError(t, err)

	cases := []struct {
		name       string
		key        string
		dotsInKeys bool
		expected   interface{}
		errMsg     string
	}{
		{
			name:     "common",
			key:      "number-1",
			expected: 20.5,
		},
		{
			name:   "key does not exist",
			key:    "not-exist",
			errMsg: errKeyNotFound.Error(),
		},
		{
			name:     "char in string",
			key:      "string.2",
			expected: string("hello")[2],
		},
		{
			name:       "dot in keys: true",
			key:        "dot.in.keys",
			dotsInKeys: true,
			expected:   true,
		},
		{
			name:       "dot in keys: false",
			key:        "object.sub-object.dot.in.keys",
			dotsInKeys: false,
			errMsg:     errKeyNotFound.Error(),
		},
		{
			name: "object",
			key:  "object",
			expected: map[string]interface{}{
				"first":  "a",
				"second": "b",
				"sub-object": map[string]interface{}{
					"array":       []interface{}{1.0, 2.0, 3.0},
					"string":      "world",
					"dot.in.keys": false,
				},
			},
		},
		{
			name:     "nested object",
			key:      "object.sub-object.array.1",
			expected: 2.0,
		},
		{
			name:     "object array",
			key:      "object-array.1.name",
			expected: "ls",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual, _, err := lookup(data, c.dotsInKeys, c.key)
			if c.errMsg != "" {
				require.EqualError(t, err, c.errMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
}

func TestRemove(t *testing.T) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	require.NoError(t, err)

	cases := []struct {
		name         string
		key          string
		dotsInKeys   bool
		expected     interface{}
		removeErrMsg string
		lookErrMsg   string
	}{
		{
			name:       "common",
			key:        "number-1",
			lookErrMsg: errKeyNotFound.Error(),
		},
		{
			name:       "key does not exist",
			key:        "not-exist",
			lookErrMsg: errKeyNotFound.Error(),
		},
		{
			name:       "dot in keys: true",
			key:        "dot.in.keys",
			dotsInKeys: true,
			lookErrMsg: errKeyNotFound.Error(),
		},
		{
			name:       "dot in keys: false",
			key:        "object.sub-object.dot.in.keys",
			dotsInKeys: false,
			lookErrMsg: errKeyNotFound.Error(),
		},
		{
			name:         "nested object",
			key:          "object.sub-object.array.1",
			expected:     2.0,
			removeErrMsg: fmt.Sprintf(errInvalidFieldTypeFmt, "slice"),
		},
		{
			name:       "object array",
			key:        "object-array.1.name",
			lookErrMsg: errKeyNotFound.Error(),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err = remove(data, c.dotsInKeys, c.key)
			if c.removeErrMsg != "" {
				require.EqualError(t, err, c.removeErrMsg)
				return
			}
			require.NoError(t, err)

			actual, _, err := lookup(data, c.dotsInKeys, c.key)
			if c.lookErrMsg != "" {
				require.EqualError(t, err, c.lookErrMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
	//fmt.Println(data)
}

func TestSet(t *testing.T) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	require.NoError(t, err)

	cases := []struct {
		name         string
		key          string
		value        interface{}
		dotsInKeys   bool
		expected     interface{}
		setErrMsg    string
		lookupErrMsg string
	}{
		{
			name:     "common",
			key:      "new-key",
			value:    "new-value",
			expected: "new-value",
		},
		{
			name:     "overwrite an existing kv",
			key:      "array",
			value:    []interface{}{9, 8, 7},
			expected: []interface{}{9, 8, 7},
		},
		{
			name:       "dot in keys",
			key:        "dot.in.keys",
			value:      "true",
			dotsInKeys: true,
			expected:   "true",
		},
		{
			name: "nested object",
			key:  "object.sub-object",
			value: map[string]interface{}{
				"a": 1,
				"b": true,
				"c": "c",
				"d": []interface{}{1, 2, 3},
			},
			expected: map[string]interface{}{
				"a": 1,
				"b": true,
				"c": "c",
				"d": []interface{}{1, 2, 3},
			},
		},
		{
			name:     "object array",
			key:      "object-array.0.age",
			value:    24,
			expected: 24,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err = set(data, c.dotsInKeys, c.key, c.value)
			if c.setErrMsg != "" {
				require.EqualError(t, err, c.setErrMsg)
				return
			}
			require.NoError(t, err)

			actual, _, err := lookup(data, c.dotsInKeys, c.key)
			if c.lookupErrMsg != "" {
				require.EqualError(t, err, c.lookupErrMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
	//fmt.Println(data)
}

func TestRename(t *testing.T) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	require.NoError(t, err)

	cases := []struct {
		name         string
		fromKey      string
		toKey        string
		dotsInKeys   bool
		expected     interface{}
		renameErrMsg string
		lookupErrMsg string
	}{
		{
			name:     "common",
			fromKey:  "number-1",
			toKey:    "number-3",
			expected: 20.5,
		},
		{
			name:         "fromKey does not exist",
			fromKey:      "not-exist",
			toKey:        "new-key",
			lookupErrMsg: errKeyNotFound.Error(),
		},
		{
			name:       "dot in keys: true",
			fromKey:    "dot.in.keys",
			toKey:      "dot-in-keys",
			dotsInKeys: true,
			expected:   true,
		},
		{
			name:    "nested object",
			fromKey: "object.sub-object",
			toKey:   "nested-object",
			expected: map[string]interface{}{
				"array":       []interface{}{1.0, 2.0, 3.0},
				"string":      "world",
				"dot.in.keys": false,
			},
		},
		{
			name:     "nested key",
			fromKey:  "number-2",
			toKey:    "one.two.three.number",
			expected: 10.0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err = rename(data, c.dotsInKeys, c.fromKey, c.toKey)
			if c.renameErrMsg != "" {
				require.EqualError(t, err, c.renameErrMsg)
				return
			}
			require.NoError(t, err)

			actual, _, err := lookup(data, c.dotsInKeys, c.toKey)
			if c.lookupErrMsg != "" {
				require.EqualError(t, err, c.lookupErrMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
	//fmt.Println(data)
}

func TestReplace(t *testing.T) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	require.NoError(t, err)

	cases := []struct {
		name          string
		key           string
		newValue      interface{}
		dotsInKeys    bool
		expected      interface{}
		replaceErrMsg string
		lookupErrMsg  string
	}{
		{
			name:     "common",
			key:      "string",
			newValue: "hello world",
			expected: "hello world",
		},
		{
			name:         "key does not exist",
			key:          "not-exist",
			newValue:     "test-value",
			lookupErrMsg: errKeyNotFound.Error(),
		},
		{
			name:       "dot in keys",
			key:        "dot.in.keys",
			newValue:   "false",
			dotsInKeys: true,
			expected:   "false",
		},
		{
			name:     "nested object",
			key:      "object.sub-object",
			newValue: "child",
			expected: "child",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err = replace(data, c.dotsInKeys, c.key, c.newValue)
			if c.replaceErrMsg != "" {
				require.EqualError(t, err, c.replaceErrMsg)
				return
			}
			require.NoError(t, err)

			actual, _, err := lookup(data, c.dotsInKeys, c.key)
			if c.lookupErrMsg != "" {
				require.EqualError(t, err, c.lookupErrMsg)
				return
			}
			//fmt.Println(data)
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
	//fmt.Println(data)
}

func TestAdd(t *testing.T) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	require.NoError(t, err)

	cases := []struct {
		name         string
		key          string
		value        interface{}
		dotsInKeys   bool
		expected     interface{}
		addErrMsg    string
		lookupErrMsg string
	}{
		{
			name:     "common",
			key:      "add-key",
			value:    "add-value",
			expected: "add-value",
		},
		{
			name:       "key already exist",
			key:        "dot.in.keys",
			value:      "false",
			dotsInKeys: true,
			expected:   true,
		},
		{
			name:     "nested object",
			key:      "object.new-key",
			value:    "new-value",
			expected: "new-value",
		},
		{
			name:     "nested object field already exist",
			key:      "object.sub-object.array",
			value:    []interface{}{"a", "b", "c"},
			expected: []interface{}{1.0, 2.0, 3.0},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err = add(data, c.dotsInKeys, c.key, c.value)
			if c.addErrMsg != "" {
				require.EqualError(t, err, c.addErrMsg)
				return
			}
			require.NoError(t, err)

			actual, _, err := lookup(data, c.dotsInKeys, c.key)
			if c.lookupErrMsg != "" {
				require.EqualError(t, err, c.lookupErrMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
	//fmt.Println(data)
}

func TestAppend(t *testing.T) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	require.NoError(t, err)

	cases := []struct {
		name         string
		key          string
		value        interface{}
		dotsInKeys   bool
		expected     interface{}
		appendErrMsg string
		lookupErrMsg string
	}{
		{
			name:     "common",
			key:      "number-1",
			value:    10.0,
			expected: []interface{}{20.5, 10.0},
		},
		{
			name:     "different types",
			key:      "boolean",
			value:    "10",
			expected: true,
		},
		{
			name:     "key does not exist",
			key:      "new-key",
			value:    "new-value",
			expected: "new-value",
		},
		{
			name:     "append one to array",
			key:      "unique-array",
			value:    4.0,
			expected: []interface{}{1.0, 2.0, 3.0, 4.0},
		},
		{
			name:  "append array to array",
			key:   "object.sub-object.array",
			value: []interface{}{10.0, 9.0, 8.0},
			expected: []interface{}{
				1.0, 2.0, 3.0,
				10.0, 9.0, 8.0,
			},
		},
		{
			name:     "append one to empty array",
			key:      "empty-array-1",
			value:    100.0,
			expected: 100.0,
		},
		{
			name:     "append array to empty array",
			key:      "empty-array-2",
			value:    []interface{}{1.1, 2.2, 3.3},
			expected: []interface{}{1.1, 2.2, 3.3},
		},
		{
			name:       "dot in keys",
			key:        "my.dot.in.keys",
			value:      []interface{}{1.0, 2.0, 3.0},
			dotsInKeys: true,
			expected:   []interface{}{1.0, 2.0, 3.0},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err = append_(data, c.dotsInKeys, c.key, c.value)
			if c.appendErrMsg != "" {
				require.EqualError(t, err, c.appendErrMsg)
				return
			}
			require.NoError(t, err)

			actual, _, err := lookup(data, c.dotsInKeys, c.key)
			if c.lookupErrMsg != "" {
				require.EqualError(t, err, c.lookupErrMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
	//fmt.Println(data)
}

func TestMap(t *testing.T) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	require.NoError(t, err)

	cases := []struct {
		name         string
		fromKey      string
		toKey        string
		dotsInKeys   bool
		expected     interface{}
		mapErrMsg    string
		lookupErrMsg string
	}{
		{
			name:     "common",
			fromKey:  "number-1",
			toKey:    "map-number",
			expected: 20.5,
		},
		{
			name:         "from key does not exist",
			fromKey:      "from-key",
			toKey:        "to-key",
			lookupErrMsg: errKeyNotFound.Error(),
		},
		{
			name:     "map array",
			fromKey:  "array",
			toKey:    "map-array",
			expected: []interface{}{"1", "2", "3", "3", "2", "1"},
		},
		{
			name:    "map nested object",
			fromKey: "object.sub-object",
			toKey:   "object.map-sub-object",
			expected: map[string]interface{}{
				"array":       []interface{}{1.0, 2.0, 3.0},
				"string":      "world",
				"dot.in.keys": false,
			},
		},
		{
			name:       "dot in keys",
			fromKey:    "dot.in.keys",
			toKey:      "map-dot-in-keys",
			dotsInKeys: true,
			expected:   true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err = map_(data, c.dotsInKeys, c.fromKey, c.toKey)
			if c.mapErrMsg != "" {
				require.EqualError(t, err, c.mapErrMsg)
				return
			}
			require.NoError(t, err)

			actual, _, err := lookup(data, c.dotsInKeys, c.toKey)
			if c.lookupErrMsg != "" {
				require.EqualError(t, err, c.lookupErrMsg)
				return
			}
			//fmt.Println(data)
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
}

func TestDedupe(t *testing.T) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	require.NoError(t, err)

	cases := []struct {
		name         string
		key          string
		strategy     string
		dotsInKeys   bool
		expected     interface{}
		dedupeErrMsg string
		lookupErrMsg string
	}{
		{
			name:     "retain unique",
			key:      "array",
			strategy: "RETAIN_UNIQUE",
			expected: []interface{}{"1", "2", "3"},
		},
		{
			name:     "retain last",
			key:      "object.sub-object.array",
			strategy: "RETAIN_LAST",
			expected: 3.0,
		},
		{
			name:     "retain first",
			key:      "unique-array",
			strategy: "RETAIN_FIRST",
			expected: 1.0,
		},
		{
			name:         "key does not exist",
			key:          "not-exist",
			lookupErrMsg: errKeyNotFound.Error(),
		},
		{
			name:       "dot in keys",
			key:        "dot.in.keys",
			strategy:   "RETAIN_FIRST",
			dotsInKeys: true,
			expected:   true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err = dedupe(data, c.dotsInKeys, c.key, c.strategy)
			if c.dedupeErrMsg != "" {
				require.EqualError(t, err, c.dedupeErrMsg)
				return
			}
			require.NoError(t, err)

			actual, _, err := lookup(data, c.dotsInKeys, c.key)
			if c.lookupErrMsg != "" {
				require.EqualError(t, err, c.lookupErrMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
	//fmt.Println(data)
}

func TestValTypeIsSame(t *testing.T) {
	cases := []struct {
		name     string
		vala     interface{}
		valb     interface{}
		expected bool
	}{
		{
			name:     "common",
			vala:     "hello world",
			valb:     "ni hao",
			expected: true,
		},
		{
			name:     "different type",
			vala:     "hello world",
			valb:     10,
			expected: false,
		},
		{
			name:     "one interface{}",
			vala:     interface{}("hello world"),
			valb:     "ni hao",
			expected: true,
		},
		{
			name:     "one interface{} & different type",
			vala:     interface{}("hello world"),
			valb:     10,
			expected: false,
		},
		{
			name:     "all interface{}",
			vala:     interface{}("hello world"),
			valb:     interface{}("ni hao"),
			expected: true,
		},
		{
			name:     "all interface{} & different type",
			vala:     interface{}("hello world"),
			valb:     interface{}(10),
			expected: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := valTypeIsSame(reflect.ValueOf(c.vala), reflect.ValueOf(c.valb))
			require.Equal(t, c.expected, actual)
		})
	}
}

func TestSliceElemIsSame(t *testing.T) {
	cases := []struct {
		name     string
		sliceA   interface{}
		sliceB   interface{}
		expected bool
	}{
		{
			name:     "all slice",
			sliceA:   []int{1, 2, 3},
			sliceB:   []int{1, 2, 3, 4, 5},
			expected: true,
		},
		{
			name:     "all array",
			sliceA:   [3]int{1, 2, 3},
			sliceB:   [5]int{1, 2, 3, 4, 5},
			expected: true,
		},
		{
			name:     "slice and array",
			sliceA:   []int{1, 2, 3},
			sliceB:   [5]int{1, 2, 3, 4, 5},
			expected: true,
		},
		{
			name:     "all []interface{}",
			sliceA:   []interface{}{1, 2, 3},
			sliceB:   [5]interface{}{1, 2, 3, 4, 5},
			expected: true,
		},
		{
			name:     "all []interface{} & all empty",
			sliceA:   []interface{}{},
			sliceB:   [5]interface{}{},
			expected: true,
		},
		{name: "all []interface{} & one empty",
			sliceA:   []interface{}{"1", "2", "3"},
			sliceB:   []interface{}{},
			expected: true,
		},
		{
			name:     "one []interface{}",
			sliceA:   []interface{}{1, 2, 3},
			sliceB:   []int{1, 2, 3, 5},
			expected: true,
		},
		{
			name:     "one []interface{} & one empty",
			sliceA:   []interface{}{},
			sliceB:   []string{"1", "2"},
			expected: true,
		},
		{
			name:     "different type",
			sliceA:   []int{1, 2, 3},
			sliceB:   []string{"1", "2", "3"},
			expected: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := sliceElemTypeIsSame(reflect.ValueOf(c.sliceA), reflect.ValueOf(c.sliceB))
			require.Equal(t, c.expected, actual)
		})
	}
}

func TestSliceElemTypeIsSameToVal(t *testing.T) {
	cases := []struct {
		name     string
		slice    interface{}
		val      interface{}
		expected bool
	}{
		{
			name:     "common",
			slice:    []int{1, 2, 3},
			val:      int(1),
			expected: true,
		},
		{
			name:     "slice []interface{}",
			slice:    []interface{}{1, 2, 3},
			val:      int(1),
			expected: true,
		},
		{
			name:     "slice []interface{} & empty",
			slice:    []interface{}{},
			val:      int(1),
			expected: true,
		},
		{
			name:     "slice []interface{} & val interface{}",
			slice:    []interface{}{1, 2, 3},
			val:      interface{}(1),
			expected: true,
		},
		{
			name:     "empty slice []interface{} & val interface{}",
			slice:    []interface{}{},
			val:      interface{}(1),
			expected: true,
		},
		{
			name:     "val interface{}",
			slice:    []int{1, 2, 3},
			val:      interface{}(1),
			expected: true,
		},
		{
			name:     "different type",
			slice:    []int{1, 2, 3},
			val:      "1",
			expected: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := sliceElemTypeIsSameToVal(reflect.ValueOf(c.slice), reflect.ValueOf(c.val))
			require.Equal(t, c.expected, actual)
		})
	}
}

func TestParseQueryByPath(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		expected map[string][]string
		errMsg   string
	}{
		{
			name: "common",
			path: "/get?k1=v1&k2=v2&k3=v3",
			expected: map[string][]string{
				"k1": {"v1"},
				"k2": {"v2"},
				"k3": {"v3"},
			},
		},
		{
			name:     "empty query",
			path:     "www.example.com/get",
			expected: map[string][]string{},
		},
		{
			name: "multiple values",
			path: "www.example.com/get?k1=v11&k1=v12&k2=v2&k1=v13",
			expected: map[string][]string{
				"k1": {"v11", "v12", "v13"},
				"k2": {"v2"},
			},
		},
		{
			name: "encoded url",
			path: "/get%20with%3Freserved%20characters?key=Hello+World",
			expected: map[string][]string{
				"key": {"Hello World"},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual, err := parseQueryByPath(c.path)
			if c.errMsg != "" {
				require.EqualError(t, err, c.errMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
}

func TestConstructPath(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		qs       map[string][]string
		expected string
		errMsg   string
	}{
		{
			name: "common",
			path: "/get",
			qs: map[string][]string{
				"k1": {"v1"},
				"k2": {"v2"},
				"k3": {"v3"},
			},
			expected: "/get?k1=v1&k2=v2&k3=v3",
		},
		{
			name:     "empty query",
			path:     "www.example.com/get",
			qs:       map[string][]string{},
			expected: "www.example.com/get",
		},
		{
			name: "encoded url",
			path: "/get with?",
			qs: map[string][]string{
				"key": {"Hello World"},
			},
			expected: "/get%20with?key=Hello+World",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual, err := constructPath(c.path, c.qs)
			if c.errMsg != "" {
				require.EqualError(t, err, c.errMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
}

func TestParseBody(t *testing.T) {
	cases := []struct {
		name      string
		mediaType string
		body      []byte
		expected  interface{}
		errMsg    string
	}{
		{
			name:      "application/json",
			mediaType: "application/json",
			body: []byte(`
{
  "k1": "v2",
  "k2": 20,
  "k3": true,
  "k4": [1, 2, 3],
  "k5": {
    "k6": "v6"
  }
}`),
			expected: map[string]interface{}{
				"k1": "v2",
				"k2": 20.0,
				"k3": true,
				"k4": []interface{}{1.0, 2.0, 3.0},
				"k5": map[string]interface{}{
					"k6": "v6",
				},
			},
		},
		{
			name:      "application/x-www-form-urlencoded",
			mediaType: "application/x-www-form-urlencoded",
			body:      []byte("k1=v11&k1=v12&k2=v2&k3=v3"),
			expected: map[string][]string{
				"k1": {"v11", "v12"},
				"k2": {"v2"},
				"k3": {"v3"},
			},
		},
		{
			name:      "multipart/form-data",
			mediaType: "multipart/form-data; boundary=--------------------------962785348548682888818907",
			body:      []byte("----------------------------962785348548682888818907\r\nContent-Disposition: form-data; name=\"k1\"\r\n\r\nv11\r\n----------------------------962785348548682888818907\r\nContent-Disposition: form-data; name=\"k1\"\r\n\r\nv12\r\n----------------------------962785348548682888818907\r\nContent-Disposition: form-data; name=\"k2\"\r\n\r\nv2\r\n----------------------------962785348548682888818907--\r\n"),
			expected: map[string][]string{
				"k1": {"v11", "v12"},
				"k2": {"v2"},
			},
		},
		{
			name:      "unsupported media type",
			mediaType: "plain/text",
			body:      []byte(`qwe`),
			errMsg:    "unsupported media type: plain/text",
		},
		{
			name:      "empty body",
			mediaType: "application/json",
			body:      []byte(``),
			errMsg:    errEmptyBody.Error(),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual, err := parseBody(c.mediaType, c.body)
			if c.errMsg != "" {
				require.EqualError(t, err, c.errMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
}

func TestConstructBody(t *testing.T) {
	cases := []struct {
		name      string
		mediaType string
		body      interface{}
		expected  []byte
		errMsg    string
	}{
		{
			name:      "application/json",
			mediaType: "application/json",
			body: map[string]interface{}{
				"k1": map[string]interface{}{
					"k2": []interface{}{1.0, 2.0, 3.0},
				},
			},
			expected: []byte(`{
 "k1": {
  "k2": [
   1,
   2,
   3
  ]
 }
}`),
		},
		{
			name:      "application/x-www-form-urlencoded",
			mediaType: "application/x-www-form-urlencoded",
			body: map[string][]string{
				"k1": {"v11", "v12"},
			},
			expected: []byte("k1=v11&k1=v12"),
		},
		{
			name:      "multipart/form-data",
			mediaType: "multipart/form-data; boundary=--------------------------962785348548682888818907",
			body: map[string][]string{
				"k1": {"v11", "v12"},
			},
			expected: []byte("----------------------------962785348548682888818907\r\nContent-Disposition: form-data; name=\"k1\"\r\n\r\nv11\r\n----------------------------962785348548682888818907\r\nContent-Disposition: form-data; name=\"k1\"\r\n\r\nv12\r\n----------------------------962785348548682888818907--\r\n"),
		},
		{
			name:      "unsupported media type",
			mediaType: "plain/text",
			body:      []byte(`qwe`),
			errMsg:    "unsupported media type: plain/text",
		},
		{
			name:      "empty body",
			mediaType: "application/json",
			body:      map[string]interface{}{},
			expected:  []byte(`{}`),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual, err := constructBody(c.mediaType, c.body)
			if c.errMsg != "" {
				require.EqualError(t, err, c.errMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.expected, actual)
		})
	}
}

func TestConvertByJsonType(t *testing.T) {
	cases := []struct {
		name     string
		valueTyp string
		value    string
		expected interface{}
	}{
		{
			name:     "boolean",
			valueTyp: "boolean",
			value:    "true",
			expected: true,
		},
		{
			name:     "boolean: failed",
			valueTyp: "boolean",
			value:    "null",
			expected: "null", // default string
		},
		{
			name:     "number",
			valueTyp: "number",
			value:    "10",
			expected: float64(10),
		},
		{
			name:     "string",
			valueTyp: "string",
			value:    "hello world",
			expected: "hello world",
		},
		{
			name:     "unsupported type",
			valueTyp: "integer",
			value:    "10",
			expected: "10", // default string
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := convertByJsonType(c.valueTyp, c.value)
			require.Equal(t, c.expected, actual)
		})
	}
}
