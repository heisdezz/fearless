#!/usr/bin/env bash
set -euo pipefail

# INFRA: reject GUI modules in core/ and app/ dependency graphs.
# core/ and app/ must not depend on MyGo, Lobe MyGo, Gio, or Fyne.
# Per go-architecture.md §1 (+ lobe-mygo, D-002).

if ! command -v go >/dev/null 2>&1; then
  echo "go not found" >&2
  exit 1
fi

patterns='github.com/egoist/mygo|github.com/lobehub/lobe-mygo|gioui.org|fyne.io'
# The layering roots are under internal/ (see docs/progress/ownership.md).
# internal/history and internal/platform are guarded too: they are headless, so a
# GUI import there is a layering violation even though no presenter consumes them
# yet (B-009, B-010).
roots=(
  ./internal/core/...
  ./internal/app/...
  ./internal/ui/presenters/...
  ./internal/history/...
  ./internal/platform/...
)
existing=()
for r in "${roots[@]}"; do
  # Only check roots that exist as packages in this module.
  if go list "$r" >/dev/null 2>&1; then
    existing+=("$r")
  fi
done

if [ ${#existing[@]} -eq 0 ]; then
  echo "dependency-policy: no guarded packages found; skipping"
  exit 0
fi

# shellcheck disable=SC2046
bad="$(go list -deps "${existing[@]}" 2>/dev/null | grep -E "$patterns" || true)"
if [ -n "$bad" ]; then
  echo "dependency-policy: forbidden GUI dependency found in a guarded graph:" >&2
  echo "$bad" >&2
  exit 1
fi

echo "dependency-policy: ok"
