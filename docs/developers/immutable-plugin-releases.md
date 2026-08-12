# Immutable Go/Rust plugin release operation

This procedure applies only to official Go extensions in
`plugins/wasm-go/extensions/` and official Rust extensions in
`plugins/wasm-rust/extensions/`. C++, AssemblyScript, Go examples, Rust
examples, and explicitly unmanaged downstream defaults are not republished or
rewritten by this path.

## Required administrative cutover

Before a non-dry-run release, an administrator must configure all of the
following. The checked-in workflows deliberately fail closed until these
controls exist.

- Make every public plugin tag other than `latest` immutable in the registry.
  Retain candidate artifacts only in a separate content-addressed namespace
  with an expiry policy.
- Give the protected `plugin-release-production` environment the sole
  credential able to write `plugins/<plugin>:<VERSION>` and `:latest`; remove
  equivalent public-write access from legacy Wasm workflows and users.
- Give the protected `plugin-server-production` environment the sole
  credential able to write `higress/plugin-server:<gateway-version>`. The
  plugin-server repository publisher may write only its development/candidate
  namespace.
- Create a scoped GitHub App for each downstream repository dispatch/PR
  receiver. It may update the deterministic dependency PR but may not merge,
  tag, or release.
- Protect `higress-release-manager` with human approval and a tag ruleset that
  permits only its release-manager App to create `vMAJOR.MINOR.PATCH` tags.
- Record negative checks for every retained credential: a conflicting public
  version tag and gateway-version image overwrite must be refused.

Environment protection, registry ACLs, lifecycle policy, tag rulesets, and
credential revocation are external administration. They cannot be asserted by
repository source alone.

## Preparation and promotion

1. At code freeze, record the exact 40-character commit at the current `main`
   head. Do not open or update the normal release metadata/Helm/submodule PR
   (for example, a PR like #4019) yet. `prepare-plugin-release` verifies that
   `target_ref` is exactly this main head; its generated branch therefore adds
   only plugin preparation changes when opened back against `main`.
2. Run `prepare-plugin-release` using that exact main/code-freeze commit, a
   stable gateway version, and `dry_run=true`. It validates the complete
   catalog and writes a deterministic plan. Version overrides are only
   proposed stable `VERSION` edits in the generated review branch; a publisher
   must never trust the dispatch payload as a public tag. For the first managed
   release only, set `bootstrap_comparison_base` to the exact 40-character
   commit of the last stable Higress release (`v2.2.3` is
   `39ec41aab6eb1d40499bed2847085696de0ebb96`). The workflow captures current
   public refs/digests read-only, then compares artifact inputs from that
   ancestor; ordinary snapshots reject this one-time override.
3. Build/test every plan entry through the protected candidate publisher. Each
   candidate is keyed by plan ID/input hash and reports its manifest digest,
   source commit, and input hash. Render the snapshot only after every changed
   entry has matching evidence. A later source or `VERSION` edit invalidates
   that evidence.
4. Review and merge the one preparation PR containing `VERSION` edits and
   `plugins/release/snapshots/<gateway-version>.json`. The snapshot carries all
   release-eligible plugins, including unchanged entries carried forward by
   digest. It is the only downstream input.
5. Run `promote-plugin-release` for that exact merged commit and snapshot
   hash. A production publisher may create a missing public version tag or
   accept an identical existing digest only. It must fail before mutation on a
   conflicting digest. Advance `latest` only after the full batch verifies and
   only when its stable SemVer cannot move backwards.
6. Build `higress/plugin-server:<gateway-version>` from the exact approved
   plugin-server commit and snapshot. Verify image platforms, labels, response
   content hashes, and the snapshot hash before downstream dispatch.
7. Dispatch Console from the exact snapshot-carrying Higress commit, then merge
   its generated PR and wait for the Console chart containing that lock to be
   released. Only now create or update the normal #4019-style release PR from
   current `main`, limiting it to release metadata, Helm dependency, release
   notes, and intended submodule pins. Set the released Console dependency and
   run `helm dependency update helm/higress` so `Chart.lock` is regenerated;
   review both `Chart.yaml` and `Chart.lock`. Never manually edit managed plugin
   `VERSION` files there. If this release PR changes any catalog-declared plugin
   artifact input, stop and reprepare the snapshot/plugin-server/Console chain
   from the new exact main commit. Merge the release PR only after these exact
   dependencies and readiness checks converge, immediately before exact tag
   authorization. After the Higress release, dispatch Standalone with
   immutable release/chart/image identities. It derives a deterministic PR key
   and must not read a newer branch head.

The pre-tag `authorize-higress-release-tag` workflow rechecks the exact merged
release commit and promoted snapshot. It derives the identical snapshot-source
Higress commit from every plugin-server platform label, requires it to be an
ancestor of the release commit, and proves snapshot bytes and committed plugin
inputs at both commits are unchanged. It also resolves the released Console
chart through the HTTPS repository declared identically by `Chart.yaml` and
`Chart.lock`, then requires those package bytes to equal the unique Helm content
layer under the provenance-pinned OCI manifest digest. Human approval and the
tag ruleset are mandatory; ordinary tag-triggered workflows are post-tag
publishers and cannot satisfy this gate.

## Canonical release evidence and downstream order

Release-critical dispatches use canonical JSON (`jq -cS`) and a SHA-256 key of
the canonical bytes. The Higress `release.published` sender resolves its own
release ID/tag/peeled commit, snapshot hash, Console release ID/tag/commit and
chart digest, and plugin-server ref/digest before it sends that evidence to
Standalone. Standalone re-resolves each identity and records the exact
Standalone base SHA in `release-provenance.json`; retries reuse only a branch
whose provenance bytes match. It never rebases onto a newer `main`.

The required configuration is `PLUGIN_SERVER_REGISTRY`, protected environment
registry credentials, and the scoped App variables/secrets named by the
workflows. The Console chart must be published before tag authorization. The
operator supplies its immutable GitHub release ID plus the pinned plugin-server
source/digest; readiness re-resolves the release, provenance asset, both chart
distribution paths, plugin-server index, and child labels and emits canonical
release-evidence JSON. The dry-run artifact/hash is reviewed first. Formal
authorization requires that approved SHA-256 and refuses if re-resolved
canonical bytes differ. Neither path accepts hand-assembled evidence. A
Standalone `release.published` event carries
the release ID, dereferenced tag commit, archive and installer checksums and a
canonical key back to Higress OSS packaging. Repository-dispatch is production
by design; only the manual workflow path honours `dry_run=true`.

## Recovery and rollback

Retries use the same immutable identifiers. A retry may accept an existing
tag only when it resolves to the expected digest. Never overwrite a version
tag. Recover by creating a new reviewed snapshot/release that refers to an
earlier verified digest set. If `latest` needs repair, use the serialized
promotion flow with an audited recovery approval; released Console, gateway,
and Standalone defaults remain pinned and do not depend on `latest`.

## Exact Standalone packaging

`deploy-standalone-to-oss` is manual and requires one verified Standalone
`vX.Y.Z` tag plus SHA-256 values for both its archive and
`src/get-higress.sh`. It downloads both from that exact tag and fails for a
missing tag or checksum mismatch. It never selects a release from HTML or
downloads an installer from `main`.
