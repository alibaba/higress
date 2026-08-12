// Copyright 2026 Higress Authors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

var (
	safeIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:[a-z0-9._-]*[a-z0-9])?$`)
	digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

type ociManifest struct {
	Digest      string
	Annotations map[string]string
}

// ociManifestResolver is a seam for the bootstrap and verification tests. It
// resolves the supplied reference itself, rather than trusting a digest copied
// from workflow input.
var ociManifestResolver = resolveOCIManifest

func loadCatalog(path string) (Catalog, []byte, error) {
	var c Catalog
	data, err := readJSON(path, &c)
	return c, data, err
}

func validateCatalog(root, path string) error {
	c, _, err := loadCatalog(path)
	if err != nil {
		return err
	}
	if c.SchemaVersion != catalogSchemaVersion {
		return fmt.Errorf("catalog schemaVersion must be %d", catalogSchemaVersion)
	}
	if c.Registry == "" || strings.Contains(c.Registry, "://") {
		return errors.New("catalog registry must be a host without a URL scheme")
	}
	seenLogical, seenSource, seenImage := map[string]bool{}, map[string]bool{}, map[string]bool{}
	dirClassifications := map[string]bool{}
	consumerMappings := map[string]map[string]string{"console": {}, "pluginServer": {}}
	aliases := map[string]string{}
	for _, p := range c.Plugins {
		if !safeIDPattern.MatchString(p.LogicalID) {
			return fmt.Errorf("plugin has unsafe logicalId %q", p.LogicalID)
		}
		if seenLogical[p.LogicalID] {
			return fmt.Errorf("duplicate logicalId %q", p.LogicalID)
		}
		seenLogical[p.LogicalID] = true
		if seenSource[p.SourceDir] {
			return fmt.Errorf("duplicate sourceDir %q", p.SourceDir)
		}
		seenSource[p.SourceDir] = true
		if seenImage[p.Image] {
			return fmt.Errorf("duplicate image %q", p.Image)
		}
		seenImage[p.Image] = true
		expectedPrefix := "plugins/wasm-" + p.Implementation + "/extensions/"
		if p.Implementation != "go" && p.Implementation != "rust" {
			return fmt.Errorf("%s: implementation must be go or rust", p.LogicalID)
		}
		if !strings.HasPrefix(p.SourceDir, expectedPrefix) || strings.Contains(p.SourceDir, "/examples/") || strings.Contains(p.SourceDir, "/example/") {
			return fmt.Errorf("%s: sourceDir %q is outside the official %s extension root", p.LogicalID, p.SourceDir, p.Implementation)
		}
		if _, err := os.Stat(filepath.Join(root, p.SourceDir)); err != nil {
			return fmt.Errorf("%s: sourceDir: %w", p.LogicalID, err)
		}
		dirClassifications[p.SourceDir] = true
		if !safeIDPattern.MatchString(filepath.Base(p.Image)) || !strings.HasPrefix(p.Image, "plugins/") {
			return fmt.Errorf("%s: unsafe or non-plugin image %q", p.LogicalID, p.Image)
		}
		if p.ReleaseEligible {
			if p.UnmanagedReason != "" {
				return fmt.Errorf("%s: release-eligible plugin cannot have unmanagedReason", p.LogicalID)
			}
			versionData, err := os.ReadFile(filepath.Join(root, p.SourceDir, "VERSION"))
			if err != nil {
				return fmt.Errorf("%s: release-eligible plugin lacks VERSION: %w", p.LogicalID, err)
			}
			if _, err := parseSemver(strings.TrimSpace(string(versionData))); err != nil {
				return fmt.Errorf("%s: %w", p.LogicalID, err)
			}
			versionPath := p.SourceDir + "/VERSION"
			// In a Git worktree, reject ignored/local VERSION files so CI sees the
			// same release inputs as a clean checkout. A git archive intentionally
			// has no .git directory; its extracted contents are already the exact
			// tracked tree and remain valid for archive-based verification.
			if err := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree").Run(); err == nil {
				tracked := exec.Command("git", "-C", root, "ls-files", "--error-unmatch", "--", versionPath)
				if err := tracked.Run(); err != nil {
					return fmt.Errorf("%s: release-eligible VERSION must be tracked by Git: %s", p.LogicalID, versionPath)
				}
			}
		} else if strings.TrimSpace(p.UnmanagedReason) == "" {
			return fmt.Errorf("%s: release-ineligible plugin requires unmanagedReason", p.LogicalID)
		}
		if len(p.ArtifactInputs) == 0 {
			return fmt.Errorf("%s: artifactInputs is empty", p.LogicalID)
		}
		for _, group := range p.SharedInputGroups {
			if _, ok := c.SharedInputGroups[group]; !ok {
				return fmt.Errorf("%s: unknown shared input group %q", p.LogicalID, group)
			}
		}
		if p.Consumers.Console != nil {
			cc := p.Consumers.Console
			if cc.PropertyKey == "" || cc.ResourceDir == "" || cc.URLForm != "oci" {
				return fmt.Errorf("%s: incomplete Console mapping", p.LogicalID)
			}
			if other := consumerMappings["console"][cc.PropertyKey]; other != "" {
				return fmt.Errorf("Console key %q belongs to both %s and %s", cc.PropertyKey, other, p.LogicalID)
			}
			consumerMappings["console"][cc.PropertyKey] = p.LogicalID
		}
		if p.Consumers.PluginServer != nil {
			pc := p.Consumers.PluginServer
			if pc.InventoryKey == "" || pc.HTTPPath == "" {
				return fmt.Errorf("%s: incomplete plugin-server mapping", p.LogicalID)
			}
			if other := consumerMappings["pluginServer"][pc.InventoryKey]; other != "" {
				return fmt.Errorf("plugin-server key %q belongs to both %s and %s", pc.InventoryKey, other, p.LogicalID)
			}
			consumerMappings["pluginServer"][pc.InventoryKey] = p.LogicalID
			for _, alias := range pc.Aliases {
				if other := aliases[alias]; other != "" {
					return fmt.Errorf("plugin-server alias %q belongs to both %s and %s", alias, other, p.LogicalID)
				}
				aliases[alias] = p.LogicalID
			}
		}
	}
	for _, implementation := range []string{"go", "rust"} {
		rootDir := filepath.Join(root, "plugins", "wasm-"+implementation, "extensions")
		entries, err := os.ReadDir(rootDir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				source := filepath.ToSlash(filepath.Join("plugins", "wasm-"+implementation, "extensions", entry.Name()))
				if !dirClassifications[source] {
					return fmt.Errorf("unclassified official extension %s", source)
				}
			}
		}
	}
	for consumer, entries := range c.ConsumerInventories {
		mapping, ok := consumerMappings[consumer]
		if !ok {
			return fmt.Errorf("unknown consumer inventory %q", consumer)
		}
		seen := map[string]bool{}
		for _, entry := range entries {
			if seen[entry.Key] {
				return fmt.Errorf("%s inventory has duplicate %q", consumer, entry.Key)
			}
			seen[entry.Key] = true
			switch entry.Classification {
			case "managed":
				if entry.LogicalID == "" || mapping[entry.Key] != entry.LogicalID {
					return fmt.Errorf("%s inventory key %q has unresolved managed mapping %q", consumer, entry.Key, entry.LogicalID)
				}
			case "unmanaged":
				if entry.Reason == "" || entry.LogicalID != "" {
					return fmt.Errorf("%s inventory key %q requires only an unmanaged reason", consumer, entry.Key)
				}
			default:
				return fmt.Errorf("%s inventory key %q has invalid classification %q", consumer, entry.Key, entry.Classification)
			}
		}
		for key := range mapping {
			if !seen[key] {
				return fmt.Errorf("%s managed key %q is absent from classified inventory", consumer, key)
			}
		}
	}
	return nil
}

func buildPlan(root, catalogPath, previousPath, baseRef, targetRef, gatewayVersion, overridesPath string) (Plan, error) {
	if _, err := parseSemver(gatewayVersion); err != nil {
		return Plan{}, fmt.Errorf("gateway version: %w", err)
	}
	if err := validateCatalog(root, catalogPath); err != nil {
		return Plan{}, err
	}
	if previousPath == "" {
		return Plan{}, errors.New("no previous snapshot: import a reviewed digest baseline with bootstrap-snapshot before planning; bootstrap never rebuilds existing public tags")
	}
	c, catalogData, err := loadCatalog(catalogPath)
	if err != nil {
		return Plan{}, err
	}
	target, err := resolveCommit(root, targetRef)
	if err != nil {
		return Plan{}, err
	}
	var previous Snapshot
	previousEntries := map[string]SnapshotEntry{}
	explicitBase := baseRef != ""
	if previousPath != "" {
		if _, err := readJSON(previousPath, &previous); err != nil {
			return Plan{}, err
		}
		if previous.ProvenanceMode == "bootstrap-public" {
			if !explicitBase {
				return Plan{}, errors.New("a bootstrap-public previous snapshot requires one exact --base comparison commit")
			}
		} else if explicitBase {
			return Plan{}, errors.New("--base is allowed only with a bootstrap-public previous snapshot")
		}
		for _, entry := range previous.Plugins {
			previousEntries[entry.LogicalID] = entry
		}
		if baseRef == "" {
			baseRef = previous.SourceCommit
		}
	}
	base := ""
	if baseRef != "" {
		if explicitBase && !commitPattern.MatchString(baseRef) {
			return Plan{}, errors.New("bootstrap comparison --base must be a full lowercase 40-character commit")
		}
		base, err = resolveCommit(root, baseRef)
		if err != nil {
			return Plan{}, err
		}
		if explicitBase {
			if err := requireAncestor(root, base, target); err != nil {
				return Plan{}, err
			}
		}
	}
	paths, err := changedPaths(root, base, target)
	if err != nil {
		return Plan{}, err
	}
	overrides := map[string]string{}
	if overridesPath != "" {
		if _, err := readJSON(overridesPath, &overrides); err != nil {
			return Plan{}, err
		}
	}
	plugins := append([]Plugin(nil), c.Plugins...)
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].LogicalID < plugins[j].LogicalID })
	plan := Plan{SchemaVersion: planSchemaVersion, GatewayVersion: gatewayVersion, SourceCommit: target, BaseCommit: base,
		PreviousRelease: previous.GatewayVersion, CatalogSHA256: sha256Hex(catalogData)}
	for _, p := range plugins {
		if !p.ReleaseEligible {
			continue
		}
		currentRaw, err := fileAtCommit(root, target, p.SourceDir+"/VERSION")
		if err != nil {
			return Plan{}, fmt.Errorf("%s VERSION: %w", p.LogicalID, err)
		}
		current := strings.TrimSpace(currentRaw)
		if _, err := parseSemver(current); err != nil {
			return Plan{}, fmt.Errorf("%s: %w", p.LogicalID, err)
		}
		changed := []string{}
		for _, path := range paths {
			if pluginInputMatches(c, p, path) || path == p.SourceDir+"/VERSION" {
				changed = append(changed, path)
			}
		}
		prev, hasPrevious := previousEntries[p.LogicalID]
		_, hasOverride := overrides[p.LogicalID]
		affected := base == "" || len(changed) > 0 || hasOverride
		if previous.ProvenanceMode == "bootstrap-public" && hasPrevious {
			previousVersion, parseErr := parseSemver(prev.Version)
			if parseErr != nil {
				return Plan{}, fmt.Errorf("%s previous bootstrap version: %w", p.LogicalID, parseErr)
			}
			// Do not carry a legacy public prerelease into the first managed
			// snapshot: promote it once and build immutable candidate evidence.
			affected = affected || previousVersion.prerelease != ""
		}
		if !hasPrevious && previousPath != "" {
			affected = true
		}
		if !affected {
			continue
		}
		version := current
		previousVersion := ""
		if hasPrevious {
			previousVersion = prev.Version
			if previous.ProvenanceMode == "bootstrap-public" {
				version, err = nextBootstrapVersion(prev.Version, current)
			} else {
				version, err = nextVersion(prev.Version, current)
			}
			if err != nil {
				return Plan{}, fmt.Errorf("%s: %w", p.LogicalID, err)
			}
		}
		if override, ok := overrides[p.LogicalID]; ok {
			overrideVersion, err := parseSemver(override)
			if err != nil || overrideVersion.prerelease != "" {
				return Plan{}, fmt.Errorf("%s: override %q must be stable SemVer", p.LogicalID, override)
			}
			if hasPrevious {
				prevVersion, _ := parseSemver(prev.Version)
				if compareSemver(overrideVersion, prevVersion) <= 0 {
					return Plan{}, fmt.Errorf("%s: override %q must be greater than previous %q", p.LogicalID, override, prev.Version)
				}
			}
			version = override
		}
		hash, err := inputHash(root, target, version, c, p)
		if err != nil {
			return Plan{}, fmt.Errorf("%s: %w", p.LogicalID, err)
		}
		plan.Plugins = append(plan.Plugins, PlanEntry{LogicalID: p.LogicalID, Implementation: p.Implementation,
			SourceDir: p.SourceDir, Image: p.Image, PreviousVersion: previousVersion, Version: version,
			InputHash: hash, ChangedPaths: changed})
	}
	for id := range overrides {
		found := false
		for _, entry := range plan.Plugins {
			found = found || entry.LogicalID == id
		}
		if !found {
			return Plan{}, fmt.Errorf("override references unknown or release-ineligible plugin %q", id)
		}
	}
	plan.PlanID = "sha256:" + canonicalObjectHash(plan, true)
	return plan, nil
}

func canonicalObjectHash(value any, clearPlanID bool) string {
	if clearPlanID {
		if plan, ok := value.(Plan); ok {
			plan.PlanID = ""
			value = plan
		}
	}
	data, _ := json.Marshal(value)
	return sha256Hex(data)
}

func applyPlan(root string, plan Plan) error {
	for _, entry := range plan.Plugins {
		path := filepath.Join(root, entry.SourceDir, "VERSION")
		if err := os.WriteFile(path, []byte(entry.Version+"\n"), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func renderSnapshot(catalogPath, planPath, previousPath, evidencePath string) (Snapshot, error) {
	c, catalogData, err := loadCatalog(catalogPath)
	if err != nil {
		return Snapshot{}, err
	}
	var plan Plan
	if _, err := readJSON(planPath, &plan); err != nil {
		return Snapshot{}, err
	}
	if plan.CatalogSHA256 != sha256Hex(catalogData) {
		return Snapshot{}, errors.New("plan catalogSha256 does not match catalog")
	}
	if plan.SchemaVersion != planSchemaVersion || !commitPattern.MatchString(plan.SourceCommit) || !digestPattern.MatchString(plan.PlanID) {
		return Snapshot{}, errors.New("plan has an unsupported schema or invalid immutable provenance")
	}
	var previous Snapshot
	previousEntries := map[string]SnapshotEntry{}
	if previousPath != "" {
		if _, err := readJSON(previousPath, &previous); err != nil {
			return Snapshot{}, err
		}
		for _, entry := range previous.Plugins {
			previousEntries[entry.LogicalID] = entry
		}
	}
	var evidence CandidateEvidenceFile
	if _, err := readJSON(evidencePath, &evidence); err != nil {
		return Snapshot{}, err
	}
	planEntries := map[string]PlanEntry{}
	for _, entry := range plan.Plugins {
		if _, exists := planEntries[entry.LogicalID]; exists {
			return Snapshot{}, fmt.Errorf("plan contains duplicate plugin %s", entry.LogicalID)
		}
		planEntries[entry.LogicalID] = entry
	}
	plugins := append([]Plugin(nil), c.Plugins...)
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].LogicalID < plugins[j].LogicalID })
	snapshot := Snapshot{SchemaVersion: snapshotSchemaVersion, GatewayVersion: plan.GatewayVersion,
		SourceCommit: plan.SourceCommit, PreviousRelease: plan.PreviousRelease, CatalogSHA256: plan.CatalogSHA256, PlanID: plan.PlanID, ProvenanceMode: "candidate"}
	for _, p := range plugins {
		if !p.ReleaseEligible {
			continue
		}
		if planned, ok := planEntries[p.LogicalID]; ok {
			e, ok := evidence.Plugins[p.LogicalID]
			if !ok {
				return Snapshot{}, fmt.Errorf("missing candidate evidence for %s", p.LogicalID)
			}
			if !digestPattern.MatchString(e.Digest) || e.SourceCommit != plan.SourceCommit || e.InputHash != planned.InputHash || !strings.HasSuffix(e.CandidateRef, "@"+e.Digest) {
				return Snapshot{}, fmt.Errorf("candidate evidence for %s does not match plan provenance", p.LogicalID)
			}
			snapshot.Plugins = append(snapshot.Plugins, SnapshotEntry{LogicalID: p.LogicalID, Implementation: p.Implementation,
				SourceDir: p.SourceDir, Image: p.Image, Version: planned.Version,
				OCIRef: c.Registry + "/" + p.Image + ":" + planned.Version, Digest: e.Digest,
				InputHash: planned.InputHash, SourceCommit: plan.SourceCommit, CandidateRef: e.CandidateRef,
				ProvenanceMode: "candidate", Consumers: cloneConsumers(p.Consumers)})
			continue
		}
		old, ok := previousEntries[p.LogicalID]
		if !ok {
			return Snapshot{}, fmt.Errorf("release-eligible plugin %s is neither planned nor present in previous snapshot", p.LogicalID)
		}
		old.Consumers = cloneConsumers(p.Consumers)
		if old.ProvenanceMode == "" {
			old.ProvenanceMode = entryProvenance(old)
		}
		if old.ProvenanceMode == "public" {
			snapshot.ProvenanceMode = "mixed"
		}
		snapshot.Plugins = append(snapshot.Plugins, old)
	}
	return snapshot, nil
}

func verifySnapshot(root, catalogPath, snapshotPath, expectedSource, committedSource string, resolve bool, ociSource string) error {
	if err := validateCatalog(root, catalogPath); err != nil {
		return err
	}
	c, catalogData, err := loadCatalog(catalogPath)
	if err != nil {
		return err
	}
	var snapshot Snapshot
	if _, err := readJSON(snapshotPath, &snapshot); err != nil {
		return err
	}
	if snapshot.SchemaVersion != snapshotSchemaVersion || snapshot.CatalogSHA256 != sha256Hex(catalogData) {
		return errors.New("snapshot schema or catalog hash mismatch")
	}
	if _, err := parseSemver(snapshot.GatewayVersion); err != nil {
		return fmt.Errorf("snapshot gatewayVersion: %w", err)
	}
	if !commitPattern.MatchString(snapshot.SourceCommit) {
		return errors.New("snapshot sourceCommit must be a full lowercase Git commit")
	}
	if snapshot.ProvenanceMode != "candidate" && snapshot.ProvenanceMode != "bootstrap-public" && snapshot.ProvenanceMode != "mixed" {
		return fmt.Errorf("snapshot has unsupported provenanceMode %q", snapshot.ProvenanceMode)
	}
	if ociSource != "candidate" && ociSource != "public" {
		return fmt.Errorf("OCI source must be candidate or public, got %q", ociSource)
	}
	if expectedSource != "" {
		commit, err := resolveCommit(root, expectedSource)
		if err != nil || commit != snapshot.SourceCommit {
			return fmt.Errorf("snapshot sourceCommit %s does not match expected source %s", snapshot.SourceCommit, expectedSource)
		}
	}
	committed := ""
	if committedSource != "" {
		var err error
		committed, err = resolveCommit(root, committedSource)
		if err != nil {
			return err
		}
	}
	plugins := map[string]Plugin{}
	eligible := 0
	for _, p := range c.Plugins {
		plugins[p.LogicalID] = p
		if p.ReleaseEligible {
			eligible++
		}
	}
	if len(snapshot.Plugins) != eligible {
		return fmt.Errorf("snapshot has %d plugins, catalog has %d release-eligible plugins", len(snapshot.Plugins), eligible)
	}
	seen := map[string]bool{}
	last := ""
	for _, entry := range snapshot.Plugins {
		p, ok := plugins[entry.LogicalID]
		if !ok || !p.ReleaseEligible || seen[entry.LogicalID] {
			return fmt.Errorf("snapshot has unknown, ineligible, or duplicate plugin %q", entry.LogicalID)
		}
		if last != "" && entry.LogicalID < last {
			return errors.New("snapshot plugins are not sorted by logicalId")
		}
		last, seen[entry.LogicalID] = entry.LogicalID, true
		if entry.Image != p.Image || entry.SourceDir != p.SourceDir || entry.Implementation != p.Implementation {
			return fmt.Errorf("%s snapshot identity differs from catalog", entry.LogicalID)
		}
		if !reflect.DeepEqual(entry.Consumers, p.Consumers) {
			return fmt.Errorf("%s snapshot consumer mappings differ from catalog", entry.LogicalID)
		}
		if _, err := parseSemver(entry.Version); err != nil {
			return fmt.Errorf("%s: %w", entry.LogicalID, err)
		}
		expectedRef := c.Registry + "/" + p.Image + ":" + entry.Version
		if entry.OCIRef != expectedRef || !digestPattern.MatchString(entry.Digest) || !digestPattern.MatchString(entry.InputHash) {
			return fmt.Errorf("%s has invalid OCI or provenance fields", entry.LogicalID)
		}
		provenance := entryProvenance(entry)
		if provenance == "candidate" && !strings.HasSuffix(entry.CandidateRef, "@"+entry.Digest) {
			return fmt.Errorf("%s candidate provenance does not pin the snapshot digest", entry.LogicalID)
		}
		if provenance == "public" && entry.CandidateRef != "" {
			return fmt.Errorf("%s bootstrap snapshot must not claim candidate provenance", entry.LogicalID)
		}
		if provenance != "candidate" && provenance != "public" {
			return fmt.Errorf("%s has invalid artifact provenance", entry.LogicalID)
		}
		hash, err := inputHash(root, entry.SourceCommit, entry.Version, c, p)
		if err != nil || hash != entry.InputHash {
			return fmt.Errorf("%s input hash does not recompute from sourceCommit and proposed version", entry.LogicalID)
		}
		if committed != "" {
			versionAtCommit, err := fileAtCommit(root, committed, p.SourceDir+"/VERSION")
			if err != nil || strings.TrimSpace(versionAtCommit) != entry.Version {
				return fmt.Errorf("%s VERSION at committed source does not equal snapshot version", entry.LogicalID)
			}
			committedHash, err := inputHash(root, committed, entry.Version, c, p)
			if err != nil || committedHash != entry.InputHash {
				return fmt.Errorf("%s committed source artifact inputs differ from snapshot", entry.LogicalID)
			}
		}
		if resolve {
			if ociSource == "candidate" && provenance == "public" {
				continue
			}
			if err := verifyOCI(entry, provenance, snapshot.ProvenanceMode, ociSource); err != nil {
				return fmt.Errorf("%s: %w", entry.LogicalID, err)
			}
		}
	}
	return nil
}

func entryProvenance(entry SnapshotEntry) string {
	if entry.ProvenanceMode != "" {
		return entry.ProvenanceMode
	}
	if entry.CandidateRef != "" {
		return "candidate"
	}
	return "public"
}

func verifyOCI(entry SnapshotEntry, provenanceMode, snapshotMode, ociSource string) error {
	ref := entry.OCIRef
	if ociSource == "candidate" {
		ref = entry.CandidateRef
	}
	manifest, err := ociManifestResolver(ref)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", ref, err)
	}
	if manifest.Digest != entry.Digest {
		return fmt.Errorf("resolved digest %s does not match snapshot digest %s", manifest.Digest, entry.Digest)
	}
	// Historical production images may predate release annotations. The
	// reviewed bootstrap evidence plus the resolved public tag is the only
	// provenance available for that one import path.
	if snapshotMode == "bootstrap-public" {
		if ociSource != "public" || provenanceMode != "public" || entry.CandidateRef != "" {
			return errors.New("bootstrap snapshots may resolve only exact public artifacts")
		}
		return nil
	}
	if manifest.Annotations["org.opencontainers.image.revision"] != entry.SourceCommit ||
		manifest.Annotations["io.higress.plugin.input-hash"] != entry.InputHash ||
		manifest.Annotations["org.opencontainers.image.version"] != entry.Version {
		return errors.New("OCI provenance annotations do not match snapshot")
	}
	return nil
}

func resolveOCIManifest(ref string) (ociManifest, error) {
	descriptorCmd := exec.Command("oras", "manifest", "fetch", ref, "--descriptor", "--format", "json")
	descriptorOut, err := descriptorCmd.Output()
	if err != nil {
		return ociManifest{}, err
	}
	var descriptor struct {
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal(descriptorOut, &descriptor); err != nil {
		return ociManifest{}, err
	}
	manifestCmd := exec.Command("oras", "manifest", "fetch", ref)
	manifestOut, err := manifestCmd.Output()
	if err != nil {
		return ociManifest{}, err
	}
	var manifest struct {
		Annotations map[string]string `json:"annotations"`
	}
	if err := json.Unmarshal(manifestOut, &manifest); err != nil {
		return ociManifest{}, err
	}
	return ociManifest{Digest: descriptor.Digest, Annotations: manifest.Annotations}, nil
}

func digestReference(ociRef, digest string) (string, error) {
	ref := strings.TrimPrefix(ociRef, "oci://")
	slash := strings.LastIndex(ref, "/")
	colon := strings.LastIndex(ref, ":")
	if slash < 0 || colon <= slash || !digestPattern.MatchString(digest) {
		return "", fmt.Errorf("invalid OCI reference or digest")
	}
	return ref[:colon] + "@" + digest, nil
}

func parseCommon(name string, args []string) (*flag.FlagSet, *string, *string) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	root := fs.String("root", ".", "repository root")
	catalog := fs.String("catalog", "plugins/release/catalog.json", "catalog path")
	return fs, root, catalog
}

func commandValidate(args []string) error {
	fs, root, catalog := parseCommon("validate-catalog", args)
	if err := fs.Parse(args); err != nil {
		return err
	}
	return validateCatalog(*root, *catalog)
}

func commandPlan(args []string) error {
	fs, root, catalog := parseCommon("plan", args)
	previous := fs.String("previous", "", "previous snapshot")
	base := fs.String("base", "", "base Git ref (defaults to previous sourceCommit)")
	target := fs.String("target", "HEAD", "exact target Git ref")
	gateway := fs.String("gateway-version", "", "target gateway version")
	overrides := fs.String("overrides", "", "reviewed logicalId-to-version JSON")
	output := fs.String("output", "-", "plan output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	plan, err := buildPlan(*root, *catalog, *previous, *base, *target, *gateway, *overrides)
	if err != nil {
		return err
	}
	return writeCanonical(*output, plan)
}

// captureBootstrapEvidence resolves the exact public tag selected by every
// release-eligible VERSION in the target tree. The evidence intentionally has
// no source/input fields: embedding the commit of the PR that commits the
// evidence would be self-referential. bootstrap-snapshot recomputes source and
// input provenance independently when it consumes these reviewed public refs.
func captureBootstrapEvidence(root, catalogPath, source string) (BootstrapEvidenceFile, error) {
	if !commitPattern.MatchString(source) {
		return BootstrapEvidenceFile{}, errors.New("bootstrap evidence source must be a full lowercase 40-character commit")
	}
	if err := validateCatalog(root, catalogPath); err != nil {
		return BootstrapEvidenceFile{}, err
	}
	c, _, err := loadCatalog(catalogPath)
	if err != nil {
		return BootstrapEvidenceFile{}, err
	}
	commit, err := resolveCommit(root, source)
	if err != nil {
		return BootstrapEvidenceFile{}, err
	}
	plugins := append([]Plugin(nil), c.Plugins...)
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].LogicalID < plugins[j].LogicalID })
	evidence := BootstrapEvidenceFile{Plugins: map[string]BootstrapEvidence{}}
	for _, p := range plugins {
		if !p.ReleaseEligible {
			continue
		}
		raw, err := fileAtCommit(root, commit, p.SourceDir+"/VERSION")
		if err != nil {
			return BootstrapEvidenceFile{}, fmt.Errorf("%s VERSION: %w", p.LogicalID, err)
		}
		version := strings.TrimSpace(raw)
		if _, err := parseSemver(version); err != nil {
			return BootstrapEvidenceFile{}, fmt.Errorf("%s: %w", p.LogicalID, err)
		}
		publicRef := c.Registry + "/" + p.Image + ":" + version
		manifest, err := ociManifestResolver(publicRef)
		if err != nil {
			return BootstrapEvidenceFile{}, fmt.Errorf("resolve bootstrap public artifact %s: %w", p.LogicalID, err)
		}
		if !digestPattern.MatchString(manifest.Digest) {
			return BootstrapEvidenceFile{}, fmt.Errorf("bootstrap public artifact %s returned invalid digest %q", p.LogicalID, manifest.Digest)
		}
		evidence.Plugins[p.LogicalID] = BootstrapEvidence{PublicRef: publicRef, Digest: manifest.Digest}
	}
	return evidence, nil
}

func commandCaptureBootstrapEvidence(args []string) error {
	fs, root, catalog := parseCommon("capture-bootstrap-evidence", args)
	source := fs.String("source", "HEAD", "exact target source commit")
	output := fs.String("output", "-", "canonical public evidence output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	evidence, err := captureBootstrapEvidence(*root, *catalog, *source)
	if err != nil {
		return err
	}
	return writeCanonical(*output, evidence)
}

// bootstrap-snapshot imports already-published production artifacts into the
// first reviewed baseline. It never builds or writes an OCI tag: evidence must
// resolve to the current public digest and match the checked-in VERSION/input.
func commandBootstrap(args []string) error {
	fs, root, catalog := parseCommon("bootstrap-snapshot", args)
	gateway := fs.String("gateway-version", "", "baseline gateway version")
	source := fs.String("source", "HEAD", "exact source commit")
	evidencePath := fs.String("existing-evidence", "", "reviewed public digest evidence JSON")
	output := fs.String("output", "-", "snapshot output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *evidencePath == "" {
		return errors.New("--existing-evidence is required")
	}
	if _, err := parseSemver(*gateway); err != nil {
		return err
	}
	if err := validateCatalog(*root, *catalog); err != nil {
		return err
	}
	c, bytes, err := loadCatalog(*catalog)
	if err != nil {
		return err
	}
	commit, err := resolveCommit(*root, *source)
	if err != nil {
		return err
	}
	var evidence BootstrapEvidenceFile
	if _, err := readJSON(*evidencePath, &evidence); err != nil {
		return err
	}
	snapshot := Snapshot{SchemaVersion: snapshotSchemaVersion, GatewayVersion: *gateway, SourceCommit: commit, CatalogSHA256: sha256Hex(bytes), PlanID: "bootstrap:" + commit, ProvenanceMode: "bootstrap-public"}
	plugins := append([]Plugin(nil), c.Plugins...)
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].LogicalID < plugins[j].LogicalID })
	for _, p := range plugins {
		if !p.ReleaseEligible {
			continue
		}
		raw, err := fileAtCommit(*root, commit, p.SourceDir+"/VERSION")
		if err != nil {
			return err
		}
		version := strings.TrimSpace(raw)
		if _, err := parseSemver(version); err != nil {
			return err
		}
		hash, err := inputHash(*root, commit, version, c, p)
		if err != nil {
			return err
		}
		e, ok := evidence.Plugins[p.LogicalID]
		publicRef := c.Registry + "/" + p.Image + ":" + version
		if !ok || !digestPattern.MatchString(e.Digest) || e.PublicRef != publicRef {
			return fmt.Errorf("bootstrap evidence for %s is incomplete or mismatched", p.LogicalID)
		}
		resolved, err := ociManifestResolver(e.PublicRef)
		if err != nil || resolved.Digest != e.Digest {
			if err != nil {
				return fmt.Errorf("bootstrap public artifact %s: %w", p.LogicalID, err)
			}
			return fmt.Errorf("bootstrap public artifact %s resolved digest %s, expected %s", p.LogicalID, resolved.Digest, e.Digest)
		}
		snapshot.Plugins = append(snapshot.Plugins, SnapshotEntry{LogicalID: p.LogicalID, Implementation: p.Implementation, SourceDir: p.SourceDir, Image: p.Image, Version: version, OCIRef: publicRef, Digest: e.Digest, InputHash: hash, SourceCommit: commit, ProvenanceMode: "public", Consumers: cloneConsumers(p.Consumers)})
	}
	return writeCanonical(*output, snapshot)
}

func commandApply(args []string) error {
	fs := flag.NewFlagSet("apply-plan", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root")
	planPath := fs.String("plan", "", "plan path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var plan Plan
	if *planPath == "" {
		return errors.New("--plan is required")
	}
	if _, err := readJSON(*planPath, &plan); err != nil {
		return err
	}
	return applyPlan(*root, plan)
}

func commandRender(args []string) error {
	fs, _, catalog := parseCommon("render-snapshot", args)
	plan := fs.String("plan", "", "plan path")
	previous := fs.String("previous", "", "previous snapshot")
	evidence := fs.String("candidate-evidence", "", "candidate evidence path")
	output := fs.String("output", "-", "snapshot output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *plan == "" || *evidence == "" {
		return errors.New("--plan and --candidate-evidence are required")
	}
	snapshot, err := renderSnapshot(*catalog, *plan, *previous, *evidence)
	if err != nil {
		return err
	}
	return writeCanonical(*output, snapshot)
}

func commandVerify(args []string) error {
	fs, root, catalog := parseCommon("verify-snapshot", args)
	snapshot := fs.String("snapshot", "", "snapshot path")
	expected := fs.String("expected-source", "", "expected exact pre-merge source ref")
	committed := fs.String("committed-source", "", "exact merged source ref whose VERSION files must match")
	resolve := fs.Bool("resolve", false, "resolve OCI manifests and verify provenance annotations")
	ociSource := fs.String("oci-source", "candidate", "OCI source to resolve: candidate or public")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *snapshot == "" {
		return errors.New("--snapshot is required")
	}
	return verifySnapshot(*root, *catalog, *snapshot, *expected, *committed, *resolve, *ociSource)
}

func commandCompare(args []string) error {
	fs := flag.NewFlagSet("semver-compare", flag.ContinueOnError)
	current := fs.String("current", "", "current stable version")
	candidate := fs.String("candidate", "", "candidate stable version")
	if err := fs.Parse(args); err != nil {
		return err
	}
	a, err := parseSemver(*current)
	if err != nil || a.prerelease != "" {
		return errors.New("--current must be stable SemVer")
	}
	b, err := parseSemver(*candidate)
	if err != nil || b.prerelease != "" {
		return errors.New("--candidate must be stable SemVer")
	}
	if compareSemver(b, a) < 0 {
		return fmt.Errorf("candidate %s would move latest backwards from %s", *candidate, *current)
	}
	return nil
}
