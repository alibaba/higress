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
	"text/template"

	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/option"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/types"

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
	source, err := types.GetAbsolutePath(c.FromPath)
	if err != nil {
		return errors.Wrapf(err, "invalid build products path %q", c.FromPath)
	}
	c.FromPath = source

	target, err := types.GetAbsolutePath(c.TestPath)
	if err != nil {
		return errors.Wrapf(err, "invalid test path %q", c.TestPath)
	}
	c.TestPath = target

	fields := testTmplFields{}

	// 1. extract the parameters from spec.yaml and convert them to PluginConf
	fields.PluginConf, err = c.extractPluginConfFromSpec()
	if err != nil {
		return errors.Wrapf(err, "failed to get the parameters of plugin-conf.yaml from %s/spec.yaml", c.FromPath)
	}

	// 2. get DockerCompose instance
	fields.DockerCompose = &DockerCompose{
		TestPath:    c.TestPath,
		ProductPath: c.FromPath,
	}

	// 3. get Envoy instance
	var obj interface{}
	err = yaml.Unmarshal([]byte(fields.PluginConf.Example), &obj)
	if err != nil {
		return errors.Wrap(err, "failed to get the example of wasm plugin")
	}
	b, err := json.MarshalIndent(obj, "", strings.Repeat(" ", 2))
	if err != nil {
		return errors.Wrap(err, "failed to marshal example to json")
	}
	jsExample := addIndent(string(b), strings.Repeat(" ", 30))
	fields.Envoy = &Envoy{JSONExample: jsExample}

	// 4. generate corresponding test files
	err = os.MkdirAll(target, 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create the test environment")
	}
	err = c.genTestConfFiles(fields)
	if err != nil {
		return errors.Wrap(err, "failed to create the test environment")
	}

	fmt.Fprintf(c.w, "Created the test environment in %q\n", target)

	return nil
}

type testTmplFields struct {
	PluginConf    *PluginConf    // for plugin-conf.yaml
	DockerCompose *DockerCompose // for docker-compose.yaml
	Envoy         *Envoy         // for envoy.yaml
}

type PluginConf struct {
	Name        string
	Namespace   string
	Title       string
	Description string
	IconUrl     string
	Version     string
	Category    string
	Phase       string
	Priority    int64
	Example     string
}

type DockerCompose struct {
	TestPath    string
	ProductPath string
}

type Envoy struct {
	JSONExample string
}

func (c *creator) extractPluginConfFromSpec() (*PluginConf, error) {
	path := fmt.Sprintf("%s/spec.yaml", c.FromPath)
	spec, err := types.ParseSpecYAML(path)
	if err != nil {
		return nil, err
	}

	example, err := spec.GetConfigExample()
	if err != nil {
		return nil, err
	}

	pc := &PluginConf{
		Name:        spec.Info.Name,
		Namespace:   "higress-system",
		Title:       spec.Info.Title,
		Description: spec.Info.Description,
		IconUrl:     spec.Info.IconUrl,
		Version:     spec.Info.Version,
		Category:    string(spec.Info.Category),
		Phase:       string(spec.Spec.Phase),
		Priority:    spec.Spec.Priority,
		Example:     example,
	}
	pc.withDefaultValue()

	return pc, nil
}

func (pc *PluginConf) withDefaultValue() {
	if pc.Name == "" {
		pc.Name = "unnamed"
	}
	if pc.Namespace == "" {
		pc.Namespace = "higress-system"
	}
	if pc.Title == "" {
		pc.Title = "untitled"
	}
	if pc.Description == "" {
		pc.Description = "no description"
	}
	if pc.IconUrl == "" {
		pc.IconUrl = types.Category2IconUrl(types.Category(pc.Category))
	}
	if pc.Version == "" {
		pc.Version = "0.1.0"
	}
	if pc.Category == "" {
		pc.Category = string(types.CategoryDefault)
	}
	if pc.Phase == "" {
		pc.Phase = string(types.PhaseDefault)
	}

}

func (c *creator) genTestConfFiles(fields testTmplFields) error {
	err := GenPluginConfYAML(fields.PluginConf, c.TestPath)
	if err != nil {
		return err
	}

	err = genDockerComposeYAML(fields.DockerCompose, c.TestPath)
	if err != nil {
		return err
	}

	err = genEnvoyYAML(fields.Envoy, c.TestPath)
	if err != nil {
		return err
	}

	return nil
}

func GenPluginConfYAML(p *PluginConf, target string) error {
	path := fmt.Sprintf("%s/plugin-conf.yaml", target)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = template.Must(template.New("PluginConfYAML").Parse(PluginConfYAML)).Execute(f, p); err != nil {
		return err
	}

	return nil
}

func genDockerComposeYAML(d *DockerCompose, target string) error {
	path := fmt.Sprintf("%s/docker-compose.yaml", target)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = template.Must(template.New("DockerComposeYAML").Parse(DockerComposeYAML)).Execute(f, d); err != nil {
		return err
	}

	return nil
}

func genEnvoyYAML(e *Envoy, target string) error {
	path := fmt.Sprintf("%s/envoy.yaml", target)
	f, err := os.Create(path)
	if err != nil {
		panic(fmt.Sprintf("failed to create %q: %v\n", path, err))
	}
	defer f.Close()

	if err = template.Must(template.New("EnvoyYAML").Parse(EnvoyYAML)).Execute(f, e); err != nil {
		return err
	}

	return nil
}

func addIndent(str, indent string) string {
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
