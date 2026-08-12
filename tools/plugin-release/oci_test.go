package main

import "testing"

func TestDigestReferencePreservesRegistryPort(t *testing.T) {
	got, err := digestReference("oci://registry.example:5000/plugins/demo:1.2.3", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil || got != "registry.example:5000/plugins/demo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("digestReference=%q, %v", got, err)
	}
}
