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

package annotations

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"istio.io/istio/pilot/pkg/model"
)

func TestExtraSecret(t *testing.T) {
	inputCases := []struct {
		input  string
		expect model.NamespacedName
	}{
		{
			input:  "test/test",
			expect: model.NamespacedName{},
		},
		{
			input:  "kubernetes-ingress://test/test",
			expect: model.NamespacedName{},
		},
		{
			input: "kubernetes-ingress://cluster/foo/bar",
			expect: model.NamespacedName{
				Namespace: "foo",
				Name:      "bar",
			},
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			if !reflect.DeepEqual(inputCase.expect, extraSecret(inputCase.input)) {
				t.Fatal("Should be equal")
			}
		})
	}
}

func TestSplitBySeparator(t *testing.T) {
	testCases := []struct {
		input  string
		sep    string
		expect []string
	}{
		{
			input:  "a b c d",
			sep:    " ",
			expect: []string{"a", "b", "c", "d"},
		},
		{
			input:  ".1.2.3.4.",
			sep:    ".",
			expect: []string{"1", "2", "3", "4"},
		},
		{
			input:  "1....2....3....4",
			sep:    ".",
			expect: []string{"1", "2", "3", "4"},
		},
	}

	for _, tt := range testCases {
		got := splitBySeparator(tt.input, tt.sep)
		if diff := cmp.Diff(tt.expect, got); diff != "" {
			t.Errorf("TestSplitBySeparator() mismatch (-want +got):\n%s", diff)
		}
	}

}
