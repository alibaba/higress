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

package plugin

import (
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/build"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/config"
	plugininit "github.com/alibaba/higress/pkg/cmd/hgctl/plugin/init"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/install"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/ls"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/test"
	"github.com/alibaba/higress/pkg/cmd/hgctl/plugin/uninstall"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	pluginCommand := &cobra.Command{
		Use:     "plugin",
		Aliases: []string{"plg", "p"},
		Short:   "For the Golang WASM plugin",
	}

	pluginCommand.AddCommand(build.NewCommand())
	pluginCommand.AddCommand(install.NewCommand())
	pluginCommand.AddCommand(uninstall.NewCommand())
	pluginCommand.AddCommand(ls.NewCommand())
	pluginCommand.AddCommand(test.NewCommand())
	pluginCommand.AddCommand(config.NewCommand())
	pluginCommand.AddCommand(plugininit.NewCommand())

	return pluginCommand
}
