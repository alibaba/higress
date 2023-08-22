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

package plugininit

import (
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/types"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func NewCommand() *cobra.Command {
	var target string

	initCmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"i"},
		Short:   "Initialize a Golang WASM plugin project",
		Example: `  hgctl plugin init`,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(runInit(cmd.OutOrStdout(), target))
		},
	}

	initCmd.PersistentFlags().StringVarP(&target, "target", "t", "./", "Specify the project target path")

	return initCmd
}

func runInit(w io.Writer, target string) (err error) {
	ans := answer{}
	err = survey.Ask(questions, &ans)
	if err != nil {
		if errors.Is(err, terminal.InterruptErr) {
			fmt.Println("interrupted initialization")
			return nil
		}
		return fmt.Errorf("failed to initialize the project: %w", err)
	}

	path := fmt.Sprintf("%s/%s", target, ans.Name)
	err = os.MkdirAll(path, 0755)
	defer func() {
		if err != nil {
			os.RemoveAll(path)
			err = fmt.Errorf("failed to initialize the project: %w", err)

		}
	}()
	if err != nil {
		return
	}
	err = genGoMain(&ans, path)
	if err != nil {
		return
	}
	err = genGoMod(&ans, path)
	if err != nil {
		return
	}
	err = genGitIgnore(path)
	if err != nil {
		return
	}
	err = genOptionYAML(path)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "Successfully initialized the project in %q\n", path)

	return nil
}

var questions = []*survey.Question{
	{
		Name: "Name",
		Prompt: &survey.Input{
			Message: "Plugin name:",
			Default: "hello-world",
		},
		Validate: survey.Required,
	},
	{
		Name: "Category",
		Prompt: &survey.Select{
			Message: "Choose a plugin category:",
			Options: []string{
				string(types.CategoryCustom),
				string(types.CategoryAuth),
				string(types.CategorySecurity),
				string(types.CategoryProtocol),
				string(types.CategoryFlowControl),
				string(types.CategoryFlowMonitor),
			},
			Default: string(types.CategoryCustom),
		},
		Validate: survey.Required,
	},
	{
		Name: "Phase",
		Prompt: &survey.Select{
			Message: "Choose a execution phase:",
			Options: []string{
				string(types.PhaseUnspecified),
				string(types.PhaseAuthn),
				string(types.PhaseAuthz),
				string(types.PhaseStats),
			},
			Default: string(types.PhaseUnspecified),
		},
		Validate: survey.Required,
	},
	{
		Name: "Priority",
		Prompt: &survey.Input{
			Message: "Execution priority:",
			Default: "0",
		},
		Validate: survey.Required,
	},
	{
		Name: "I18nType",
		Prompt: &survey.Select{
			Message: "Choose i18n type:",
			Options: []string{
				string(types.I18nZH_CN),
				string(types.I18nEN_US),
			},
			Default: string(types.I18nDefault),
		},
		Validate: survey.Required,
	},
	{
		Name: "Title",
		Prompt: &survey.Input{
			Message: "Display name in the plugin market:",
			Default: "Hello World",
		},
		Validate: survey.Required,
	},
	{
		Name: "Description",
		Prompt: &survey.Input{
			Message: "Description of the plugin functionality:",
			Default: "This is a demo plugin",
		},
	},
	{
		Name: "IconUrl",
		Prompt: &survey.Input{
			Message: "Display icon in the plugin market:",
			Default: "",
		},
	},
	{
		Name: "Version",
		Prompt: &survey.Input{
			Message: "Plugin version:",
			Default: "0.1.0",
		},
		Validate: survey.Required,
	},
	{
		Name: "ContactName",
		Prompt: &survey.Input{
			Message: "Name of plugin developer:",
			Default: "",
		},
	},
	{
		Name: "ContactUrl",
		Prompt: &survey.Input{
			Message: "Web home of developer:",
			Default: "",
		},
	},
	{
		Name: "ContactEmail",
		Prompt: &survey.Input{
			Message: "Email of developer:",
			Default: "",
		},
	},
}

type answer struct {
	Name        string
	Category    string
	Phase       string
	Priority    int64
	I18nType    string
	Title       string
	Description string
	IconUrl     string
	Version     string

	ContactName  string
	ContactUrl   string
	ContactEmail string
}

func genGoMain(ans *answer, target string) error {
	path := fmt.Sprintf("%s/main.go", target)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create main.go: %w", err)
	}
	defer f.Close()

	t, err := template.New("GoMain").Parse(GoMain)
	if err != nil {
		return fmt.Errorf("failed to create main.go: %w", err)
	}

	if err = t.Execute(f, ans); err != nil {
		return fmt.Errorf("failed to create main.go: %w", err)
	}

	return nil
}

func genGoMod(ans *answer, target string) error {
	path := fmt.Sprintf("%s/go.mod", target)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}
	defer f.Close()

	t, err := template.New("GoMod").Parse(GoMod)
	if err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	if err = t.Execute(f, ans); err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	return nil
}

func genGitIgnore(target string) error {
	path := fmt.Sprintf("%s/.gitignore", target)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(GitIgnore)
	if err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	return nil
}

func genOptionYAML(target string) error {
	path := fmt.Sprintf("%s/option.yaml", target)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create option.yaml: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(OptionYAML)
	if err != nil {
		return fmt.Errorf("failed to create option.yaml: %w", err)
	}

	return nil
}
