package test

import (
	"fmt"
	"io"

	"github.com/alibaba/higress/pkg/cmd/hgctl/docker"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func newStartCommand() *cobra.Command {
	var (
		name   string
		config string
		source string
		detach bool
	)

	startCmd := &cobra.Command{
		Use:     "start",
		Aliases: []string{"s"},
		Short:   "Start the test environment, similar to `docker compose up`",
		Example: `  # Run containers in the background with the option --detach(-d)
  hgctl plugin test start -d
  `,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(start(cmd.OutOrStdout(), name, config, source, detach))
		},
	}

	startCmd.PersistentFlags().StringVarP(&name, "name", "p", "wasm-test", "Test environment name, that is compose project name")
	startCmd.PersistentFlags().StringVarP(&config, "file", "f", "", "Compose configuration file")
	startCmd.PersistentFlags().StringVarP(&source, "source", "s", "./test", "Test configuration source")
	startCmd.PersistentFlags().BoolVarP(&detach, "detach", "d", false, "Detached mode: Run containers in the background")

	return startCmd
}

func start(w io.Writer, name, config, source string, detach bool) error {
	cli, err := docker.NewCompose(w)
	if err != nil {
		return fmt.Errorf("failed to build the docker compose client: %w", err)
	}

	var configs []string
	if config != "" {
		configs = []string{config}
	}

	fmt.Fprintf(w, "Start the test environment %q ...\n", name)
	err = cli.Up(w, name, configs, source, detach)
	if err != nil {
		return fmt.Errorf("failed to start test environment %q: %w", name, err)
	}

	return nil
}
