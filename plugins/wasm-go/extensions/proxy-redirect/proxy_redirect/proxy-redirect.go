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

package proxy_redirect

import (
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

func SpeculatePrefixRewrite(original, upstream string) (string, string, bool) {
	if original == upstream {
		return "", "", false
	}
	if original == "" || upstream == "" {
		return "", "", false
	}
	if original[len(original)-1] != upstream[len(upstream)-1] {
		return "", "", false
	}
	var ok = false
	for {
		path0 := path.Dir(original)
		base0 := path.Base(original)
		path1 := path.Dir(upstream)
		base1 := path.Base(upstream)
		if base0 != base1 {
			if ok {
				return original, upstream, ok
			}
			return "", "", false
		}
		ok = true
		original, upstream = path0, path1
	}
}

func ReplaceSubstitution(expr, raw, substitution string) (string, bool, error) {
	if strings.HasPrefix(expr, "~*") {
		expr = "(?i)" + strings.TrimSuffix(expr, "~*")
	} else {
		expr = strings.TrimPrefix(expr, "~")
	}

	re, err := regexp.Compile(expr)
	if err != nil {
		return "", false, errors.Wrap(err, "invalid regex expression")
	}
	if ok := re.MatchString(raw); !ok {
		return "", false, nil
	}
	return re.ReplaceAllString(raw, substitution), true, nil
}
