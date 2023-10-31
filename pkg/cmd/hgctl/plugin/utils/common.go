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

package utils

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// GetAbsolutePath returns the absolute path, e.g.:
// - ~/foo -> /home/user/foo
// - ./foo -> /current/dir/foo
// - /foo/ -> /foo
func GetAbsolutePath(path string) (newPath string, err error) {
	if strings.HasPrefix(path, "~") {
		newPath, err = homedir.Expand(path)
		if err != nil {
			return "", errors.Wrapf(err, "failed to expand path: %q", path)
		}
	} else {
		newPath, err = filepath.Abs(path)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get absolute path of %q", path)
		}
	}

	l := len(newPath)
	if l > 1 && newPath[l-1] == '/' { // if l == 1, the path might be "/"
		newPath = newPath[:l-1]
	}

	return newPath, nil
}

// AddIndent for each line of str
func AddIndent(str, indent string) string {
	ret := ""
	ss := strings.Split(str, "\n")
	for i, s := range ss {
		if i == 0 {
			ret = fmt.Sprintf("%s%s", indent, s)
		} else {
			ret = fmt.Sprintf("%s\n%s%s", ret, indent, s)
		}
	}

	return ret
}

// MarshalYamlWithIndent marshals v to yaml with indent, specify space width with spaces
func MarshalYamlWithIndent(v interface{}, spaces int) ([]byte, error) {
	w := new(bytes.Buffer)
	ec := yaml.NewEncoder(w)
	defer ec.Close()
	ec.SetIndent(spaces)
	if err := ec.Encode(v); err != nil {
		return w.Bytes(), err
	}

	return w.Bytes(), nil
}

// MarshalYamlWithIndentTo marshals v to yaml with indent, specify space width with spaces, and output to w
func MarshalYamlWithIndentTo(w io.Writer, v interface{}, spaces int) error {
	ec := yaml.NewEncoder(w)
	defer ec.Close()
	ec.SetIndent(spaces)
	if err := ec.Encode(v); err != nil {
		return err
	}

	return nil
}
