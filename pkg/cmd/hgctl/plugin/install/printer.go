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

package install

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	DefaultOut   = os.Stdout
	DefaultIdent = NewIndent(strings.Repeat(" ", 2), 0)
	DefaultYes   = color.New(color.FgHiGreen)
	DefaultNo    = color.New(color.FgRed)
)

type Printer struct {
	out     io.Writer
	indent  *Indent
	yes, no *color.Color
}

func NewPrinter(out io.Writer, indent *Indent, yes, no *color.Color) *Printer {
	return &Printer{
		out:    out,
		indent: indent,
		yes:    yes,
		no:     no,
	}
}

func DefaultPrinter() *Printer {
	return NewPrinter(DefaultOut, DefaultIdent, DefaultYes, DefaultNo)
}

func (p *Printer) Printf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(p.out, format, a...)
}

func (p *Printer) Println(a ...interface{}) (int, error) {
	return fmt.Fprintln(p.out, a...)
}

func (p *Printer) PrintWithIndentf(format string, a ...interface{}) (int, error) {
	format = fmt.Sprintf("%s%s", p.indent, format)
	return fmt.Fprintf(p.out, format, a...)
}

func (p *Printer) PrintWithIndentln(a ...interface{}) (int, error) {
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

func (p *Printer) Yesf(format string, a ...interface{}) (int, error) {
	return p.yes.Fprintf(p.out, format, a...)
}

func (p *Printer) Yesln(a ...interface{}) (int, error) {
	return p.yes.Fprintln(p.out, a...)
}

func (p *Printer) YesWithIndentf(format string, a ...interface{}) (int, error) {
	format = fmt.Sprintf("%s%s", p.indent, format)
	return p.yes.Fprintf(p.out, format, a...)
}

func (p *Printer) YesWithIndentln(a ...interface{}) (int, error) {
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

func (p *Printer) Nof(format string, a ...interface{}) (int, error) {
	return p.no.Fprintf(p.out, format, a...)
}

func (p *Printer) Noln(a ...interface{}) (int, error) {
	return p.no.Fprintln(p.out, a...)
}

func (p *Printer) NoWithIndentf(format string, a ...interface{}) (int, error) {
	format = fmt.Sprintf("%s%s", p.indent, format)
	return p.no.Fprintf(p.out, format, a...)
}

func (p *Printer) NoWithIndentln(a ...interface{}) (int, error) {
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

func (p *Printer) Ident() string { return p.indent.String() }

func (p *Printer) IncIdentRepeat() { p.indent.IncRepeat() }

func (p *Printer) DecIndentRepeat() { p.indent.DecRepeat() }

func (p *Printer) SetIdentRepeat(v int) { p.indent.SetRepeat(v) }

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
