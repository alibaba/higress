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

// TestBasicDocTag This is a test struct for documents(comments) and tags
type TestBasicDocTag struct {
	// Name, specify username
	Name string `yaml:"name" required:"true" minLength:"1" maxLength:"32"`

	// Age, specify age
	Age uint `yaml:"age" required:"true" minimum:"0" maximum:"140" `

	// Married, specify marital status [true, false]
	// and optional
	Married bool `yaml:"married" required:"false"`

	// Salary, specify income status, optional
	Salary float64 `yaml:"salary" required:"false"`

	// Children, specify a list of children's names, optional
	Children []string `yaml:"children" required:"false"`

	// ignore1
	Ignore1 string `yaml:"-"`

	// ignore 2
	Ignore2 string `yaml:""`
}

type TestNestedStructDocTag struct {
	// This is the comment of the nested struct field
	Struct []*TestBasicDocTag `yaml:"struct" required:"true" minItems:"1" maxItems:"10"`
}
