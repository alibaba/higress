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
