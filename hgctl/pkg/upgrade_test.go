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
