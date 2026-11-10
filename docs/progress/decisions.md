# Decisions Log

**Append-only.** Add a new numbered section at the end of the file. Never edit or delete an existing entry. If a decision changes, append a new entry that says which one it supersedes.

Use this for anything that crosses a package boundary: new exported symbols, interface changes, shared type changes, agreed behaviour where the docs were ambiguous.

---

## Template

```md
## D-XXX — <short title>

- **Date:** YYYY-MM-DDTHH:MMZ
- **Raised by:** <package-id> / <agent>
- **Affects:** <package-id>, <package-id>
- **Decision:** <what was decided, in one or two sentences>
- **Rationale:** <why>
- **Action required:** <who does what, and by when if relevant>
- **Supersedes:** <D-YYY or —>
```

---

## D-001 — Progress tracking protocol adopted

- **Date:** 2026-11-10T20:30Z
- **Raised by:** `INFRA` / project setup
- **Affects:** all packages
- **Decision:** Work is split into the packages in `packages.md`, each owning a disjoint file set in `ownership.md`. Packages are claimed by editing a single row in `board.md`.
- **Rationale:** Enables several agents to work simultaneously without edit collisions.
- **Action required:** Read `README.md` before starting any work.
- **Supersedes:** —

## D-002 — `lobe-mygo` added to the GUI dependency guard

- **Date:** 2026-11-10T20:30Z
- **Raised by:** `INFRA` / design review
- **Affects:** `INFRA`, all `UI-*` packages
- **Decision:** The CI check that forbids GUI modules in `core/` and `app/` must reject `github.com/lobehub/lobe-mygo` as well as `github.com/egoist/mygo`, `gioui.org` and `fyne.io`.
- **Rationale:** Lobe MyGo is now the component layer; a stray import from a presenter would silently couple the core to the GUI.
- **Action required:** `INFRA` implements the check in the CI workflow.
- **Supersedes:** —

## D-003 — Presenters may not import Lobe or MyGo

- **Date:** 2026-11-10T20:30Z
- **Raised by:** `UI-PRESENT-REQ` / design review
- **Affects:** `UI-PRESENT-REQ`, `UI-PRESENT-NAV`, `UI-PRESENT-SET`
- **Decision:** Presenter packages import only plain Go types from `core/model` and `core/events`. Widgets and screens import the GUI libraries.
- **Rationale:** Keeps every presenter unit-testable headlessly and keeps the toolkit swappable.
- **Action required:** `INFRA` extends the guard to `./internal/ui/presenters/...`.
- **Supersedes:** —

---

<!-- Append new decisions below this line. -->

## D-004 — MyGo must be the pseudo-version Lobe requires

- **Date:** 2026-11-10T21:20Z
- **Raised by:** `INFRA`
- **Affects:** `INFRA`, all `UI-*` packages
- **Decision:** Pin `github.com/egoist/mygo` to `v0.8.4-0.20261108082913-9233663736ca`, not the tagged `v0.8.3`.
- **Rationale:** `lobe-mygo` requires that exact, later version; `v0.8.3` fails with a minimum-version conflict. This confirms the docs' warning that MyGo and Lobe must be bumped together.
- **Action required:** Any future MyGo bump must be validated against Lobe's `go.mod` requirement. Bump both in one change.
- **Supersedes:** —

## D-005 — Verified pinned versions

- **Date:** 2026-11-10T21:20Z
- **Raised by:** `INFRA`
- **Affects:** `INFRA`, `UI-*`
- **Decision:** `github.com/egoist/mygo v0.8.4-0.20261108082913-9233663736ca`; `github.com/lobehub/lobe-mygo v0.0.0-20261110162706-5e5fea5ef9e2` (no tags exist).
- **Rationale:** Resolved and compiled successfully; `theme.Apply`, `theme.TokensOf`, `action.*`, `field.*`, `display.*`, `feedback.*`, `overlay.*`, `nav.*` and `code.Highlighter` all type-check exactly as documented.
- **Action required:** —
- **Supersedes:** —

## D-006 — Layering roots live under internal/

- **Date:** 2026-11-10T21:40Z
- **Raised by:** zed-agent-1
- **Affects:** `INFRA`, all packages
- **Decision:** The GUI-free layering roots are `./internal/core`, `./internal/app` and `./internal/ui/presenters`. The `./core/...` / `./app/...` form quoted in `go-architecture.md` §1 and `interface-design.md` §9 is shorthand, not a path.
- **Rationale:** `package-structure.md` and `ownership.md` both place these under `internal/`. Using the shorthand in CI made the guard vacuous (see B-002).
- **Action required:** Anyone writing a `go list` check must use the `internal/` paths.
- **Supersedes:** —

## D-007 — `model.Draft` does not exist; `StateActor.Apply` needs a real signature

- **Date:** 2026-11-10T22:15Z
- **Raised by:** zed-agent-1 (found while implementing `EVENTS`)
- **Affects:** `BUS`, `UI-PRESENT-*`
- **Decision:** **Unresolved — needs a ruling before `BUS` starts.**
- **Rationale:** `interface-design.md` §6 declares `Apply(mutator func(*model.Draft))`, but there is no `model.Draft` type. `model.Item` already models the draft duality as `Saved *Request` / `Draft *Request` (`request.go`), so either the doc is stale or a new aggregate type is intended.
- **Options:**
  1. `Apply(func(*model.WorkspaceState))` — the actor owns the whole state; simplest, matches "single-owner actor" in `go-architecture.md` §3.
  2. `Apply(func(*model.Item))` — scoped to one item; safer, but cannot express tab or collection changes.
  3. Introduce `model.Draft` as an aggregate holding the item plus tab context — closest to the doc, but invents a type nothing else needs.
- **Recommendation:** option 1. It is what `go-architecture.md` describes and needs no new type.
- **Action required:** whoever takes `BUS` should confirm and implement; append a superseding decision.
- **Supersedes:** —

## D-008 — `ErrorOccurred` is a value type

- **Date:** 2026-11-10T22:15Z
- **Raised by:** zed-agent-1
- **Affects:** `EVENTS`, `BUS`, all `UI-PRESENT-*`, `CLI`
- **Decision:** `ErrorOccurred` satisfies `Event` as a value (`func (ErrorOccurred) Kind()`), matching every other event and the shapes in `go-architecture.md` §4.
- **Rationale:** a pointer-only event would force one branch of every presenter switch to differ, which is exactly the kind of inconsistency that causes nil-dereference bugs in event loops. `Err error` staying in the struct is what the architecture doc specifies.
- **Action required:** Because the receiver is a value, `errors.As` matches only a value target: declare `var target ErrorOccurred` and pass `&target`. A `*ErrorOccurred` target does **not** match (verified). Consumers that need the cause should prefer `errors.Is` on the wrapped error.
- **Supersedes:** —

## D-007 — RESOLVED: `StateActor.Apply` takes the whole workspace state

- **Date:** 2026-11-10T22:45Z
- **Raised by:** zed-agent-1 (found while implementing `EVENTS`)
- **Affects:** `BUS`, `UI-PRESENT-*`, `CMD-*`
- **Decision:** `Apply(mutator func(*model.WorkspaceState))`. Option 1 of the original entry. No `model.Draft` type is introduced.
- **Rationale:** `go-architecture.md` §3 already describes the actor as owning `WorkspaceState` and publishing immutable snapshots; a per-item mutator cannot express tab, collection or settings changes, and a new aggregate type would exist only to match a stale sentence in `interface-design.md`. `model.Item.Draft` remains the draft duality, unchanged.
- **Action required:** `interface-design.md` §6 still shows `func(*model.Draft)`; treat this decision as authoritative until the doc is amended.
- **Supersedes:** D-007 (original unresolved entry)
