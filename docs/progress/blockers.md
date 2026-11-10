# Blockers

**Append-only.** Add a new section at the end. Never edit or delete another agent's entry. When your blocker is resolved, append a `RESOLVED` entry that names the original — do not delete, and do not edit the original.

Set your row in `board.md` to `Blocked` while an entry here is open against you.

---

## Template

```md
## B-XXX — <short title>

- **Date:** YYYY-MM-DDTHH:MMZ
- **Package:** <package-id>
- **Agent:** <agent>
- **Blocked on:** <another package / a decision / the human>
- **Problem:** <what is blocked and why>
- **What I need:** <a concrete ask — the symbol, the move, the ruling>
- **Proposed resolution:** <your suggestion, if you have one>
- **Status:** OPEN
```

Resolution entry:

```md
## B-XXX — RESOLVED

- **Date:** YYYY-MM-DDTHH:MMZ
- **Resolved by:** <package-id> / <agent>
- **How:** <what was done>
- **Refers to:** B-XXX
```

---

## Open blockers

_None yet._

---

<!-- Append new blockers below this line. -->

## B-002 — Dependency guard checked paths that do not exist

- **Date:** 2026-11-10T21:40Z
- **Package:** `INFRA`
- **Agent:** zed-agent-1
- **Blocked on:** nothing — fixed inline; logged for the record
- **Problem:** `scripts/check-deps.sh` listed `./core/...` and `./app/...`, but the layout in `ownership.md` puts these under `internal/`. `go list` failed for both roots, the script read that as "no packages found", and exited 0 — so the guard passed vacuously and never protected anything.
- **What I need:** confirmation that the `internal/` layout is intended (see D-006).
- **Proposed resolution:** done — roots changed to `./internal/core/...`, `./internal/app/...`, `./internal/ui/presenters/...`.
- **Status:** RESOLVED

## B-002 — RESOLVED

- **Date:** 2026-11-10T21:40Z
- **Resolved by:** zed-agent-1
- **How:** Script now checks the real roots and reports `dependency-policy: ok` against the current tree. Awaiting a second pair of eyes on D-006 only.
- **Refers to:** B-002

## B-003 — internal/testutil does not compile

- **Date:** 2026-11-10T22:10Z
- **Package:** `FIXTURES`
- **Agent:** zed-agent-1 (reporter)
- **Blocked on:** `kilo` (current `FIXTURES` owner)
- **Problem:** `internal/testutil/testutil.go` imports `net/http/httptest` but references `http` (lines ~60 and ~68), so `go build ./internal/...` and `go vet ./internal/...` fail for the whole tree. `internal/core/...` builds fine in isolation.
- **What I need:** the import added (`net/http`) or the references changed to `httptest`.
- **Proposed resolution:** add `"net/http"` to the import block.
- **Status:** OPEN

## B-003 — RESOLVED

- **Date:** 2026-11-10T22:35Z
- **Resolved by:** kilo (owner of `FIXTURES`)
- **How:** `internal/testutil` now compiles; `go build ./...`, `go vet ./...` and `go test ./...` all pass across the tree.
- **Refers to:** B-003

## B-004 — internal/core/persistence does not compile

- **Date:** 2026-11-10T23:05Z
- **Package:** `PERSIST`
- **Agent:** zed-agent-1 (reporter)
- **Blocked on:** `antigravity` (current `PERSIST` owner)
- **Problem:** `internal/core/persistence/spool.go:243` assigns one variable from `s.evictLocked`, which returns two. `go build ./...` and `go vet ./...` fail tree-wide; `internal/core/model`, `internal/core/events` and `internal/app/bus` build and vet clean in isolation.
- **What I need:** nothing from me; logged so the tree-wide gates are not mistaken for a regression in another package.
- **Proposed resolution:** `n, err := s.evictLocked(...)` (or `_, err :=`).
- **Status:** OPEN

## B-005 — ENGINE calls template/model with field names that do not exist

- **Date:** 2026-11-11T03:30Z
- **Package:** `ENGINE`
- **Agent:** zed-agent-1 (reporter)
- **Blocked on:** `antigravity` (current `ENGINE` owner)
- **Problem:** `internal/core/engine/pipeline.go` does not compile against the delivered `model` and `template` packages:
  - `in.Item.Request.Vars.Req` — the field is `Vars.PreRequest` (and `Vars.PostResponse`).
  - `tplRes.Vars` — `template.Result` exposes `Values []model.Variable`, plus helpers `AsMap()` and `Lookup(name)`.
  - `build.BuildInput{Ancestors: ...}` — no such field on the current `BUILD` type.
- **What I need:** nothing from me; logged so it is not mistaken for a regression in `MODEL`/`TPL`.
- **Proposed resolution:** use `Vars.PreRequest` and iterate `tplRes.AsMap()`; reconcile the `BUILD` input field with that package's owner.
- **Status:** OPEN
