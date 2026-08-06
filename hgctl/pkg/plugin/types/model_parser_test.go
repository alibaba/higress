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

package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetModel(t *testing.T) {
	var (
		BasicStructField = []Model{
			{
				Name: "Name",
				Type: "string",
			},
			{
				Name: "Age",
				Type: "integer",
			},
			{
				Name: "Married",
				Type: "boolean",
			},
			{
				Name: "Salary",
				Type: "number",
			},
		}

		ExternalStructField = []Model{
			{
				Name: "one",
				Type: "string",
			},
			{
				Name: "two",
				Type: "integer",
			},
			{
				Name: "three",
				Type: "array of boolean",
			},
		}

		NestedStructField = []Model{
			{
				Name: "Simple",
				Type: "string",
			},
			{
				Name: "Complex",
				Type: "array of integer",
			},
		}
	)

	cases := []struct {
		name     string
		expected *Model
		errMsg   string
	}{
		{
			name: "TestBasicStruct",
			expected: &Model{
				Name:   "TestBasicStruct",
				Type:   "object",
				Fields: BasicStructField,
			},
		},
		{
			name: "TestComplexStruct",
			expected: &Model{
				Name: "TestComplexStruct",
				Type: "object",
				Fields: []Model{
					{
						Name: "Array",
						Type: "array of integer",
					},
					{
						Name: "Slice",
						Type: "array of string",
					},
					{
						Name: "Pointer",
						Type: "string",
					},
					{
						Name: "PPPointer",
						Type: "boolean",
					},
					{
						Name: "ArrayPointer",
						Type: "array of integer",
					},
					{
						Name: "SlicePointer",
						Type: "array of integer",
					},
					{
						Name:   "StructPointerSlice",
						Type:   "array of object",
						Fields: BasicStructField,
					},
					{
						Name:   "StructArrayPointer",
						Type:   "array of object",
						Fields: BasicStructField,
					},
				},
			},
		},
		{
			name: "TestAliasStruct",
			expected: &Model{
				Name: "TestAliasStruct",
				Type: "object",
				Fields: []Model{
					{
						Name: "MyString",
						Type: "string",
					},
					{
						Name: "MyPointerInt",
						Type: "integer",
					},
					{
						Name:   "MyStruct",
						Type:   "object",
						Fields: BasicStructField,
					},
				},
			},
		},
		{
			name: "TestExternalStruct",
			expected: &Model{
				Name: "TestExternalStruct",
				Type: "object",
				Fields: []Model{
					{
						Name: "InternalFloat",
						Type: "number",
					},
					{
						Name:   "ExStruct",
						Type:   "object",
						Fields: ExternalStructField,
					},
					{
						Name: "ExternalInt",
						Type: "integer",
					},
					{
						Name: "ExBool",
						Type: "boolean",
					},
					{
						Name: "ExSlice",
						Type: "array of string",
					},
				},
			},
		},
		{
			name: "TestNestedStruct",
			expected: &Model{
				Name: "TestNestedStruct",
				Type: "object",
				Fields: []Model{
					{
						Name: "NestedStruct",
						Type: "object",
						Fields: []Model{
							{
								Name:   "NestedStruct",
								Type:   "object",
								Fields: NestedStructField,
							},
							{
								Name: "NestedInt",
								Type: "integer",
							},
							{
								Name: "NestedString",
								Type: "string",
							},
						},
					},
				},
			},
		},
		{
			name: "ext.TestExStruct",
			expected: &Model{
				Name:   "ext.TestExStruct",
				Type:   "object",
				Fields: ExternalStructField,
			},
		},
		{
			name: "ext.TestNestedStruct",
			expected: &Model{
				Name: "ext.TestNestedStruct",
				Type: "object",
				Fields: []Model{
					{
						Name:   "NestedStruct",
						Type:   "object",
						Fields: NestedStructField,
					},
					{
						Name: "NestedInt",
						Type: "integer",
					},
					{
						Name: "NestedString",
						Type: "string",
					},
				},
			},
		},
	}

	p, err := NewModelParser("./testdata/types")
	require.NoError(t, err)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual, err := p.GetModel(c.name)
			if c.errMsg != "" {
				require.EqualError(t, err, c.errMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, c.expected, actual)
			}
		})
	}
}

func TestParseStructAndAlias(t *testing.T) {
	cases := []struct {
		name            string
		dir             string
		expectedStructs map[string]struct{}
		expectedAlias   map[string]string
	}{
		{
			name: "Basic",
			dir:  "./testdata/types",
			expectedStructs: map[string]struct{}{
				"TestBasicStruct":         {},
				"TestComplexStruct":       {},
				"TestAliasStruct":         {},
				"TestExternalStruct":      {},
				"TestNestedStruct":        {},
				"ext.TestExStruct":        {},
				"ext.TestNestedStruct":    {},
				"nested.TestNestedStruct": {},
			},
			expectedAlias: map[string]string{
				"MyString":            "string",
				"MyPointerInt":        "int",
				"MyStruct":            "TestBasicStruct",
				"NestedAlias":         "nested.TestNestedStruct",
				"NestedBasicAlias":    "bool",
				"ext.ExAlias":         "nested.TestNestedStruct",
				"ext.ExPointerInt":    "int",
				"ext.ExBool":          "bool",
				"ext.ExSlice":         "string",
				"nested.NestedInt":    "int",
				"nested.NestedString": "string",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := NewModelParser(c.dir)
			require.NoError(t, err)

			actualStructs := make(map[string]struct{})
			for _, s := range p.structs {
				actualStructs[s.name] = struct{}{}
			}
			require.Equal(t, c.expectedStructs, actualStructs)

			actualAlias := make(map[string]string)
			for name, alias := range p.alias {
				actualAlias[name] = alias.name
			}
			require.Equal(t, c.expectedAlias, actualAlias)
		})
	}
}

func TestStructFieldDocAndTag(t *testing.T) {
	var BasicStructField = []Model{
		{
			Name: "Name",
			Type: "string",
			Doc:  "Name, specify username",
			Tag:  `yaml:"name" required:"true" minLength:"1" maxLength:"32"`,
		},
		{
			Name: "Age",
			Type: "integer",
			Doc:  "Age, specify age",
			Tag:  `yaml:"age" required:"true" minimum:"0" maximum:"140"`,
		},
		{
			Name: "Married",
			Type: "boolean",
			Doc:  "Married, specify marital status [true, false]\nand optional",
			Tag:  `yaml:"married" required:"false"`,
		},
		{
			Name: "Salary",
			Type: "number",
			Doc:  "Salary, specify income status, optional",
			Tag:  `yaml:"salary" required:"false"`,
		},
		{
			Name: "Children",
			Type: "array of string",
			Doc:  "Children, specify a list of children's names, optional",
			Tag:  `yaml:"children" required:"false"`,
		},
	}

	cases := []struct {
		name     string
		model    string
		expected []Model
	}{
		{
			name:     "TestBasicDocTag",
			model:    "TestBasicDocTag",
			expected: BasicStructField,
		},
		{
			name:  "TestNestedStructDocTag",
			model: "TestNestedStructDocTag",
			expected: []Model{
				{
					Name:   "Struct",
					Type:   "array of object",
					Doc:    "This is the comment of the nested struct field",
					Tag:    `yaml:"struct" required:"true" minItems:"1" maxItems:"10"`,
					Fields: BasicStructField,
				},
			},
		},
	}

	p, err := NewModelParser("./testdata/doc_tag")
	require.NoError(t, err)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, err := p.GetModel(c.model)
			require.NoError(t, err)
			require.Equal(t, c.expected, m.Fields)
		})
	}
}
