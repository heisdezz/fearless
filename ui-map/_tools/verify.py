#!/usr/bin/env python3
"""Verify the ui-map's promises hold. Run from the repo root:

    python3 ui-map/_tools/verify.py

Checks:
  1. every `screens/*.md` / `components/*.md` target named in overview.md exists
  2. every internal doc cross-link resolves
  3. every citation `path:line` resolves to a file, and (when the path uniquely
     identifies a file) the line is within it

Exit code is non-zero if any check fails, so it can gate a change.
"""
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parents[2]
MAP = ROOT / "ui-map"
SRC = ROOT / "bruno/packages/bruno-app/src"
PKG = ROOT / "bruno/packages"

DOC_TARGET = re.compile(r"`((?:screens|components)/[a-z-]+\.md)`")
CITE = re.compile(r"([A-Za-z0-9_./-]+\.(?:jsx?|tsx?)):(\d+)")


def all_src_files():
    return [p for p in SRC.rglob("*") if p.is_file()]


def resolve_cite(path, files):
    """Return (file, ambiguous). Matches exact path, package-relative, else
    any file whose path ends with `path`."""
    for base in (SRC, ROOT / "bruno/packages/bruno-app", PKG):
        c = base / path
        if c.exists():
            return c, False
    hits = [p for p in files if str(p).endswith("/" + path.lstrip("./"))]
    if len(hits) == 1:
        return hits[0], False
    return None, True


def nlines(p):
    with p.open(errors="ignore") as f:
        return sum(1 for _ in f)


def main():
    fails = []
    files = all_src_files()
    mds = sorted(MAP.rglob("*.md"))

    # 1. checklist / doc targets
    for m in DOC_TARGET.finditer((MAP / "overview.md").read_text()):
        target = MAP / m.group(1)
        if not target.exists():
            fails.append(f"overview.md names missing {m.group(1)}")

    # 2. internal cross-links (whole map)
    for md in mds:
        for m in DOC_TARGET.finditer(md.read_text()):
            if not (MAP / m.group(1)).exists():
                fails.append(f"{md.name}: broken doc link {m.group(1)}")

    # 3. citations
    citations = 0
    unresolved = 0
    past_eof = 0
    for md in mds:
        for m in CITE.finditer(md.read_text()):
            citations += 1
            p, ambiguous = resolve_cite(m.group(1), files)
            if p is None:
                if not ambiguous:
                    unresolved += 1
                    fails.append(f"{md.name}: citation {m.group(1)} not found")
                continue
            line = int(m.group(2))
            if not ambiguous and line > nlines(p):
                past_eof += 1
                fails.append(
                    f"{md.name}: {m.group(1)}:{line} is past EOF ({nlines(p)} lines)"
                )

    print(f"doc targets checked; citations: {citations}")
    print(f"unresolved: {unresolved}  past-EOF: {past_eof}")
    if fails:
        print(f"\n{len(fails)} problem(s):")
        for f in fails[:40]:
            print("  ", f)
        return 1
    print("OK")
    return 0


if __name__ == "__main__":
    sys.exit(main())
