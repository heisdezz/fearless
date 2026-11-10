# Unresolved Questions

Status: all entries are **UNKNOWN** unless marked otherwise. Each entry states what is unknown, why it matters, how it was investigated, and what would resolve it. Nothing here is a guess presented as fact.

Two kinds of entry:

- **UQ-B**: a question about Bruno's behaviour that source inspection did not settle.
- **UQ-D**: a **decision** the Go port must make that cannot be derived from Bruno (needs the project owner's input).

---

## UQ-B: Bruno behaviour not fully established

### UQ-B1 — Header deduplication across the folder chain

**Question.** When a request defines header `X-Token` and its parent folder and collection also define `X-Token`, which wins, and is the merged result case-sensitive?

`utils/collection.js` merges headers from collection → folder → request, and child values appear to win. What is not established:

- whether comparison is case-insensitive (HTTP says it must be; a naive `===` would not be),
- whether duplicates are dropped or appended,
- whether a header explicitly set to empty in the child deletes the inherited one.

**Why it matters.** `logic/request-builder.md` merges headers the same way. Getting the rule wrong silently changes outgoing requests.

**Investigated.** Read `packages/bruno-electron/src/utils/collection.js` merge helpers and `prepare-request.js`. The merge is present; the case-handling rule was not conclusively read for every path.

**Would resolve.** A focused unit test in the Bruno repo, or inspection of `packages/bruno-electron/tests/utils` for a headers-merge spec, plus a live experiment.

**Go port interim decision (PROPOSED).** Merge case-insensitively, last-writer-wins from collection → folder → request, and treat an explicit empty value as a deletion. Document the choice in `logic/request-builder.md`.

### UQ-B2 — WPAD / PAC fallback order

**Question.** When system proxy settings say "auto-detect", what is the exact discovery order and timeout?

**Why it matters.** `logic/http-engine.md` §6 requires PAC/WPAD behaviour; `mygo/netproxy` provides an implementation whose fallback order may differ from Chromium's.

**Investigated.** Proxy logic lives partly in `bruno-requests/src/network/system-proxy/` and partly in Electron/Node internals, which are not in this repository.

**Would resolve.** Reading Electron's `session.resolveProxy` documentation, or empirical capture on Windows/macOS/Linux.

### UQ-B3 — Does `snapshotMiddleware` retain full bodies?

**Question.** Is the snapshot mechanism intended to keep full response bodies in state, or does it retain only metadata?

**Why it matters.** The Go port stores only `BodyRef`. If Bruno's snapshots intentionally keep bodies (e.g. for undo), that is a feature the port would lose.

**Investigated.** `providers/ReduxStore/index.js` registers `snapshotMiddleware`; the retention rule depends on the snapshot service (`services/snapshot`), which was not read line by line.

**Would resolve.** Read `packages/bruno-electron/src/services/snapshot`.

### UQ-B4 — Exact multipart part ordering and boundary generation

**Question.** Is part order the file's declaration order after interpolation, and is the boundary random per request?

**Why it matters.** Servers that hash the raw body (some signature-scheme APIs) are sensitive to both.

**Investigated.** `utils/form-data.js` builds the parts; the ordering was consistent with declaration order in every fixture read, but no explicit sort step was confirmed.

**Go port interim decision (PROPOSED).** Preserve declaration order and use `multipart.Writer`'s random boundary.

### UQ-B5 — Which Node built-ins are exposed to scripts

**Question.** The complete set of modules a user script may `require`.

**Why it matters.** `goja` provides no Node built-ins. `logic/scripting.md` lists the ones observed in fixtures; a script relying on an unlisted module would fail in the port.

**Investigated.** `bruno-js/src/sandbox/*` defines the module allow-list; the list was read but not exhaustively cross-checked against every e2e fixture under `tests/scripting/inbuilt-libraries`.

**Would resolve.** Enumerate `inbuilt-libraries` fixtures and compare against `core/scripting` shims.

### UQ-B6 — Does `bru.sendRequest` participate in the request timeline?

**Question.** Are requests issued from a script (including the follow-on request in a "request chaining" flow) recorded in the timeline and in any history?

**Why it matters.** The Go port's `ScriptedRequest` event assumes they are surfaced distinctly. If Bruno hides them, behaviour differs.

**Investigated.** `flows/request-lifecycle.md` lists `bru.sendRequest` as an in-script transport call; timeline recording was not confirmed.

### UQ-B7 — OAuth2 token refresh in a headless run

**Question.** In `bru run` (CLI) with OAuth2 configured, how is an interactive authorization handled — fail, or open a system browser?

**Why it matters.** `implementation/go-architecture.md` §12 proposes an injected `AuthBrowser`; the CLI branch says "print the URL". If Bruno fails outright, parity is simpler and the port is a superset.

**Investigated.** `bruno-electron/src/ipc/network/authorize-user-in-window.js` handles the GUI case; the CLI path was not traced.

### UQ-B8 — Client-certificate precedence vs proxy CONNECT

**Question.** When a client cert is configured, is it presented to the proxy's CONNECT tunnel as well, or only to the origin server?

**Why it matters.** `logic/tls-and-certificates.md` documents collection-vs-global precedence (pinned by `tests/ssl/client-certs`), but the proxy interaction is untested upstream.

---

## UQ-D: Decisions requiring project-owner input

### UQ-D1 — GUI toolkit finality

**Context.** `plan_update.md` originally named "Shirei", which is not a Go module; the follow-up instruction resolved the toolkit to **MyGo** (`github.com/egoist/mygo`). MyGo is still at `v0.8.x`, so minor releases may break the widget API.

**Decision needed.** Not _which_ framework — that is settled — but **how tightly to bind** to it: pin a version and accept that upgrades are deliberate, maintain a continuously-green upgrade branch, or wrap every widget behind internal interfaces (which costs boilerplate for little gain given that MyGo is a declarative `View` API rather than an object tree).

**Documented in.** `implementation/dependency-recommendations.md` (scoring table), `architecture/system-overview.md` §8.5.

**Recommended default.** MyGo, exact-version pinned in `go.mod`, with `ui/presenters` kept toolkit-free so that a future toolkit swap touches only `ui/widgets` and `ui/layout`. Do not wrap individual widgets.

### UQ-D2 — Should response bodies be exposed through a local HTTP endpoint?

**Context.** Some workflows (and Bruno's own HTML preview) want to hand a body to an external viewer.

**Decision needed.** Spool-to-disk and open the file, or serve bodies from a loopback server? The latter widens the attack surface.

**Recommended default.** Spool to disk; open with the OS handler.

### UQ-D3 — History retention policy

**Status: DECIDED** in `progress/decisions.md` D-011 — metadata always; bodies on failure or explicit save; prune by age (30 days) and size (200 MB), both configurable.

**Context.** History is a new feature (`flows/history-flow.md`). Unbounded growth is unacceptable; over-aggressive pruning loses the audit trail users expect.

**Decision needed.** Retention by count, by age, by size, or a combination; and whether bodies are retained or only metadata.

**Recommended default.** Metadata always; bodies only for failures and for requests explicitly saved; prune by (age, total size).

### UQ-D4 — Do we implement a `.bru` writer or only a reader?

**Status: DECIDED** in `progress/decisions.md` D-011 — semantics-preserving writer. Measured: only 5 of 335 upstream files (1 %) are byte-identical after a round trip, so byte-exactness was rejected; model round-trip holds at 335/335.

**Context.** Full compatibility implies round-trip fidelity including formatting and comments. A reader-only implementation can load and run any collection but cannot save edits without destroying layout.

**Decision needed.** Byte-exact round-trip (expensive) vs semantics-preserving rewrite (cheaper, diffs noisier).

**Recommended default.** Semantics-preserving writer with stable section ordering, plus a golden round-trip test corpus from the pinned commit.

### UQ-D5 — Script compatibility target: Node-vm mode or QuickJS safe mode?

**Context.** The two upstream sandboxes diverge (`research/limitations.md` §2.6). A single `goja` runtime can match only one.

**Decision needed.** Which one, and whether to expose a "strict" flag.

**Recommended default.** Match Node-vm mode (it is what most collections were authored against) and document the deviations.

### UQ-D6 — Feature scope of v1

**Status: DECIDED** in `progress/decisions.md` D-011 — v1 is HTTP + collections + environments + auth (through OAuth2) + scripting + runner + Postman/curl import. Deferred: gRPC, WebSocket, mock server, AI assistant, terminal, OpenAPI sync, WSDL, docs editor.

**Context.** `implementation/feature-parity-matrix.md` assigns priorities. Several features are marked `deferred`.

**Decision needed.** Confirm that gRPC, WebSocket, mock server, AI assistant, terminal and OpenAPI sync are outside v1.

### UQ-D7 — Secrets backend on Linux

**Context.** MyGo exposes `Secrets` over the OS credential store; Linux may have no keyring in some environments.

**Decision needed.** Accept the failure and require a keyring, or add the AES-GCM fallback with a machine-derived key (weaker, but always available).

**Recommended default.** Try the keyring; on failure prompt once and fall back, storing a marker that the entry is degraded.

### UQ-D8 — Package/module path and licence notices

**Context.** `go.mod` declares `module fearless`. Collections and fixtures from Bruno are MIT-licensed and would be used as test corpora.

**Decision needed.** Final module path, and whether vendored Bruno fixtures are committed (with `NOTICE`) or generated by a script that clones the pinned commit.
