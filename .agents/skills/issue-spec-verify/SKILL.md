---
name: issue-spec-verify
description: Run final issue-spec verification across exact-current review, test, check, rationale, and traceability evidence.
license: MIT
compatibility: Requires issue-spec CLI.
metadata:
  author: issue-spec
  version: "1.0"
  generatedBy: "issue-spec"
---

# Issue Spec Verify

Coordinator: use issue-spec-workflow for final routing. In repository durable mode, materialize the projection on the implementation branch before dispatch and seal the built-in issue-spec/durable-spec check into the verification assignment. Forecast with status --gate final --summary --json, resolve its detail actions, then run authoritative issue-spec verify --summary --json and full --json before merge. Change-bearing nodes require backend-appropriate rationale and REVIEW completion evidence. Status forecast and final verify use the same authoritative validator. The validator owns exact identity, revision, freshness, and legacy compatibility.

## Verification Role Packet

1. Accept only the sealed verification assignment for the exact immutable subject revision, affected scenarios, required test commands/check selectors, and result schema. Do not load proposal/Design bodies, the complete DAG, link matrices, post-merge policy, or provider routing.
2. Run only the required focused tests/checks against the exact revision. Keep local self-reported test evidence distinct from provider-owned check identity and conclusion; never invent externally observed check evidence.
3. Generate/submit the bounded VERIFY receipt under the real verifier identity. Record command/check identity, revision, result, and failures. Do not collect or pass runtime-specific session IDs.
4. A failed, pending, stale, or mismatched check is a blocker with a focused refresh/remediation result. Verification does not create or refresh REVIEW, infer links from prose, or replace independent review.

## Project Workflow

- Workflow Source: `builtin`
- Workflow Schema: `issue-spec`
- Workflow Config: `issue-spec/config.yaml`
- Workflow Diagnostics:

Project workflow templates are declarative only. Active proposal, design, implement, SPEC, TASK, PROCESS, QUESTION, REVIEW, and VERIFY artifacts remain in the selected issue backend's issue-native storage; repository-mode durable specs are materialized and checked on the implementation branch.
