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
