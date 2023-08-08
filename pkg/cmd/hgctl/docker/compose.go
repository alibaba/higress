package docker

import (
	"context"
	"io"
	"strings"

	"github.com/compose-spec/compose-go/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/cmd/formatter"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
)

type Compose struct {
	client *api.ServiceProxy
}

func NewCompose(w io.Writer) (*Compose, error) {
	dockerCli, err := command.NewDockerCli(
		command.WithCombinedStreams(w),
		command.WithDefaultContextStoreConfig(),
	)
	if err != nil {
		return nil, err
	}

	opts := flags.NewClientOptions()
	err = dockerCli.Initialize(opts)
	if err != nil {
		return nil, err
	}
	serviceProxy := api.NewServiceProxy().WithService(compose.NewComposeService(dockerCli))

	return &Compose{client: serviceProxy}, nil
}

func (c Compose) Up(w io.Writer, name string, configs []string, source string, detach bool) error {
	pOpts, err := cli.NewProjectOptions(
		configs,
		cli.WithWorkingDirectory(source),
		cli.WithDefaultConfigPath,
		cli.WithName(name),
	)
	if err != nil {
		return err
	}

	project, err := cli.ProjectFromOptions(pOpts)
	if err != nil {
		return err
	}

	for i, s := range project.Services {
		s.CustomLabels = map[string]string{
			api.ProjectLabel:     project.Name,
			api.ServiceLabel:     s.Name,
			api.VersionLabel:     api.ComposeVersion,
			api.WorkingDirLabel:  project.WorkingDir,
			api.ConfigFilesLabel: strings.Join(project.ComposeFiles, ","),
			api.OneoffLabel:      "False",
		}
		project.Services[i] = s
	}
	project.WithoutUnnecessaryResources()

	// for log
	ctx := context.TODO()
	var consumer api.LogConsumer
	if !detach {
		consumer = formatter.NewLogConsumer(ctx, w, w, true, true, false)
	}
	attachTo := make([]string, 0)
	for _, svc := range project.Services {
		attachTo = append(attachTo, svc.Name)
	}

	return c.client.Up(ctx, project, api.UpOptions{
		Start: api.StartOptions{
			Attach:   consumer,
			AttachTo: attachTo,
		},
	})
}

func (c Compose) List() ([]api.Stack, error) {
	return c.client.List(context.TODO(), api.ListOptions{})
}

func (c Compose) Down(name string) error {
	return c.client.Down(context.TODO(), name, api.DownOptions{})
}
