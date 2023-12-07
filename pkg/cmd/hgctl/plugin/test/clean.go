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
	"os"

	"github.com/alibaba/higress/pkg/cmd/hgctl/docker"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/option"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/utils"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type cleaner struct {
	optionFile string
	option.TestOptions

	w io.Writer
}

func newCleanCommand() *cobra.Command {
	var c cleaner
	v := viper.New()

	cleanCmd := &cobra.Command{
		Use:     "clean",
		Aliases: []string{"cl"},
		Short:   "Clean the test environment, that is remove the source of test configuration",
		Example: `  hgctl plugin test clean`,
		PreRun: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(c.config(v, cmd))
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(c.clean())
		},
	}

	flags := cleanCmd.PersistentFlags()
	option.AddOptionFileFlag(&c.optionFile, flags)
	v.BindPFlags(flags)

	flags.StringP("name", "p", "wasm-test", "Test environment name")
	v.BindPFlag("test.name", flags.Lookup("name"))
	v.SetDefault("test.name", "wasm-test")

	// TODO(WeixinX): Obtain the test configuration source directory based on the test environment name (hgctl plugin test ls)
	flags.StringP("test-path", "t", "./test", "Test configuration source")
	v.BindPFlag("test.test-path", flags.Lookup("test-path"))
	v.SetDefault("test.test-path", "./test")

	return cleanCmd
}

func (c *cleaner) config(v *viper.Viper, cmd *cobra.Command) error {
	allOpt, err := option.ParseOptions(c.optionFile, v, cmd.PersistentFlags())
	if err != nil {
		return err
	}
	c.TestOptions = allOpt.Test

	c.w = cmd.OutOrStdout()

	return nil
}

func (c *cleaner) clean() error {
	cli, err := docker.NewCompose(c.w)
	if err != nil {
		return errors.Wrap(err, "failed to build the docker compose client")
	}

	err = cli.Down(context.TODO(), c.Name)
	if err != nil {
		return errors.Wrapf(err, "failed to stop the test environment %q", c.Name)
	}
	fmt.Fprintf(c.w, "Stopped the test environment %q\n", c.Name)

	source, err := utils.GetAbsolutePath(c.TestPath)
	if err != nil {
		return errors.Wrapf(err, "invalid test configuration source %q", c.TestPath)
	}
	err = os.RemoveAll(source)
	if err != nil {
		return errors.Wrapf(err, "failed to remove the test configuration source %q", source)
	}
	fmt.Fprintf(c.w, "Removed the source %q\n", source)

	return nil
}
