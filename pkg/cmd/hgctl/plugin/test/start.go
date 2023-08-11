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

	"github.com/alibaba/higress/pkg/cmd/hgctl/docker"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func newStartCommand() *cobra.Command {
	var (
		name   string
		config string
		source string
		detach bool
	)

	startCmd := &cobra.Command{
		Use:     "start",
		Aliases: []string{"s"},
		Short:   "Start the test environment, similar to `docker compose up`",
		Example: `  # Run containers in the background with the option --detach(-d)
  hgctl plugin test start -d
  `,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(start(cmd.OutOrStdout(), name, config, source, detach))
		},
	}

	startCmd.PersistentFlags().StringVarP(&name, "name", "p", "wasm-test", "Test environment name, that is compose project name")
	startCmd.PersistentFlags().StringVarP(&config, "file", "f", "", "Compose configuration file")
	startCmd.PersistentFlags().StringVarP(&source, "source", "s", "./test", "Test configuration source")
	startCmd.PersistentFlags().BoolVarP(&detach, "detach", "d", false, "Detached mode: Run containers in the background")

	return startCmd
}

func start(w io.Writer, name, config, source string, detach bool) error {
	cli, err := docker.NewCompose(w)
	if err != nil {
		return fmt.Errorf("failed to build the docker compose client: %w", err)
	}

	var configs []string
	if config != "" {
		configs = []string{config}
	}

	fmt.Fprintf(w, "Start the test environment %q ...\n", name)
	err = cli.Up(w, name, configs, source, detach)
	if err != nil {
		return fmt.Errorf("failed to start test environment %q: %w", name, err)
	}

	return nil
}
