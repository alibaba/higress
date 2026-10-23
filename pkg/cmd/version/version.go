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

package version

import (
	"encoding/json"
	"fmt"
	"io"

	"sigs.k8s.io/yaml"
)

type Info struct {
	Type           string `json:"type,omitempty" yaml:"type,omitempty"`
	HigressVersion string `json:"higressVersion,omitempty" yaml:"higressVersion,omitempty"`
	GitCommitID    string `json:"gitCommitID,omitempty" yaml:"gitCommitID,omitempty"`
	GatewayVersion string `json:"gatewayVersion,omitempty" yaml:"gatewayVersion,omitempty"`
}

func Get() Info {
	return Info{
		HigressVersion: higressVersion,
		GitCommitID:    gitCommitID,
	}
}

var (
	higressVersion string
	gitCommitID    string
)

// Print shows the versions of the Envoy Gateway.
func Print(w io.Writer, format string) error {
	v := Get()
	switch format {
	case "json":
		if marshalled, err := json.MarshalIndent(v, "", "  "); err == nil {
			_, _ = fmt.Fprintln(w, string(marshalled))
		}
	case "yaml":
		if marshalled, err := yaml.Marshal(v); err == nil {
			_, _ = fmt.Fprintln(w, string(marshalled))
		}
	default:
		_, _ = fmt.Fprintf(w, "HIGRESS_VERSION: %s\n", v.HigressVersion)
		_, _ = fmt.Fprintf(w, "GIT_COMMIT_ID: %s\n", v.GitCommitID)
	}

	return nil
}
