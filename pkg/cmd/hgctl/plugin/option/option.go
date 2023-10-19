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

package option

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Option struct {
	Version string         `json:"version" yaml:"version" mapstructure:"version"`
	Build   BuildOptions   `json:"build" yaml:"build" mapstructure:"build"`
	Test    TestOptions    `json:"test" yaml:"test" mapstructure:"test"`
	Install InstallOptions `json:"install" yaml:"install" mapstructure:"install"`
}

type BuildOptions struct {
	Builder    BuilderVersion `json:"builder" yaml:"builder" mapstructure:"builder"`
	Input      string         `json:"input" yaml:"input" mapstructure:"input"`
	Output     Output         `json:"output" yaml:"output" mapstructure:"output"`
	DockerAuth string         `json:"docker-auth" yaml:"docker-auth" mapstructure:"docker-auth"`
	ModelDir   string         `json:"model-dir" yaml:"model-dir" mapstructure:"model-dir"`
	Model      string         `json:"model" yaml:"model" mapstructure:"model"`
	Debug      bool           `json:"debug" yaml:"debug" mapstructure:"debug"`
}

type TestOptions struct {
	Name        string `json:"name" yaml:"name" mapstructure:"name"`
	FromPath    string `json:"from-path" yaml:"from-path" mapstructure:"from-path"`
	TestPath    string `json:"test-path" yaml:"test-path" mapstructure:"test-path"`
	ComposeFile string `json:"compose-file" yaml:"compose-file" mapstructure:"compose-file"`
	Detach      bool   `json:"detach" yaml:"detach" mapstructure:"detach"`
}

type InstallOptions struct {
	Namespace string `json:"namespace" yaml:"namespace" mapstructure:"namespace"`
	SpecYaml  string `json:"spec-yaml" yaml:"spec-yaml" mapstructure:"spec-yaml"`
	FromYaml  string `json:"from-yaml" yaml:"from-yaml" mapstructure:"from-yaml"`
	FromGoSrc string `json:"from-go-src" yaml:"from-go-src" mapstructure:"from-go-src"`
	Debug     bool   `json:"debug" yaml:"debug" mapstructure:"debug"`
}

type BuilderVersion struct {
	Go     string `json:"go" yaml:"go" mpastructure:"go"`
	TinyGo string `json:"tinygo" yaml:"tinygo" mapstructure:"tinygo"`
	Oras   string `json:"oras" yaml:"oras" mapstructure:"oras"`
}

type Output struct {
	Type string `json:"type" yaml:"type" mapstructure:"type"`
	Dest string `json:"dest" yaml:"dest" mapstructure:"dest"`
}

// ParseOptions reads `option.yaml` and parses it into Option struct
func ParseOptions(optionFile string, v *viper.Viper, flags *pflag.FlagSet) (*Option, error) {
	_, err := os.Stat(optionFile)
	if err != nil {
		// `option-file` is explicitly specified, but the given file does not exist
		if errors.Is(err, os.ErrNotExist) && flags.Changed("option-file") {
			return nil, errors.Errorf("option file does not exist: %q", optionFile)
		}
	} else {
		v.SetConfigFile(optionFile)
		if err = v.ReadInConfig(); err != nil {
			return nil, errors.Wrapf(err, "failed to read option file %q", optionFile)
		}
	}

	var opt Option
	if err = v.Unmarshal(&opt); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal option file %q", optionFile)
	}

	return &opt, nil
}

// AddOptionFileFlag adds `option-file` flag
func AddOptionFileFlag(optionFile *string, flags *pflag.FlagSet) {
	flags.StringVarP(optionFile, "option-file", "f", "./option.yaml",
		"Option file for build, test and install")
}
