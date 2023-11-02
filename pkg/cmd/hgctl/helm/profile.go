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

package helm

import (
	"errors"
	"fmt"
	"strings"

	"istio.io/istio/operator/pkg/util"
	"sigs.k8s.io/yaml"
)

type InstallMode string

const (
	InstallK8s         InstallMode = "k8s"
	InstallLocalK8s    InstallMode = "local-k8s"
	InstallLocalDocker InstallMode = "local-docker"
	InstallLocal       InstallMode = "local"
)

type Profile struct {
	Profile            string            `json:"profile,omitempty"`
	InstallPackagePath string            `json:"installPackagePath,omitempty"`
	Global             ProfileGlobal     `json:"global,omitempty"`
	Console            ProfileConsole    `json:"console,omitempty"`
	Gateway            ProfileGateway    `json:"gateway,omitempty"`
	Controller         ProfileController `json:"controller,omitempty"`
	Storage            ProfileStorage    `json:"storage,omitempty"`
	Values             map[string]any    `json:"values,omitempty"`
	Charts             ProfileCharts     `json:"charts,omitempty"`
}

type ProfileGlobal struct {
	Install          InstallMode `json:"install,omitempty"`
	IngressClass     string      `json:"ingressClass,omitempty"`
	EnableIstioAPI   bool        `json:"enableIstioAPI,omitempty"`
	EnableGatewayAPI bool        `json:"enableGatewayAPI,omitempty"`
	Namespace        string      `json:"namespace,omitempty"`
}

func (p ProfileGlobal) SetFlags(install InstallMode) ([]string, error) {
	sets := make([]string, 0)
	if install == InstallK8s || install == InstallLocalK8s {
		sets = append(sets, fmt.Sprintf("global.ingressClass=%s", p.IngressClass))
		sets = append(sets, fmt.Sprintf("global.enableIstioAPI=%t", p.EnableIstioAPI))
		sets = append(sets, fmt.Sprintf("global.enableGatewayAPI=%t", p.EnableGatewayAPI))
		if install == InstallLocalK8s {
			sets = append(sets, fmt.Sprintf("global.local=%t", true))
		}
	}
	return sets, nil
}

func (p ProfileGlobal) Validate(install InstallMode) []error {
	errs := make([]error, 0)
	// now only support k8s, local-k8s, local-docker installation mode
	if install != InstallK8s && install != InstallLocalK8s && install != InstallLocalDocker {
		errs = append(errs, errors.New("global.install only can be set to k8s, local-k8s or local-docker"))
	}
	if install == InstallK8s || install == InstallLocalK8s {
		if len(p.IngressClass) == 0 {
			errs = append(errs, errors.New("global.ingressClass can't be empty"))
		}
		if len(p.Namespace) == 0 {
			errs = append(errs, errors.New("global.namespace can't be empty"))
		}
	}
	return errs
}

type ProfileConsole struct {
	Port        uint32 `json:"port,omitempty"`
	Replicas    uint32 `json:"replicas,omitempty"`
	O11yEnabled bool   `json:"o11YEnabled,omitempty"`
}

func (p ProfileConsole) SetFlags(install InstallMode) ([]string, error) {
	sets := make([]string, 0)
	if install == InstallK8s || install == InstallLocalK8s {
		sets = append(sets, fmt.Sprintf("higress-console.replicaCount=%d", p.Replicas))
		sets = append(sets, fmt.Sprintf("higress-console.o11y.enabled=%t", p.O11yEnabled))
	}
	return sets, nil
}

func (p ProfileConsole) Validate(install InstallMode) []error {
	errs := make([]error, 0)
	if install == InstallK8s || install == InstallLocalK8s {
		if p.Replicas <= 0 {
			errs = append(errs, errors.New("console.replica need be large than zero"))
		}
	}

	if install == InstallLocalDocker {
		if p.Port <= 0 {
			errs = append(errs, errors.New("console.port need be large than zero"))
		}
	}

	return errs
}

type ProfileGateway struct {
	Replicas    uint32 `json:"replicas,omitempty"`
	HttpPort    uint32 `json:"httpPort,omitempty"`
	HttpsPort   uint32 `json:"httpsPort,omitempty"`
	MetricsPort uint32 `json:"metricsPort,omitempty"`
}

func (p ProfileGateway) SetFlags(install InstallMode) ([]string, error) {
	sets := make([]string, 0)
	if install == InstallK8s || install == InstallLocalK8s {
		sets = append(sets, fmt.Sprintf("higress-core.gateway.replicas=%d", p.Replicas))
	}
	return sets, nil
}

func (p ProfileGateway) Validate(install InstallMode) []error {
	errs := make([]error, 0)
	if install == InstallK8s || install == InstallLocalK8s {
		if p.Replicas <= 0 {
			errs = append(errs, errors.New("gateway.replica need be large than zero"))
		}
	}

	if install == InstallLocalDocker {
		if p.HttpPort <= 0 {
			errs = append(errs, errors.New("gateway.httpPort need be large than zero"))
		}
		if p.HttpsPort <= 0 {
			errs = append(errs, errors.New("gateway.httpsPort need be large than zero"))
		}
		if p.MetricsPort <= 0 {
			errs = append(errs, errors.New("gateway.MetricsPort need be large than zero"))
		}
	}
	return errs
}

type ProfileController struct {
	Replicas uint32 `json:"replicas,omitempty"`
}

func (p ProfileController) SetFlags(install InstallMode) ([]string, error) {
	sets := make([]string, 0)
	if install == InstallK8s || install == InstallLocalK8s {
		sets = append(sets, fmt.Sprintf("higress-core.controller.replicas=%d", p.Replicas))
	}
	return sets, nil
}

func (p ProfileController) Validate(install InstallMode) []error {
	errs := make([]error, 0)
	if install == InstallK8s || install == InstallLocalK8s {
		if p.Replicas <= 0 {
			errs = append(errs, errors.New("controller.replica need be large than zero"))
		}
	}
	return errs
}

type ProfileStorage struct {
	Url        string `json:"url,omitempty"`
	Ns         string `json:"ns,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	DataEncKey string `json:"DataEncKey,omitempty"`
}

func (p ProfileStorage) Validate(install InstallMode) []error {
	errs := make([]error, 0)
	if install == InstallLocalDocker {
		if len(p.Url) == 0 {
			errs = append(errs, errors.New("storage.url can't be empty"))
		}
		if len(p.Ns) == 0 {
			errs = append(errs, errors.New("storage.ns can't be empty"))
		}

		if !strings.HasPrefix(p.Url, "nacos://") && !strings.HasPrefix(p.Url, "file://") {
			errs = append(errs, fmt.Errorf("invalid storage url: %s", p.Url))
		} else {
			// check localhost or 127.0.0.0
			if strings.Contains(p.Url, "localhost") || strings.Contains(p.Url, "/127.") {
				errs = append(errs, errors.New("localhost or loopback addresses in nacos url won't work"))
			}
		}

		if len(p.DataEncKey) > 0 && len(p.DataEncKey) != 32 {
			errs = append(errs, fmt.Errorf("expecting 32 characters for dataEncKey, but got %d length", len(p.DataEncKey)))
		}

		if len(p.Username) > 0 && len(p.Password) == 0 || len(p.Username) == 0 && len(p.Password) > 0 {
			errs = append(errs, errors.New("both nacos username and password should be provided"))
		}
	}
	return errs
}

type Chart struct {
	Url     string `json:"url,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type ProfileCharts struct {
	Higress    Chart `json:"higress,omitempty"`
	Standalone Chart `json:"standalone,omitempty"`
}

func (p ProfileCharts) Validate(install InstallMode) []error {
	errs := make([]error, 0)

	return errs
}

func (p *Profile) ValuesYaml() (string, error) {
	setFlags := make([]string, 0)
	// Get global setting
	globalFlags, _ := p.Global.SetFlags(p.Global.Install)
	setFlags = append(setFlags, globalFlags...)

	// Get console setting
	consoleFlags, _ := p.Console.SetFlags(p.Global.Install)
	setFlags = append(setFlags, consoleFlags...)

	// Get gateway setting
	gatewayFlags, _ := p.Gateway.SetFlags(p.Global.Install)
	setFlags = append(setFlags, gatewayFlags...)

	// Get controller setting
	controllerFlags, _ := p.Controller.SetFlags(p.Global.Install)
	setFlags = append(setFlags, controllerFlags...)

	valueOverlayYAML := ""
	if p.Values != nil {
		out, err := yaml.Marshal(p.Values)
		if err != nil {
			return "", err
		}
		valueOverlayYAML = string(out)
	}

	flagsYAML, err := overlaySetFlagValues("", setFlags)
	if err != nil {
		return "", err
	}
	// merge values and setFlags
	overlayYAML, err := util.OverlayYAML(flagsYAML, valueOverlayYAML)
	if err != nil {
		return "", err
	}
	return overlayYAML, nil
}

func (p *Profile) IstioEnabled() bool {
	if (p.Global.Install == InstallK8s || p.Global.Install == InstallLocalK8s) && p.Global.EnableIstioAPI {
		return true
	}
	return false
}

func (p *Profile) GatewayAPIEnabled() bool {
	if (p.Global.Install == InstallK8s || p.Global.Install == InstallLocalK8s) && p.Global.EnableGatewayAPI {
		return true
	}
	return false
}

func (p *Profile) GetIstioNamespace() string {
	if valuesGlobal, ok1 := p.Values["global"]; ok1 {
		if global, ok2 := valuesGlobal.(map[string]any); ok2 {
			if istioNamespace, ok3 := global["istioNamespace"]; ok3 {
				if namespace, ok4 := istioNamespace.(string); ok4 {
					return namespace
				}
			}
		}
	}
	return ""
}

func (p *Profile) Validate() error {
	errs := make([]error, 0)
	errsGlobal := p.Global.Validate(p.Global.Install)
	if len(errsGlobal) > 0 {
		errs = append(errs, errsGlobal...)
	}
	errsConsole := p.Console.Validate(p.Global.Install)
	if len(errsConsole) > 0 {
		errs = append(errs, errsConsole...)
	}
	errsGateway := p.Gateway.Validate(p.Global.Install)
	if len(errsGateway) > 0 {
		errs = append(errs, errsGateway...)
	}
	errsController := p.Controller.Validate(p.Global.Install)
	if len(errsController) > 0 {
		errs = append(errs, errsController...)
	}
	errsStorage := p.Storage.Validate(p.Global.Install)
	if len(errsStorage) > 0 {
		errs = append(errs, errsStorage...)
	}
	errsCharts := p.Charts.Validate(p.Global.Install)
	if len(errsCharts) > 0 {
		errs = append(errs, errsCharts...)
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.New(ToString(errs, "\n"))
}

// ToString returns a string representation of errors, with elements separated by separator string. Any nil errors in the
// slice are skipped.
func ToString(errors []error, separator string) string {
	var out string
	for i, e := range errors {
		if e == nil {
			continue
		}
		if i != 0 {
			out += separator
		}
		out += e.Error()
	}
	return out
}
