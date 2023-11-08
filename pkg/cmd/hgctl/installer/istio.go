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
	"fmt"
	"io"
	"strings"

	"github.com/alibaba/higress/pkg/cmd/hgctl/helm"
	"github.com/alibaba/higress/pkg/cmd/hgctl/kubernetes"
	"github.com/alibaba/higress/pkg/cmd/hgctl/manifests"
)

const (
	Istio ComponentName = "istio"
)

type IstioCRDComponent struct {
	profile  *helm.Profile
	started  bool
	opts     *ComponentOptions
	renderer helm.Renderer
	writer   io.Writer
	kubeCli  kubernetes.CLIClient
}

func NewIstioCRDComponent(kubeCli kubernetes.CLIClient, profile *helm.Profile, writer io.Writer, opts ...ComponentOption) (Component, error) {
	newOpts := &ComponentOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}

	var renderer helm.Renderer
	var err error

	// Istio can be installed by embed type or remote type
	if strings.HasPrefix(newOpts.RepoURL, "embed://") {
		chartDir := strings.TrimPrefix(newOpts.RepoURL, "embed://")
		renderer, err = helm.NewLocalChartRenderer(
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
	} else {
		renderer, err = helm.NewRemoteRenderer(
			helm.WithName(newOpts.ChartName),
			helm.WithNamespace(newOpts.Namespace),
			helm.WithRepoURL(newOpts.RepoURL),
			helm.WithVersion(newOpts.Version),
			helm.WithCapabilities(newOpts.Capabilities),
			helm.WithRestConfig(kubeCli.RESTConfig()),
		)
		if err != nil {
			return nil, err
		}
	}

	istioComponent := &IstioCRDComponent{
		profile:  profile,
		renderer: renderer,
		opts:     newOpts,
		writer:   writer,
		kubeCli:  kubeCli,
	}
	return istioComponent, nil
}

func (i *IstioCRDComponent) ComponentName() ComponentName {
	return Istio
}

func (i *IstioCRDComponent) Namespace() string {
	return i.opts.Namespace
}

func (i *IstioCRDComponent) Enabled() bool {
	return true
}

func (i *IstioCRDComponent) Run() error {
	if !i.opts.Quiet {
		fmt.Fprintf(i.writer, "🏄 Downloading Istio Helm Chart version: %s, url: %s\n", i.opts.Version, i.opts.RepoURL)
	}
	if err := i.renderer.Init(); err != nil {
		return err
	}
	i.started = true
	return nil
}

func (i *IstioCRDComponent) RenderManifest() (string, error) {
	if !i.started {
		return "", nil
	}
	if !i.opts.Quiet {
		fmt.Fprintf(i.writer, "📦 Rendering Istio Helm Chart\n")
	}
	values := make(map[string]any)
	manifest, err := renderComponentManifest(values, i.renderer, false, i.ComponentName(), i.opts.Namespace)
	if err != nil {
		return "", err
	}
	return manifest, nil
}
