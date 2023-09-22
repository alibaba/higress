// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"testing"
)

func TestLicenseFilter(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: `# license line
content line`,
			want: `content line`,
		},
		{
			input: `# license line`,
			want:  "",
		},
		{
			input: `# license line
content line
# comment line`,
			want: `content line
# comment line`,
		},
	}

	for _, test := range tests {
		res := LicenseFilter(test.input)
		if res != test.want {
			t.Errorf("want %s\n but got %s", test.want, res)
		}
	}
}

func TestSpaceFilter(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: `
content line
`,
			want: "content line",
		},
	}

	for _, test := range tests {
		res := SpaceFilter(test.input)
		if res != test.want {
			t.Errorf("want %s\n but got %s", test.want, res)
		}
	}
}

func TestFormatterFilter(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: `key1: val1  `,
			want: `key1: val1
`,
		},
		{
			input: `key1:
    key2: val2`,
			want: `key1:
  key2: val2
`,
		},
	}

	for _, test := range tests {
		res := FormatterFilter(test.input)
		if res != test.want {
			t.Errorf("want \n%s\n but got \n%s\n", test.want, res)
		}
	}
}
