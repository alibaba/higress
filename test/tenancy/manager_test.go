package tenancy_test

import (
	"testing"

	"github.com/alibaba/higress/pkg/tenancy"
)

func TestIsolateRoutes(t *testing.T) {
	m := &tenancy.TenantManager{}
	if err := m.IsolateRoutes("team-a"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := m.IsolateRoutes("..bad"); err == nil {
		t.Fatalf("expected error for invalid namespace")
	}
}

func TestAllowedNamespace(t *testing.T) {
	m := &tenancy.TenantManager{}
	if !m.AllowedNamespace("a", []string{}) {
		t.Fatalf("empty tenant list should allow all")
	}
	if !m.AllowedNamespace("ns1", []string{"ns1", "ns2"}) {
		t.Fatalf("ns1 should be allowed")
	}
	if m.AllowedNamespace("ns3", []string{"ns1", "ns2"}) {
		t.Fatalf("ns3 should not be allowed")
	}
}