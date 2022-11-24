// Copyright (c) 2022 Terminus, Inc.
//
// This program is free software: you can use, redistribute, and/or modify
// it under the terms of the GNU Affero General Public License, version 3
// or later ("AGPL"), as published by the Free Software Foundation.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package proxy_redirect_test

import (
	"testing"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/proxy-redirect/proxy_redirect"
)

func TestGetPrefixRewrite(t *testing.T) {
	type Case struct {
		Original        string
		Upstream        string
		Prefix          string
		Rewrite         string
		HasCommonSuffix bool
	}
	var cases = []Case{
		{
			Original:        "/one/two/three",
			Upstream:        "/a/two/three",
			Prefix:          "/one",
			Rewrite:         "/a",
			HasCommonSuffix: true,
		}, {
			Original:        "/one/two/three",
			Upstream:        "/one/two/three",
			Prefix:          "",
			Rewrite:         "",
			HasCommonSuffix: false,
		}, {
			Original:        "/one/two/three",
			Upstream:        "/a/b/c",
			Prefix:          "",
			Rewrite:         "",
			HasCommonSuffix: false,
		}, {
			Original:        "/one/two/three",
			Upstream:        "/one/two/c",
			Prefix:          "",
			Rewrite:         "",
			HasCommonSuffix: false,
		}, {
			Original:        "/one/two/three/",
			Upstream:        "/one/two/three",
			Prefix:          "",
			Rewrite:         "",
			HasCommonSuffix: false,
		},
	}
	for i, item := range cases {
		prefix, rewrite, ok := proxy_redirect.SpeculatePrefixRewrite(item.Original, item.Upstream)
		if prefix != item.Prefix || rewrite != item.Rewrite || ok != item.HasCommonSuffix {
			t.Fatalf("[%v] expects prefix: %s, rewrite: %s, hasCommonSuffix: %v, get prefix: %s, rewrite: %s, hasCommonSuffix: %v\n",
				i, item.Prefix, item.Rewrite, item.HasCommonSuffix, prefix, rewrite, ok)
		}
	}
}

func TestReplaceSubstitution(t *testing.T) {
	type Case struct {
		Expr   string
		Raw    string
		Subs   string
		Result string
	}
	var cases = []Case{
		{
			Expr:   `~^(http://[^:]+):\d+(/.+)$`,
			Raw:    "http://localhost:8080/one/two/three",
			Subs:   "$1$2",
			Result: "http://localhost/one/two/three",
		},
		{
			Expr:   "~*/user/([^/]+)/(.+)$",
			Raw:    "/user/one/two/three",
			Subs:   "http://$1.example.com/$2",
			Result: "http://one.example.com/two/three",
		},
	}
	for i, item := range cases {
		s, ok, err := proxy_redirect.ReplaceSubstitution(item.Expr, item.Raw, item.Subs)
		if err != nil {
			t.Fatalf("failure to ReplaceSubstitution for cases[%v], expr: %s, raw: %s, sub: %s, err: %v",
				i, item.Expr, item.Raw, item.Subs, err)
		}
		if !ok {
			t.Fatalf("not match for cases[%v], expr: %s, raw: %s, sub: %s, err: %v",
				i, item.Expr, item.Raw, item.Subs, err)
		}
		if s != item.Result {
			t.Fatalf("result error for cases[%v], expr: %s, raw: %s, sub: %s, expects: %s, got: %s",
				i, item.Expr, item.Raw, item.Subs, item.Result, s)
		}
	}
}
