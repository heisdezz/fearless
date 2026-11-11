#!/usr/bin/env bash
set -euo pipefail

# INFRA: does HEAD build from its tracked files alone?
#
# Every agent (and every CI job that is not this one) builds the *working tree*,
# where an untracked directory is present and compiles. That is how
# internal/core/cookies came to be imported by three committed packages while
# being absent from git (B-021): `go build ./...` was green for everyone, and a
# fresh clone did not compile.
#
# `git archive HEAD` exports only tracked files, so building it asks the
# question that matters: can someone who clones this repository build it?
#
# Run locally before merging a branch, and in CI as `tracked-tree-build`.

if ! command -v go >/dev/null 2>&1; then
  echo "go not found" >&2
  exit 1
fi
if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "not a git repository" >&2
  exit 1
fi

# Work from the repository root whatever directory this was invoked from.
cd "$(git rev-parse --show-toplevel)"

dest=$(mktemp -d)
trap 'rm -rf "$dest"' EXIT

git archive HEAD | tar -x -C "$dest"

if ! ( cd "$dest" && go build ./... ); then
  echo "::error::HEAD does not build from tracked files alone." >&2
  echo "" >&2
  echo "A package is imported by committed code but was never committed." >&2
  echo "Find it with:" >&2
  echo "  git status --short | grep '^??'" >&2
  echo "then commit the directory (see B-021 for the original case)." >&2
  exit 1
fi

echo "tracked-tree: ok (HEAD builds from its tracked files alone)"
