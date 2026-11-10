# Implementation Roadmap

Status: **PROPOSED**. Constraint: **Go-only application, GUI included** (no React/Vue/Svelte/HTML-CSS frontend, no Electron, no Wails; JavaScript only inside the embedded script engine), built with **MyGo native UI** (`github.com/egoist/mygo/ui`). Architecture: `implementation/go-architecture.md`; packages: `implementation/package-structure.md`; libraries: `implementation/dependency-recommendations.md`.

Each phase lists deliverables, dependencies, acceptance criteria and testing. Phases 1–2 and 5 can overlap; **the CLI (`cmd/fearless-cli`) is built first** because it exercises the whole core without a GUI and is the fastest way to prove engine parity.

---

## Phase 1 — Core architecture & HTTP engine

**Deliverables**

| Package                                | Contents                                                                                                       |
| -------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| `core/model`                           | Collection, Item, Request, Body, Auth, Result, Timeline, BodyRef                                               |
| `core/format/bru`                      | Recursive-descent parser + writer for `.bru` (request, collection, folder, env)                                |
| `core/format`                          | `Codec` interface + format detection                                                                           |
| `core/vars`                            | Ordered scopes, provenance, `Interpolate` (recursive, JSON-escape mode), faker mocks, `{{?Prompt}}` extraction |
| `core/engine/build`                    | Tree merge, headers (ordered, duplicates), body modes, path params, URL normalisation                          |
| `core/engine/transport`                | `http.Client` with `CheckRedirect: ErrUseLastResponse`, manual redirect loop, `httptrace` timeline             |
| `core/engine/proxy`, `core/engine/tls` | off/on/system/PAC, CA bundle, client certs                                                                     |
| `core/cookies`                         | Jar + trustworthy-origin policy                                                                                |
| `core/engine/auth`                     | Registry + `none/basic/bearer/apikey/wsse`                                                                     |
| `app/commands`                         | `SendRequest`, `CancelRequest`; `app/bus` typed events                                                         |
| `cmd/fearless-cli`                     | `run` command: load a collection, execute one request, print status/timing                                     |

**Dependencies:** none (foundational).

**Acceptance criteria**

1. Parses every fixture under `bruno-tests/collection` without error and round-trips `.bru` byte-stably (modulo key order) for ≥ 95 % of files.
2. `Interpolate` passes a port of Bruno's `interpolate.spec.ts` cases, including recursive resolution and unresolved reporting.
3. Sending GET/POST/JSON/multipart/file bodies reproduces headers, body bytes and status against `httptest` servers.
4. Redirect matrix passes: 301/302/303 → GET, 307/308 preserve, limit exceeded, relative `Location`.
5. Core packages have **zero** imports of `ui/*` or GUI modules (CI check in `go-architecture.md §1`).

**Testing:** golden-file parser tests; table-driven interpolation tests; `httptest` transport suite; `go vet`, `staticcheck`, `govulncheck`.

---

## Phase 2 — Request editor & response viewer (GUI)

**Deliverables**

| Area               | Contents                                                                                                                                                               |
| ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ui/theme`         | Bruno token tree → `*ui.Theme` (Light/Dark, per-method colors, radii, spacing, font sizes)                                                                             |
| `ui/layout`        | `Split`/`SplitVertical`/`SplitPercent` composition, console dock, region budget                                                                                        |
| `ui/screens`       | Shell (title bar, sidebar, tab strip, status row), request editor (11 sub-tabs), response viewer                                                                       |
| `ui/widgets`       | Thin MyGo wrappers: `MethodBadge`, variable-aware text inputs, request tree rows, editable KeyValue table cells                                                        |
| `ui/presenters`    | Request/response/tab/sidebar/preferences view-models                                                                                                                   |
| Editors            | **`ui/editor`** for JSON/XML/JS bodies, scripts, tests and assertions; `ui.TextArea`/`TextAreaBuffer` for plain text; `ui.Autocomplete` for `{{variable}}` suggestions |
| Tables             | `ui.Table` + `ListState` (drags, sorting, per-column widths) for params, headers, form bodies, variables, assertions                                                   |
| Trees              | `ui.Outline`/`OutlineTable` with `Key` for the collection tree (virtualized)                                                                                           |
| Response           | `ui.List` (timeline), `ui.Table` (headers), `ui/editor` (body), `ui/chart` (optional timings)                                                                          |
| Dialogs            | `ui.Modal`, `ui.AlertDialog`, `ui.ContextMenu`, `ui.PopoverMenu`, `ui.Toast`                                                                                           |
| `core/model/state` | Draft/saved duality + per-tab UI sub-state                                                                                                                             |

**Dependencies:** Phase 1 (presenters call commands; no engine changes expected).

**Acceptance criteria**

1. The URL⇄params reducer algorithm (UID preservation, disabled rows retained, path params) behaves identically to `requestUrlChanged`.
2. Sub-tab indicator dots/counts appear exactly as in `ui/request-editor.md §1`.
3. A 10 000-row response table scrolls at ≥ 55 fps and never blocks the event loop (bodies are `BodyRef`, not inline).
4. Request/response split supports stacked and side-by-side with persisted ratios.
5. Draft edits never touch disk until save; dirty indicators match the model state.

**Testing:** headless presenter unit tests; widget smoke tests; golden screenshots per theme at 100/150/200 % DPI; manual parity checklist against a Bruno instance.

---

## Phase 3 — Collections & persistence

**Deliverables**

- `core/collection`: loader, `mygo.WatchFiles`-backed watcher (debounced, `Ignore` for `node_modules`/`.git`), UID-from-path mapping, `seq` ordering, 2.5 MB redaction.
- `core/persistence/settings` (JSON, migrations), `secrets` (`mygo.Secrets`), `body` (spool).
- Sidebar screen (`ui.Outline`/`Sidebar`), drag/reorder, rename/clone/delete.
- Save/open flows via `mygo.Dialog`; per-collection environment selection; window state via `mygo.WindowStateStore`.

**Dependencies:** Phase 1 (models/parser), Phase 2 (tree widget).

**Acceptance criteria**

1. External file edits reload without losing drafts; deleted files remove items; renamed files remap open tabs.
2. 1 000-request collection opens in < 2 s and the tree scrolls smoothly.
3. Two app instances cannot corrupt one workspace (lock file).
4. Secrets never appear in collection files or logs.

**Testing:** watcher integration tests with a temp dir; redaction tests; explicit secret-leak scan of written files; crash-recovery test (kill during save).

---

## Phase 4 — Authentication & environments

**Deliverables**

- Remaining auth providers: `digest`, `ntlm`, `oauth1`, `oauth2` (4 grants + PKCE + token cache), `awsv4`, `akamai-edgegrid`.
- `core/oauth2` token service with `TokenCache`; `platform/browser` for the authorization-code flow (system browser + loopback callback).
- Environments: file parser/writer, `extends` resolution, secret store, global environments, `.env` (`process.env`).
- Prompt-variable modal.

**Dependencies:** Phase 1 (auth registry, cookie jar), Phase 3 (env files).

**Acceptance criteria**

1. Digest and NTLM succeed against a challenge server using a retry `RoundTripper`.
2. OAuth2 authorization-code completes end-to-end without a WebView; tokens cached per `credentialsId` and refreshable.
3. AWS SigV4 signatures match a reference implementation for a fixed clock/credential.
4. Environment precedence and `extends` semantics match `flows/environment-resolution.md`, including cycle detection.

**Testing:** provider conformance suite; recorded OAuth token exchanges against a local fake authorization server; SigV4 golden vectors.

---

## Phase 5 — Scripting & assertions

**Deliverables**

- `core/scripting/goja`: sandbox, globals, `bru`/`req`/`res`/`test`/`expect`, promises, `goja` interruption deadlines.
- `core/engine/post`: `vars:post-response`, scripts, assertion operator table (28 operators), tests.
- Script error mapping back to `.bru` line numbers.
- Runner control (`setNextRequest`, `skipRequest`, `stopExecution`) and `bru.runRequest`, `bru.sendRequest`.

**Dependencies:** Phase 1 (engine hooks), Phase 2 (script editors).

**Acceptance criteria**

1. Bruno script fixtures produce identical variable deltas, test results and console output.
2. Runaway scripts terminate at the deadline without leaking goroutines.
3. Partial results are preserved when a script throws.
4. `bru.runRequest` nested requests appear as scripted timeline entries.

**Testing:** fixture replay suite; fuzz the parser for scripts-in-`.bru`; race detector (`-race`) across concurrent script runs.

---

## Phase 6 — Import/export & Bruno compatibility

**Deliverables**

- `convert/postman`, `convert/insomnia`, `convert/openapi`, `convert/curl`.
- `core/format/yaml` (opencollection) read/write + migration from `.bru`.
- Code generator (curl/Go/JS/Python…).
- Compatibility audit: run a real Bruno collection through both apps and diff results.

**Dependencies:** Phases 1–5.

**Acceptance criteria**

1. Importing Bruno's converter fixtures yields structurally equivalent collections; every dropped construct is reported.
2. A collection exported to OpenCollection YAML and re-opened loses no behaviour.
3. 30 representative requests (auth × body × redirect × script) produce identical status/headers/body hashes in Bruno and the Go app.

**Testing:** converter fixture tests; cross-app differential harness (record once from Bruno, replay in Go).

---

## Phase 7 — Advanced functionality & extensibility

**Deliverables**

- `gRPC` (grpc-go), `WebSocket` (`coder/websocket`), GraphQL helpers.
- Collection runner (folder/collection, tags, delay) + CLI reporters.
- History (SQLite) and large-body streaming previews (new capability, see `flows/history-flow.md`).
- Extension registries: `HTTPHook`, `BodyEncoder`, `VariableResolver`, `Importer`/`Exporter` (`plugins/extensibility-proposal.md`).
- Proxy PAC parity, system proxy detection, workspace support.

**Dependencies:** Phases 1–6.

**Acceptance criteria**

1. Runner respects tags/selection and reports failures with exit codes.
2. Streaming responses render incrementally without buffering the whole body.
3. Extension registries are exercised by at least one internal implementation each.

**Testing:** long-running stream tests; runner integration against mock servers; extension conformance tests.

---

## Cross-phase gates

| Gate                                                                  | When                                      |
| --------------------------------------------------------------------- | ----------------------------------------- |
| Core-imports-GUI check + `govulncheck`                                | Continuous, phases 1+                     |
| Differential harness vs Bruno                                         | Phase 2 (requests), Phase 6 (collections) |
| Golden screenshots (all themes/DPIs)                                  | Phases 2, 3, 7                            |
| Resource benchmarks (open collection, scroll, memory with 10 MB body) | Phases 3, 7                               |

## Known risk items

1. **MyGo pre-1.0 churn** — pin version, isolate in `internal/ui` + `internal/platform`.
2. **Text editor parity** — `ui/editor` covers syntax, completions and diffs, but Bruno's CodeMirror behaviours (variable chips, lint markers, error-line focus) still need mapping; prototype in Phase 2 before committing to other editor work.
3. **Script library parity** — bundled JS libs (lodash, moment, chai, cheerio) must be shipped into the goja sandbox; scope this early in Phase 5.
4. **No HTML app UI** — HTML/PDF response previews may use a response-only web view; the tiptap docs tab becomes `ui/markdown`/`ui/richtext`; the in-app terminal is replaced by opening the OS terminal (`mygo.Shell`).
