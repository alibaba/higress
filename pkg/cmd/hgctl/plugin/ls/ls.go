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

package ls

import (
	"context"
	"fmt"
	"io"
	"time"

	k8s "github.com/alibaba/higress/pkg/cmd/hgctl/kubernetes"
	"github.com/alibaba/higress/pkg/cmd/options"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/printers"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func NewCommand() *cobra.Command {
	lsCmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"l"},
		Short:   "List all installed WASM plugins",
		Example: `  hgctl plugin ls`,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(runLs(cmd.OutOrStdout()))
		},
	}

	flags := lsCmd.PersistentFlags()
	options.AddKubeConfigFlags(flags)
	k8s.AddHigressNamespaceFlags(flags)

	return lsCmd
}

func runLs(w io.Writer) error {
	dynCli, err := k8s.NewDynamicClient(options.DefaultConfigFlags.ToRawKubeConfigLoader())
	if err != nil {
		return errors.Wrap(err, "failed to build kubernetes client")
	}
	cli := k8s.NewWasmPluginClient(dynCli)

	list, err := cli.List(context.TODO())
	if err != nil {
		return errors.Wrap(err, "failed to list all wasm plugins")
	}

	printer := printers.GetNewTabWriter(w)
	now := time.Now()
	fmt.Fprintf(printer, "NAME\tAGE\n")
	for _, item := range list.Items {
		fmt.Fprintf(printer, "%s\t%s\n", item.GetName(), getAge(now, item.GetCreationTimestamp().Time))
	}
	if err = printer.Flush(); err != nil {
		return errors.Wrap(err, "failed to flush output")
	}

	return nil
}

func getAge(now time.Time, create time.Time) string {
	return duration.ShortHumanDuration(now.Sub(create))
}
