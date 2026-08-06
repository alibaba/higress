---
name: issue-spec-propose
description: Create or continue proposal, SPEC, QUESTION, design, and TASK artifacts for an issue-spec change.
license: MIT
compatibility: Requires issue-spec CLI.
metadata:
  author: issue-spec
  version: "1.0"
  generatedBy: "issue-spec"
---

# Issue Spec Propose

Use when the user asks for /issue-spec:propose, proposal, Design, SPEC, QUESTION, or TASK authoring. Use issue-spec-workflow for shared reads, provider routing, and recovery.

1. Validate workflow config, search related issues, and open only selected discussions. If the issue is already in a later phase, continue that phase rather than duplicating it.
2. Keep unconfirmed investigation, reproduction, or triage notes in a simple issue with issue-spec issue create simple; a proposal states the confirmed problem and the intended change, so never promote an investigation issue into the proposal or attach SPEC/Design to it. Create phase issues with concrete body files, beginning with issue-spec issue create proposal --repo higress-group/higress --body-file <file>. Follow the workflow `rules.language` and `rules.language_instructions` for every Issue title. When those rules require a localized or non-English title, pass an explicit `--title` for Proposal, Design, and Implement; do not rely on the derived title because it retains an English stage prefix. Otherwise use the standardized Proposal:, Design:, and Implement: title family. Do not perform style-only title rewrites after creation.
3. Perform the Proposal's first QUESTION discovery/create pass. Record each genuine unresolved decision as a blocking typed QUESTION with issue-spec question create, attaching a choice model when credible options exist; never leave an open decision as body or projection prose. Do not manufacture a question or reopen a settled choice; keep unresolved decisions distinct from evidence-dependent items.
4. Upsert `proposal-choice-brief` after that pass and before complete SPEC authoring. Lead with a representative human or operator scene and a concrete before/after case, then cover the problem, outcome, success signal, boundaries, non-goals, assumptions, risks, decisions, alternatives, and expected SPEC coverage. Distinguish settled, needs-evidence, and needs-decision items; show how options change the case. With no open decision, keep the other review dimensions visible. The projection is ordinary and statusless.
5. Generate canonical SPEC comments with issue-spec comment generate --type SPEC. Requirements must be testable and include WHEN/THEN scenarios. --allow-noncanonical is a migration bypass, not normal authoring.
6. Persist the authoritative self-contained Design, perform its first QUESTION discovery/create pass, then upsert `design-explainer` before complete TASK planning. Lead with a concrete request or operator case and observable outcome, then trace its normal and failure paths through architecture, invariants, interfaces, state, alternatives, compatibility, rollout, risks, verification, and active SPEC traceability. Use purposeful interaction to make the complete review surface easier to navigate.
7. Generate TASK comments with issue-spec comment generate --type TASK. Execution Planning must identify Design-invariant cohesion and major entry points, bounded role-context pressure, stable interfaces, owned areas, shared touchpoints, dependencies, coupling, and acceptance consequences. File ownership and parallelism are scheduling context, not semantic PROCESS boundaries. Execution modes such as coordinator-owned describe scheduling only; they never authorize coordinator-inline implementation of an agent-executed change-bearing PROCESS.
8. Link SPEC <-> TASK, verify links, and run status --gate proposal/design/implement --summary --json as appropriate. Do not enter Implement while a semantic boundary decision is unresolved; block and ask a human.

## Human Review Projections

Before generating or updating `proposal-choice-brief` or `design-explainer`, read [Human Review Projection Generation](../issue-spec-workflow/references/human-review-projections.md) completely and apply the matching phase recipe. Build the phase coverage ledger and generate a coverage-complete `projection.md` from current authoritative inputs before running `projection upsert`; never rely on the reviewer already knowing omitted design information.

## Project Workflow

- Workflow Source: `builtin`
- Workflow Schema: `issue-spec`
- Workflow Config: `issue-spec/config.yaml`
- Workflow Diagnostics:

Project workflow templates are declarative only. Active proposal, design, implement, SPEC, TASK, PROCESS, QUESTION, REVIEW, and VERIFY artifacts remain in the selected issue backend's issue-native storage; repository-mode durable specs are materialized and checked on the implementation branch.
