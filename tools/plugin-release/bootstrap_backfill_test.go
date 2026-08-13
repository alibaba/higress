// Copyright 2026 Higress Authors
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

package main

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestClassifyOCIFailureDistinguishesUnauthorizedFromAbsent(t *testing.T) {
	cases := []struct {
		msg  string
		want ociFailureClass
	}{
		{"401 Unauthorized", ociFailureUnauthorized},
		{"403 Forbidden", ociFailureUnauthorized},
		{"denied: requested access to the resource is denied", ociFailureUnauthorized},
		{"UNAUTHORIZED: authentication required", ociFailureUnauthorized},
		{"authorization required", ociFailureUnauthorized},
		{"404 Not Found", ociFailureNotFound},
		{"manifest unknown", ociFailureNotFound},
		{"name unknown", ociFailureNotFound},
		{"repository does not exist", ociFailureNotFound},
		{"Error response from daemon: not found", ociFailureNotFound},
		{"connection refused", ociFailureOther},
		{"i/o timeout", ociFailureOther},
		// Authorization always wins over absence: a 401/403 is never an
		// absent artifact, even when the registry also says "unknown".
		{"403: manifest unknown", ociFailureUnauthorized},
		{"401: not found", ociFailureUnauthorized},
	}
	for _, tc := range cases {
		if got := classifyOCIFailure(errors.New(tc.msg)); got != tc.want {
			t.Fatalf("classifyOCIFailure(%q) = %d, want %d", tc.msg, got, tc.want)
		}
	}
}

func TestCaptureBootstrapEvidenceDefersAlphaWithoutResolution(t *testing.T) {
	root, catalog, commit := backfillRepo(t, map[string]string{"alpha": "1.0.0-alpha.1", "demo": "1.0.0"})
	digest := "sha256:" + strings.Repeat("a", 64)
	withManifestResolver(t, func(ref string) (ociManifest, error) {
		if strings.HasPrefix(ref, "registry.example/plugins/alpha:") {
			t.Fatalf("deferred alpha prerelease must never be resolved, got %q", ref)
		}
		if ref != "registry.example/plugins/demo:1.0.0" {
			t.Fatalf("resolved %q, want the stable demo ref", ref)
		}
		return ociManifest{Digest: digest}, nil
	})
	evidence, err := captureBootstrapEvidence(root, catalog, commit)
	if err != nil {
		t.Fatal(err)
	}
	if got := evidence.Plugins["alpha"]; got.Status != "deferred" || got.Version != "1.0.0-alpha.1" || got.PublicRef != "" || got.Digest != "" {
		t.Fatalf("alpha prerelease must be deferred without resolution: %#v", got)
	}
	if got := evidence.Plugins["demo"]; got.Status != "public" || got.Version != "1.0.0" || got.PublicRef != "registry.example/plugins/demo:1.0.0" || got.Digest != digest {
		t.Fatalf("stable public artifact lost reviewed provenance: %#v", got)
	}
}

func TestCaptureBootstrapEvidenceMarksStableAbsentForBackfill(t *testing.T) {
	root, catalog, commit := backfillRepo(t, map[string]string{"demo": "1.0.0"})
	withManifestResolver(t, func(string) (ociManifest, error) {
		return ociManifest{}, errors.New("404: manifest unknown")
	})
	evidence, err := captureBootstrapEvidence(root, catalog, commit)
	if err != nil {
		t.Fatal(err)
	}
	got := evidence.Plugins["demo"]
	if got.Status != "missing" || got.Version != "1.0.0" || got.PublicRef != "registry.example/plugins/demo:1.0.0" || got.Digest != "" {
		t.Fatalf("stable absent artifact must be classified missing for backfill: %#v", got)
	}
}

func TestCaptureBootstrapEvidenceFailsClosedForNonAlphaPrereleaseAbsent(t *testing.T) {
	root, catalog, commit := backfillRepo(t, map[string]string{"demo": "1.0.0-beta.1"})
	withManifestResolver(t, func(string) (ociManifest, error) {
		return ociManifest{}, errors.New("manifest unknown")
	})
	if _, err := captureBootstrapEvidence(root, catalog, commit); err == nil || !strings.Contains(err.Error(), "only stable versions may be backfilled") {
		t.Fatalf("absent non-alpha prerelease must fail closed, got %v", err)
	}
}

func TestCaptureBootstrapEvidenceTreatsUnauthorizedAsConfigurationError(t *testing.T) {
	for _, msg := range []string{"401 Unauthorized", "denied: requested access to the resource is denied"} {
		root, catalog, commit := backfillRepo(t, map[string]string{"demo": "1.0.0"})
		withManifestResolver(t, func(string) (ociManifest, error) {
			return ociManifest{}, errors.New(msg)
		})
		if _, err := captureBootstrapEvidence(root, catalog, commit); err == nil || !strings.Contains(err.Error(), "never an absent artifact") {
			t.Fatalf("%q must abort capture as an authorization/configuration error, got %v", msg, err)
		}
	}
}

func TestBootstrapBackfillProducesMixedSnapshotVerifiedFromBothSources(t *testing.T) {
	root, catalog, base := backfillRepo(t, map[string]string{"alpha": "1.0.0-alpha", "miss": "1.0.0", "pub": "1.0.0"})
	mustWrite(t, filepath.Join(root, "release-notes", "notes.md"), "notes\n")
	mustRun(t, root, "git", "add", ".")
	mustRun(t, root, "git", "commit", "-q", "-m", "unrelated")
	target, _ := resolveCommit(root, "HEAD")
	pubDigest := "sha256:" + strings.Repeat("a", 64)
	candidateDigest := "sha256:" + strings.Repeat("c", 64)
	candidateRef := "registry.example/candidates/miss@" + candidateDigest

	withManifestResolver(t, func(ref string) (ociManifest, error) {
		switch ref {
		case "registry.example/plugins/pub:1.0.0":
			return ociManifest{Digest: pubDigest}, nil
		case "registry.example/plugins/miss:1.0.0":
			return ociManifest{}, errors.New("404: manifest unknown")
		default:
			t.Fatalf("unexpected bootstrap resolve %q", ref)
			return ociManifest{}, nil
		}
	})
	evidence, err := captureBootstrapEvidence(root, catalog, target)
	if err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(root, "bootstrap-evidence.json")
	if err := writeCanonical(evidencePath, evidence); err != nil {
		t.Fatal(err)
	}
	bootstrapPath := filepath.Join(root, "bootstrap.json")
	if err := commandBootstrap([]string{"--root", root, "--catalog", catalog, "--gateway-version", "2.0.0", "--source", target, "--existing-evidence", evidencePath, "--output", bootstrapPath}); err != nil {
		t.Fatal(err)
	}
	var bootstrap Snapshot
	if _, err := readJSON(bootstrapPath, &bootstrap); err != nil {
		t.Fatal(err)
	}
	if bootstrap.ProvenanceMode != "bootstrap-public" || len(bootstrap.Plugins) != 1 || bootstrap.Plugins[0].LogicalID != "pub" {
		t.Fatalf("bootstrap must import only the resolvable public artifact: %#v", bootstrap.Plugins)
	}

	plan, err := buildPlan(root, catalog, bootstrapPath, base, target, "2.0.1", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Plugins) != 1 || plan.Plugins[0].LogicalID != "miss" || !plan.Plugins[0].Backfill ||
		plan.Plugins[0].Version != "1.0.0" || plan.Plugins[0].PreviousVersion != "" {
		t.Fatalf("stable absent plugin must be planned exactly once as a backfill: %#v", plan.Plugins)
	}
	if len(plan.Deferred) != 1 || plan.Deferred[0].LogicalID != "alpha" || plan.Deferred[0].Reason != "alpha-prerelease" {
		t.Fatalf("alpha plugin must be deferred, not planned: %#v", plan.Deferred)
	}
	planPath := filepath.Join(root, "plan.json")
	if err := writeCanonical(planPath, plan); err != nil {
		t.Fatal(err)
	}
	candidatePath := filepath.Join(root, "candidates.json")
	if err := writeCanonical(candidatePath, CandidateEvidenceFile{Plugins: map[string]CandidateEvidence{
		"miss": {CandidateRef: candidateRef, Digest: candidateDigest, SourceCommit: target, InputHash: plan.Plugins[0].InputHash},
	}}); err != nil {
		t.Fatal(err)
	}
	mixed, err := renderSnapshot(catalog, planPath, bootstrapPath, candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	if mixed.ProvenanceMode != "mixed" || len(mixed.Plugins) != 2 {
		t.Fatalf("backfill snapshot must mix historical public and candidate provenance: %#v", mixed)
	}
	backfill, carried := mixed.Plugins[0], mixed.Plugins[1]
	if backfill.LogicalID != "miss" || !backfill.Backfill || backfill.ProvenanceMode != "candidate" ||
		backfill.CandidateRef != candidateRef || backfill.Digest != candidateDigest || backfill.OCIRef != "registry.example/plugins/miss:1.0.0" {
		t.Fatalf("backfill entry lost candidate provenance: %#v", backfill)
	}
	if carried.LogicalID != "pub" || carried.Backfill || carried.ProvenanceMode != "public" || carried.Digest != pubDigest || carried.CandidateRef != "" {
		t.Fatalf("unchanged public artifact must carry forward without candidate provenance: %#v", carried)
	}
	mixedPath := filepath.Join(root, "mixed.json")
	if err := writeCanonical(mixedPath, mixed); err != nil {
		t.Fatal(err)
	}
	annotations := map[string]string{
		"org.opencontainers.image.revision": target,
		"io.higress.plugin.input-hash":      backfill.InputHash,
		"org.opencontainers.image.version":  "1.0.0",
	}

	// PR validation resolves only candidate references; the historical public
	// entry is deliberately not re-resolved or annotated there.
	withManifestResolver(t, func(ref string) (ociManifest, error) {
		if ref != candidateRef {
			t.Fatalf("candidate validation resolved %q, want only the backfill candidate", ref)
		}
		return ociManifest{Digest: candidateDigest, Annotations: annotations}, nil
	})
	if err := verifySnapshot(root, catalog, mixedPath, target, target, true, "candidate"); err != nil {
		t.Fatalf("mixed snapshot must pass candidate PR validation: %v", err)
	}

	// Post-promotion verification resolves every public tag; the backfilled
	// tag serves the copied candidate content with its provenance annotations.
	withManifestResolver(t, func(ref string) (ociManifest, error) {
		switch ref {
		case "registry.example/plugins/pub:1.0.0":
			return ociManifest{Digest: pubDigest}, nil
		case "registry.example/plugins/miss:1.0.0":
			return ociManifest{Digest: candidateDigest, Annotations: annotations}, nil
		default:
			t.Fatalf("unexpected public resolve %q", ref)
			return ociManifest{}, nil
		}
	})
	if err := verifySnapshot(root, catalog, mixedPath, target, target, true, "public"); err != nil {
		t.Fatalf("mixed snapshot must pass post-promotion public verification: %v", err)
	}

	withManifestResolver(t, func(string) (ociManifest, error) {
		return ociManifest{Digest: "sha256:" + strings.Repeat("d", 64)}, nil
	})
	if err := verifySnapshot(root, catalog, mixedPath, target, target, true, "public"); err == nil {
		t.Fatal("a public tag drifting from the reviewed digest must fail closed")
	}
}

func TestBootstrapSnapshotRejectsStaleMissingEvidence(t *testing.T) {
	root, catalog, commit := backfillRepo(t, map[string]string{"demo": "1.0.0"})
	evidence := filepath.Join(root, "evidence.json")
	if err := writeCanonical(evidence, BootstrapEvidenceFile{Plugins: map[string]BootstrapEvidence{
		"demo": {Status: "missing", Version: "1.0.0", PublicRef: "registry.example/plugins/demo:1.0.0"},
	}}); err != nil {
		t.Fatal(err)
	}
	args := []string{"--root", root, "--catalog", catalog, "--gateway-version", "2.0.0", "--source", commit, "--existing-evidence", evidence, "--output", filepath.Join(root, "out.json")}
	withManifestResolver(t, func(string) (ociManifest, error) {
		return ociManifest{Digest: "sha256:" + strings.Repeat("a", 64)}, nil
	})
	if err := commandBootstrap(args); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("a public tag appearing after review must reject the stale evidence, got %v", err)
	}
	withManifestResolver(t, func(string) (ociManifest, error) {
		return ociManifest{}, errors.New("403 Forbidden")
	})
	if err := commandBootstrap(args); err == nil || strings.Contains(err.Error(), "stale") {
		t.Fatalf("an authorization failure must abort, never confirm absence: %v", err)
	}
}

func TestRenderSnapshotRejectsDeferredPlanAbuse(t *testing.T) {
	root, catalog, commit := backfillRepo(t, map[string]string{"demo": "1.0.0"})
	digest := "sha256:" + strings.Repeat("a", 64)
	evidence := filepath.Join(root, "evidence.json")
	if err := writeCanonical(evidence, CandidateEvidenceFile{Plugins: map[string]CandidateEvidence{}}); err != nil {
		t.Fatal(err)
	}
	newPlan := func(mutate func(*Plan)) string {
		plan := Plan{SchemaVersion: 1, GatewayVersion: "2.0.0", SourceCommit: commit, CatalogSHA256: sha256Hex(mustRead(t, catalog)), PlanID: digest}
		mutate(&plan)
		path := filepath.Join(t.TempDir(), "plan.json")
		if err := writeCanonical(path, plan); err != nil {
			t.Fatal(err)
		}
		return path
	}
	entry := PlanEntry{LogicalID: "demo", Implementation: "go", SourceDir: "plugins/wasm-go/extensions/demo", Image: "plugins/demo", Version: "1.0.0", InputHash: digest}

	plan := newPlan(func(p *Plan) {
		p.Plugins = []PlanEntry{entry}
		p.Deferred = []DeferredPlugin{{LogicalID: "demo", Version: "1.0.0", Reason: "alpha-prerelease"}}
	})
	if _, err := renderSnapshot(catalog, plan, "", evidence); err == nil || !strings.Contains(err.Error(), "both plans and defers") {
		t.Fatalf("a plugin must not be both planned and deferred: %v", err)
	}
	plan = newPlan(func(p *Plan) {
		p.Deferred = []DeferredPlugin{{LogicalID: "demo", Version: "1.0.0", Reason: "manual"}}
	})
	if _, err := renderSnapshot(catalog, plan, "", evidence); err == nil || !strings.Contains(err.Error(), "unsupported reason") {
		t.Fatalf("only alpha-prerelease deferral is supported: %v", err)
	}
	plan = newPlan(func(p *Plan) {
		p.Deferred = []DeferredPlugin{{LogicalID: "demo", Version: "1.0.0", Reason: "alpha-prerelease"}, {LogicalID: "ghost", Version: "1.0.0", Reason: "alpha-prerelease"}}
	})
	if _, err := renderSnapshot(catalog, plan, "", evidence); err == nil || !strings.Contains(err.Error(), "unknown or release-ineligible") {
		t.Fatalf("deferral must reference release-eligible catalog plugins: %v", err)
	}
}

func TestDeferredAlphaCarriesForwardOnlyStablePreviousEntry(t *testing.T) {
	root, catalog, base := backfillRepo(t, map[string]string{"demo": "1.0.0"})
	digest := "sha256:" + strings.Repeat("a", 64)
	c, _, err := loadCatalog(catalog)
	if err != nil {
		t.Fatal(err)
	}
	p := c.Plugins[0]
	stableHash, err := inputHash(root, base, "1.0.0", c, p)
	if err != nil {
		t.Fatal(err)
	}
	previous := Snapshot{SchemaVersion: 1, GatewayVersion: "2.0.0", SourceCommit: base, CatalogSHA256: sha256Hex(mustRead(t, catalog)), PlanID: digest, ProvenanceMode: "candidate",
		Plugins: []SnapshotEntry{{LogicalID: p.LogicalID, Implementation: p.Implementation, SourceDir: p.SourceDir, Image: p.Image, Version: "1.0.0",
			OCIRef: "registry.example/plugins/demo:1.0.0", Digest: digest, InputHash: stableHash, SourceCommit: base,
			CandidateRef: "registry.example/candidates/demo@" + digest, ProvenanceMode: "candidate"}}}
	previousPath := filepath.Join(root, "previous.json")
	if err := writeCanonical(previousPath, previous); err != nil {
		t.Fatal(err)
	}

	// The plugin moves to an alpha VERSION: its inputs changed, yet no
	// candidate is planned and no new snapshot entry is created.
	mustWrite(t, filepath.Join(root, p.SourceDir, "VERSION"), "1.1.0-alpha\n")
	mustRun(t, root, "git", "add", ".")
	mustRun(t, root, "git", "commit", "-q", "-m", "alpha development")
	target, _ := resolveCommit(root, "HEAD")
	plan, err := buildPlan(root, catalog, previousPath, "", target, "2.0.1", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Plugins) != 0 || len(plan.Deferred) != 1 || plan.Deferred[0].LogicalID != "demo" || plan.Deferred[0].Version != "1.1.0-alpha" {
		t.Fatalf("changed alpha plugin must be deferred without a candidate: %#v %#v", plan.Plugins, plan.Deferred)
	}
	planPath := filepath.Join(root, "plan.json")
	if err := writeCanonical(planPath, plan); err != nil {
		t.Fatal(err)
	}
	evidence := filepath.Join(root, "evidence.json")
	if err := writeCanonical(evidence, CandidateEvidenceFile{Plugins: map[string]CandidateEvidence{}}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := renderSnapshot(catalog, planPath, previousPath, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Plugins) != 1 || snapshot.Plugins[0].Version != "1.0.0" || snapshot.Plugins[0].InputHash != stableHash {
		t.Fatalf("deferred plugin must carry only its earlier stable release forward: %#v", snapshot.Plugins)
	}
	snapshotPath := filepath.Join(root, "snapshot.json")
	if err := writeCanonical(snapshotPath, snapshot); err != nil {
		t.Fatal(err)
	}
	if err := verifySnapshot(root, catalog, snapshotPath, target, target, false, "candidate"); err != nil {
		t.Fatalf("stable carry-forward for a deferred plugin must verify: %v", err)
	}

	tampered := snapshot
	tampered.Plugins[0].Version = "1.1.0-alpha"
	tampered.Plugins[0].OCIRef = "registry.example/plugins/demo:1.1.0-alpha"
	tamperedPath := filepath.Join(root, "tampered.json")
	if err := writeCanonical(tamperedPath, tampered); err != nil {
		t.Fatal(err)
	}
	if err := verifySnapshot(root, catalog, tamperedPath, target, target, false, "candidate"); err == nil || !strings.Contains(err.Error(), "deferred") {
		t.Fatalf("an alpha build must never become a snapshot entry: %v", err)
	}
}

func TestManagedReleasePlansNewCatalogPluginWithoutBackfill(t *testing.T) {
	root, catalog, base := backfillRepo(t, map[string]string{"demo": "1.0.0"})
	digest := "sha256:" + strings.Repeat("a", 64)
	c, _, err := loadCatalog(catalog)
	if err != nil {
		t.Fatal(err)
	}
	p := c.Plugins[0]
	stableHash, err := inputHash(root, base, "1.0.0", c, p)
	if err != nil {
		t.Fatal(err)
	}
	previous := Snapshot{SchemaVersion: 1, GatewayVersion: "2.0.0", SourceCommit: base, CatalogSHA256: sha256Hex(mustRead(t, catalog)), PlanID: digest, ProvenanceMode: "candidate",
		Plugins: []SnapshotEntry{{LogicalID: p.LogicalID, Implementation: p.Implementation, SourceDir: p.SourceDir, Image: p.Image, Version: "1.0.0",
			OCIRef: "registry.example/plugins/demo:1.0.0", Digest: digest, InputHash: stableHash, SourceCommit: base,
			CandidateRef: "registry.example/candidates/demo@" + digest, ProvenanceMode: "candidate"}}}
	previousPath := filepath.Join(root, "previous.json")
	if err := writeCanonical(previousPath, previous); err != nil {
		t.Fatal(err)
	}

	newDir := "plugins/wasm-go/extensions/newbie"
	mustWrite(t, filepath.Join(root, newDir, "main.go"), "package main\n")
	mustWrite(t, filepath.Join(root, newDir, "VERSION"), "1.0.0\n")
	mustRun(t, root, "git", "add", ".")
	mustRun(t, root, "git", "commit", "-q", "-m", "new catalog plugin")
	target, _ := resolveCommit(root, "HEAD")
	c.Plugins = append(c.Plugins, Plugin{LogicalID: "newbie", Implementation: "go", SourceDir: newDir, Image: "plugins/newbie", ReleaseEligible: true, ArtifactInputs: []string{newDir + "/**"}})
	if err := writeCanonical(catalog, c); err != nil {
		t.Fatal(err)
	}
	plan, err := buildPlan(root, catalog, previousPath, "", target, "2.0.1", "")
	if err != nil {
		t.Fatal(err)
	}
	// A new catalog plugin in a managed release is a genuine new release: it
	// is planned through the same candidate path but is not imported history,
	// so latest advances normally for it at promotion.
	if len(plan.Plugins) != 1 || plan.Plugins[0].LogicalID != "newbie" || plan.Plugins[0].Backfill {
		t.Fatalf("new managed-release plugin must be planned without backfill marking: %#v", plan.Plugins)
	}
	if len(plan.Deferred) != 0 {
		t.Fatalf("no plugin is deferred here: %#v", plan.Deferred)
	}
}

func backfillRepo(t *testing.T, versions map[string]string) (root, catalogPath, commit string) {
	t.Helper()
	root = t.TempDir()
	mustRun(t, root, "git", "init", "-q")
	mustRun(t, root, "git", "config", "user.name", "test")
	mustRun(t, root, "git", "config", "user.email", "test@example.com")
	names := make([]string, 0, len(versions))
	for name := range versions {
		names = append(names, name)
	}
	sort.Strings(names)
	plugins := make([]Plugin, 0, len(names))
	for _, name := range names {
		dir := "plugins/wasm-go/extensions/" + name
		mustWrite(t, filepath.Join(root, dir, "main.go"), "package main\n")
		mustWrite(t, filepath.Join(root, dir, "VERSION"), versions[name]+"\n")
		plugins = append(plugins, Plugin{LogicalID: name, Implementation: "go", SourceDir: dir, Image: "plugins/" + name, ReleaseEligible: true, ArtifactInputs: []string{dir + "/**"}})
	}
	mustWrite(t, filepath.Join(root, "plugins/wasm-rust/extensions/.keep"), "")
	mustRun(t, root, "git", "add", ".")
	mustRun(t, root, "git", "commit", "-q", "-m", "source")
	commit, _ = resolveCommit(root, "HEAD")
	catalogPath = filepath.Join(root, "catalog.json")
	c := Catalog{SchemaVersion: 1, Registry: "registry.example", Plugins: plugins}
	if err := writeCanonical(catalogPath, c); err != nil {
		t.Fatal(err)
	}
	return root, catalogPath, commit
}
