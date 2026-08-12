// Copyright 2026 Higress Authors
// Licensed under the Apache License, Version 2.0.

package main

import "testing"

func TestNextVersion(t *testing.T) {
	tests := []struct{ previous, current, want string }{
		{"1.2.3", "2.0.0", "2.0.0"},
		{"1.2.3", "2.0.0-alpha.2", "2.0.0"},
		{"1.2.3", "1.2.3", "1.2.4"},
		{"1.2.3", "1.0.0-alpha", "1.2.4"},
	}
	for _, tt := range tests {
		got, err := nextVersion(tt.previous, tt.current)
		if err != nil || got != tt.want {
			t.Fatalf("nextVersion(%q, %q) = %q, %v; want %q", tt.previous, tt.current, got, err, tt.want)
		}
	}
}

func TestNextBootstrapVersionPromotesReviewedPrerelease(t *testing.T) {
	got, err := nextBootstrapVersion("1.0.0-alpha", "1.0.0-alpha")
	if err != nil || got != "1.0.0" {
		t.Fatalf("bootstrap prerelease promotion = %q, %v", got, err)
	}
	if _, err := nextBootstrapVersion("1.0.0-alpha", "0.9.0"); err == nil {
		t.Fatal("bootstrap must not move backwards from reviewed public version")
	}
}

func TestPrereleaseOrdering(t *testing.T) {
	a, _ := parseSemver("1.0.0-alpha.2")
	b, _ := parseSemver("1.0.0-alpha.10")
	stable, _ := parseSemver("1.0.0")
	if compareSemver(a, b) >= 0 || compareSemver(b, stable) >= 0 {
		t.Fatal("SemVer prerelease ordering is incorrect")
	}
}
