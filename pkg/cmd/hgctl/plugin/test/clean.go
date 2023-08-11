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
	"fmt"
	"io"
	"os"

	"github.com/alibaba/higress/pkg/cmd/hgctl/docker"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func newCleanCommand() *cobra.Command {
	var (
		name   string
		source string
	)

	cleanCmd := &cobra.Command{
		Use:     "clean",
		Aliases: []string{"c"},
		Short:   "Clean the test environment, that is remove the source of test configuration",
		Example: `  hgctl plugin test clean`,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(clean(cmd.OutOrStdout(), name, source))
		},
	}

	cleanCmd.PersistentFlags().StringVarP(&name, "name", "p", "wasm-test", "Test environment name, that is compose project name")
	cleanCmd.PersistentFlags().StringVarP(&source, "source", "s", "./test", "Test configuration source")

	return cleanCmd
}

func clean(w io.Writer, name, source string) error {
	cli, err := docker.NewCompose(w)
	if err != nil {
		return fmt.Errorf("failed to build the docker compose client: %w", err)
	}

	fmt.Fprintf(w, "Clean the test environment %q ...\n", name)
	err = cli.Down(name)
	if err != nil {
		return fmt.Errorf("failed to stop test environment %q: %w", name, err)
	}

	err = os.RemoveAll(source)
	if err != nil {
		return fmt.Errorf("failed to remove test configuration source %q: %w", source, err)
	}
	fmt.Fprintf(w, "Remove the source: %q\n", source)

	return nil
}
