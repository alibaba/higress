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
	"os"
	"strings"

	"github.com/fatih/color"
)

type YesOrNoPrinter struct {
	out     io.Writer
	indent  *Indent
	yes, no *color.Color
}

var (
	DefaultOut   = os.Stdout
	DefaultIdent = NewIndent(strings.Repeat(" ", 2), 0)
	DefaultYes   = color.New(color.FgHiGreen)
	DefaultNo    = color.New(color.FgHiRed)
)

func NewPrinter(out io.Writer, indent *Indent, yes, no *color.Color) *YesOrNoPrinter {
	return &YesOrNoPrinter{
		out:    out,
		indent: indent,
		yes:    yes,
		no:     no,
	}
}

func DefaultPrinter() *YesOrNoPrinter {
	return NewPrinter(DefaultOut, DefaultIdent, DefaultYes, DefaultNo)
}

func (p *YesOrNoPrinter) Printf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(p.out, format, a...)
}

func (p *YesOrNoPrinter) Println(a ...interface{}) (int, error) {
	return fmt.Fprintln(p.out, a...)
}

func (p *YesOrNoPrinter) PrintWithIndentf(format string, a ...interface{}) (int, error) {
	format = fmt.Sprintf("%s%s", p.indent, format)
	return fmt.Fprintf(p.out, format, a...)
}

func (p *YesOrNoPrinter) PrintWithIndentln(a ...interface{}) (int, error) {
	n1, err := fmt.Fprintf(p.out, "%s", p.indent)
	if err != nil {
		return n1, err
	}
	n2, err := fmt.Fprintln(p.out, a...)
	if err != nil {
		return n1 + n2, err
	}
	return n1 + n2, nil
}

func (p *YesOrNoPrinter) Yesf(format string, a ...interface{}) (int, error) {
	return p.yes.Fprintf(p.out, format, a...)
}

func (p *YesOrNoPrinter) Yesln(a ...interface{}) (int, error) {
	return p.yes.Fprintln(p.out, a...)
}

func (p *YesOrNoPrinter) YesWithIndentf(format string, a ...interface{}) (int, error) {
	format = fmt.Sprintf("%s%s", p.indent, format)
	return p.yes.Fprintf(p.out, format, a...)
}

func (p *YesOrNoPrinter) YesWithIndentln(a ...interface{}) (int, error) {
	n1, err := p.yes.Fprintf(p.out, "%s", p.indent)
	if err != nil {
		return n1, err
	}
	n2, err := p.yes.Fprintln(p.out, a...)
	if err != nil {
		return n1 + n2, err
	}
	return n1 + n2, nil
}

func (p *YesOrNoPrinter) Nof(format string, a ...interface{}) (int, error) {
	return p.no.Fprintf(p.out, format, a...)
}

func (p *YesOrNoPrinter) Noln(a ...interface{}) (int, error) {
	return p.no.Fprintln(p.out, a...)
}

func (p *YesOrNoPrinter) NoWithIndentf(format string, a ...interface{}) (int, error) {
	format = fmt.Sprintf("%s%s", p.indent, format)
	return p.no.Fprintf(p.out, format, a...)
}

func (p *YesOrNoPrinter) NoWithIndentln(a ...interface{}) (int, error) {
	n1, err := p.no.Fprintf(p.out, "%s", p.indent)
	if err != nil {
		return n1, err
	}
	n2, err := p.no.Fprintln(p.out, a...)
	if err != nil {
		return n1 + n2, err
	}
	return n1 + n2, nil
}

func (p *YesOrNoPrinter) Ident() string { return p.indent.String() }

func (p *YesOrNoPrinter) IncIdentRepeat() { p.indent.IncRepeat() }

func (p *YesOrNoPrinter) DecIndentRepeat() { p.indent.DecRepeat() }

func (p *YesOrNoPrinter) SetIdentRepeat(v int) { p.indent.SetRepeat(v) }

type Indent struct {
	format string
	repeat int
}

func NewIndent(format string, repeat int) *Indent {
	return &Indent{
		format: format,
		repeat: repeat,
	}
}

func (i *Indent) String() string {
	return strings.Repeat(i.format, i.repeat)
}

func (i *Indent) IncRepeat() { i.repeat++ }

func (i *Indent) DecRepeat() {
	i.repeat--
	if i.repeat < 0 {
		i.repeat = 0
	}
}

func (i *Indent) SetRepeat(v int) {
	if v < 0 {
		v = 0
	}
	i.repeat = v
}
