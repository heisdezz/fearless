# Feature Parity Matrix

Status: VERIFIED for "How Bruno implements it" and source paths; PROPOSED for the Go columns (implementation, complexity, priority, compatibility).

Priority legend: **P0** = required for a usable HTTP client; **P1** = required for Bruno compatibility; **P2** = valuable, can ship later; **deferred** = documented but out of v1 scope (see `research/limitations.md` §4.2).
Complexity legend: **L** / **M** / **H**, judged against the Go implementation, not Bruno's code.
Compatibility: **Yes** = byte/behaviour compatible is achievable and worth achieving; **Partial** = behaviour matches, representation differs; **No** = deliberately divergent (with reason).

---

## 1. Core HTTP client (P0)

| Feature                              | How Bruno implements it                                                          | Source                                                                    | Required Go implementation                                               | Cx  | Dependencies                   | Priority | Compat                 |
| ------------------------------------ | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------- | ------------------------------------------------------------------------ | --- | ------------------------------ | -------- | ---------------------- |
| HTTP methods                         | Method string on the request; UI restricts to a fixed list                       | `bruno-app/.../RequestPane/QueryUrl`                                      | Method enum + free-text escape hatch, `net/http`                         | L   | —                              | P0       | Yes                    |
| URL input + implicit scheme          | Prepends `http://` when no scheme and no leading `{{`                            | `bruno-electron/src/ipc/network/index.js` (`configureRequest`, l.118–130) | `engine/build/normalizeURL`                                              | L   | `net/url`                      | P0       | Yes                    |
| Query params                         | Ordered list of `{name, value, enabled, type}`; serialized into the URL          | `RequestPane/QueryParams`, `prepare-request.js`                           | Keep ordered slice; do **not** use `url.Values` (loses order/duplicates) | L   | —                              | P0       | Yes                    |
| Headers                              | Ordered `{name, value, enabled}`; merged collection→folder→request               | `utils/collection.js`, `RequestPane/RequestHeaders`                       | `[]Header` with ordered merge (case-insensitive, see UQ-B1)              | M   | —                              | P0       | Yes                    |
| Body: JSON                           | Free text body with `none/json/text/xml/sparql/form-urlencoded/multipart/binary` | `RequestBody`, `utils/form-data.js`                                       | `Body` union + per-type encoder                                          | M   | —                              | P0       | Yes                    |
| Body: form-urlencoded                | Key/value rows, encoded                                                          | `FormUrlEncodedParams`                                                    | `url.Values` preserving order                                            | L   | —                              | P0       | Yes                    |
| Body: multipart                      | Rows of `text`/`file`; `@file(path)`, `@contentType(...)` annotations            | `MultipartFormParams`, `utils/form-data.js`                               | `multipart.Writer` + recorded part order                                 | H   | `mime/multipart`               | P0       | Yes                    |
| Body: file (binary)                  | Single file reference                                                            | `FileBody`                                                                | Stream from disk, `Content-Length` from stat                             | M   | —                              | P0       | Yes                    |
| Body: GraphQL                        | Query + variables editor, schema introspection                                   | `GraphQLRequestPane`, `GraphQLVariables`, `GraphQLSchemaActions`          | JSON envelope `{query, variables, operationName}` + introspection call   | M   | —                              | P1       | Yes                    |
| Send / cancel                        | `axios` with an `AbortController`; cancel tokens kept per request                | `ipc/network/index.js` (`currentAbortController`), `cancelTokens`         | `context.Context` per request, cancel registry in `app/commands`         | M   | —                              | P0       | Yes                    |
| Response status/time/size            | Computed in main, rendered in `ResponsePane` status strip                        | `network/response-time.ts`, `sent-headers.ts`                             | Typed `ResultMetrics{Status, Duration, Received, Sent}`                  | L   | —                              | P0       | Yes                    |
| Response body view                   | CodeMirror with format modes + raw/preview toggle                                | `ResponsePane`, `utils/codemirror`                                        | Text widget with syntax highlighting; raw always available               | M   | —                              | P0       | Yes                    |
| Response headers view                | Table of received headers                                                        | `ResponsePane`                                                            | Table widget                                                             | L   | —                              | P0       | Yes                    |
| Redirects                            | Followed by `axios` with `maxRedirects`; hop recording                           | `logic/redirects.md`                                                      | `http.Client.CheckRedirect` + manual hop capture                         | M   | —                              | P0       | Yes                    |
| Cookies                              | `tough-cookie` jar in main, persisted debounced                                  | `bruno-requests/src/cookies`, `store/cookies.js`                          | `net/http/cookiejar` + custom persistence layer                          | M   | —                              | P0       | Yes                    |
| Proxy                                | HTTP/HTTPS/SOCKS, PAC, system proxy, per-request override                        | `bruno-requests/src/network/system-proxy`, `http-https-agents.ts`         | `http.Transport.Proxy` + `mygo/netproxy` + `x/net/proxy`                 | H   | `mygo/netproxy`, `x/net/proxy` | P1       | Partial                |
| TLS / client certs                   | Global and collection-scoped certs, CA override, passphrase                      | `ipc/network/cert-utils.js`, `tests/ssl/client-certs`                     | `tls.Config` builder + `mygo.Secrets` for passphrases                    | M   | —                              | P1       | Yes                    |
| Timeouts                             | Per-request `timeout` setting                                                    | `RequestPane/Settings`                                                    | `context.WithTimeout`                                                    | L   | —                              | P0       | Yes                    |
| Response body size cap               | **None** — unbounded buffering                                                   | `promisifyStream` (l.92)                                                  | Cap + disk spill (`BodyRef`)                                             | M   | —                              | P0       | **No** (§1 divergence) |
| Streaming (SSE)                      | Axios stream + incremental IPC events; cancel supported                          | `ipc/network/index.js`, `tests/sse`                                       | `bufio.Scanner` over the body + `ResponseChunk` events                   | M   | —                              | P1       | Yes                    |
| WebSocket                            | `ws-client.js`; multi-message `.bru` form                                        | `bruno-requests/src/ws`, `tests/websockets`                               | `coder/websocket`                                                        | M   | `coder/websocket`              | P2       | Partial                |
| gRPC                                 | `grpc-client.js`; proto/protobuf support; metadata; multi-message                | `bruno-requests/src/grpc`, `tests/grpc`                                   | `grpc-go` + `protoreflect`                                               | H   | `google.golang.org/grpc`       | P2       | Partial                |
| Request chaining (`bru.sendRequest`) | Issued from inside a script through the same transport                           | `bruno-js/src/bru.js`                                                     | Host function that calls `Engine.Do` with a child context                | M   | —                              | P1       | Yes                    |

---

## 2. Authentication (P1)

All providers are registered in `packages/bruno-requests/src/auth/` and applied by `prepare-request.js`. Bruno requires touching several places to add one (`plugins/authentication-providers.md`).

| Feature                          | Bruno source                                          | Required Go implementation                                                            | Cx  | Dependencies           | Priority | Compat  |
| -------------------------------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------- | --- | ---------------------- | -------- | ------- |
| Inherit / none                   | `utils/collection.js` auth merge                      | `AuthConfig{Mode: inherit}` resolved during build                                     | L   | —                      | P0       | Yes     |
| Basic                            | inline in `prepare-request.js`                        | `req.SetBasicAuth`                                                                    | L   | —                      | P0       | Yes     |
| Bearer                           | inline                                                | Set `Authorization` header; token may be `{{var}}`                                    | L   | —                      | P0       | Yes     |
| API key (header/query)           | inline                                                | Header or query injection                                                             | L   | —                      | P0       | Yes     |
| Digest                           | `digestauth-helper.js`                                | Custom `RoundTripper` (challenge/response, `qop=auth`)                                | H   | —                      | P1       | Yes     |
| NTLM                             | `ntlm.ts`                                             | Handshake `RoundTripper`                                                              | H   | `Azure/go-ntlmssp`     | P2       | Yes     |
| OAuth1                           | `oauth1-request-authorization.ts`                     | Signature base string + HMAC-SHA1/PLAINTEXT                                           | H   | `oauth1` lib           | P2       | Yes     |
| OAuth2                           | `oauth2-helper.ts`; tokens cached per `credentialsId` | Auth-code + PKCE + client-credentials + password + device; token cache                | H   | `golang.org/x/oauth2`  | P1       | Yes     |
| AWS SigV4                        | `awsv4auth-helper.js`                                 | Sign request; supports session token                                                  | H   | `aws-sdk-go-v2` signer | P2       | Yes     |
| WSSE                             | inline                                                | UsernameToken with nonce + created                                                    | M   | —                      | P2       | Yes     |
| Akamai EdgeGrid                  | `edgegrid-helper.js`                                  | HMAC signature scheme                                                                 | M   | —                      | P2       | Yes     |
| OAuth2 interactive authorization | Opens an Electron window                              | `AuthBrowser` interface; GUI opens system browser + loopback listener, CLI prints URL | M   | —                      | P1       | Partial |
| OAuth2 cache clearing            | `ipc` handler clearing the store                      | `ClearOAuth2Cache` command                                                            | L   | —                      | P2       | Yes     |

---

## 3. Variables and environments (P0)

| Feature                                         | How Bruno implements it                                                  | Source                                                              | Go implementation                                   | Cx  | Priority | Compat  |
| ----------------------------------------------- | ------------------------------------------------------------------------ | ------------------------------------------------------------------- | --------------------------------------------------- | --- | -------- | ------- |
| `{{var}}` interpolation                         | Recursive regex substitution with cycle guard                            | `bruno-common/src/interpolate`                                      | Same algorithm, explicit depth limit                | M   | P0       | Yes     |
| Precedence chain                                | Runtime > request > folder > collection > env > global env > process env | `logic/variable-interpolation.md`                                   | Ordered `Scopes` struct, explicit resolution list   | M   | P0       | Yes     |
| Dynamic variables                               | `$guid`, `$timestamp`, `$randomInt`, `$isoTimestamp`, `$randomIP`…       | `bruno-common/src/interpolate`                                      | Providers registered by name                        | L   | P1       | Yes     |
| Prompt variables                                | Modal collects values before send                                        | `providers/PromptVariables`, `tests/interpolation/prompt-variables` | `PromptVars` in `SendRequest`; GUI dialog; CLI flag | M   | P1       | Partial |
| Secret variables                                | Value stored encrypted per env                                           | `store/env-secrets.js`                                              | `Secrets` interface (`logic/persistence.md` §4.3)   | M   | P1       | Yes     |
| Env files (`.bru` / `.yml`)                     | Two dialects                                                             | `data/environment-format.md`                                        | Both codecs                                         | M   | P1       | Yes     |
| Env inheritance                                 | `extends` / parent chain                                                 | `flows/environment-resolution.md`                                   | Resolution walk with cycle detection                | M   | P1       | Yes     |
| `bru.setEnvVar` / `setVar` / `setCollectionVar` | Script API writing to env/collection stores                              | `bruno-js/src/bru.js`, `tests/environments/api-setEnvVar`           | Host functions calling `vars` + persistence         | M   | P1       | Yes     |
| Typed variables                                 | Values keep type where possible                                          | `tests/variable-datatypes`                                          | `any` values in scope map                           | M   | P2       | Partial |
| Unresolved-variable reporting                   | Warns before send, lists names                                           | `tests/unresolved-variables-info`                                   | `UnresolvedVariables` event (already planned)       | L   | P1       | Yes     |
| Dotenv import                                   | `.env` parsing                                                           | `dotenv` handling in main                                           | `.env` parser                                       | L   | P2       | Yes     |
| Collection/global runtime vars                  | Held in memory during a run                                              | Runner state                                                        | `RuntimeVars` map owned by command                  | L   | P1       | Yes     |

---

## 4. Scripting, tests and assertions (P1)

| Feature                                   | How Bruno implements it                                   | Source                                                          | Go implementation                   | Cx  | Priority | Compat          |
| ----------------------------------------- | --------------------------------------------------------- | --------------------------------------------------------------- | ----------------------------------- | --- | -------- | --------------- |
| Pre-request script                        | Runs in `vm`/QuickJS before send, may mutate request/vars | `bruno-js/src/runtime/*`, `RequestPane/Script`                  | `goja` + `bru`/`req` shims          | H   | P1       | Partial         |
| Post-response script                      | Read-only response, mutable vars                          | same                                                            | same                                | H   | P1       | Partial         |
| `tests {}` block                          | `test(name, fn)` + `expect`                               | `bruno-js/src/runtime/test-runtime`                             | Host `test`/`expect` implementation | H   | P1       | Yes             |
| `assert` expressions                      | Expression evaluated against `res`                        | `bruno-js/src/runtime/assert-runtime`                           | Parse-and-eval in goja              | M   | P1       | Yes             |
| `vars:pre-request` / `vars:post-response` | Declarative variable blocks                               | `bruno-lang` grammar                                            | Codec support + evaluation          | M   | P1       | Yes             |
| Script console output                     | Forwarded to console dock                                 | `ScriptOutput` IPC                                              | `ScriptOutput` event                | L   | P1       | Yes             |
| Script timeout                            | Runtime trap/interrupt                                    | `quickjs-trap-containment`                                      | `vm.Interrupt` with deadline        | M   | P1       | Partial         |
| Inbuilt libraries                         | Curated module allow-list                                 | `bruno-js/src/sandbox`, `tests/scripting/inbuilt-libraries`     | Pure-Go shims for the same set      | H   | P1       | Partial (UQ-B5) |
| Code generator (curl, HAR, languages)     | `generate-code`                                           | `bruno-common/src/generate-code`, `tests/request/generate-code` | Template-based generators           | M   | P2       | Partial         |

---

## 5. Collections and persistence (P0)

| Feature                             | How Bruno implements it                                         | Source                                          | Go implementation                                      | Cx  | Priority | Compat               |
| ----------------------------------- | --------------------------------------------------------------- | ----------------------------------------------- | ------------------------------------------------------ | --- | -------- | -------------------- |
| `.bru` parse/write                  | ohm-js grammar v1 + v2                                          | `bruno-lang/v1`, `v2`                           | Hand-written recursive-descent lexer/parser            | H   | P0       | Yes                  |
| `.yml` (OpenCollection) parse/write | `yaml` lib + per-type parsers                                   | `bruno-filestore/src/formats/yml`               | `yaml.v3` + typed structs                              | H   | P1       | Yes                  |
| Collection config                   | `bruno.json` / `opencollection.yml`                             | `data/collection-format.md`                     | Typed struct + validation                              | L   | P0       | Yes                  |
| Folder organization                 | Directories with `folder.bru`/`folder.yml`; `seq` ordering      | `data/collection-format.md`                     | Tree model with explicit ordering                      | M   | P0       | Yes                  |
| Open/save/reload                    | IPC + worker-pool parse                                         | `ipc/collection.js`, `services/pool`            | `collection` package + parse pool                      | M   | P0       | Yes                  |
| File watcher                        | `chokidar`, self-write suppression                              | `app/collection-watcher.js`                     | `mygo.WatchFiles` (recursive, debounced) + suppression | M   | P1       | Yes                  |
| Draft/saved model                   | Redux draft slices + middleware                                 | `slices/collections/index.js`                   | `Saved`/`Draft` fields + actor                         | M   | P0       | Yes                  |
| Autosave                            | `autosaveMiddleware`                                            | `ReduxStore/index.js`                           | Settings-driven debounced save command                 | L   | P2       | Yes                  |
| Response examples                   | Saved next to request as `*.example.bru`                        | `exampleReducers.js`, `tests/response-examples` | Example list on `model.Item` + codec support           | M   | P2       | Yes                  |
| Transient/scratch requests          | Unsaved detail-pane requests                                    | `tests/transient-requests`, `scratch-requests`  | Unsaved tab with `synthetic` UID                       | M   | P2       | Partial              |
| Workspaces                          | Multiple collection roots, git integration                      | `slices/workspaces/actions.js`                  | Workspace config file + roots                          | M   | P2       | Partial              |
| Preferences                         | `electron-store` preferences                                    | `store/preferences.js`                          | `preferences.json` (`logic/persistence.md` §4.2)       | L   | P0       | Yes                  |
| Window state                        | `window-state.js`                                               | same                                            | `session.json`; MyGo `WindowStateStore`                | L   | P2       | Yes                  |
| History                             | **Does not exist** (in-memory timeline only, cleared on reload) | `flows/history-flow.md`                         | SQLite history store                                   | M   | P2       | **No** (new feature) |
| Import: Postman v2/2.1              | `bruno-converters/src/postman` (101 test files)                 | same                                            | Converter package with fixture corpus                  | H   | P1       | Yes                  |
| Import: OpenAPI 3 / Swagger 2       | `bruno-converters/src/openapi`                                  | same                                            | OpenAPI parser → collection                            | H   | P2       | Partial              |
| Import: Insomnia                    | `bruno-converters/src/insomnia`                                 | same                                            | Converter                                              | M   | P2       | Partial              |
| Import: WSDL                        | `bruno-converters/src/wsdl`                                     | same                                            | Converter                                              | H   | deferred | Partial              |
| Import: URL / curl                  | URL parse, curl paste                                           | `tests/import/url-import`                       | Parsers                                                | M   | P2       | Yes                  |
| Export                              | Postman/OpenCollection writers                                  | `bruno-converters`, `bruno-io`                  | Writers                                                | M   | P2       | Partial              |

---

## 6. UI features (P0/P1)

| Feature                                     | How Bruno implements it                                                  | Source                                | Go implementation                            | Cx  | Priority | Compat  |
| ------------------------------------------- | ------------------------------------------------------------------------ | ------------------------------------- | -------------------------------------------- | --- | -------- | ------- |
| Sidebar collection tree                     | Recursive tree, dnd, context menu                                        | `components/Sidebar`                  | `ui.Sidebar` / `ui.Tree`                     | M   | P0       | Partial |
| Request tabs                                | Drag-reorder, overflow collapse, vertical/horizontal                     | `components/RequestTabs`              | `ui.TabsBase` + presenter reorder            | M   | P0       | Partial |
| Request/response split + orientation toggle | `changeLayout` (⌘/Ctrl+J)                                                | `components/RequestTabPanel`          | `ui.Split` / `ui.SplitVertical`              | L   | P0       | Partial |
| Request sub-tabs                            | Params, Body, Headers, Auth, Vars, Script, Assert, Tests, Docs, Settings | `HttpRequestPane`                     | Tabs + per-tab editors                       | M   | P0       | Partial |
| Method selector                             | Coloured dropdown                                                        | `QueryUrl`                            | `ui.MenuButton` + `MethodBadge`-style chip   | L   | P0       | Partial |
| Code editor                                 | CodeMirror 5 with variable autocomplete                                  | `MultiLineEditor`, `utils/codemirror` | MyGo `ui/editor` + completion from scope     | H   | P0       | Partial |
| Variable autocomplete                       | Triggered on `{{`                                                        | `tests/autocomplete`                  | Provider on the editor                       | M   | P1       | Partial |
| Environment selector                        | Header dropdown                                                          | `CollectionHeader`                    | `ui.MenuButton`                              | L   | P0       | —       |
| Response tabs                               | Response, Headers, Timeline, Tests                                       | `ResponsePane`                        | Tabs                                         | L   | P0       | —       |
| Timeline view                               | Per-request phase table                                                  | `tests/request/timeline`              | Table widget fed by `Result.Timeline`        | M   | P1       | —       |
| Test/assert results                         | Pass/fail rows with diffs                                                | `ResponsePane` tests tab              | Table + diff view (`ui/editor` diff)         | M   | P1       | —       |
| Keyboard shortcuts                          | `mousetrap` + per-OS map                                                 | `providers/Hotkeys/keyMappings.js`    | Presenter key router + MyGo `GlobalShortcut` | M   | P1       | Partial |
| Dialogs/modals                              | `Portal`                                                                 | `ui/Modal`, `ui/DialogBase`           | same                                         | L   | P0       | —       |
| Toasts/notifications                        | `providers/Toaster`                                                      | `Context.Toast`                       | same                                         | L   | P1       | —       |
| Theming                                     | Tailwind + CSS variables, light/dark                                     | `providers/Theme`                     | `ui.Theme` tokens (`ui/design-system.md` §7) | M   | P1       | Partial |
| Onboarding/welcome                          | `WelcomeModal`, `EmptyAppState`                                          | `tests/onboarding`                    | Empty-state screens                          | L   | P2       | —       |
| Docs editor                                 | tiptap                                                                   | `tests/richtext-editor`               | Markdown editor                              | M   | P2       | **No**  |
| HTML/PDF preview                            | Electron renderer                                                        | `ResponsePane`                        | System browser / source                      | L   | P2       | **No**  |
| Console/devtools dock                       | Electron devtools + terminal                                             | `tests/devtools`                      | Log dock (own logger)                        | M   | P2       | **No**  |
| AI assistant                                | `ipc/ai/` + `ai-sdk`                                                     | `tests/ai-assist`                     | —                                            | H   | deferred | No      |

---

## 7. CLI and runner (P1)

| Feature                        | How Bruno implements it                | Source                                                                     | Go implementation                      | Cx  | Priority | Compat  |
| ------------------------------ | -------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------- | --- | -------- | ------- |
| `bru run` collection/folder    | Reuses the same transport + JS runtime | `packages/bruno-cli/src/runner`                                            | `cmd/fearless run` over `app/commands` | M   | P1       | Yes     |
| Environment selection flags    | `--env`, `--env-var`                   | `tests/.../run-default-environment.spec.js`, `cli-global-env-var-override` | CLI flags → `SendRequest`              | L   | P1       | Yes     |
| Tag filtering                  | `--tags` on folders/requests           | `tests/.../run-folder-tags`                                                | Tag filter in runner                   | L   | P2       | Yes     |
| Iterations / delay / data file | Runner options                         | `runner` package                                                           | CSV/JSON data driver                   | M   | P2       | Yes     |
| JUnit / HTML reports           | Reporters                              | `junit.xml` outputs, `run-onfail-persistence`                              | `testing`-style reporter + JUnit XML   | M   | P1       | Partial |
| Fail-fast / `onFail`           | Runner option persisted                | `run-onfail-persistence.spec.js`                                           | Runner config                          | L   | P2       | Yes     |
| Exit codes                     | Non-zero on failure                    | CLI                                                                        | Same                                   | L   | P1       | Yes     |

---

## 8. Summary by priority

| Priority     | Count | Notes                                                               |
| ------------ | ----- | ------------------------------------------------------------------- |
| **P0**       | 26    | Everything needed to send, inspect, and edit requests               |
| **P1**       | 27    | Bruno compatibility: both formats, auth, scripting, runner, watcher |
| **P2**       | 24    | Polish and breadth: gRPC/WS, import breadth, workspaces             |
| **deferred** | 4     | WSDL, AI assistant, mock server, terminal                           |

### 8.1 Three deliberate incompatibilities

1. **Response size handling** — capped and spilled instead of unbounded (`research/limitations.md` §2.1).
2. **History** — a real persisted history instead of an in-memory timeline.
3. **DOM-dependent features** — docs editor, HTML/PDF preview, devtools/terminal.

### 8.2 Compatibility risk ranking

Highest-risk P0/P1 items, in order:

1. `.bru` round-trip parsing (grammar edge cases; 30 upstream specs as corpus).
2. Multipart body construction (`@file`/`@contentType` annotations).
3. Interpolation recursion and typed values.
4. Header merge across collection → folder → request (UQ-B1).
5. Script API surface breadth (UQ-B5).
6. Client-certificate precedence with proxy (UQ-B8).

Source (inventory basis): `packages/*/src` enumeration, `tests/**` spec directory names, and the per-module analyses in `docs/logic/**` and `docs/flows/**`.
