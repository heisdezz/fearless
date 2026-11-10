# Design Decisions (Observed)

Status: **VERIFIED** = behavior observed in code; **INFERRED** = rationale reconstructed from evidence, not from commit messages or ADRs. Bruno has no in-repo ADR directory; decisions are inferred from code comments, tests, and structure.

---

## D1 — Local-first, filesystem-backed collections

**Decision.** A "collection" is a directory of plain text files. No database owns the source of truth.

**Evidence.** `bruno.json` / `opencollection.yml` at the collection root, one file per request/folder/environment, `.bru`/`.yml` plain text, git-friendly. `bruno-filestore` is the only writer.

**Trade-off accepted.** Reading requires parsing many small files; mitigated by worker parsing and a **file index** (`services/mount`, `bruno-sqlite`).

**Go impact.** Keep it. This is Bruno's core differentiator vs. Postman. Value: near-zero migration friction.

---

## D2 — Two formats simultaneously (`bru` and `yml`)

**Decision.** Keep the legacy `.bru` DSL while migrating to `opencollection` YAML.

**Evidence.** `FORMAT_CONFIG` in `utils/collection.js`; `parseRequest(content, { format })` switching; `ipc/yml-migration.js` exists; `MigrateCollectionToYmlModal` component.

**Trade-off.** Two serializers and two grammars to maintain; every feature must be implemented twice.

**Go impact.** Implement `bru` first (it is still what nearly every existing collection uses), then `yml`. Design a single `Document` interface with two `Codec`s from day one (PROPOSED).

---

## D3 — The `.bru` DSL is grammar-defined, not hand-parsed

**Decision.** Use `ohm-js` (PEG-ish) with a declarative grammar over ad-hoc regex/state machines.

**Evidence.** `bruToJson.js` starts with `const grammar = ohm.grammar('Bru { … }')` and has a semantic action per rule.

**Trade-off.** Grammar is readable and testable, but build-time cost and error messages are noisy.

**Go impact.** A hand-written recursive-descent parser is straightforward. Prior art: `usebruno` has no Go parser; several community parsers exist. Treat the grammar as the specification (`data/bru-format.md`).

---

## D4 — All privileged work in the main process

**Decision.** Renderer never touches FS/network directly; a narrow preload bridge exposes IPC only.

**Evidence.** `contextBridge.exposeInMainWorld('ipcRenderer', ...)`, `restrictIpcToAppFrames(ipcMain)` security wrapper, no `nodeIntegration` usage in the renderer.

**Go impact.** In a single-process Go GUI this constraint disappears, but the *shape* should remain: UI → core service via interfaces, never direct I/O from UI code.

---

## D5 — Request execution is a single mutable pipeline

**Decision.** One `request` object flows through merge → script → interpolate → configure → send → post-script.

**Evidence.** `runRequest` in `ipc/network/index.js`; every stage mutates and returns the same object.

**Trade-off.** Fast and simple; ordering bugs are easy; two places handle basic auth (acknowledged by a TODO in `interpolate-vars.js`).

**Go impact.** **Deliberately diverge:** use an immutable context object per stage, documented ordering, and a stage list. Preserve semantics, not structure.

---

## D6 — Manual redirect implementation

**Decision.** `maxRedirects: 0` in axios; Bruno follows redirects in a response interceptor.

**Evidence.** `axios-instance.js`: `maxRedirects: 0` (line 127), redirect logic in the rejection handler (`redirectResponseCodes`, `redirectCount`).

**Rationale (INFERRED).** Needed to record per-hop timeline entries, save cookies per hop, strip auth on cross-origin hops, and resolve relative Locations — none of which axios exposes.

**Go impact.** Use `http.Client{CheckRedirect: func(...) error { return http.ErrUseLastResponse }}` and a loop, so behavior matches.

---

## D7 — JS sandbox is a first-class runtime with two backends

**Decision.** Execute user scripts in an isolated JS engine; default `quickjs`, optional `nodevm`.

**Evidence.** `ScriptRuntime` constructor default `'quickjs'`; `sandbox/quickjs/index.js`, `sandbox/node-vm/index.js`; `collection.brunoConfig.scripts.moduleWhitelist` for nodevm.

**Trade-off.** quickjs is fast and safe but has limited Node compat; nodevm is compatible but has larger attack surface.

**Go impact.** `goja` is the equivalent of quickjs (pure Go, no cgo). Provide a `ScriptEngine` interface so an alternative engine can be added.

---

## D8 — Object store for post-response variables

**Decision.** `vars:post-response` entries are **JS expressions** evaluated against the response, not templates.

**Evidence.** `VarsRuntime.runPostResponseVars` calls `evaluateJsExpression(v.value, context)` for each enabled var and `bru.setVar(name, value)`.

**Go impact.** Requires the JS engine even for "simple" variable capture — cannot be done with a template engine.

---

## D9 — Secrets are per-environment-key encrypted at rest

**Decision.** `vars:secret` names live in the env file; values live in an encrypted store (`env-secrets`), keyed by collection/env/key.

**Evidence.** `store/env-secrets.js`, `decryptStringSafe`, `EnvironmentSecretsStore` in `collection-watcher.js`; env files contain `vars:secret [ names… ]` arrays only.

**Go impact.** Need an OS keyring (PROPOSED: `zalando/go-keyring`) or a passphrase-derived AES key, matching UX.

---

## D10 — OAuth2 credentials are cached centrally, keyed by `credentialsId`

**Decision.** Access/refresh tokens are stored in main-process store and reused across requests/folders.

**Evidence.** `store/oauth2.js`, `getOAuth2TokenUsing*` helpers, `request.oauth2Credentials = { credentials, credentialsId, folderUid }`, `main:credentials-update` IPC, `clear-oauth2-cache` handler.

**Go impact.** Model as a `TokenCache` interface with disk persistence.

---

## D11 — Inheritance is resolved at execution time, not at parse time

**Decision.** `folder.bru`/`collection.bru` hold partial config (`headers`, `auth`, `vars`, `script`); merging happens in `prepareRequest` via the item's *tree path*.

**Evidence.** `getTreePathFromCollectionToItem`, `mergeHeaders`, `mergeVars`, `mergeAuth`, `mergeScripts` in `utils/collection.js`.

**Go impact.** Keep. Persist the tree; resolve lazily.

---

## D12 — Responses are large binary-capable and streamed

**Decision.** `responseType: 'stream'` (set in `runRequest`); SSE bodies are rebuilt from chunks; binary buffers are base64-encoded across IPC.

**Evidence.** `request.responseType = 'stream'`, `promisifyStream`, `isResponseStream`, `buildResponseBodyFromStreamChunks`, `dataBufferBase64`.

**Go impact.** `http.Response.Body` is already a stream. Define a `BodyStore` that spills large bodies to temp files to keep memory bounded (PROPOSED).

---

## D13 — Timeline as a diagnostic record

**Decision.** Every request records a timestamped timeline (preparation, DNS/connect/TLS, headers, response, redirects, errors).

**Evidence.** `timeline-agent.ts`, timeline pushes in `axios-instance.js`, `Timeline` UI component, `response.timeline` returned via IPC.

**Go impact.** `httptrace.ClientTrace` gives the hooks needed; reproduce the same entry types.

---

## D14 — Code generation and curl import are pure functions over the request model

**Evidence.** `bruno-common/src/generate-code`, renderer `utils/codegenerator` and `utils/curl`.

**Go impact.** Nice-to-have; implement after the engine.

---

## D15 — Extensibility is **not** a plugin system

**Observation (VERIFIED).** There is no plugin manifest, no dynamic loading of third-party code, no extension hook registry. Extensibility consists of:

1. **Scripts** (`script:pre-request`, `script:post-response`, `tests`, `vars:post-response`) — arbitrary user JS in a sandbox.
2. **Collection config `scripts.moduleWhitelist`** — which Node modules the `nodevm` runtime may `require`.
3. **Fixed auth provider set** in `setAuthHeaders` / `configureRequest` and matching `.bru` grammar rules.
4. **Importers/converters** compiled into `bruno-converters`.
5. **External integrations** through the sandbox: `bru.sendRequest`, `bru.runRequest`, Vault (`node-vault.ts`), AWS credential providers.

See `plugins/extension-architecture.md`. A Go port can **improve** on this with a real interface-based extension model (PROPOSED).
