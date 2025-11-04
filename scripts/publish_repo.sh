#!/usr/bin/env bash
set -euo pipefail

# One-click publish this workspace to GitHub.
# Default remote: ink-hz/higress-ai-capability-auth

REMOTE_URL="git@github.com:ink-hz/higress-ai-capability-auth.git"

echo "==> Checking git identity"
NAME=$(git config --get user.name || true)
EMAIL=$(git config --get user.email || true)
if [[ -z "$NAME" || -z "$EMAIL" ]]; then
  echo "WARN: git user.name or user.email is not set."
  echo "      Configure with: git config --global user.name \"Your Name\"; git config --global user.email \"you@example.com\""
fi

echo "==> Initializing git repo (if needed)"
if [[ ! -d .git ]]; then
  git init
fi

echo "==> Staging all files"
git add -A

echo "==> Committing (if there are changes)"
if git diff --cached --quiet; then
  echo "No staged changes to commit."
else
  git commit -m "feat: AI capability auth demo (mcp-guard + ai-proxy, deepseek/openai)"
fi

echo "==> Ensuring default branch: master"
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
if [[ "$CURRENT_BRANCH" != "master" ]]; then
  git branch -M master
fi

echo "==> Configuring remote origin: $REMOTE_URL"
if git remote | grep -q '^origin$'; then
  git remote set-url origin "$REMOTE_URL"
else
  git remote add origin "$REMOTE_URL"
fi

echo "==> Pushing to origin master"
git push -u origin master

echo "Done. Repo published at: $REMOTE_URL"

