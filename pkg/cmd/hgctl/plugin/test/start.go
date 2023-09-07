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

// TODO(WeixinX): If no test environment exists, create one first and then start
type starter struct {
	optionFile string
	option.TestOptions

	w io.Writer
}

func newStartCommand() *cobra.Command {
	var s starter
	v := viper.New()

	startCmd := &cobra.Command{
		Use:     "start",
		Aliases: []string{"s"},
		Short:   "Start the test environment",
		Example: `  # If the option.yaml file exists in the current path, do the following:
  hgctl plugin test start

  # Run containers in the background with the option --detach(-d)
  hgctl plugin test start -d
  `,
		PreRun: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(s.config(v, cmd))
		},

		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(s.start())
		},
	}

	flags := startCmd.PersistentFlags()
	option.AddOptionFileFlag(&s.optionFile, flags)
	v.BindPFlags(flags)

	flags.StringP("name", "p", "wasm-test", "Test environment name")
	v.BindPFlag("test.name", flags.Lookup("name"))
	v.SetDefault("test.name", "wasm-test")

	flags.StringP("test-path", "t", "./test", "Test configuration source")
	v.BindPFlag("test.test-path", flags.Lookup("test-path"))
	v.SetDefault("test.test-path", "./test")

	flags.StringP("compose-file", "c", "", "Docker compose configuration file")
	v.BindPFlag("test.compose-file", flags.Lookup("compose-file"))
	v.SetDefault("test.compose-file", "")

	flags.BoolP("detach", "d", false, "Detached mode: Run containers in the background")
	v.BindPFlag("test.detach", flags.Lookup("detach"))
	v.SetDefault("test.detach", false)

	return startCmd
}

func (s *starter) config(v *viper.Viper, cmd *cobra.Command) error {
	allOpt, err := option.ParseOptions(s.optionFile, v, cmd.PersistentFlags())
	if err != nil {
		return err
	}
	s.TestOptions = allOpt.Test

	s.w = cmd.OutOrStdout()

	return nil
}

func (s *starter) start() error {
	cli, err := docker.NewCompose(s.w)
	if err != nil {
		return errors.Wrap(err, "failed to build the docker compose client")
	}

	var configs []string
	if s.ComposeFile != "" {
		configs = []string{s.ComposeFile}
	}

	err = cli.Up(context.TODO(), s.Name, configs, s.TestPath, s.Detach)
	if err != nil {
		return errors.Wrap(err, "failed to start the test environment")
	}
	fmt.Fprintf(s.w, "Started the test environment %q\n", s.Name)

	return nil
}
