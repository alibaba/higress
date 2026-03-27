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

func TestStripPrefix(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		prefix string
		want   string
	}{
		{
			name:   "empty prefix keeps path",
			path:   "a/b/c",
			prefix: "",
			want:   "a/b/c",
		},
		{
			name:   "full match returns empty",
			path:   "a/b",
			prefix: "a/b",
			want:   "",
		},
		{
			name:   "prefix match with slash strips prefix",
			path:   "a/b/c.yaml",
			prefix: "a/b",
			want:   "c.yaml",
		},
		{
			name:   "prefix ending with slash is accepted",
			path:   "a/b/c.yaml",
			prefix: "a/b/",
			want:   "c.yaml",
		},
		{
			name:   "not a real prefix keeps path",
			path:   "a/b/c",
			prefix: "x/y",
			want:   "a/b/c",
		},
		{
			name:   "longer prefix keeps path",
			path:   "a",
			prefix: "a/b",
			want:   "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripPrefix(tt.path, tt.prefix)
			if got != tt.want {
				t.Fatalf("StripPrefix(%q, %q) = %q, want %q", tt.path, tt.prefix, got, tt.want)
			}
		})
	}
}
