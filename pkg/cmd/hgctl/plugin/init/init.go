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
	"os/exec"

	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/option"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/utils"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func NewCommand() *cobra.Command {
	var target string

	initCmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"ini", "i"},
		Short:   "Initialize a Golang WASM plugin project",
		Example: `  hgctl plugin init`,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(runInit(cmd.OutOrStdout(), target))
		},
	}

	initCmd.PersistentFlags().StringVarP(&target, "target", "t", "./", "Directory where the project is initialized")

	return initCmd
}

func runInit(w io.Writer, target string) (err error) {
	ans := answer{}
	err = utils.Ask(questions, &ans)
	if err != nil {
		if errors.Is(err, terminal.InterruptErr) {
			fmt.Fprintf(w, "Interrupted\n")
			return nil
		}
		return errors.Wrap(err, "failed to initialize the project")
	}

	target, err = utils.GetAbsolutePath(target)
	if err != nil {
		return errors.Wrap(err, "invalid target directory")
	}
	dir := fmt.Sprintf("%s/%s", target, ans.Name)
	err = os.MkdirAll(dir, 0755)
	defer func() {
		if err != nil {
			os.RemoveAll(dir)
			err = errors.Wrap(err, "failed to initialize the project")

		}
	}()
	if err != nil {
		return
	}
	if err = genGoMain(&ans, dir); err != nil {
		return errors.Wrap(err, "failed to create main.go")
	}
	if err = genGoMod(&ans, dir); err != nil {
		return errors.Wrap(err, "failed to create go.mod")
	}
	if err = genGitIgnore(dir); err != nil {
		return errors.Wrap(err, "failed to create .gitignore")
	}
	if err = option.GenOptionYAML(dir); err != nil {
		return errors.Wrap(err, "failed to create option.yaml")
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to run go mod tidy")
	}

	fmt.Fprintf(w, "Initialized the project in %q\n", dir)

	return nil
}
