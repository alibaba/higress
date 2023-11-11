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

package installer

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/alibaba/higress/pkg/cmd/hgctl/helm"
	"github.com/alibaba/higress/pkg/cmd/hgctl/kubernetes"
	"github.com/alibaba/higress/pkg/cmd/hgctl/manifests"
)

const (
	GatewayAPI ComponentName = "gatewayAPI"
)

type GatewayAPIComponent struct {
	profile  *helm.Profile
	started  bool
	opts     *ComponentOptions
	renderer helm.Renderer
	writer   io.Writer
	kubeCli  kubernetes.CLIClient
}

func NewGatewayAPIComponent(kubeCli kubernetes.CLIClient, profile *helm.Profile, writer io.Writer, opts ...ComponentOption) (Component, error) {
	newOpts := &ComponentOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}

	if !strings.HasPrefix(newOpts.RepoURL, "embed://") {
		return nil, errors.New("GatewayAPI Url need start with embed://")
	}

	chartDir := strings.TrimPrefix(newOpts.RepoURL, "embed://")
	// GatewayAPI can only be installed by embed type
	renderer, err := helm.NewLocalFileRenderer(
		helm.WithName(newOpts.ChartName),
		helm.WithNamespace(newOpts.Namespace),
		helm.WithRepoURL(newOpts.RepoURL),
		helm.WithVersion(newOpts.Version),
		helm.WithFS(manifests.BuiltinOrDir("")),
		helm.WithDir(chartDir),
		helm.WithCapabilities(newOpts.Capabilities),
		helm.WithRestConfig(kubeCli.RESTConfig()),
	)
	if err != nil {
		return nil, err
	}

	gatewayAPIComponent := &GatewayAPIComponent{
		profile:  profile,
		renderer: renderer,
		opts:     newOpts,
		writer:   writer,
		kubeCli:  kubeCli,
	}
	return gatewayAPIComponent, nil
}

func (i *GatewayAPIComponent) ComponentName() ComponentName {
	return GatewayAPI
}

func (i *GatewayAPIComponent) Namespace() string {
	return i.opts.Namespace
}

func (i *GatewayAPIComponent) Enabled() bool {
	return true
}

func (i *GatewayAPIComponent) Run() error {
	if !i.opts.Quiet {
		fmt.Fprintf(i.writer, "🏄 Downloading GatewayAPI Yaml Files version: %s, url: %s\n", i.opts.Version, i.opts.RepoURL)
	}
	if err := i.renderer.Init(); err != nil {
		return err
	}
	i.started = true
	return nil
}

func (i *GatewayAPIComponent) RenderManifest() (string, error) {
	if !i.started {
		return "", nil
	}
	if !i.opts.Quiet {
		fmt.Fprintf(i.writer, "📦 Rendering GatewayAPI Yaml Files\n")
	}
	values := make(map[string]any)
	manifest, err := renderComponentManifest(values, i.renderer, false, i.ComponentName(), i.opts.Namespace)
	if err != nil {
		return "", err
	}
	return manifest, nil
}
