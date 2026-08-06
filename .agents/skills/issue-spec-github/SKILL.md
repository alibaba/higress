---
name: issue-spec-github
description: Use GitHub CLI for GitHub issues, pull requests, CI runs, and API queries that issue-spec does not wrap.
license: MIT
compatibility: Requires GitHub CLI (gh).
metadata:
  author: issue-spec
  version: "1.0"
  generatedBy: "issue-spec"
---

# GitHub CLI

Use the gh CLI only for GitHub operations outside issue-spec's workflow and discussion surfaces.

## Use

- Inspect PR status, reviews, mergeability, CI, workflow runs, releases, labels, and repository metadata.
- Use structured --json/--jq output. Use git directly for local repository operations.
- Ordinary issue discussion writes: write a body file and run issue-spec comment create --repo owner/repo --issue 42 --body-file reply.md --json. The selected issue backend owns the write. Never use GitHub CLI or a raw issue-comment API write.
- issue-spec owns the proposal, design, implement, typed comments, review, verify, durable projection, and closure workflow. Do not use GitHub endpoints for non-GitHub providers.

## Setup and examples

    gh auth login
    gh auth status
    gh pr view 17 --repo owner/repo --json number,title,state,url
    gh pr checks 17 --repo owner/repo
    gh run view <run-id> --repo owner/repo --log-failed
    gh api repos/owner/repo/labels --jq '.[].name'
