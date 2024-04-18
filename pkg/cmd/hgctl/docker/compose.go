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
	w      io.Writer
}

func NewCompose(w io.Writer) (*Compose, error) {
	c := &Compose{w: w}

	dockerCli, err := command.NewDockerCli(
		command.WithCombinedStreams(c.w),
		// command.WithDefaultContextStoreConfig(), Deprecated, set during NewDockerCli
	)
	if err != nil {
		return nil, err
	}

	opts := flags.NewClientOptions()
	err = dockerCli.Initialize(opts)
	if err != nil {
		return nil, err
	}
	c.client = api.NewServiceProxy().WithService(compose.NewComposeService(dockerCli.Client(), dockerCli.ConfigFile()))

	return c, nil
}

func (c Compose) Up(ctx context.Context, name string, configs []string, source string, detach bool) error {
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
		// TODO(WeixinX): Change from `Label` to `CustomLabels` after upgrading the dependency library github.com/compose-spec/compose-go
		s.Labels = map[string]string{
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
	var consumer api.LogConsumer
	if !detach {
		// TODO(WeixinX): Change to `formatter.NewLogConsumer(ctx, c.w, c.w, true, true, false)` after upgrading the dependency library github.com/compose-spec/compose-go
		consumer = formatter.NewLogConsumer(ctx, c.w, true, true)
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

func (c Compose) List(ctx context.Context) ([]api.Stack, error) {
	return c.client.List(ctx, api.ListOptions{})
}

func (c Compose) Down(ctx context.Context, name string) error {
	return c.client.Down(ctx, name, api.DownOptions{})
}

func (c Compose) Ps(ctx context.Context, name string) ([]api.ContainerSummary, error) {
	return c.client.Ps(ctx, name, api.PsOptions{})
}
