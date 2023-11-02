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
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/alibaba/higress/pkg/cmd/hgctl/helm"
	"github.com/alibaba/higress/pkg/cmd/options"
)

type HelmRelease struct {
	appVersion string `json:"app_version,omitempty"`
	chart      string `json:"chart,omitempty"`
	name       string `json:"name,omitempty"`
	namespace  string `json:"namespace,omitempty"`
	revision   string `json:"revision,omitempty"`
	status     string `json:"status,omitempty"`
	updated    string `json:"updated,omitempty"`
}

type HelmAgent struct {
	profile        *helm.Profile
	writer         io.Writer
	helmBinaryName string
	quiet          bool
}

func NewHelmAgent(profile *helm.Profile, writer io.Writer, quiet bool) *HelmAgent {
	return &HelmAgent{
		profile:        profile,
		writer:         writer,
		helmBinaryName: "helm",
		quiet:          quiet,
	}
}

func (h *HelmAgent) IsHigressInstalled() (bool, error) {
	args := []string{"list", "-n", h.profile.Global.Namespace, "-f", "higress"}
	if len(*options.DefaultConfigFlags.KubeConfig) > 0 {
		args = append(args, fmt.Sprintf("--kubeconfig=%s", *options.DefaultConfigFlags.KubeConfig))
	}
	if len(*options.DefaultConfigFlags.Context) > 0 {
		args = append(args, fmt.Sprintf("--kube-context=%s", *options.DefaultConfigFlags.Context))
	}
	if !h.quiet {
		fmt.Fprintf(h.writer, "\n📦 Running command: %s  %s\n\n", h.helmBinaryName, strings.Join(args, "  "))
	}
	cmd := exec.Command(h.helmBinaryName, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return false, nil
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err == nil {
			content := out.String()
			if !h.quiet {
				fmt.Fprintf(h.writer, "\n%s\n", content)
			}
			if strings.Contains(content, "deployed") {
				return true, nil
			}
		}
	}
	return false, nil
}
