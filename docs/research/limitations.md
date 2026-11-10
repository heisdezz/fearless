# Limitations

Status: VERIFIED unless marked. This file records **what Bruno cannot do or does badly**, so the Go port does not inherit defects by accident. It is the companion to `research/unresolved-questions.md` (what *we* could not establish) and `implementation/feature-parity-matrix.md` (what to build).

Three categories:

1. **Product limitations** — missing features.
2. **Implementation limitations** — present but constrained by Bruno's architecture.
3. **Analysis limitations** — what this documentation set does not cover.

---

## 1. Product limitations (VERIFIED: absence in source)

### 1.1 No request history

There is no persistent request/response history. What exists instead:

- An **in-memory** per-collection `timeline` (cleared on reload) — see `flows/history-flow.md`.
- **Saved response examples** written to disk next to the request.

INFERRED: a user who closes a tab loses the response body. The Go port adds a real history store (`logic/persistence.md`), which is an **addition**, not parity work, and therefore carries no backward-compatibility obligation.

### 1.2 No plugin system

VERIFIED: no plugin manifest, loader, registry, or sandbox-for-plugins exists. Extension happens only through scripts and through first-party code. See `plugins/extension-architecture.md` and `plugins/plugin-system.md`.

### 1.3 No HTML/PDF response rendering without Chromium

Response preview for `text/html` and PDF relies on the Electron renderer. A Go-only application has no DOM, so this degrades to "open in system browser / show source". Recorded as a conflict in `implementation/go-architecture.md` §12.

### 1.4 Documentation editor is DOM-bound

The docs tab uses tiptap (ProseMirror). Fifteen Playwright specs pin the *behaviour*, but the implementation is unrecoverable outside a DOM. The Go port substitutes a Markdown editor.

### 1.5 AI assistant is outside the HTTP-client scope

`ipc/ai/` plus the `ai-sdk` dependency implement chat/assist features. Not analyzed in depth and not proposed for parity (see §4.2).

---

## 2. Implementation limitations (VERIFIED in source)

### 2.1 Unbounded response buffering

`promisifyStream` accumulates every chunk into an array and resolves with `Buffer.concat(chunks)` only on stream close (or first chunk when `closeOnFirst`). There is **no size cap**, no `maxContentLength`, and no spill-to-disk.

Consequences:

- A multi-gigabyte download is fully materialised in the main process.
- The buffer is then transferred over IPC to the renderer, so peak memory is roughly **2×** the body size.
- Nothing prevents an accidental `GET` of a large ISO from OOMing the app.

Source: `packages/bruno-electron/src/ipc/network/index.js`
Relevant symbols: `promisifyStream` (line 92), `Buffer.concat` (line 97), call sites at lines 1080, 1102, 1911, 1953

**Go port requirement:** cap in-memory bodies (PROPOSED: configurable, default ~32 MiB) and spill to a temp file with a `BodyRef` handle (`logic/persistence.md` §4). This directly contradicts Bruno, so `research/limitations.md` is cited from `logic/http-engine.md` §6.

### 2.2 Large monolithic orchestrator

`ipc/network/index.js` is **2320 lines** containing request preparation, auth, interpolation, script execution, transport, redirects, cookie handling, response parsing, streaming, gRPC and WebSocket paths. There is no unit spec for it. For the port this means the send pipeline has to be reconstructed from behaviour, not from a module contract — hence `flows/request-lifecycle.md` spells out the phase order explicitly.

### 2.3 Worker-pool parsing, but single-writer persistence

`bruno-filestore` parses in web workers (`services/pool`), yet writes go through a single `bruno-io` pipeline. INFERRED: parse throughput scales, write throughput does not. This is a design choice, not a defect, but it constrains how fast a 10k-request collection can be saved. The Go port replaces it with one writer goroutine (`implementation/go-architecture.md` §7).

### 2.4 Two on-disk dialects (`.bru` + `.yml`) with parallel code paths

Two complete codecs exist (`bruno-filestore/src/formats/bru` and `.../yml`). Every feature must be implemented twice and the e2e matrix for client certs alone has four combinations (`tests/ssl/client-certs/tests/{global,collection}-client-certs-{enabled,disabled}` × bru/yml). The Go port keeps both codecs for compatibility but normalises to one internal model (`data/collection-format.md`).

### 2.5 Interpolation is string-substitution based

`interpolate` resolves `{{var}}` by regex and string replacement, recursively. Consequences observed in the specs:

- A value containing `{{` can cause unintended second-order substitution.
- Type information is lost before interpolation (`tests/variable-datatypes` documents the workarounds).
- `encodeUrl` must run afterwards to repair encoding.

The Go port keeps the same observable semantics (compatibility) but is documented in `logic/variable-interpolation.md`, including the recursion depth guard.

### 2.6 Script sandbox differences between runtimes

Two script sandboxes ship: a Node `vm`-based one and a QuickJS "safe mode". They do **not** have identical semantics — `tests/scripting/{quickjs-safe-mode,nodevm-developer-mode}` exist precisely because behaviour differs, and `tests/bruno-cli/tests/integration/quickjs-trap-containment` pins a trap behaviour that is QuickJS-specific.

INFERRED: scripts that work in one mode can fail in the other. The Go port has exactly one runtime (`goja`) and therefore must document which mode it approximates; recorded in `research/unresolved-questions.md`.

### 2.7 No streaming to the UI for ordinary responses

`promisifyStream` resolves once. Only the SSE/WS paths emit incremental data. So a slow 100 MB JSON response shows **nothing** until it completes. The Go port's `ResponseChunk` event (§4 of `implementation/go-architecture.md`) fixes this; it is an improvement, not parity.

### 2.8 Proxy support depends on Node/Electron

PAC evaluation, WPAD discovery and system-proxy detection come from Node libraries and Electron APIs. There is no pure-Go drop-in, so the port uses `mygo/netproxy` (first-party, system proxy incl. PAC/WPAD) plus `x/net/proxy` for SOCKS5. Residual risk: OS-specific proxy edge cases differ from Chromium's network stack; recorded in `research/unresolved-questions.md`.

### 2.9 Memory pressure from Redux state

VERIFIED via `architecture/dependency-map.md`: the collections slice (`slices/collections/index.js`, 4549 lines) holds response bodies inside application state. Every response is cloned through Redux middleware (`snapshotMiddleware`, `autosaveMiddleware`). Combined with §2.1 this means large responses are held at least twice in the renderer too.

Source: `packages/bruno-app/src/providers/ReduxStore/index.js`
Relevant symbols: `snapshotMiddleware`, `autosaveMiddleware`, `draftDetectMiddleware`

**Go port requirement (already designed):** state holds `BodyRef{ID, Size, ContentType}` only, never bytes (`implementation/go-architecture.md` §3).

---

## 3. Portability limitations

| Constraint | Detail | Consequence |
| --- | --- | --- |
| Electron-only features | Chromium rendering, `node-pty` terminal, `webContents` devtools | Terminal and HTML preview have no Go equivalent; both are candidates for omission |
| Native module build | `node-addon-api`, `better-sqlite3`-style deps, `node:sqlite` behind `--experimental-sqlite` | Why `bruno-sqlite` tests need `NODE_OPTIONS`; irrelevant to Go |
| Node ≥ version pinned in `package.json` `engines` | Scripts may rely on modern Node APIs | goja coverage of Node built-ins is partial — see `logic/scripting.md` |
| Desktop-window assumption | Layout assumes ≥ ~900 px (`ui/application-layout.md` §6) | Go port may add a narrow mode; explicit addition |

---

## 4. Analysis limitations of this documentation set

### 4.1 Depth by module

| Module | Depth | Reason |
| --- | --- | --- |
| `.bru` / `.yml` codecs | High — grammar-level | Needed for byte-compatible parsing |
| Interpolation, auth, cookies, redirects, TLS | High | Core parity risk |
| Scripting (`bru`/`req`/`res`) | High, but API-surface level | Reproducing every edge of the JS runtime is out of scope |
| Import/export | Medium — format-level | 159 source files; behaviour summarised, not line-ported |
| gRPC / WebSocket / SSE | Medium | Documented as features, not internals |
| AI assistant | Low | Out of scope (§1.5) |
| Mock server | Low | Noted only |
| OpenAPI sync / API spec editor | Low | Feature-level entry in the parity matrix |
| Git integration | Low | Feature-level entry |

### 4.2 Deliberate non-goals (PROPOSED)

The following are documented but **proposed as out of scope** for the first Go release, because their value is low relative to a DOM/Chromium dependency:

- AI assistant
- Mock server
- Integrated terminal
- API spec editor with OpenAPI sync
- HTML/PDF rendered preview

Each is listed in `implementation/feature-parity-matrix.md` with priority `deferred`.

### 4.3 Claims we could not verify

Anything that remained unproven after source inspection is not asserted here; it lives in `research/unresolved-questions.md` with an explicit UNKNOWN label. Notably: exact WPAD fallback order, the precise deduplication rule for merged `headers` arrays across folder levels, and whether `snapshotMiddleware` retains full bodies by design or by accident.

---

## 5. Deviations the Go port will make on purpose

Consolidated list so reviewers can see them in one place:

| # | Bruno behaviour | Go port behaviour | Justification |
| --- | --- | --- | --- |
| 1 | Unbounded response buffering | Capped + spooled to disk | Prevents OOM (§2.1) |
| 2 | Responses stored in app state | `BodyRef` handles only | Memory (§2.9) |
| 3 | No persistent history | SQLite history store | User value; addition not parity |
| 4 | No streaming render for normal responses | `ResponseChunk` incrementally | Responsiveness (§2.7) |
| 5 | Two script sandboxes with divergent semantics | One `goja` runtime | Simplicity; divergence documented |
| 6 | HTML/PDF preview via Chromium | System browser / source view | No DOM in a Go-only app |
| 7 | Tiptap docs editor | Markdown editor | No DOM |
| 8 | JS IPC between processes | In-process typed events | Go-only architecture |
| 9 | No plugin system | Minimal compile-time extension points proposed | `plugins/extensibility-proposal.md` |
