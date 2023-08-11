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
	"k8s.io/cli-runtime/pkg/printers"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func newLsCommand() *cobra.Command {
	lsCmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"l"},
		Short:   "List all test environments, similar to `docker compose ls`",
		Example: `  hgctl plugin test ls`,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(runLs(cmd.OutOrStdout()))
		},
	}

	return lsCmd
}

func runLs(w io.Writer) error {
	cli, err := docker.NewCompose(w)
	if err != nil {
		return fmt.Errorf("failed to build the docker compose client: %w", err)
	}

	list, err := cli.List()
	if err != nil {
		return fmt.Errorf("failed to list all test environments: %w", err)
	}

	printer := printers.GetNewTabWriter(w)
	fmt.Fprintf(printer, "NAME\tSTATUS\tCONFIG FILES\n")
	for _, stack := range list {
		fmt.Fprintf(printer, "%s\t%s\t%s\n", stack.Name, stack.Status, stack.ConfigFiles)
	}
	printer.Flush()

	return nil
}
