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
	"context"
	"fmt"
	"io"

	"github.com/alibaba/higress/pkg/cmd/hgctl/docker"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/option"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type stopper struct {
	optionFile string
	option.TestOptions

	w io.Writer
}

func newStopCommand() *cobra.Command {
	var s stopper
	v := viper.New()

	stopCmd := &cobra.Command{
		Use:     "stop",
		Aliases: []string{"st"},
		Short:   "Stop the test environment",
		Example: `  # Stop responding to the compose containers with the option --name(-p)
  hgctl plugin test stop -p wasm-test
  `,

		PreRun: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(s.config(v, cmd))
		},

		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(s.stop())
		},
	}

	flags := stopCmd.PersistentFlags()
	option.AddOptionFileFlag(&s.optionFile, flags)
	v.BindPFlags(flags)

	flags.StringP("name", "p", "wasm-test", "Test environment name")
	v.BindPFlag("test.name", flags.Lookup("name"))
	v.SetDefault("test.name", "wasm-test")

	return stopCmd
}

func (s *stopper) config(v *viper.Viper, cmd *cobra.Command) error {
	allOpt, err := option.ParseOptions(s.optionFile, v, cmd.PersistentFlags())
	if err != nil {
		return err
	}
	s.TestOptions = allOpt.Test

	s.w = cmd.OutOrStdout()

	return nil
}

func (s *stopper) stop() error {
	cli, err := docker.NewCompose(s.w)
	if err != nil {
		return errors.Wrap(err, "failed to build the docker compose client")
	}

	err = cli.Down(context.TODO(), s.Name)
	if err != nil {
		return errors.Wrapf(err, "failed to stop the test environment %q", s.Name)
	}
	fmt.Fprintf(s.w, "Stopped the test environment %q\n", s.Name)

	return nil
}
