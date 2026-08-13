// Copyright 2026 Higress Authors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestPluginServerWorkflowFailsClosedBeforeCandidateMutation(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/build-plugin-server-from-snapshot.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	lookup := strings.Index(workflow, "public tag lookup failed; refusing candidate/public mutation")
	build := strings.Index(workflow, "docker buildx build --platform")
	if lookup < 0 || build < 0 || lookup > build {
		t.Fatal("public lookup must fail closed before the candidate build/push")
	}
	for _, required := range []string{
		"docker/setup-qemu-action@29109295f81e9208d7d86ff1c6c12d2833863392", "docker/setup-buildx-action@e468171a9de216ec08956ac3ada2f0791b6bd435", "oras-project/setup-oras@8d34698a59f5ffe24821f0b48ab62a3de8b64b20",
		"length == 2", "linux/amd64", "linux/arm64", "org.opencontainers.image.revision",
		"snapshot-inventory.json", "jsonrpc-converter", "unmanaged-plugins.lock.json",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("workflow lacks required acceptance contract %q", required)
		}
	}
	buildData, err := os.ReadFile("../../.github/workflows/build-plugin-server-from-snapshot.yaml")
	if err != nil || !strings.Contains(string(buildData), "plugin-release-batches:$SNAPSHOT_SHA256") {
		t.Fatal("plugin-server build must require the immutable latest-completion marker")
	}
}

func TestReleaseWorkflowsPairORASSetupMetadataWithPinnedCLI(t *testing.T) {
	const orasSetup = "oras-project/setup-oras@8d34698a59f5ffe24821f0b48ab62a3de8b64b20 # v1.2.3"
	const orasSetupWithCLI = orasSetup + "\n        with:\n          version: 1.2.3"
	const supersededSetup = "oras-project/setup-oras@ca28077386065e263c03428f4ae0c09024817c93"

	expectedCallers := map[string]int{
		"prepare-plugin-release.yaml":            2,
		"promote-plugin-release.yaml":            2,
		"build-plugin-server-from-snapshot.yaml": 1,
		"authorize-higress-release-tag.yaml":     1,
		"dispatch-standalone-release.yaml":       1,
		"validate-plugin-preparation-pr.yaml":    1,
	}
	for name, expected := range expectedCallers {
		data, err := os.ReadFile("../../.github/workflows/" + name)
		if err != nil {
			t.Fatal(err)
		}
		workflow := string(data)
		if strings.Contains(workflow, supersededSetup) {
			t.Fatalf("%s retains the setup-oras v1.2.0 metadata action", name)
		}
		if got := strings.Count(workflow, "version: 1.2.3"); got != expected {
			t.Fatalf("%s has %d ORAS CLI 1.2.3 declarations, want %d", name, got, expected)
		}
		if got := strings.Count(workflow, orasSetupWithCLI); got != expected {
			t.Fatalf("%s has %d setup-oras v1.2.3 action/CLI pairs, want %d", name, got, expected)
		}
	}
}

func TestReleaseAuthorizerRequiresCompleteImmutableEvidence(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/authorize-higress-release-tag.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	if strings.Contains(workflow, "inputs.release_evidence") || strings.Contains(workflow, "RELEASE_EVIDENCE:") {
		t.Fatal("release authorizer must not accept operator-assembled release evidence JSON")
	}
	for _, required := range []string{
		"console_release_id", "schemaVersion", "gatewayVersion", "releaseCommit", "snapshotSha256", "consoleReleaseId", "consoleReleaseTag", "consoleReleaseCommit", "consoleChartOciRef", "consoleChartDigest", "consoleChartPackageSha256", "pluginServerCommit", "pluginServerHigressCommit", "pluginServerImage", "consoleProvenanceAssetId", "snapshot-console-map.json",
		"repos/higress-group/higress-console/releases/$CONSOLE_RELEASE_ID", "git/ref/tags/$CONSOLE_RELEASE_TAG",
		"oras manifest fetch \"$console_chart_repo@$CONSOLE_CHART_DIGEST\"", "git/ref/tags/$tag", "GitHub tag lookup failed", "snapshot-console-map.json", "jq -cnS", "/tmp/release-evidence.json",
		"resolved_plugin_server_digest", `[[ "$PLUGIN_SERVER_REGISTRY" != *"://"* ]]`, "helm/higress/Chart.yaml", "helm/higress/Chart.lock", "plugin-release-provenance.json",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("release authorizer lacks immutable evidence contract %q", required)
		}
	}
}

func TestReleaseAuthorizerBindsPluginServerToSnapshotAncestor(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/authorize-higress-release-tag.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		`child_higress_commit=$(jq -er '."io.higress.higress-source-commit"'`,
		`test "$child_higress_commit" = "$PLUGIN_SERVER_HIGRESS_COMMIT"`,
		`git merge-base --is-ancestor "$PLUGIN_SERVER_HIGRESS_COMMIT" "$RELEASE_COMMIT"`,
		`git show "$PLUGIN_SERVER_HIGRESS_COMMIT:$SNAPSHOT_PATH"`,
		`cmp /tmp/plugin-server-source-snapshot.json "$SNAPSHOT_PATH"`,
		`--committed-source "$PLUGIN_SERVER_HIGRESS_COMMIT"`,
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("release authorizer lacks snapshot-source ancestor contract %q", required)
		}
	}
	if strings.Contains(workflow, `--arg source "$RELEASE_COMMIT"`) {
		t.Fatal("plugin-server immutable labels must not be required to name the later release metadata commit")
	}
}

func TestReleaseAuthorizerRequiresApprovedDryRunEvidenceHash(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/authorize-higress-release-tag.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		"approved_readiness_evidence_sha256", `if [ "$DRY_RUN" = true ]`,
		`test -z "$APPROVED_EVIDENCE_SHA256"`,
		`[[ "$APPROVED_EVIDENCE_SHA256" =~ ^[0-9a-f]{64}$ ]]`,
		`test "$evidence_sha256" = "$APPROVED_EVIDENCE_SHA256"`,
		"Upload canonical pre-tag evidence",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("formal authorization can rebaseline dry-run evidence: missing %q", required)
		}
	}
}

func TestReleaseAuthorizerBindsHTTPSChartPackageToOCIContent(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/authorize-higress-release-tag.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		"azure/setup-helm@1a275c3b69536ee54be43f2070a358922e12c8d4",
		"console_dependency()", `test "$chart_yaml_dependency" = "$chart_lock_dependency"`,
		`helm pull higress-console --repo "$CONSOLE_HELM_REPOSITORY" --version "$CONSOLE_CHART_VERSION"`,
		`test "$(helm show chart "$CONSOLE_HTTP_CHART"`,
		`layers | length == 1`, "application/vnd.cncf.helm.chart.content.v1.tar+gzip",
		`oras blob fetch "$console_chart_repo@$CONSOLE_CHART_DIGEST" "$console_chart_layer"`,
		`test "sha256:$CONSOLE_CHART_PACKAGE_SHA256" = "$console_chart_layer"`,
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("Console Helm dependency is not cryptographically bound to OCI provenance: missing %q", required)
		}
	}
}

func TestPreparationBootstrapPlansFromExactHistoricalBase(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/prepare-plugin-release.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		"bootstrap_comparison_base", `[[ "$BOOTSTRAP_BASE" =~ ^[0-9a-f]{40}$ ]]`,
		`test "$(git rev-parse refs/remotes/origin/main)" = "$TARGET_REF"`,
		`git merge-base --is-ancestor "$BOOTSTRAP_BASE" "$TARGET_REF"`,
		"capture-bootstrap-evidence", "--existing-evidence /tmp/bootstrap-evidence.json",
		"--previous /tmp/bootstrap-snapshot.json", `--base "$BOOTSTRAP_BASE"`,
		"apply-plan", "Build and publish content-addressed candidates",
		// The first managed snapshot carries the explicit committed bootstrap
		// evidence marker: render receives the same evidence bytes that are
		// committed at the deterministic gateway-versioned path.
		"--bootstrap-evidence /tmp/bootstrap-evidence.json",
		`cp /tmp/bootstrap-evidence.json "plugins/release/bootstrap-evidence/$GATEWAY_VERSION.json"`,
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("preparation bootstrap lacks first-release contract %q", required)
		}
	}
	if strings.Count(workflow, `test "$(git rev-parse refs/remotes/origin/main)" = "$TARGET_REF"`) != 2 {
		t.Fatal("preparation must verify the exact main/code-freeze commit both before planning and before opening its base-main PR")
	}
	for _, forbidden := range []string{"inputs.existing_evidence", "PRODUCTION_REGISTRY_PASSWORD", "jq -n --arg gateway"} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("preparation bootstrap retains unsafe/empty-plan contract %q", forbidden)
		}
	}
}

func TestPreparationVersionOverridesPreserveCanonicalObjectAndRejectMalformedInputs(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/prepare-plugin-release.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	const prefix = `jq -ceS '`
	const suffix = `' > /tmp/overrides.json`
	start := strings.Index(workflow, prefix)
	if start < 0 {
		t.Fatal("preparation workflow lacks canonical validate-and-return version override filter")
	}
	start += len(prefix)
	endOffset := strings.Index(workflow[start:], suffix)
	if endOffset < 0 {
		t.Fatal("preparation workflow does not write validated version overrides to the planner input")
	}
	filter := workflow[start : start+endOffset]
	if strings.Contains(filter, `then true`) || !strings.Contains(filter, `then .`) {
		t.Fatal("version override validation must return the validated object, not a boolean predicate")
	}

	jq, err := exec.LookPath("jq")
	if err != nil {
		t.Fatal("jq is required to execute the preparation workflow contract")
	}
	run := func(input string) ([]byte, error) {
		cmd := exec.Command(jq, "-ceS", filter)
		cmd.Stdin = strings.NewReader(input)
		return cmd.CombinedOutput()
	}

	valid := map[string]string{
		`{}`: "{}\n",
		`{"z-plugin":"2.0.0","a_plugin":"1.2.3"}`: `{"a_plugin":"1.2.3","z-plugin":"2.0.0"}` + "\n",
	}
	for input, want := range valid {
		got, err := run(input)
		if err != nil {
			t.Fatalf("valid version overrides %s were rejected: %v\n%s", input, err, got)
		}
		if string(got) != want {
			t.Fatalf("valid version overrides %s were not preserved canonically: got %q, want %q", input, got, want)
		}
	}

	invalid := []string{
		`[]`,
		`{"Unsafe/Plugin":"1.2.3"}`,
		`{"safe-plugin":1}`,
		`{"safe-plugin":"1.2.3-alpha"}`,
		`{"safe-plugin":"01.2.3"}`,
		`{"safe-plugin":`,
	}
	for _, input := range invalid {
		if output, err := run(input); err == nil {
			t.Fatalf("invalid version overrides %s were accepted as %s", input, output)
		}
	}
}

func TestPreparationPreflightsReadOnlyDeterministicCatalogPublicManifest(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/prepare-plugin-release.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	preflightStart := strings.Index(workflow, "  preflight:\n")
	prepareStart := strings.Index(workflow, "  prepare:\n")
	bootstrap := strings.Index(workflow, "capture-bootstrap-evidence")
	if preflightStart < 0 || prepareStart < 0 || bootstrap < 0 || preflightStart > prepareStart || prepareStart > bootstrap {
		t.Fatal("read-only ORAS preflight job must run before the protected full bootstrap job")
	}
	preflight := workflow[preflightStart:prepareStart]
	if !strings.Contains(workflow[prepareStart:], "needs: preflight") {
		t.Fatal("protected preparation job must wait for the read-only preflight")
	}
	if strings.Contains(preflight, "environment:") || strings.Contains(preflight, "create-github-app-token") || strings.Contains(preflight, "oras login") {
		t.Fatal("ORAS preflight must remain read-only and outside the protected credential environment")
	}
	for _, required := range []string{
		"select(.releaseEligible)", "[$catalog.registry, .image, .sourceDir] | @tsv",
		`[ "${prerelease%%.*}" = "alpha" ]`, `attempted=$((attempted + 1))`,
		`descriptor=$(oras manifest fetch "$public_ref" --descriptor`, `jq -er '.digest | strings and test("^sha256:[0-9a-f]{64}$")'`,
		`present=$((present + 1))`, `absent=$((absent + 1))`, `authenticated_capture=0`,
		`authenticated_capture=$((authenticated_capture + 1))`,
		`grep -Fq 'Error response from registry:'`, `grep -Fq "$public_ref: not found"`,
		"public registry preflight requires authenticated capture; authorization failure is never absence",
		"public registry preflight failed without explicit absence evidence",
		"Read-only ORAS preflight classified $attempted deterministic public references",
		"$authenticated_capture require authenticated capture",
	} {
		if !strings.Contains(preflight, required) {
			t.Fatalf("preflight lacks required read-only contract %q", required)
		}
	}
	if strings.Contains(preflight, "sort_by(.logicalId) | .[0]") || strings.Contains(preflight, "break\n") {
		t.Fatal("preflight must classify every deterministic non-alpha reference, not select or stop at the first entry")
	}
	if strings.Contains(preflight, "--descriptor --format") {
		t.Fatal("ORAS 1.2.3 descriptor preflight must not combine --descriptor and --format")
	}
	authStart := strings.Index(preflight, "if grep -Eiq '401|403|")
	absenceStart := strings.Index(preflight, "if grep -Eiq '404|manifest unknown|")
	if authStart < 0 || absenceStart < 0 || authStart > absenceStart {
		t.Fatal("uncredentialed preflight must classify authorization before explicit absence")
	}
	authBranch := preflight[authStart:absenceStart]
	if !strings.Contains(authBranch, "continue") || strings.Contains(authBranch, "exit 1") {
		t.Fatal("uncredentialed preflight must defer 401/403 to authenticated capture and continue without claiming absence")
	}
}

func TestPreparationPRValidatorPinsAllActions(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/validate-plugin-preparation-pr.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, pinned := range []string{
		"actions/checkout@11d5960a326750d5838078e36cf38b85af677262 # v4",
		"actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff # v5",
		"oras-project/setup-oras@8d34698a59f5ffe24821f0b48ab62a3de8b64b20 # v1.2.3",
	} {
		if !strings.Contains(workflow, pinned) {
			t.Fatalf("preparation PR validator lacks pinned action %q", pinned)
		}
	}
	for _, floating := range []string{"actions/checkout@v4", "actions/setup-go@v5"} {
		if strings.Contains(workflow, floating) {
			t.Fatalf("preparation PR validator retains floating action %q", floating)
		}
	}
}

func TestRegistryAbsenceShellClassifiersRequireStructuredQualifiedEvidence(t *testing.T) {
	files := map[string][]string{
		"prepare-plugin-release.yaml":            {`grep -Fq 'Error response from registry:'`, `grep -Fq "$public_ref: not found"`},
		"promote-plugin-release.yaml":            {`grep -Fq 'Error response from registry:'`, `grep -Fq "$ref: not found"`},
		"build-plugin-server-from-snapshot.yaml": {`grep -Fq 'Error response from registry:'`, `grep -Fq "$image: not found"`},
	}
	for name, required := range files {
		data, err := os.ReadFile("../../.github/workflows/" + name)
		if err != nil {
			t.Fatal(err)
		}
		workflow := string(data)
		for _, marker := range required {
			if !strings.Contains(workflow, marker) {
				t.Fatalf("%s lacks structured registry absence marker %q", name, marker)
			}
		}
		if strings.Contains(workflow, "not found|repository does not exist") {
			t.Fatalf("%s accepts generic not-found text", name)
		}
	}
}

func TestLatestPromotionMarkerSupportsFreshPartialAndCompletedRetries(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/promote-plugin-release.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	if strings.Count(workflow, `marker="$REGISTRY/candidates/plugin-release-batches:$SNAPSHOT_SHA256"`) != 2 {
		t.Fatal("both version and latest jobs must independently define the completion marker")
	}
	for _, required := range []string{
		`phase:"version-complete"`, `phase == "latest-complete"`,
		`/tmp/plugin-release-version-$SNAPSHOT_SHA256.json`,
		`/tmp/plugin-release-latest-$SNAPSHOT_SHA256.json`,
		`no mutable latest tags were written`, `immutable latest marker conflict`,
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("latest marker state machine lacks %q", required)
		}
	}
}

func TestPromotionBackfillsVersionTagAndJoinsMonotonicLatest(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/promote-plugin-release.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	// The version phase iterates every snapshot entry; an absent tag is created
	// only from candidate provenance (which is how a backfill gets its stable
	// tag), an identical existing digest is accepted, and any conflict or a
	// missing carried-forward public artifact fails closed.
	for _, required := range []string{
		`done < <(jq -c '.plugins[]' "$SNAPSHOT_PATH")`,
		`select(.preflight == "absent" and .provenanceMode == "candidate")`,
		`test "$existing" = "$digest" || { echo "immutable tag conflict: $ref" >&2; exit 1; }`,
		`state=already-present`,
		`previous public artifact is missing`,
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("version promotion lacks the backfill/idempotence contract %q", required)
		}
	}
	// Every selected stable plugin participates in the serialized monotonic
	// latest policy, including historical bootstrap backfill entries: create
	// when absent, accept an identical digest without a write, advance an older
	// reliably annotated stable version, and fail closed on an unclassifiable
	// alias, a downgrade, or a same-version digest conflict. The final
	// re-resolve rechecks the complete set, which keeps retries idempotent.
	for _, required := range []string{
		`existing latest lacks a stable version annotation`,
		`semver-compare --current "$old" --candidate "$version"`,
		`latest alias conflict`,
		`state=identical`,
		`select(.preflight != "identical")`,
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("latest promotion lacks the monotonic complete-set contract %q", required)
		}
	}
	if strings.Count(workflow, `done < <(jq -c '.plugins[]' "$SNAPSHOT_PATH")`) != 2 {
		t.Fatal("version and latest phases must both iterate the complete selected snapshot set")
	}
	if strings.Contains(workflow, `select(.backfill != true)`) {
		t.Fatal("backfill is provenance/migration state, never a blanket exclusion from latest")
	}
	latestDigest := strings.Index(workflow, `if [ "$old_digest" = "$digest" ]; then`)
	latestAnnotation := strings.Index(workflow, `old=$(oras manifest fetch "$latest" --format json`)
	if latestDigest < 0 || latestAnnotation < 0 || latestDigest > latestAnnotation {
		t.Fatal("latest promotion must accept an identical digest before requiring a legacy version annotation")
	}
}

func TestPromotionDryRunResolvesCandidateProvenanceWithoutMutation(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/promote-plugin-release.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	verifyStart := strings.Index(workflow, "      - name: Verify exact snapshot provenance\n")
	dryRunBoundary := strings.Index(workflow, "      - name: Confirm dry-run boundary\n")
	if verifyStart < 0 || dryRunBoundary < 0 || verifyStart > dryRunBoundary {
		t.Fatal("promotion must verify exact provenance before reaching its dry-run boundary")
	}
	verify := workflow[verifyStart:dryRunBoundary]
	for _, required := range []string{
		`--plan "../../$plan"`, `--committed-source "$SOURCE_COMMIT"`,
		`--resolve --oci-source candidate`, `args+=(--previous "../../$previous_path")`,
	} {
		if !strings.Contains(verify, required) {
			t.Fatalf("promotion dry run lacks exact provenance resolution contract %q", required)
		}
	}
	for _, mutation := range []string{"oras login", "oras cp", "oras push"} {
		if strings.Contains(verify, mutation) {
			t.Fatalf("promotion dry-run verification contains registry mutation %q", mutation)
		}
	}
}

func TestPluginServerDryRunChecksExactBuildInputsAndCandidateProvenance(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/build-plugin-server-from-snapshot.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	dryRunBoundary := strings.Index(workflow, "      - name: Confirm dry-run boundary\n")
	if dryRunBoundary < 0 {
		t.Fatal("plugin-server workflow lacks a dry-run boundary")
	}
	verify := workflow[:dryRunBoundary]
	for _, required := range []string{
		"oras-project/setup-oras@8d34698a59f5ffe24821f0b48ab62a3de8b64b20 # v1.2.3",
		`repository: higress-group/plugin-server`, `ref: ${{ inputs.plugin_server_commit }}`,
		`[[ "$GATEWAY_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]`,
		`test "$SNAPSHOT_PATH" = "plugins/release/snapshots/$GATEWAY_VERSION.json"`,
		`test "$(jq -er .gatewayVersion "$SNAPSHOT_PATH")" = "$GATEWAY_VERSION"`,
		`test "$(git -C plugin-server rev-parse HEAD)" = "$PLUGIN_SERVER_COMMIT"`,
		`(cd plugin-server && python3 -m unittest test_pull_plugins.py)`,
		`--plan "../../$plan"`, `--resolve --oci-source candidate`,
		`args+=(--previous "../../$previous_path")`,
	} {
		if !strings.Contains(verify, required) {
			t.Fatalf("plugin-server dry run lacks exact input contract %q", required)
		}
	}
	for _, mutation := range []string{"docker login", "docker buildx build", "docker buildx imagetools create", "gh api --method POST"} {
		if strings.Contains(verify, mutation) {
			t.Fatalf("plugin-server dry-run verification crosses mutation boundary with %q", mutation)
		}
	}
}

func TestReleaseAuthorizerBindsConsoleLockToCanonicalPluginServer(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/authorize-higress-release-tag.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	image := strings.Index(workflow, `PLUGIN_SERVER_IMAGE="$plugin_server_tag@$resolved_plugin_server_digest"`)
	lock := strings.Index(workflow, `.pluginLock.pluginServerCommit == $plugin and .pluginLock.pluginServerImage == $image`)
	inspect := strings.Index(workflow, `oras manifest fetch "$PLUGIN_SERVER_IMAGE" --raw`)
	if image < 0 || lock < 0 || inspect < 0 || image > lock || lock > inspect {
		t.Fatal("release authorizer must bind the Console lock to the canonical plugin-server image before inspecting it")
	}
}

func TestStandaloneDispatchBindsUniqueConsoleEvidenceToPluginServer(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/dispatch-standalone-release.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		`test "$(jq '[.assets[] | select(.name == "plugin-release-provenance.json")] | length' <<<"$console_release")" = 1`,
		`test "$(jq -er .gatewayVersion "$snapshot")" = "${TAG#v}"`,
		`SNAPSHOT_SOURCE_COMMIT=$(jq -er .sourceCommit "$snapshot")`,
		`git merge-base --is-ancestor "$SNAPSHOT_SOURCE_COMMIT" "$COMMIT"`,
		`git show "$SNAPSHOT_SOURCE_COMMIT:$snapshot"`,
		`cmp /tmp/plugin-server-source-snapshot.json "$snapshot"`,
		`plugin_server_image="$plugin_server_ref@$plugin_server"`,
		`PLUGIN_SERVER_COMMIT=$(jq -er .pluginLock.pluginServerCommit /tmp/console-provenance.json)`,
		`.pluginLock.sourceCommit == $sourceCommit and .pluginLock.snapshotSha256 == $snapshot`,
		`.pluginLock.pluginServerCommit == $pluginCommit and .pluginLock.pluginServerImage == $pluginImage`,
		`test "$digest" = "$plugin_server"`,
		`jq -cn --arg ref "$ref" --arg digest "$digest"`,
		`plugin_json=$(inspect_plugin_server "$plugin_server_ref")`,
		`io.higress.higress-source-commit`,
		`org.opencontainers.image.revision`,
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("standalone dispatch lacks exact Console/plugin-server binding %q", required)
		}
	}
	commit := strings.Index(workflow, `PLUGIN_SERVER_COMMIT=$(jq -er .pluginLock.pluginServerCommit`)
	inspect := strings.Index(workflow, `inspect_plugin_server() {`)
	if commit < 0 || inspect < 0 || commit > inspect {
		t.Fatal("standalone dispatch must derive the plugin-server commit from signed Console provenance before image inspection")
	}
	if strings.Contains(workflow, `.pluginLock.snapshotSha256 != null`) {
		t.Fatal("standalone dispatch must compare the exact snapshot hash, not merely require a non-null lock field")
	}
	if strings.Contains(workflow, `jq -cn --arg ref "$plugin_server_image"`) {
		t.Fatal("standalone evidence must keep pluginServer.ref as the gateway-version tag and carry its digest separately")
	}
	if strings.Contains(workflow, `--arg sourceCommit "$COMMIT"`) || strings.Contains(workflow, `--arg source "$COMMIT" --arg snapshot`) {
		t.Fatal("standalone dispatch must bind plugin-server provenance to the snapshot source ancestor, not the later release commit")
	}
}

func TestPreparationBootstrapReusesProtectedCandidateRegistryCredential(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/prepare-plugin-release.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		`CANDIDATE_REGISTRY: ${{ vars.PLUGIN_CANDIDATE_REGISTRY }}`,
		`REGISTRY_USERNAME: ${{ secrets.CANDIDATE_REGISTRY_USERNAME }}`,
		`REGISTRY_PASSWORD: ${{ secrets.CANDIDATE_REGISTRY_PASSWORD }}`,
		`[[ "$CANDIDATE_REGISTRY" =~ ^[a-z0-9][a-z0-9.-]*(\:[0-9]+)?$ ]]`,
		`test "$CANDIDATE_REGISTRY" = "$(jq -er .registry plugins/release/catalog.json)"`,
		`test -n "$REGISTRY_USERNAME"; test -n "$REGISTRY_PASSWORD"`,
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("bootstrap capture does not reuse the protected candidate registry contract %q", required)
		}
	}
	branch := strings.Index(workflow, `if [ -z "$PREVIOUS" ]; then`)
	login := strings.Index(workflow, `echo "$REGISTRY_PASSWORD" | oras login "$CANDIDATE_REGISTRY" -u "$REGISTRY_USERNAME" --password-stdin`)
	capture := strings.Index(workflow, "capture-bootstrap-evidence")
	if branch < 0 || login < 0 || capture < 0 || branch > login || login > capture {
		t.Fatal("the existing candidate registry login must stay inside the bootstrap branch and precede read-only evidence capture")
	}
	for _, obsolete := range []string{"PLUGIN_REGISTRY_READER_USERNAME", "PLUGIN_REGISTRY_READER_PASSWORD", "PLUGIN_PUBLIC_REGISTRY"} {
		if strings.Contains(workflow, obsolete) {
			t.Fatalf("preparation must not require obsolete registry configuration %q", obsolete)
		}
	}
}

func TestPreparationPRValidationBindsMixedSnapshotToBootstrapEvidence(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/validate-plugin-preparation-pr.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		"candidate | mixed)",
		// The jq gate checks exact backfill equality between plan and snapshot
		// in both directions before the Go verifier runs.
		`(($published.backfill // false) == ($entry.backfill // false))`,
		// Bootstrap mode is explicit committed provenance: the snapshot marker
		// names the deterministic evidence file the preparation PR carries, and
		// verify-snapshot recomputes its digest and binds every claim to it.
		`marker=$(jq -r '.bootstrapEvidence.path // empty' "$snapshot")`,
		`[[ "$marker" =~ ^plugins/release/bootstrap-evidence/[0-9]+\.[0-9]+\.[0-9]+\.json$ ]]`,
		`test -f "$marker"`,
		// The snapshot is always bound to the exact committed plan; a managed
		// predecessor is bound through --previous only when no marker is
		// present (the bootstrap baseline itself is never committed).
		`--plan "../../$plan"`,
		`previous_path="plugins/release/snapshots/$previous.json"`,
		`args+=(--previous "../../$previous_path")`,
		"--oci-source candidate",
		"--oci-source public",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("preparation PR validation lacks the committed-evidence binding contract %q", required)
		}
	}
	// Bootstrap mode must never be inferred from a previous snapshot file that
	// the prepare workflow never commits; that inference silently skipped the
	// evidence cross-check.
	if strings.Contains(workflow, `provenanceMode // empty`) {
		t.Fatal("validation must not infer bootstrap mode from a previous snapshot provenanceMode file lookup")
	}
	if strings.Count(workflow, "verify-snapshot") != 2 {
		t.Fatal("candidate/mixed and bootstrap-public snapshots must each keep one verify-snapshot gate")
	}
	// verify-snapshot --resolve executes the oras CLI and the workflow runs the
	// release tool tests, so this required PR gate must install the pinned ORAS
	// 1.2.3 before either step: without it the resolver fails closed with
	// executable-not-found and the executable descriptor contract test would
	// silently skip instead of exercising the pinned CLI.
	orasSetup := strings.Index(workflow, "oras-project/setup-oras@8d34698a59f5ffe24821f0b48ab62a3de8b64b20 # v1.2.3")
	toolTests := strings.Index(workflow, "go test ./...")
	verify := strings.Index(workflow, "verify-snapshot")
	if orasSetup < 0 || toolTests < 0 || verify < 0 || orasSetup > toolTests || orasSetup > verify {
		t.Fatal("validation must install the pinned ORAS 1.2.3 CLI before running the release tool tests and verify-snapshot --resolve")
	}
}

func TestGatewayStableAliasesAreStagedAndImmutable(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/build-image-and-push.yaml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		`release-build-$GITHUB_SHA`, `release-provenance-$GITHUB_SHA`,
		`immutable stable image tag conflict`, `:v$version`,
		`BUILDX_NO_DEFAULT_ATTESTATIONS=1`, `stable image tag lookup failed; refusing mutation`,
		`authorization failure is never absence`,
		`401|403|unauthorized|forbidden|denied|authentication required|authorization required`,
		`404|manifest unknown|name unknown|repository does not exist`,
		`verify_release_staging "$staging"`,
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("gateway release publication lacks %q", required)
		}
	}
	if strings.Contains(workflow, "not found|repository does not exist") {
		t.Fatal("generic \"not found\" text is never registry absence evidence; only explicit 404-class markers qualify")
	}
	if strings.Count(workflow, "descriptor_or_absent()") != 3 || strings.Count(workflow, "verify_release_staging()") != 3 {
		t.Fatal("every controller/pilot/gateway publisher must use its own fail-closed lookup and staging acceptance helper")
	}
}
