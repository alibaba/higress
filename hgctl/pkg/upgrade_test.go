// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hgctl

import (
	"testing"

	"github.com/alibaba/higress/hgctl/pkg/helm"
)

func TestValidateLocalDockerUpgradeOverlay(t *testing.T) {
	baseline := &helm.Profile{
		Global:  helm.ProfileGlobal{Install: helm.InstallLocalDocker},
		Gateway: helm.ProfileGateway{HttpPort: 80},
	}

	if err := validateLocalDockerUpgradeOverlay(baseline, "gateway:\n  httpPort: 18080\n"); err == nil || err.Error() != "local-docker upgrade does not support overlay fields: gateway.httpPort" {
		t.Fatalf("validateLocalDockerUpgradeOverlay() error = %v", err)
	}
	if err := validateLocalDockerUpgradeOverlay(baseline, "charts:\n  standalone:\n    url: https://example.com/get-higress.sh\n"); err != nil {
		t.Fatalf("validateLocalDockerUpgradeOverlay() allowed URL error = %v", err)
	}
}
