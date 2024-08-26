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
	"bufio"
	"io"
	"strings"

	"github.com/google/yamlfmt/formatters/basic"
)

var (
	formatterConfig = func() *basic.Config {
		cfg := basic.DefaultConfig()
		return cfg
	}()
	formatter = &basic.BasicFormatter{
		Config:   formatterConfig,
		Features: basic.ConfigureFeaturesFromConfig(formatterConfig),
	}
)

// FilterFunc is used to filter some contents of manifest.
type FilterFunc func(string) string

func ApplyFilters(input string, filters ...FilterFunc) string {
	for _, filter := range filters {
		input = filter(input)
	}
	return input
}

// LicenseFilter assumes that license is at the beginning.
// So we just remove all the leading comments until the first non-comment line appears.
func LicenseFilter(input string) string {
	var index int
	buf := bufio.NewReader(strings.NewReader(input))
	for {
		line, err := buf.ReadString('\n')
		if !strings.HasPrefix(line, "#") {
			return input[index:]
		}
		index += len(line)
		if err == io.EOF {
			return input[index:]
		}
	}
}

// SpaceFilter removes all leading and trailing space.
func SpaceFilter(input string) string {
	return strings.TrimSpace(input)
}

// SpaceLineFilter removes all space lines.
func SpaceLineFilter(input string) string {
	var builder strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	return builder.String()
}

// FormatterFilter uses github.com/google/yamlfmt to format yaml file
func FormatterFilter(input string) string {
	resBytes, err := formatter.Format([]byte(input))
	// todo: think about log
	if err != nil {
		return input
	}
	return string(resBytes)
}
