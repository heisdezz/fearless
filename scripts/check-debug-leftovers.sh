#!/usr/bin/env bash
set -euo pipefail

# INFRA: reject throwaway debug files that would otherwise be committed.
#
# The convention is that a file named zz_* is a scratch probe: a dump, a capture,
# a timing harness. It is meant to be deleted before the work is committed. Three
# of them reached the tree during development and each cost someone time:
#
#   internal/core/engine/transport/zz_probe_test.go   (B-012 investigation)
#   internal/core/format/bru/zz_dump_test.go          (B-019 — wrong package,
#                                                      no assertions, would have
#                                                      been swept up by git add -A)
#   internal/ui/app/zz_capture_test.go                (UI-APP — broke `go vet`
#                                                      tree-wide while present)
#
# This checks the index, not the working tree: an untracked scratch file is
# expected and fine, it just must not be staged. A *tracked* zz_* file is
# unambiguous evidence it was committed by accident.
#
# Legitimate long-lived test fixtures should be named for what they test, not
# zz_-prefixed. If a file genuinely belongs in the repo, rename it.

if ! command -v git >/dev/null 2>&1; then
  echo "git not found" >&2
  exit 1
fi

tracked=$(git ls-files -- 'zz_*' '*/zz_*')

if [ -n "$tracked" ]; then
  echo "::error::Tracked debug leftovers. These zz_* files were committed:" >&2
  echo "$tracked" >&2
  echo "" >&2
  echo "Delete them, or rename them for the behaviour they test." >&2
  echo "See scripts/check-debug-leftovers.sh." >&2
  exit 1
fi

echo "debug-leftovers: ok"
