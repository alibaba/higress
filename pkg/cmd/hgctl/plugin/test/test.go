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
