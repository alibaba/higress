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
	Install        InstallMode `json:"install,omitempty"`
	IngressClass   string      `json:"ingressClass,omitempty"`
	WatchNamespace string      `json:"watchNamespace,omitempty"`
	DisableAlpnH2  bool        `json:"disableAlpnH2,omitempty"`
	EnableStatus   bool        `json:"enableStatus,omitempty"`
	EnableIstioAPI bool        `json:"enableIstioAPI,omitempty"`
	Namespace      string      `json:"namespace,omitempty"`
	IstioNamespace string      `json:"istioNamespace,omitempty"`
}

func (p ProfileGlobal) SetFlags(install InstallMode) ([]string, error) {
	sets := make([]string, 0)
	sets = append(sets, fmt.Sprintf("global.ingressClass=%s", p.IngressClass))
	sets = append(sets, fmt.Sprintf("global.watchNamespace=%s", p.WatchNamespace))
	sets = append(sets, fmt.Sprintf("global.disableAlpnH2=%t", p.DisableAlpnH2))
	sets = append(sets, fmt.Sprintf("global.enableStatus=%t", p.EnableStatus))
	sets = append(sets, fmt.Sprintf("global.enableIstioAPI=%t", p.EnableIstioAPI))
	sets = append(sets, fmt.Sprintf("global.istioNamespace=%s", p.IstioNamespace))
	if install == InstallLocalK8s {
		sets = append(sets, fmt.Sprintf("global.local=%t", true))
	}
	return sets, nil
}

func (p ProfileGlobal) Validate(install InstallMode) []error {
	errs := make([]error, 0)
	// now only support k8s and local-k8s installation mode
	if p.Install != InstallK8s && p.Install != InstallLocalK8s {
		errs = append(errs, errors.New("global.install only can be set to k8s or local-k8s"))
	}
	if len(p.IngressClass) == 0 {
		errs = append(errs, errors.New("global.ingressClass can't be empty"))
	}
	if len(p.Namespace) == 0 {
		errs = append(errs, errors.New("global.namespace can't be empty"))
	}
	if len(p.IstioNamespace) == 0 {
		errs = append(errs, errors.New("global.istioNamespace can't be empty"))
	}
	return errs
}

type ProfileConsole struct {
	Port                uint32 `json:"port,omitempty"`
	Replicas            uint32 `json:"replicas,omitempty"`
	ServiceType         string `json:"serviceType,omitempty"`
	Domain              string `json:"domain,omitempty"`
	TlsSecretName       string `json:"tlsSecretName,omitempty"`
	WebLoginPrompt      string `json:"webLoginPrompt,omitempty"`
	AdminPasswordValue  string `json:"adminPasswordValue,omitempty"`
	AdminPasswordLength uint32 `json:"adminPasswordLength,omitempty"`
	O11yEnabled         bool   `json:"o11YEnabled,omitempty"`
	PvcRwxSupported     bool   `json:"pvcRwxSupported,omitempty"`
}

func (p ProfileConsole) SetFlags(install InstallMode) ([]string, error) {
	sets := make([]string, 0)
	sets = append(sets, fmt.Sprintf("higress-console.replicaCount=%d", p.Replicas))
	sets = append(sets, fmt.Sprintf("higress-console.service.type=%s", p.ServiceType))
	sets = append(sets, fmt.Sprintf("higress-console.domain=%s", p.Domain))
	sets = append(sets, fmt.Sprintf("higress-console.tlsSecretName=%s", p.TlsSecretName))
	sets = append(sets, fmt.Sprintf("higress-console.web.login.prompt=%s", p.WebLoginPrompt))
	sets = append(sets, fmt.Sprintf("higress-console.admin.password.value=%s", p.AdminPasswordValue))
	sets = append(sets, fmt.Sprintf("higress-console.admin.password.length=%d", p.AdminPasswordLength))
	sets = append(sets, fmt.Sprintf("higress-console.o11y.enabled=%t", p.O11yEnabled))
	sets = append(sets, fmt.Sprintf("higress-console.pvc.rwxSupported=%t", p.PvcRwxSupported))
	return sets, nil
}

func (p ProfileConsole) Validate(install InstallMode) []error {
	errs := make([]error, 0)
	if p.Replicas <= 0 {
		errs = append(errs, errors.New("console.replica need be large than zero"))
	}

	if p.ServiceType != "ClusterIP" && p.ServiceType != "NodePort" && p.ServiceType != "LoadBalancer" {
		errs = append(errs, errors.New("console.serviceType can only be set to ClusterIP, NodePort or LoadBalancer"))
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
	sets = append(sets, fmt.Sprintf("higress-core.gateway.replicas=%d", p.Replicas))
	return sets, nil
}

func (p ProfileGateway) Validate(install InstallMode) []error {
	errs := make([]error, 0)
	if p.Replicas <= 0 {
		errs = append(errs, errors.New("gateway.replica need be large than zero"))
	}

	return errs
}

type ProfileController struct {
	Replicas uint32 `json:"replicas,omitempty"`
}

func (p ProfileController) SetFlags(install InstallMode) ([]string, error) {
	sets := make([]string, 0)
	sets = append(sets, fmt.Sprintf("higress-core.controller.replicas=%d", p.Replicas))
	return sets, nil
}

func (p ProfileController) Validate(install InstallMode) []error {
	errs := make([]error, 0)
	if p.Replicas <= 0 {
		errs = append(errs, errors.New("controller.replica need be large than zero"))
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
	return errs
}

type Chart struct {
	Url     string `json:"url,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type ProfileCharts struct {
	Higress Chart `json:"higress,omitempty"`
	Istio   Chart `json:"istio,omitempty"`
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
	// merge values and setFlags
	overlayYAML, err := overlaySetFlagValues(valueOverlayYAML, setFlags)
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
	if len(errsController) > 0 {
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
