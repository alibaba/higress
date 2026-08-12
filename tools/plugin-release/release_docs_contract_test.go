// Copyright 2026 Higress Authors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseDocsCreateMetadataPRAfterDependenciesConverge(t *testing.T) {
	paths := []string{
		"../../docs/developers/immutable-plugin-releases.md",
		"../../RELEASE.md",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		doc := strings.Join(strings.Fields(string(data)), " ")
		for _, required := range []string{
			"#4019", "exact main", "snapshot", "Console",
			"manually edit", "plugin `VERSION`", "Console",
			"helm dependency update", "Chart.lock", "tag authorization", "reprepare",
		} {
			if !strings.Contains(doc, required) {
				t.Fatalf("%s lacks release-cut ordering contract %q", path, required)
			}
		}
		freeze := strings.Index(doc, "exact main")
		snapshot := strings.Index(doc[freeze:], "snapshot")
		console := strings.Index(doc[freeze:], "Console")
		releasePR := strings.Index(doc[freeze:], "create or update")
		chartLock := strings.Index(doc[freeze:], "Chart.lock")
		authorize := strings.Index(doc[freeze:], "tag authorization")
		if freeze < 0 || snapshot < 0 || console < 0 || releasePR < 0 || chartLock < 0 || authorize < 0 || snapshot > console || console > releasePR || releasePR > chartLock || chartLock > authorize {
			t.Fatalf("%s does not order exact main -> snapshot -> Console -> #4019 -> Chart.lock -> authorization", path)
		}
	}
}
