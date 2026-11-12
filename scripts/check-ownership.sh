#!/usr/bin/env bash
set -euo pipefail

# Enforce docs/progress/ownership.md against a change set.
#
#   scripts/check-ownership.sh                 # working tree + staged vs HEAD
#   scripts/check-ownership.sh --range A..B    # commits between A and B
#   scripts/check-ownership.sh --pkg UI-THEME  # only allow one package's paths
#
# An agent names its package; a file outside that package's owned paths in
# ownership.md is a contract violation and the command exits non-zero.
#
# Shared, always-writable files (board row, own log, append-only decisions and
# blockers) are allowed for everyone by design — that is what keeps concurrent
# work conflict-free.

if ! command -v git >/dev/null 2>&1; then
  echo "git not found" >&2
  exit 1
fi

MAP="docs/progress/ownership.md"
range=""
pkg=""

while [ $# -gt 0 ]; do
  case "$1" in
    --range) range="$2"; shift 2 ;;
    --pkg)   pkg="$2"; shift 2 ;;
    -h|--help)
      sed -n '3,16p' "$0" | sed 's/^# \{0,1\}//'
      exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

if [ ! -f "$MAP" ]; then
  echo "ownership: $MAP not found; run from the repo root" >&2
  exit 1
fi

if [ -n "$range" ]; then
  changed="$(git diff --name-only "$range")"
else
  # Staged and unstaged, plus untracked, relative to HEAD.
  changed="$(git --no-optional-locks status --porcelain=v1 | sed -E 's/^.. //; s/^.* -> //')"
fi

if [ -z "$changed" ]; then
  echo "ownership: no changed files"
  exit 0
fi

# Files any agent may write while working (see README §6).
is_shared() {
  case "$1" in
    docs/progress/board.md) return 0 ;;
    docs/progress/decisions.md) return 0 ;;
    docs/progress/blockers.md) return 0 ;;
    docs/progress/log/*.md) return 0 ;;
    "") return 0 ;;
    *) return 1 ;;
  esac
}

# Read-only contract files (ownership.md §6). Edits are not an agent's normal
# right; they are reported separately so a deliberate, decision-backed change
# (e.g. D-021) can be told apart from an accidental one.
is_contract() {
  case "$1" in
    docs/progress/ownership.md|docs/progress/README.md|docs/progress/packages.md|docs/*.md|docs/ui/*.md) return 0 ;;
    *) return 1 ;;
  esac
}

# Turn a path glob from an ownership row into a gitignore-style regex.
glob_to_re() {
  local g="$1"
  # Escape regex metacharacters; \\& inserts a single backslash before the char.
  g="$(printf '%s' "$g" | sed -E 's/[][(){}.+?^$|]/\\&/g')"
  g="${g//\*\*/__DS__}"
  g="${g//\*/[^/]*}"
  g="${g//__DS__/.*}"
  printf '^%s$' "$g"
}

# Collect every owned path from the table rows, remembering the package.
owned_for_pkg() {
  local want="$1"
  awk -v want="$want" '
    /^\| `[A-Z0-9-]+`/ {
      pkg = $0
      sub(/^\| `/, "", pkg)
      sub(/`.*/, "", pkg)
      line = $0
      sub(/^\| `[A-Z0-9-]+`[^|]*\| */, "", line)
      sub(/ *\|.*$/, "", line)
      if (pkg == want) print line
    }
  ' "$MAP"
}

all_pkgs() {
  # A path is owned if ANY package declares it; this mode answers "is this file
  # anyone's?" rather than "is it my package's?".
  awk '
    /^\| `[A-Z0-9-]+`/ {
      line = $0
      sub(/^\| `[A-Z0-9-]+`[^|]*\| */, "", line)
      sub(/ *\|.*$/, "", line)
      print line
    }
  ' "$MAP"
}

# Split a cell like "`a/**`, `b*.go` (`note`)" into bare globs.
split_globs() {
  # Drop parenthetical notes first: they contain commas of their own.
  sed -E 's/\([^)]*\)//g' | tr ',' '\n' | sed -E 's/`//g; s/^ +//; s/ +$//' | grep -v '^$'
}

matches_any() {
  local file="$1"
  local globs="$2"
  local g
  while IFS= read -r g; do
    [ -z "$g" ] && continue
    if [[ "$file" =~ $(glob_to_re "$g") ]]; then return 0; fi
  done <<< "$globs"
  return 1
}

violations=0
contract_edits=0
checked=0

if [ -n "$pkg" ]; then
  globs="$(owned_for_pkg "$pkg" | split_globs)"
  if [ -z "$globs" ]; then
    echo "ownership: package '$pkg' has no row in $MAP" >&2
    exit 2
  fi
  while IFS= read -r f; do
    [ -z "$f" ] && continue
    is_shared "$f" && continue
    checked=$((checked + 1))
    if is_contract "$f"; then
      echo "CONTRACT: $f is a read-only contract file (§6); needs a decision entry"
      contract_edits=$((contract_edits + 1))
      continue
    fi
    if ! matches_any "$f" "$globs"; then
      echo "VIOLATION: $f is not owned by $pkg"
      violations=$((violations + 1))
    fi
  done <<< "$changed"
else
  globs="$(all_pkgs | split_globs)"
  while IFS= read -r f; do
    [ -z "$f" ] && continue
    is_shared "$f" && continue
    checked=$((checked + 1))
    if is_contract "$f"; then
      echo "CONTRACT: $f is a read-only contract file (§6); needs a decision entry"
      contract_edits=$((contract_edits + 1))
      continue
    fi
    if ! matches_any "$f" "$globs"; then
      echo "UNOWNED: $f is not in any package's row"
      violations=$((violations + 1))
    fi
  done <<< "$changed"
fi

if [ "$violations" -gt 0 ]; then
  echo
  echo "ownership: $violations violation(s) across $checked file(s)."
  echo "Fix the file set, or raise the split in docs/progress/blockers.md."
  exit 1
fi

echo "ownership: OK ($checked file(s)${pkg:+, package $pkg}${contract_edits:+; $contract_edits contract file(s) edited — cite the decision})"
exit 0
