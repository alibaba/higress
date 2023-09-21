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
	"os"

	"github.com/alibaba/higress/pkg/cmd/hgctl/helm"
	"github.com/alibaba/higress/pkg/cmd/hgctl/util"
	"sigs.k8s.io/yaml"
)

type ComponentName string

const (
	Higress ComponentName = "higress"
	Istio   ComponentName = "istio"
)

var ComponentMap = map[string]ComponentName{
	"higress": Higress,
	"istio":   Istio,
}

type Component interface {
	// ComponentName returns the name of the component.
	ComponentName() ComponentName
	// Namespace returns the namespace for the component.
	Namespace() string
	// Enabled reports whether the component is enabled.
	Enabled() bool
	// Run starts the component. Must be called before the component is used.
	Run() error
	RenderManifest() (string, error)
}

type ComponentOptions struct {
	Name      string
	Namespace string
	// local
	ChartPath string
	// remote
	RepoURL   string
	ChartName string
	Version   string
}

type ComponentOption func(*ComponentOptions)

func WithComponentNamespace(namespace string) ComponentOption {
	return func(opts *ComponentOptions) {
		opts.Namespace = namespace
	}
}

func WithComponentChartPath(path string) ComponentOption {
	return func(opts *ComponentOptions) {
		opts.ChartPath = path
	}
}

func WithComponentChartName(chartName string) ComponentOption {
	return func(opts *ComponentOptions) {
		opts.ChartName = chartName
	}
}

func WithComponentRepoURL(url string) ComponentOption {
	return func(opts *ComponentOptions) {
		opts.RepoURL = url
	}
}

func WithComponentVersion(version string) ComponentOption {
	return func(opts *ComponentOptions) {
		opts.Version = version
	}
}

type HigressComponent struct {
	profile  *helm.Profile
	started  bool
	opts     *ComponentOptions
	renderer helm.Renderer
	writer   io.Writer
}

func (h *HigressComponent) ComponentName() ComponentName {
	return Higress
}

func (h *HigressComponent) Namespace() string {
	return h.opts.Namespace
}

func (h *HigressComponent) Enabled() bool {
	return true
}

func (h *HigressComponent) Run() error {
	// Parse latest version
	if h.opts.Version == helm.RepoLatestVersion {
		fmt.Fprintf(h.writer, "start to get higress helm chart latest version ......")
	}
	latestVersion, err := helm.ParseLatestVersion(h.opts.RepoURL, h.opts.Version)
	if err != nil {
		return err
	}
	fmt.Fprintf(h.writer, "latest version is %s\n", latestVersion)

	// Reset helm chart version
	h.opts.Version = latestVersion
	h.renderer.SetVersion(latestVersion)
	fmt.Fprintf(h.writer, "start to download higress helm chart version: %s, url: %s\n", h.opts.Version, h.opts.RepoURL)

	if err := h.renderer.Init(); err != nil {
		return err
	}
	h.started = true
	return nil
}

func (h *HigressComponent) RenderManifest() (string, error) {
	if !h.started {
		return "", nil
	}
	fmt.Fprintf(h.writer, "start to render higress helm chart......\n")
	valsYaml, err := h.profile.ValuesYaml()
	if err != nil {
		return "", err
	}
	manifest, err2 := renderComponentManifest(valsYaml, h.renderer, true, h.ComponentName(), h.opts.Namespace)
	if err2 != nil {
		return "", err
	}
	return manifest, nil
}

func NewHigressComponent(profile *helm.Profile, writer io.Writer, opts ...ComponentOption) (Component, error) {
	newOpts := &ComponentOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}

	var renderer helm.Renderer
	var err error
	if newOpts.RepoURL != "" {
		renderer, err = helm.NewRemoteRenderer(
			helm.WithName(newOpts.ChartName),
			helm.WithNamespace(newOpts.Namespace),
			helm.WithRepoURL(newOpts.RepoURL),
			helm.WithVersion(newOpts.Version),
		)
		if err != nil {
			return nil, err
		}
	} else {
		renderer, err = helm.NewLocalRenderer(
			helm.WithName(newOpts.ChartName),
			helm.WithNamespace(newOpts.Namespace),
			helm.WithVersion(newOpts.Version),
			helm.WithFS(os.DirFS(newOpts.ChartPath)),
			helm.WithDir(string(Higress)),
		)
		if err != nil {
			return nil, err
		}
	}

	higressComponent := &HigressComponent{
		profile:  profile,
		renderer: renderer,
		opts:     newOpts,
		writer:   writer,
	}
	return higressComponent, nil
}

type IstioCRDComponent struct {
	profile  *helm.Profile
	started  bool
	opts     *ComponentOptions
	renderer helm.Renderer
	writer   io.Writer
}

func NewIstioCRDComponent(profile *helm.Profile, writer io.Writer, opts ...ComponentOption) (Component, error) {
	newOpts := &ComponentOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}

	var renderer helm.Renderer
	var err error
	if newOpts.RepoURL != "" {
		renderer, err = helm.NewRemoteRenderer(
			helm.WithName(newOpts.ChartName),
			helm.WithNamespace(newOpts.Namespace),
			helm.WithRepoURL(newOpts.RepoURL),
			helm.WithVersion(newOpts.Version),
		)
		if err != nil {
			return nil, err
		}
	} else {
		renderer, err = helm.NewLocalRenderer(
			helm.WithName(newOpts.ChartName),
			helm.WithNamespace(newOpts.Namespace),
			helm.WithVersion(newOpts.Version),
			helm.WithFS(os.DirFS(newOpts.ChartPath)),
			helm.WithDir(string(Istio)),
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
	fmt.Fprintf(i.writer, "start to download istio helm chart version: %s, url: %s\n", i.opts.Version, i.opts.RepoURL)
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
	fmt.Fprintf(i.writer, "start to render istio helm chart......\n")
	values := make(map[string]any)
	manifest, err := renderComponentManifest(values, i.renderer, false, i.ComponentName(), i.opts.Namespace)
	if err != nil {
		return "", err
	}
	return manifest, nil
}

func renderComponentManifest(spec any, renderer helm.Renderer, addOn bool, name ComponentName, namespace string) (string, error) {
	var valsBytes []byte
	var valsYaml string
	var err error
	if yamlString, ok := spec.(string); ok {
		valsYaml = yamlString
	} else {
		if !util.IsValueNil(spec) {
			valsBytes, err = yaml.Marshal(spec)
			if err != nil {
				return "", err
			}
			valsYaml = string(valsBytes)
		}
	}
	final, err := renderer.RenderManifest(valsYaml)
	if err != nil {
		return "", err
	}
	return final, nil
}
