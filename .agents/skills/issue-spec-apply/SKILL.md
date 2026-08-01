---
name: issue-spec-apply
description: Implement PROCESS comments for an issue-spec change and keep implementation-change traceability synchronized.
license: MIT
compatibility: Requires issue-spec CLI.
metadata:
  author: issue-spec
  version: "1.0"
  generatedBy: "issue-spec"
---

# Issue Spec Apply

Coordinator: persist the Implement issue, perform its first QUESTION discovery/create pass, then complete PROCESS planning. Issue bodies and typed artifacts remain authoritative.

Complete DAG planning, workspace lifecycle, integration, links, review, recovery, and final evidence by following the backend-appropriate routing in issue-spec-workflow. For every agent-executed change-bearing PROCESS, seal the implementation assignment and dispatch a real non-Coordinator implementation worker with the Implementation Role Packet below; this is required regardless of node size, file count, or serial versus parallel scheduling, and the coordinator never implements, tests, or commits that PROCESS inline or uses workspace_management: independent as an escape hatch. A compatible real worker may execute successive serial or repair nodes; for each assignment give that worker only the parent TASK context plus the predecessor handoff, never the coordinator's accumulated context. Non-change-bearing orchestration and the narrow direct one-file PR path without a PROCESS DAG retain their existing policies. Run the authoritative final sync by following issue-spec-review. After that sync, explicitly link the REVIEW to its review PROCESS, every covered change-bearing PROCESS, and every covered active SPEC. Follow issue-spec-workflow for the backend-appropriate rationale command. Each owning worker authors its own rationale under that worker's --agent. Do not copy that policy into a worker packet.

## Implementation Role Packet

This packet is addressed to the dispatched worker subagent. Relay it verbatim with the sealed assignment; do not execute it in the coordinator context.

1. Accept only the sealed implementation assignment for the exact PROCESS, base revision, worktree, write ownership, focused tests, generators, result schema, and design_context. Do not load proposal bodies, the complete DAG, link matrices, post-merge policy, provider routing, or unrelated artifacts.
2. Before code changes, require design_context.read_mode=complete-issue-body and conflict_policy=design-authoritative-stop. Read the complete Design with issue-spec read issue --repo higress-group/higress --issue <design_context.source_url> without comments, timeline, history, or gates. Stop and report any conflict; do not reinterpret or summarize the packet.
3. Work only in the assigned worktree and owned paths. Preserve the named invariant, decisions, must_preserve, must_not, and minimum_verification exactly. Do not collect or pass runtime-specific session IDs.
4. Implement the owned invariant, run the assigned generators exactly, and run focused verification. If the assignment cannot fit a bounded end-to-end working set, stop with the concrete stable-interface split options and acceptance consequences; do not split by path, command, finding, or token formula.
5. Produce exactly one DCO commit when required. Return only the result commit, changed paths, generator outputs, focused test results, decisions, risks, and bounded handoff/result receipt. Do not integrate, clean up, publish Coordinator artifacts, review your own code, or create final rationale before independent review converges.

## PROCESS Write Ownership

- A bare repository-relative ownership path is one exact file.
- A directory subtree requires an explicit trailing /** declaration, for example internal/templates/**.
- Legacy bare directory declarations remain readable, but workspace prepare may reject them; correct the PROCESS or pass an explicit recursive ownership value before allocation.

## Project Workflow

- Workflow Source: `builtin`
- Workflow Schema: `issue-spec`
- Workflow Config: `issue-spec/config.yaml`
- Workflow Diagnostics:

Project workflow templates are declarative only. Active proposal, design, implement, SPEC, TASK, PROCESS, QUESTION, REVIEW, and VERIFY artifacts remain in the selected issue backend's issue-native storage; repository-mode durable specs are materialized and checked on the implementation branch.
