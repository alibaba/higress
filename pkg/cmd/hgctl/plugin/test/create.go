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

package test

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/config"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/option"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/types"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/utils"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type creator struct {
	optionFile string
	option.TestOptions

	w io.Writer
}

func newCreateCommand() *cobra.Command {
	var c creator
	v := viper.New()

	createCmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"c"},
		Short:   "Create the test environment",
		Example: `  # If the option.yaml file exists in the current path, do the following:
  hgctl plugin test create

  # Explicitly specify the source of the parameters (directory of the build 
    products) and the directory where the test configuration files is stored
  hgctl plugin test create -d ./out -t ./test
  `,

		PreRun: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(c.config(v, cmd))
		},

		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(c.create())
		},
	}

	flags := createCmd.PersistentFlags()
	option.AddOptionFileFlag(&c.optionFile, flags)
	v.BindPFlags(flags)

	flags.StringP("from-path", "d", "./out", "Path of storing the build products")
	v.BindPFlag("test.from-path", flags.Lookup("from-path"))
	v.SetDefault("test.from-path", "./out")

	flags.StringP("test-path", "t", "./test", "Path for storing the test configuration")
	v.BindPFlag("test.test-path", flags.Lookup("test-path"))
	v.SetDefault("test.test-path", "./test")

	return createCmd
}

func (c *creator) config(v *viper.Viper, cmd *cobra.Command) error {
	allOpt, err := option.ParseOptions(c.optionFile, v, cmd.PersistentFlags())
	if err != nil {
		return err
	}
	c.TestOptions = allOpt.Test

	c.w = cmd.OutOrStdout()

	return nil
}

func (c *creator) create() (err error) {
	source, err := utils.GetAbsolutePath(c.FromPath)
	if err != nil {
		return errors.Wrapf(err, "invalid build products path %q", c.FromPath)
	}
	c.FromPath = source

	target, err := utils.GetAbsolutePath(c.TestPath)
	if err != nil {
		return errors.Wrapf(err, "invalid test path %q", c.TestPath)
	}
	c.TestPath = target

	fields := testTmplFields{}

	// 1. extract the parameters from spec.yaml and convert them to PluginConf
	path := fmt.Sprintf("%s/spec.yaml", c.FromPath)
	spec, err := types.ParseSpecYAML(path)
	if err != nil {
		return errors.Wrapf(err, "failed to parse %s", path)
	}
	fields.PluginConf, err = config.ExtractPluginConfFrom(spec, "", "")
	if err != nil {
		return errors.Wrapf(err, "failed to get the parameters of plugin-conf.yaml from %s", path)
	}

	// 2. get DockerCompose instance
	fields.DockerCompose = &DockerCompose{
		TestPath:    c.TestPath,
		ProductPath: c.FromPath,
	}

	// 3. get Envoy instance
	var obj interface{}
	conf := spec.GetConfigExample()
	err = yaml.Unmarshal([]byte(conf), &obj)
	if err != nil {
		return errors.Wrap(err, "failed to get the example of wasm plugin")
	}
	b, err := json.MarshalIndent(obj, "", strings.Repeat(" ", 2))
	if err != nil {
		return errors.Wrap(err, "failed to marshal example to json")
	}
	jsExample := utils.AddIndent(string(b), strings.Repeat(" ", 30))
	fields.Envoy = &Envoy{JSONExample: jsExample}

	// 4. generate corresponding test files
	if err = os.MkdirAll(target, 0755); err != nil {
		return errors.Wrap(err, "failed to create the test environment")

	}
	if err = c.genTestConfFiles(fields); err != nil {
		return errors.Wrap(err, "failed to create the test environment")
	}

	fmt.Fprintf(c.w, "Created the test environment in %q\n", target)

	return nil
}

type testTmplFields struct {
	PluginConf    *config.PluginConf // for plugin-conf.yaml
	DockerCompose *DockerCompose     // for docker-compose.yaml
	Envoy         *Envoy             // for envoy.yaml
}

func (c *creator) genTestConfFiles(fields testTmplFields) (err error) {
	if err = config.GenPluginConfYAML(fields.PluginConf, c.TestPath); err != nil {
		return err
	}

	if err = genDockerComposeYAML(fields.DockerCompose, c.TestPath); err != nil {
		return err
	}

	if err = genEnvoyYAML(fields.Envoy, c.TestPath); err != nil {
		return err
	}

	return nil
}
