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

import "github.com/alibaba/higress/pkg/cmd/hgctl/plugin/types/testdata/types/ext"

type TestBasicStruct struct {
	Name    string
	Age     uint
	Married bool
	Salary  float64
}

type TestComplexStruct struct {
	Array              [2]int
	Slice              []string
	Pointer            *string
	PPPointer          ***bool
	ArrayPointer       [2]*int
	SlicePointer       []*int
	StructPointerSlice []*TestBasicStruct
	StructArrayPointer *[]TestBasicStruct
	_                  struct {
		one int
		two string
	}
}

type TestAliasStruct struct {
	MyString     *MyString
	MyPointerInt MyPointerInt
	MyStruct     MyStruct
}

type MyString string
type MyPointerInt *int
type MyStruct TestBasicStruct
type NestedAlias ext.ExAlias
type NestedBasicAlias ext.ExBool

type TestExternalStruct struct {
	InternalFloat float64
	ExStruct      ext.TestExStruct
	ExternalInt   ext.ExPointerInt
	ExBool        ext.ExBool
	ExSlice       ext.ExSlice
}

type TestNestedStruct struct {
	NestedStruct *ext.TestNestedStruct
}

type MyInterface interface {
}

var MyConst bool

var MyVar int
