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
	"sigs.k8s.io/yaml"
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

func (h *HelmAgent) GetHigressInformance() (bool, map[string]any, error) {
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
		return false, nil, nil
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
			split := strings.Split(content, "\n")
			for i, line := range split {
				if i == 0 {
					continue
				}
				param := strings.Split(line, "\t")
				if len(param) != 7 {
					continue
				}
				// check chart contains higress
				if strings.Contains(param[5], "higress") && strings.Contains(param[4], "deployed") {
					releaseName := param[0]
					valueFlag := h.getValueFlag(releaseName)
					return true, valueFlag, nil
				}
			}
		}
	}
	return false, nil, nil
}

func (h *HelmAgent) getValueFlag(releaseName string) map[string]any {
	args := []string{"status", releaseName, "-n", h.profile.Global.Namespace, "-o", "yaml"}
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
		return nil
	}

	done := make(chan error, 1)

	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err == nil {
			content := out.String()
			statusMap := make(map[string]any)
			err = yaml.Unmarshal([]byte(content), &statusMap)
			if err != nil {
				return nil
			}
			if config, ok := statusMap["config"]; ok {
				if valueMap, ok := config.(map[string]any); ok {
					return valueMap
				}
			}
		}
	}
	return nil
}
