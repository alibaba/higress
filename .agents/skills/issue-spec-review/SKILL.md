---
name: issue-spec-review
description: Review an issue-spec implementation change, create line findings, reply after fixes, and sync REVIEW comments.
license: MIT
compatibility: Requires issue-spec CLI.
metadata:
  author: issue-spec
  version: "1.0"
  generatedBy: "issue-spec"
---

# Issue Spec Review

Coordinator: follow issue-spec-workflow to prepare the immutable review snapshot, dispatch a real independent reviewer, route repairs to the invariant owner, and link accepted evidence. On GitHub add --pr <number>; on a self-hosted profile omit --pr and add --revision <exact-head>. Sync authoritatively captures current rationale and emits one stable done REVIEW completion even with zero findings. Run these commands after the final review sync: issue-spec link --repo higress-group/higress --from REVIEW-<n> --from-issue <implement-issue> --to PROCESS-<n> --to-issue <implement-issue>, then issue-spec link --repo higress-group/higress --from REVIEW-<n> --from-issue <implement-issue> --to SPEC-<n> --to-issue <proposal-issue>. Do not copy Coordinator lifecycle or provider policy into the review packet.

## Review Role Packet

1. Accept only the sealed review assignment for the exact subject revision, immutable snapshot/diff, code authors, owned invariant, affected scenarios, review scope, focused checks, result schema, and design_context.
2. Require design_context.read_mode=complete-issue-body and conflict_policy=design-authoritative-stop. Before inspecting code, read the complete Design with issue-spec read issue --repo higress-group/higress --issue <design_context.source_url> without comments, timeline, history, or gates. Stop on conflict; do not collect or pass runtime-specific session IDs.
3. Review the invariant end to end at the exact revision. Verify required actions, stops, compatibility, tests, and major entry points. Do not expand into unrelated proposal history, DAGs, links, post-merge policy, or provider routing.
4. Under the real review agent identity, report actionable findings with severity, exact file/line, affected SPEC/scenario, owner PROCESS, and suggested fix, or an explicit no-finding verdict. Never fabricate evidence or let the Coordinator author findings for you.
5. After a fix, re-check the exact current revision and own the resolved reply/conversation resolution. Submit only the bounded review receipt/sync result and focused verification evidence. P0/P1 findings remain blocking until reviewer-owned resolution; the author cannot review its own work.

## Project Workflow

- Workflow Source: `builtin`
- Workflow Schema: `issue-spec`
- Workflow Config: `issue-spec/config.yaml`
- Workflow Diagnostics:

Project workflow templates are declarative only. Active proposal, design, implement, SPEC, TASK, PROCESS, QUESTION, REVIEW, and VERIFY artifacts remain in the selected issue backend's issue-native storage; repository-mode durable specs are materialized and checked on the implementation branch.
