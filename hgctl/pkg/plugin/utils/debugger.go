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
	"fmt"
	"io"
)

type Debugger interface {
	Debugf(format string, a ...any) (int, error)
	Debugln(a ...any) (int, error)
}

type DefaultDebugger struct {
	debug bool
	w     io.Writer
}

func NewDefaultDebugger(debug bool, w io.Writer) *DefaultDebugger {
	return &DefaultDebugger{debug: debug, w: w}
}

func (d DefaultDebugger) Debugf(format string, a ...any) (int, error) {
	l := len(format)
	if l > 0 && format[l-1] != '\n' {
		format += "\n"
	}
	if d.debug {
		format = "[debug] " + format
		return fmt.Fprintf(d.w, format, a...)
	}
	return 0, nil
}

func (d DefaultDebugger) Debugln(a ...any) (int, error) {
	if d.debug {
		n1, err1 := fmt.Fprintf(d.w, "[debug] ")
		if err1 != nil {
			return n1, err1
		}
		n2, err2 := fmt.Fprintln(d.w, a...)
		return n1 + n2, err2
	}
	return 0, nil
}
