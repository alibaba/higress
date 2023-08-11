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

func newStopCommand() *cobra.Command {
	var name string

	stopCmd := &cobra.Command{
		Use:     "stop",
		Aliases: []string{"st"},
		Short:   "Stop the test environment, similar to `docker compose down`",
		Example: `  # Stop responding to the compose containers with the option --name(-p)
  hgctl plugin test stop -p wasm-test
  `,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(stop(cmd.OutOrStdout(), name))
		},
	}

	stopCmd.PersistentFlags().StringVarP(&name, "name", "p", "wasm-test", "Test environment name, that is compose project name")

	return stopCmd
}

func stop(w io.Writer, name string) error {
	cli, err := docker.NewCompose(w)
	if err != nil {
		return fmt.Errorf("failed to build the docker compose client: %w", err)
	}

	fmt.Fprintf(w, "Stop the test environment %q ...\n", name)
	err = cli.Down(name)
	if err != nil {
		return fmt.Errorf("failed to stop test environment %q: %w", name, err)
	}

	return nil
}
