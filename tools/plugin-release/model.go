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

import "encoding/json"

const (
	catalogSchemaVersion  = 1
	planSchemaVersion     = 1
	snapshotSchemaVersion = 1
)

type Catalog struct {
	SchemaVersion       int                            `json:"schemaVersion"`
	Registry            string                         `json:"registry"`
	SharedInputGroups   map[string][]string            `json:"sharedInputGroups"`
	ConsumerInventories map[string][]ConsumerInventory `json:"consumerInventories"`
	Plugins             []Plugin                       `json:"plugins"`
}

type ConsumerInventory struct {
	Key            string `json:"key"`
	Classification string `json:"classification"`
	LogicalID      string `json:"logicalId,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

type Plugin struct {
	LogicalID         string          `json:"logicalId"`
	Implementation    string          `json:"implementation"`
	SourceDir         string          `json:"sourceDir"`
	Image             string          `json:"image"`
	ReleaseEligible   bool            `json:"releaseEligible"`
	UnmanagedReason   string          `json:"unmanagedReason,omitempty"`
	ArtifactInputs    []string        `json:"artifactInputs"`
	SharedInputGroups []string        `json:"sharedInputGroups,omitempty"`
	Consumers         PluginConsumers `json:"consumers,omitempty"`
}

type PluginConsumers struct {
	Console      *ConsoleConsumer      `json:"console,omitempty"`
	PluginServer *PluginServerConsumer `json:"pluginServer,omitempty"`
}

type ConsoleConsumer struct {
	PropertyKey string `json:"propertyKey"`
	ResourceDir string `json:"resourceDir"`
	URLForm     string `json:"urlForm"`
}

type PluginServerConsumer struct {
	InventoryKey string   `json:"inventoryKey"`
	HTTPPath     string   `json:"httpPath"`
	Aliases      []string `json:"aliases,omitempty"`
}

type Plan struct {
	SchemaVersion   int              `json:"schemaVersion"`
	GatewayVersion  string           `json:"gatewayVersion"`
	SourceCommit    string           `json:"sourceCommit"`
	BaseCommit      string           `json:"baseCommit,omitempty"`
	PreviousRelease string           `json:"previousRelease,omitempty"`
	CatalogSHA256   string           `json:"catalogSha256"`
	PlanID          string           `json:"planId"`
	Plugins         []PlanEntry      `json:"plugins"`
	Deferred        []DeferredPlugin `json:"deferred,omitempty"`
}

type PlanEntry struct {
	LogicalID       string   `json:"logicalId"`
	Implementation  string   `json:"implementation"`
	SourceDir       string   `json:"sourceDir"`
	Image           string   `json:"image"`
	PreviousVersion string   `json:"previousVersion,omitempty"`
	Version         string   `json:"version"`
	InputHash       string   `json:"inputHash"`
	ChangedPaths    []string `json:"changedPaths"`
	// Backfill marks a bootstrap release entry whose stable public tag was
	// absent from the reviewed baseline: promotion creates the tag from this
	// candidate. The marker is bootstrap-only provenance and migration state;
	// after the complete version batch verifies, the entry participates in the
	// same serialized monotonic latest policy as every selected stable plugin.
	Backfill bool `json:"backfill,omitempty"`
}

// DeferredPlugin records a release-eligible catalog entry that was ignored
// for release selection. The only supported reason is an alpha prerelease
// VERSION, which denotes a development build without a public artifact.
type DeferredPlugin struct {
	LogicalID string `json:"logicalId"`
	Version   string `json:"version"`
	Reason    string `json:"reason"`
}

type CandidateEvidenceFile struct {
	Plugins map[string]CandidateEvidence `json:"plugins"`
}

type CandidateEvidence struct {
	CandidateRef string `json:"candidateRef"`
	Digest       string `json:"digest"`
	SourceCommit string `json:"sourceCommit"`
	InputHash    string `json:"inputHash"`
}

// BootstrapEvidenceFile is deliberately separate from candidate evidence. A
// bootstrap imports already-public immutable artifacts and must not pretend
// that historical images were built by the candidate workflow.
type BootstrapEvidenceFile struct {
	Plugins map[string]BootstrapEvidence `json:"plugins"`
}

// BootstrapEvidence classifies one release-eligible plugin's public artifact:
// "public" resolves to a reviewed digest, "missing" is a stable VERSION whose
// public tag is genuinely absent and needs a candidate backfill, and
// "deferred" is an alpha prerelease VERSION excluded from release selection.
// An authorization failure is never recorded here; it aborts the capture.
type BootstrapEvidence struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	PublicRef string `json:"publicRef,omitempty"`
	Digest    string `json:"digest,omitempty"`
}

type Snapshot struct {
	SchemaVersion   int    `json:"schemaVersion"`
	GatewayVersion  string `json:"gatewayVersion"`
	SourceCommit    string `json:"sourceCommit"`
	PreviousRelease string `json:"previousRelease,omitempty"`
	CatalogSHA256   string `json:"catalogSha256"`
	PlanID          string `json:"planId"`
	ProvenanceMode  string `json:"provenanceMode"`
	// BootstrapEvidence declares that this snapshot is the first managed
	// release rendered from the one-time bootstrap baseline and names the
	// committed bootstrap evidence the preparation PR must carry. Validation
	// never infers bootstrap mode from a missing previous-snapshot file.
	BootstrapEvidence *SnapshotBootstrapEvidence `json:"bootstrapEvidence,omitempty"`
	Plugins           []SnapshotEntry            `json:"plugins"`
}

// SnapshotBootstrapEvidence binds the first managed release to the exact
// committed bootstrap evidence file it was reviewed against. Path is the
// deterministic repository-relative committed location and SHA256 is the
// lowercase hex SHA-256 of its canonical bytes.
type SnapshotBootstrapEvidence struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type SnapshotEntry struct {
	LogicalID      string `json:"logicalId"`
	Implementation string `json:"implementation"`
	SourceDir      string `json:"sourceDir"`
	Image          string `json:"image"`
	Version        string `json:"version"`
	OCIRef         string `json:"ociRef"`
	Digest         string `json:"digest"`
	InputHash      string `json:"inputHash"`
	SourceCommit   string `json:"sourceCommit"`
	CandidateRef   string `json:"candidateRef"`
	ProvenanceMode string `json:"provenanceMode"`
	// Backfill marks an entry imported by building a candidate for a stable
	// VERSION whose public tag was absent at bootstrap. Promotion creates the
	// version tag from the candidate; the marker records bootstrap provenance
	// and migration state and is not an exclusion from the serialized
	// monotonic latest policy.
	Backfill  bool            `json:"backfill,omitempty"`
	Consumers PluginConsumers `json:"consumers,omitempty"`
}

func cloneConsumers(in PluginConsumers) PluginConsumers {
	data, _ := json.Marshal(in)
	var out PluginConsumers
	_ = json.Unmarshal(data, &out)
	return out
}
