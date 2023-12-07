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
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	testCmd := &cobra.Command{
		Use:     "test",
		Aliases: []string{"t"},
		Short:   "Test WASM plugin locally",
	}

	testCmd.AddCommand(newCreateCommand())
	testCmd.AddCommand(newStartCommand())
	testCmd.AddCommand(newStopCommand())
	testCmd.AddCommand(newCleanCommand())
	testCmd.AddCommand(newLsCommand())

	return testCmd
}
