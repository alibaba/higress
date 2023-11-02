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
	"github.com/alibaba/higress/pkg/cmd/hgctl/helm"
	"github.com/alibaba/higress/pkg/cmd/hgctl/util"
	"io"
	"os"
)

type DockerInstaller struct {
	started    bool
	standalone *StandaloneComponent
	profile    *helm.Profile
	writer     io.Writer
}

func (d *DockerInstaller) Install() error {
	fmt.Fprintf(d.writer, "\n⌛️ Processing installation... \n\n")

	if err := d.standalone.Install(); err != nil {
		return err
	}

	profileName, _ := GetInstalledYamlPath()
	fmt.Fprintf(d.writer, "\n✔️ Wrote Profile: \"%s\" \n", profileName)
	if err := util.WriteFileString(profileName, util.ToYAML(d.profile), 0o644); err != nil {
		return err
	}

	fmt.Fprintf(d.writer, "\n🎊 Install All Resources Complete!\n")
	return nil
}

func (d *DockerInstaller) UnInstall() error {

	fmt.Fprintf(d.writer, "\n⌛️ Processing uninstallation... \n\n")

	if err := d.standalone.UnInstall(); err != nil {
		return err
	}

	profileName, _ := GetInstalledYamlPath()
	fmt.Fprintf(d.writer, "\n✔️ Removed Profile: \"%s\" \n", profileName)
	os.Remove(profileName)

	fmt.Fprintf(d.writer, "\n🎊 Uninstall All Resources Complete!\n")
	return nil
}

func (d *DockerInstaller) Upgrade() error {
	fmt.Fprintf(d.writer, "\n⌛️ Processing upgrade... \n\n")

	if err := d.standalone.Upgrade(); err != nil {
		return err
	}

	fmt.Fprintf(d.writer, "\n🎊 Install All Resources Complete!\n")
	return nil
}

func NewDockerInstaller(profile *helm.Profile, writer io.Writer, quiet bool) (*DockerInstaller, error) {
	if profile == nil {
		return nil, errors.New("install profile is empty")
	}
	// initialize components
	opts := []ComponentOption{
		WithComponentVersion(profile.Charts.Standalone.Version),
		WithComponentRepoURL(profile.Charts.Standalone.Url),
		WithComponentChartName(profile.Charts.Standalone.Name),
	}
	if quiet {
		opts = append(opts, WithQuiet())
	}
	standaloneComponent, err := NewStandaloneComponent(profile, writer, opts...)
	if err != nil {
		return nil, fmt.Errorf("NewStandaloneComponent failed, err: %s", err)
	}

	op := &DockerInstaller{
		profile:    profile,
		standalone: standaloneComponent,
		writer:     writer,
	}
	return op, nil
}
