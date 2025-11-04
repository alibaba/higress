package decision

import "testing"

func TestCheckAccess_NoSubject(t *testing.T) {
    cfg := Config{
        AllowedCapabilities:       []string{"cap.a"},
        SubjectPolicy:             map[string][]string{"tenantA": {"cap.a"}},
        RequestedCapabilityHeader: "X-MCP-Capability",
    }
    in := Input{Headers: map[string]string{"X-MCP-Capability": "cap.a"}}
    r := CheckAccess(cfg, in)
    if r.Allowed {
        t.Fatalf("expected deny, got allow")
    }
    if r.Reason != "no-subject" {
        t.Fatalf("unexpected reason: %s", r.Reason)
    }
}

func TestCheckAccess_AllowedIntersection(t *testing.T) {
    cfg := Config{
        AllowedCapabilities:       []string{"cap.a", "cap.b"},
        SubjectPolicy:             map[string][]string{"tenantA": {"cap.a"}},
        RequestedCapabilityHeader: "X-MCP-Capability",
    }
    in := Input{Headers: map[string]string{"X-Subject": "tenantA", "X-MCP-Capability": "cap.a"}}
    r := CheckAccess(cfg, in)
    if !r.Allowed {
        t.Fatalf("expected allow, got deny: %s", r.Reason)
    }
}

func TestCheckAccess_DenyWrongCap(t *testing.T) {
    cfg := Config{
        AllowedCapabilities:       []string{"cap.a", "cap.b"},
        SubjectPolicy:             map[string][]string{"tenantA": {"cap.a"}},
        RequestedCapabilityHeader: "X-MCP-Capability",
    }
    in := Input{Headers: map[string]string{"X-Subject": "tenantA", "X-MCP-Capability": "cap.c"}}
    r := CheckAccess(cfg, in)
    if r.Allowed || r.Reason != "requested-cap-not-allowed" {
        t.Fatalf("expected deny requested-cap-not-allowed, got allowed=%v reason=%s", r.Allowed, r.Reason)
    }
}

func TestCheckAccess_AllowedWhenReqCapEmpty(t *testing.T) {
    cfg := Config{
        AllowedCapabilities: []string{"cap.a"},
        SubjectPolicy:       map[string][]string{"tenantA": {"cap.a"}},
    }
    in := Input{Headers: map[string]string{"X-Subject": "tenantA"}}
    r := CheckAccess(cfg, in)
    if !r.Allowed {
        t.Fatalf("expected allow, got deny: %s", r.Reason)
    }
}

