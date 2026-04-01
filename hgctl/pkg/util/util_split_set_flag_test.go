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

package util

import "testing"

func TestSplitSetFlag(t *testing.T) {
	tests := []struct {
		name string
		in   string
		key  string
		val  string
	}{
		{
			name: "normal pair",
			in:   "a=b",
			key:  "a",
			val:  "b",
		},
		{
			name: "no separator",
			in:   "a",
			key:  "a",
			val:  "",
		},
		{
			name: "value contains equals",
			in:   "token=abc=def==",
			key:  "token",
			val:  "abc=def==",
		},
		{
			name: "trim spaces",
			in:   " key = value=1 ",
			key:  "key",
			val:  "value=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, val := SplitSetFlag(tt.in)
			if key != tt.key || val != tt.val {
				t.Fatalf("SplitSetFlag(%q)=(%q,%q), want (%q,%q)", tt.in, key, val, tt.key, tt.val)
			}
		})
	}
}
