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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/types"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func newCreateCommand() *cobra.Command {
	var (
		source string
		target string
	)

	createCmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"c"},
		Short:   "Create the test environment, that is create the source of test configuration",
		Example: `  # The following commands are equivalent to 'hgctl plugin test create -f ./out -t ./test'
  hgctl plugin test create
  `,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(create(cmd.OutOrStdout(), source, target))
		},
	}

	createCmd.PersistentFlags().StringVarP(&source, "from-path", "f", "./out", "The path to build the products, that is parameter source")
	createCmd.PersistentFlags().StringVarP(&target, "to-path", "t", "./test", "Test configuration source")

	return createCmd
}

func create(w io.Writer, source, target string) error {
	source, err := types.GetAbsolutePath(source)
	if err != nil {
		return fmt.Errorf("invalid products path: %w", err)
	}

	target, err = types.GetAbsolutePath(target)
	if err != nil {
		return fmt.Errorf("invalid test path: %w", err)
	}

	fields := templateFields{}

	// 1. extract the parameters from spec.yaml and convert them to PluginConf
	fields.PluginConf, err = extractFromSpec(source)
	if err != nil {
		return fmt.Errorf("failed to extract the parameters from `spec.yaml` and convert them: %w", err)
	}

	// 2. get DockerCompose instance
	fields.DockerCompose = &DockerCompose{
		TestPath:    target,
		ProductPath: source,
	}

	// 3. get Envoy instance
	var obj interface{}
	err = yaml.Unmarshal([]byte(fields.PluginConf.Example), &obj)
	if err != nil {
		return fmt.Errorf("failed to get example: %w", err)
	}
	b, err := json.MarshalIndent(obj, "", strings.Repeat(" ", 2))
	if err != nil {
		return fmt.Errorf("failed to mashal example to json format: %w", err)
	}
	jsExample := addIndent(string(b), strings.Repeat(" ", 30))
	fields.Envoy = &Envoy{JSONExample: jsExample}

	// 4. generate corresponding test files
	fmt.Fprintf(w, "Create the test environment in %q ...\n", target)
	err = os.MkdirAll(target, 0755)
	if err != nil {
		return fmt.Errorf("failed to create the test environment: %w", err)
	}
	err = genTestConfFiles(fields, target)
	if err != nil {
		return fmt.Errorf("failed to create the test environment: %w", err)
	}

	return nil
}

func extractFromSpec(source string) (*PluginConf, error) {
	path := fmt.Sprintf("%s/spec.yaml", source)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var spec types.WasmPluginMeta
	dc := k8syaml.NewYAMLOrJSONDecoder(f, 4096)
	if err = dc.Decode(&spec); err != nil {
		return nil, err
	}

	example := ""
	schema := spec.Spec.ConfigSchema.OpenAPIV3Schema
	if schema != nil && schema.Example != nil {
		// schema.Example like:
		// {"allow":["consumer2"],"consumers":[{"credential":"admin:123456","name":"consumer1"},{"credential":"guest:abc","name":"consumer2"}]}
		var obj interface{}
		err = json.Unmarshal(schema.Example.Raw, &obj)
		if err != nil {
			return nil, err
		}

		buf := new(bytes.Buffer)
		ec := yaml.NewEncoder(buf)
		defer ec.Close()
		ec.SetIndent(2)
		if err = ec.Encode(obj); err != nil {
			return nil, err
		}
		example = addIndent(buf.String(), strings.Repeat(" ", 4))
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

	pc.WithDefaultValue()

	return pc, nil
}

type templateFields struct {
	PluginConf    *PluginConf
	DockerCompose *DockerCompose
	Envoy         *Envoy
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

func (pc *PluginConf) WithDefaultValue() {
	if pc.Name == "" {
		pc.Name = "unnamed"
	}
	if pc.Title == "" {
		pc.Title = "untitled"
	}
	if pc.Version == "" {
		pc.Version = "0.1.0"
	}
	if pc.Phase == "" {
		pc.Phase = string(types.PhaseUnspecified)
	}
	if pc.Category == "" {
		pc.Category = string(types.CategoryCustom)
	}
	if pc.IconUrl == "" {
		switch types.Category(pc.Category) {
		case types.CategoryAuth:
			pc.IconUrl = types.IconAuth
		case types.CategorySecurity:
			pc.IconUrl = types.IconSecurity
		case types.CategoryProtocol:
			pc.IconUrl = types.IconProtocol
		case types.CategoryFlowControl:
			pc.IconUrl = types.IconFlowControl
		case types.CategoryFlowMonitor:
			pc.IconUrl = types.IconFlowMonitor
		case types.CategoryCustom:
			pc.IconUrl = types.IconCustom
		default:
			pc.IconUrl = types.IconCustom
		}
	}
}

func genTestConfFiles(fields templateFields, target string) error {
	err := genPluginConfYAML(fields.PluginConf, target)
	if err != nil {
		return err
	}

	err = genDockerComposeYAML(fields.DockerCompose, target)
	if err != nil {
		return err
	}

	err = genEnvoyYAML(fields.Envoy, target)
	if err != nil {
		return err
	}

	return nil
}

func genPluginConfYAML(p *PluginConf, target string) error {
	path := fmt.Sprintf("%s/plugin-conf.yaml", target)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	t, err := template.New("PluginConfYAML").Parse(PluginConfYAML)
	if err != nil {
		return err
	}

	if err = t.Execute(f, p); err != nil {
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

	t, err := template.New("DockerComposeYAML").Parse(DockerComposeYAML)
	if err != nil {
		return err
	}

	if err = t.Execute(f, d); err != nil {
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

	t, err := template.New("EnvoyYAML").Parse(EnvoyYAML)
	if err != nil {
		return err
	}

	if err = t.Execute(f, e); err != nil {
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
