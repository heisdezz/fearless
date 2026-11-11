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

## D-009 — WSSE interpolates before hashing (divergence from Bruno)

- **Date:** 2026-11-11T01:35Z
- **Raised by:** zed-agent-1 (implementing `AUTH-CORE`)
- **Affects:** `AUTH-CORE`
- **Decision:** The WSSE digest is computed from the **interpolated** password. Bruno computes it before interpolation, so a password written as `{{var}}` is hashed literally and the request fails.
- **Rationale:** Bruno's behaviour is a defect, not a feature: no collection can depend on "the digest is wrong", and interpolating first is what the user plainly intends. `docs/logic/authentication.md` §5 already flags this and proposes the same fix.
- **Action required:** None. `TestWSSEInterpolatesBeforeHashing` pins the new behaviour so it cannot regress silently.
- **Supersedes:** —

## D-010 — `model.RequestDefaults` needs the cookie toggles

- **Date:** 2026-11-10T23:00Z
- **Raised by:** `COOKIES` / mimo-flash
- **Affects:** `MODEL`, `COOKIES`, `APP` (wiring), `UI-SETTINGS`
- **Decision:** `model.RequestDefaults` should gain two fields: `SendCookies bool` (`json:"sendCookies"`) and `StoreCookies bool` (`json:"storeCookies"`), both set to `true` in the default settings (Bruno's default).
- **Rationale:** `docs/ui/settings.md` lines 39–40 document the preferences `request.sendCookies` / `request.storeCookies` (Bruno `preferencesUtil.shouldSendCookies()` / `shouldStoreCookies()`), and `docs/logic/cookies.md` §7 requires both. `COOKIES` exposes `Jar.SetSendCookies` / `Jar.SetStoreCookies` (both default `true`) but has no settings fields to bind to; `settings.go` belongs to `MODEL`, which is `Done`.
- **Action required:** The `MODEL` owner (or whoever reopens it) adds the two fields with `true` defaults; `APP` / `UI-SETTINGS` wires them to the jar setters at startup and on change. Until then the jar behaves as Bruno does by default (both enabled), so nothing is blocked.
- **Supersedes:** —

## D-010 — `vars:post-response` is evaluated natively for its declarative subset

- **Date:** 2026-11-11T02:45Z
- **Raised by:** zed-agent-1 (implementing `TPL`)
- **Affects:** `TPL`, `SCRIPT-POST`, `ENGINE`
- **Decision:** `internal/core/template` evaluates `vars:pre-request` and `vars:post-response` without goja, supporting the three forms found in Bruno's own collections: `$res.<path>`, `{{res.<path>}}` / `{{$dynamic}}`, and plain literals. Anything else is reported as unresolved.
- **Rationale:** `packages.md` describes `TPL` as keeping these blocks "separate from JS scripting so `VARS` need not depend on the script engine". Inspecting the fixtures shows both forms in real use: `$res.body.transactionId` (an expression) and `{{res.id}}` / `{{$timestamp}}` (templates). The templates need no engine at all, and `$res.body.<path>` is a dotted path that `vars.Lookup` already resolves — so the common cases are handled natively.
- **Consequence:** `docs/flows/scripting-flow.md` §1 describes both blocks as running through JS runtimes (`VarsRuntime`). That remains true for genuine expressions. `TPL` therefore supersedes nothing but covers the declarative subset, and `SCRIPT-POST` keeps responsibility for arbitrary JavaScript in these blocks.
- **Action required:** `SCRIPT-POST` should call `template.PostResponse` first and fall back to the engine only for values reported unresolved, so variable blocks stay cheap in the common case.
- **Supersedes:** —

## D-011 — Rulings on the open items (D-006, UQ-D3, UQ-D4, UQ-D6)

- **Date:** 2026-11-11T03:20Z
- **Raised by:** zed-agent-1, on instruction to decide rather than defer
- **Affects:** `INFRA`, `FORMAT-BRU`, `ADV-HISTORY`, all packages
- **Supersedes:** D-006 (confirmed), UQ-D3 / UQ-D4 / UQ-D6 from `research/unresolved-questions.md`

### D-006 — CONFIRMED: the layout is `internal/`

The code lives at `internal/core`, `internal/app` and `internal/ui`, and the CI guard checks those paths. The `./core/... ./app/...` form in `go-architecture.md` §1 is shorthand that was never a real path — and reading it literally made the guard pass vacuously (B-002).

Nothing at the repository root conflicts: `internal/app` is the application layer and the root `app/` directory is empty and untracked.

**No further action.** Treat `./internal/...` as the only correct form in scripts.

### UQ-D4 — DECIDED: semantics-preserving writer, not byte-exact

Measured: **5 of 335** upstream requests (1 %) reproduce byte-for-byte through parse → write. Byte-exact fidelity would mean replicating Bruno's writer quirks — key ordering anomalies, inconsistent blank lines, ad-hoc `'''` handling — and it would break on the first formatting fix, because the "correct" output would then differ from the fixture.

The property that matters is that no information is lost, and that guarantee holds at **335/335**. Saving a file may reformat it; it will not lose a field.

**Consequence:** users who keep a Bruno collection in git will see a formatting diff on first save in this application. That is expected and acceptable. Never claim byte-compatibility.

### UQ-D3 — DECIDED: metadata always, bodies on failure or explicit save

- Every execution keeps a metadata row: timestamp, collection/item UIDs, environment, method, URL, status, duration, size, error.
- **Bodies are retained** when the request failed, or when the user explicitly saves the result as an example. Successful bodies are referenced by `BodyRef` and are subject to the normal spool GC.
- **Retention:** prune by age _and_ total size — rows older than 30 days, then oldest-first until the store is under 200 MB. Both are settings (`PrivacySettings.HistoryRetainDays`, plus a size cap), so a user who wants everything can disable pruning.
- **Rationale:** the audit value of history is in _what was called and what came back when it broke_. Keeping every successful body doubles disk usage for data nobody re-reads, and the spool already deduplicates by content hash.

### UQ-D6 — DECIDED: v1 scope is HTTP + collections + scripting

**In v1:** HTTP requests (all body modes), collections in both dialects, environments and variables, auth through OAuth2, scripting/tests/assertions, the collection runner, and import from Postman and curl.

**Deferred, explicitly:** gRPC, WebSocket, mock server, AI assistant, integrated terminal, OpenAPI sync, WSDL import, and the docs/tiptap editor.

**Rationale:** every deferred item either needs a subsystem with its own risk profile (gRPC/WS transports, mock server) or was excluded by the Go-only constraint (terminal, AI, Chromium-rendered previews). Deferring them keeps the compatibility work — both codecs, scripting, auth — on the critical path rather than diluted.

- **Action required:** `board.md` Phase 7 items stay `Ready` but must not block any Phase 1–6 milestone; the roadmap's acceptance gates are the definition of v1.

### D-012 — OAuth1 `oauth_body_hash` is appended after signing (RFC-literal)

Bruno pushes `oauth_body_hash` into its parameter set _before_ calling `authorize`, so the hash appears in the signature base string. RFC draft-eaton-oauth-bodyhash-00 §3.2 defines the parameter but leaves its participation in the base string ambiguous.

**Decision:** `oauth1Sign` appends the hash _after_ the signature is computed, so it is sent but not signed.

**Rationale:** the opt-in `includeBodyHash` path is rare, and the RFC-literal reading is the defensible one. Signing a value derived from a body we have finalised adds nothing an attacker could not recompute.

**Reversibility:** the change is local to `oauth1Provider.Prepare`. If replay compatibility with Bruno-signed body-hash requests ever matters, pass the hash into `oauth1Input.BodyParams` instead. No other package is affected.

### D-013 — `auth.Prepared` may carry a replacement body

OAuth1's `body` placement rewrites the payload (the signed parameters _become_ the form body). `Prepared` gained a `Body []byte` field where `nil` means "leave the request body alone".

**Rationale:** the alternative — a second `TransportProvider`-style hook — would put payload mutation in the transport layer, after signing, which is the wrong order. Keeping it in `Prepared` means every provider composes the request in one place.

**Constraint for future providers:** "no body" must be an empty non-nil slice, never `nil`, or the override will be ignored.

### D-014 — `expect` boolean assertions are methods, not properties

chai's `expect(x).to.be.true` is a getter. goja's `DefineAccessorProperty` has no error return, so host code cannot raise a **catchable** exception from one — only a Go `panic`, which escapes the runtime and aborts the process.

**Decision:** the boolean assertions (`ok`, `true`, `false`, `null`, `undefined`, `exist`, `empty`) are methods: `expect(x).to.be.true()`. Everything else (`equal`, `eql`, `above`, `include`, `match`, `a`, …) needs no change.

**Consequence:** Bruno scripts using the property form must gain `()`. This is a visible source-level incompatibility, chosen over the alternative (interrupting the runtime from a getter), which would make even a passing chain abort.

**Revisit if:** goja gains an error-returning accessor, or the fixture corpus shows the property form is common enough to justify a preprocessor.

### D-015 — assertion failures are `GoError`, not `AssertionError`

Bruno's `test()` branches on `error instanceof chai.AssertionError` to report actual/expected. goja's `*Exception` value field is unexported, so a host function cannot construct an `AssertionError` instance.

**Decision:** assertion failures are raised as Go errors (goja converts them to `GoError`) whose message names both operands. `test()` reports them as failures with that message.

**Consequence:** a script that catches assertions and inspects `error.name` will see `GoError`. The user-facing pass/fail result is unaffected.

### D-016 — `main.go` links UI-LAYOUT's demo until UI-APP lands

`main.go` appears in no package's ownership row; `packages.md` puts its final shape under `UI-APP` ("`mygo.App.Run()` called from `main`"). Until someone claims `UI-APP`, its `Content:` is wired to `ui.View(demo.View)` from `internal/ui/layout/demo` (UI-LAYOUT's row), so the MyGo + Lobe stack is runnable end to end from `go run .`.

**Consequence:** whoever claims `UI-APP` replaces that call with the real root view; after that, the `demo` package may be deleted by UI-LAYOUT's owner. No other package imports `demo`.

**Raised by:** mimo-flash-2 (UI-LAYOUT), 2026-11-11T01:04Z.

### D-017 — the app adopts the Graphite Console visual world (replaces Lobe's defaults)

Lobe MyGo's default tokens are an unopinionated neutral palette. Rendered, the app reads as a generic admin panel, and dark mode is not usable: in the assembled frame (`internal/ui/app`) input surfaces paint light while the ground paints dark.

**Decision:** the runtime token source becomes our own **Graphite Console** world, compiled in `internal/ui/theme` and applied through the existing `theme.Adapter`:

- graphite ground, two panel steps, seam-only elevation (never shadow);
- one accent, earned by the primary action and the selected state;
- small-caps panel headers, monospaced figures for data;
- Bruno's per-method palette stays a data table (unchanged from §7.2.2 rule 4);
- Bruno theme JSON stays an import format, and an import overrides the mode it declares.

**Constraint:** no screen restyles itself. Everything paints from `theme.TokensOf` / `theme.MethodColorFor`. A screen that hard-codes a colour is a defect.

**Consequence:** `DESIGN.md` is written from the built world at the end of the redesign; `docs/ui/design-system.md` §7.2.2 remains the record of the Lobe layer and gains a pointer to `DESIGN.md`.

**Raised by:** zed-design (UI-THEME), 2026-11-11T04:24Z.

### D-018 — dark is the shipped appearance; light is a finished companion, not a peer

Graphite Console is a dark-first world. The product owner chose: dark is what the app ships, light is good but secondary. Existing behaviour stays (the theme follows the system appearance; an appearance switch may still be offered), but:

- dark is the appearance every golden, screenshot and review pass is **judged** on;
- light must stay legible and complete — a light frame with an illegible control is a defect — and gets token assertions, not pixel review;
- a component whose dark and light treatments disagree in meaning (not merely in tone) is a defect in whichever one the system does not ship.

**Raised by:** zed-design (UI-THEME), 2026-11-11T04:24Z.

### D-019 — controls are shimmed over Lobe's field/action on an app-owned surface

D-017 recorded the white-input defect but not its cause. It is not a token bug: Lobe's `field.Input`, `field.Select` and `action.Button` apply their theme background to a container node that they do not return, so the element that actually paints is left with MyGo's default near-white surface. It reproduces with Lobe's own `theme.Apply` and no Graphite code at all, and it is independent of which tokens are installed.

A second, real token-plumbing bug sat on top of it: `theme.Adapter.Apply` built an ad-hoc `ui.Theme` that Lobe's `TokensOf` could not resolve back to tokens, so any widget that asked for tokens got Lobe's light defaults. `Apply` now installs the theme from a `theme.NewConfig` pair, which `TokensOf` resolves by pointer.

**Decision:** the app never builds a raw `field.*` or `action.Button` control on screen. `internal/ui/widgets/controls.go` provides `FieldInput`, `FieldSelect` and `Button`, each painted on an app-owned element from world tokens and building the Lobe control borderless inside it. Every screen routes through these.

**Consequence:** this is a workaround for an upstream bug. When Lobe/MyGo fixes the detached-surface paint, the wrappers stay (they are the app's control surface) but their internals can be reconsidered. A screen that calls `field.Input`/`field.Select`/`action.Button` directly is a defect.

**Raised by:** zed-design (UI-THEME), 2026-11-11T05:40Z.

### D-020 — `ui-map/` is the Bruno UI reconstruction spec

The Go port was being built against prose (`docs/ui/*.md`) that predates a rebuild pass and against screenshots from an older Bruno build. Reconstructing closely enough to look like Bruno needs exact values and cited evidence.

**Decision:** `ui-map/` holds a framework-agnostic UI specification generated from the pinned Bruno source (`bruno/` at `60d7517`): `overview.md` (package structure + a checklist indexing all 104 `src/components` folders), `design-system.md` (light/dark tokens resolved to hex by `ui-map/_tools/dump-theme.mjs`), `screens/*`, `components/*`, `widgets.md` and `go-ui-notes.md`. Every claim carries a `file:line` citation; ambiguities are collected in `go-ui-notes.md` §5 rather than guessed.

**Consequence:** where `refs/` screenshots disagree with source, source wins (the captures are from a different build); this is recorded in `go-ui-notes.md`. Token values are machine-verifiable — re-run `dump-theme.mjs` and diff `tokens.json`.

**Raised by:** zed-design, 2026-11-11T16:30Z.

### D-021 — `ui-map/` is a first-class work package, and ownership is enforced by script

`ui-map/` (the Bruno reconstruction spec) and `refs/` (its reference captures)
were created outside every work package, so no row owned them: two agents could
edit the same spec file and `ownership.md` would not notice. The contract also
had no checker — it relied on each agent self-policing §7 by eye.

**Decision:**
- New package `UI-MAP` owns `ui-map/**` and `refs/**`. It is the only writer of
  the map, including `design-system.md` and the shared `overview.md`.
- `scripts/check-ownership.sh` enforces the map against a change set:
  `--pkg <ID>` fails on any file outside that package's row; with no argument
  it fails on any file no package owns; `--range A..B` checks commits. The §6
  shared files (`board.md`, own `log/`, `decisions.md`, `blockers.md`) are
  always allowed.
- The protocol (§9 of `docs/progress/README.md`) requires running it before
  reporting completion, alongside `ui-map/_tools/verify.py` for the map itself.

**Consequence:** `ownership.md` remains read-only for agents; this entry is the
agreement that changed it. Adding a new top-level documentation or asset tree
from now on must include a `UI-MAP`-style row and pass the checker.

**Raised by:** zed-design (UI-MAP), 2026-11-11T19:05Z.
