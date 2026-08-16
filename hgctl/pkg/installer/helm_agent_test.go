package installer

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/alibaba/higress/hgctl/pkg/helm"
)

func TestHelmAgentIsHigressInstalled(t *testing.T) {
	profile := &helm.Profile{Global: helm.ProfileGlobal{Namespace: "higress-system"}}
	tests := []struct {
		name          string
		mode          string
		wantInstalled bool
		wantErr       string
	}{
		{name: "missing binary", mode: "missing", wantErr: "start helm ownership check"},
		{name: "command failure", mode: "failure", wantErr: "helm ownership check failed"},
		{name: "not installed", mode: "absent"},
		{name: "deployed release", mode: "deployed", wantInstalled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binary := os.Args[0]
			if tt.mode == "missing" {
				binary = "helm-not-found-for-test"
			} else {
				t.Setenv("HGCTL_HELM_TEST_MODE", tt.mode)
			}
			agent := NewHelmAgent(profile, &bytes.Buffer{}, true, WithHelmBinaryName(binary))
			installed, err := agent.IsHigressInstalled()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("IsHigressInstalled() error = %v, want substring %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("IsHigressInstalled() error = %v", err)
			}
			if installed != tt.wantInstalled {
				t.Fatalf("IsHigressInstalled() = %t, want %t", installed, tt.wantInstalled)
			}
		})
	}
}

func TestMain(m *testing.M) {
	mode := os.Getenv("HGCTL_HELM_TEST_MODE")
	if mode != "" {
		runHelmHelper(mode)
		return
	}
	os.Exit(m.Run())
}

func runHelmHelper(mode string) {
	switch mode {
	case "failure":
		fmt.Fprint(os.Stderr, "helm cluster access failed")
		os.Exit(1)
	case "absent":
		fmt.Fprint(os.Stdout, "NAME\tNAMESPACE\tREVISION\tUPDATED\tSTATUS\tCHART\tAPP VERSION\n")
	case "deployed":
		fmt.Fprint(os.Stdout, "higress\thigress-system\t1\t2026-01-01\tdeployed\thigress\t2.2.4\n")
	default:
		fmt.Fprintf(os.Stderr, "unknown test mode %q", mode)
		os.Exit(2)
	}
}
