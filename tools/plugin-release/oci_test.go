package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestDigestReferencePreservesRegistryPort(t *testing.T) {
	got, err := digestReference("oci://registry.example:5000/plugins/demo:1.2.3", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil || got != "registry.example:5000/plugins/demo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("digestReference=%q, %v", got, err)
	}
}

func TestResolveOCIManifestUsesORAS123CompatibleDescriptorInvocation(t *testing.T) {
	ref := "registry.example/plugins/demo:1.2.3"
	digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	var calls [][]string
	withORASRunner(t, func(args ...string) ([]byte, string, error) {
		calls = append(calls, append([]string(nil), args...))
		switch len(calls) {
		case 1:
			return []byte(`{"digest":"` + digest + `"}`), "", nil
		case 2:
			return []byte(`{"annotations":{"org.opencontainers.image.version":"1.2.3"}}`), "", nil
		default:
			t.Fatalf("unexpected ORAS invocation %v", args)
			return nil, "", nil
		}
	})

	manifest, err := resolveOCIManifest(ref)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Digest != digest || manifest.Annotations["org.opencontainers.image.version"] != "1.2.3" {
		t.Fatalf("unexpected manifest %#v", manifest)
	}
	want := [][]string{
		{"manifest", "fetch", ref, "--descriptor"},
		{"manifest", "fetch", ref},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("ORAS calls = %#v, want %#v", calls, want)
	}
}

func TestResolveOCIManifestReportsSanitizedORASStderr(t *testing.T) {
	withORASRunner(t, func(args ...string) ([]byte, string, error) {
		return nil, "denied: token=fixture-token authorization: Bearer fixture-auth authorization: Basic Zml4dHVyZS11c2VyOmZpeHR1cmUtcGFzc3dvcmQ= https://user:fixture-password@registry.example", errors.New("exit status 1")
	})

	_, err := resolveOCIManifest("registry.example/plugins/demo:1.2.3")
	if err == nil {
		t.Fatal("expected descriptor resolution failure")
	}
	message := err.Error()
	for _, required := range []string{"oras manifest fetch --descriptor failed", "denied:", "token=[REDACTED]", "authorization: [REDACTED]", "https://[REDACTED]@registry.example"} {
		if !strings.Contains(message, required) {
			t.Fatalf("error %q lacks %q", message, required)
		}
	}
	for _, secret := range []string{"fixture-token", "fixture-auth", "Zml4dHVyZS11c2VyOmZpeHR1cmUtcGFzc3dvcmQ=", "fixture-password"} {
		if strings.Contains(message, secret) {
			t.Fatalf("error leaked test credential %q: %q", secret, message)
		}
	}
}

func withORASRunner(t *testing.T, runner func(args ...string) ([]byte, string, error)) {
	t.Helper()
	previous := orasRunner
	orasRunner = runner
	t.Cleanup(func() { orasRunner = previous })
}
