# Package Structure

Status: **PROPOSED**. Constraint: Go-only application (GUI included) with **MyGo** (`github.com/egoist/mygo`, native `ui` package) as the GUI toolkit; core GUI-independent (see `implementation/go-architecture.md`, `implementation/dependency-recommendations.md`).

Layout follows the standard Go project pattern: `cmd/` for binaries, `internal/` for non-exported code, small interfaces at consumer sites.

---

## 1. Tree

```text
fearless/
├── go.mod
├── cmd/
│   ├── fearless/                 # GUI binary: main() wires presenter + toolkit
│   │   └── main.go
│   └── fearless-cli/             # headless runner (parity with `bru run`)
│       └── main.go
├── internal/
│   ├── core/                     # GUI-FREE. no toolkit imports, no image rendering
│   │   ├── model/
│   │   │   ├── collection.go     # Collection, Folder, Item, ordering
│   │   │   ├── request.go        # Request, Body, KeyValue, Param, Settings
│   │   │   ├── auth.go           # Auth modes + config structs
│   │   │   ├── result.go         # Result, TimelineEntry, TestResult, BodyRef
│   │   │   ├── env.go            # Environment, Variable
│   │   │   └── state.go          # WorkspaceState, Tab
│   │   ├── format/
│   │   │   ├── codec.go          # Codec interface + registry + Format enum
│   │   │   ├── bru/              # lexer.go parser.go writer.go annotations.go
│   │   │   ├── yaml/             # opencollection codec
│   │   │   └── env/              # environment file codec (bru + yml)
│   │   ├── vars/
│   │   │   ├── scope.go          # ordered scopes + provenance
│   │   │   ├── interpolate.go    # recursive resolver, visited set, JSON escaping
│   │   │   ├── faker.go          # {{$name}} mocks
│   │   │   └── prompt.go         # {{?Label}} extraction
│   │   ├── engine/
│   │   │   ├── engine.go         # Engine interface + orchestration
│   │   │   ├── build/            # merge tree, body, headers, params, url
│   │   │   ├── auth/             # provider registry + built-ins
│   │   │   ├── transport/        # client, roundtrippers, redirects
│   │   │   ├── proxy/            # manual/system/pac + no_proxy
│   │   │   ├── tls/              # CA bundle, client certs
│   │   │   ├── timeline/         # httptrace → entries
│   │   │   └── post/             # vars, scripts, assertions, tests
│   │   ├── scripting/
│   │   │   ├── runtime.go        # ScriptRuntime interface
│   │   │   ├── goja/             # engine impl + shims (bru, req, res, test)
│   │   │   └── sandbox/          # globals whitelist, timeouts, modules
│   │   ├── cookies/              # jar, trustworthiness policy, persistence
│   │   ├── oauth2/               # token service, cache, pkce, grants
│   │   ├── collection/           # loader, watcher, merge, uid mapping
│   │   ├── persistence/
│   │   │   ├── settings/         # JSON settings + migrations
│   │   │   ├── secrets/          # keyring / AES-GCM fallback
│   │   │   ├── body/             # body spool (temp files + TTL)
│   │   │   └── history/          # sqlite (optional feature)
│   │   ├── events/               # typed event definitions
│   │   └── log/                  # structured logging (no GUI)
│   ├── app/                      # GUI-FREE orchestration
│   │   ├── commands/             # SendRequest, SaveItem, ImportCollection…
│   │   └── bus/                  # event bus (buffered, per-topic)
│   ├── ui/                       # GUI-BOUND — the only place MyGo is imported
│   │   ├── app/                  # window, menu bar, keybindings → mygo.App/NewWindow/ui.View
│   │   ├── presenters/           # view-models; read events, call commands, no drawing
│   │   ├── screens/              # compose MyGo widgets per region
│   │   │   ├── shell.go          # AppTitleBar · Sidebar · Tabs · panes · status row
│   │   │   ├── request/          # 11 request sub-tabs (TextArea, Table, ui/editor)
│   │   │   ├── response/         # body viewer, headers Table, Timeline List, Tests
│   │   │   ├── sidebar/          # ui.Outline/OutlineTable or Sidebar + Tree
│   │   │   ├── preferences/      # ui.Form, Field, Switch, Select
│   │   │   └── dialogs/          # ui.Modal, AlertDialog, ContextMenu
│   │   ├── widgets/              # thin MyGo wrappers (MethodBadge, VariableEditor…)
│   │   ├── theme/                # Bruno token tree → *ui.Theme adapter
│   │   └── layout/               # Split/SplitVertical composition, region budget
│   ├── platform/                 # OS integration via MyGo + stdlib
│   │   ├── dialog/               # mygo.Dialog.Open/Save
│   │   ├── secrets/              # mygo.Secrets
│   │   ├── browser/              # mygo.Shell.OpenExternal + URL scheme callback
│   │   ├── watcher/              # mygo.WatchFiles
│   │   └── windowstate/          # mygo.WindowStateStore
│   └── convert/                  # importers/exporters (core, GUI-free)
│       ├── postman/
│       ├── insomnia/
│       ├── openapi/
│       └── curl/
├── pkg/                          # only if something must be importable externally
└── testdata/
    ├── collections/              # copied Bruno fixtures for conformance
    └── golden/                   # bru/yaml golden files
```

---

## 2. Placement rules

| Rule                                                                                    | Rationale                                                          |
| --------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| `core` and `app` never import `ui/*` or `github.com/egoist/mygo`                        | Keeps the engine GUI-independent (hard requirement)                |
| `ui` imports `app` + `core/model` (types only)                                          | Presenters may read models; screens may not touch engine internals |
| MyGo is imported **only** in `internal/ui/*`, `internal/platform/*`, and `cmd/fearless` | One place to absorb MyGo API changes, one place to swap toolkits   |
| Parsers live in `core/format`, not in `collection`                                      | Codecs are pure and reusable by the CLI                            |
| Auth providers are separate files in `engine/auth/<mode>.go` with `init()` registration | Adding a provider is one file (contrast: five places in Bruno)     |
| No `init()` side effects outside registries                                             | Predictable graph                                                  |
| `platform` isolates OS-specific code behind interfaces                                  | Testable in CI without a display/keyring                           |

---

## 3. Interface ownership

Define interfaces **where they are consumed**, not where implemented:

```text
ui/presenters  defines      Engine   (satisfied by core/engine)
ui/presenters  defines      Store    (satisfied by app state actor)
app/commands   defines      Codec, Repository, Secrets, CookieJar
core/engine    defines      ScriptRuntime, HookSet, EventSink
core/scripting defines      BruAPI, RequestProxy, ResponseProxy
```

---

## 4. Naming conventions mirroring Bruno concepts

| Bruno                         | Go                               |
| ----------------------------- | -------------------------------- |
| `Item`                        | `model.Item`                     |
| `item.draft`                  | `model.Item.Draft`               |
| `request` (runtime shape)     | `model.Request`                  |
| `axiosRequest` (compiled)     | `engine.BuiltRequest`            |
| `response` (transport result) | `model.Result`                   |
| `timeline`                    | `model.TimelineEntry`            |
| `bru.config` (bruno.json)     | `model.CollectionConfig`         |
| `presets`                     | `model.CollectionConfig.Presets` |
| `vars:pre-request`            | `model.Request.Vars.Req`         |
| `vars:post-response`          | `model.Request.Vars.Res`         |

Keeping names aligned with the source docs makes cross-referencing (`data/*.md`) unambiguous.

---

## 5. Anti-patterns to avoid

1. A `gui` package importing `net/http` directly — network always goes through `core/engine`.
2. Presenters doing file I/O — they call commands.
3. Passing `*model.Result` bodies to widgets — pass `BodyRef` and let a reader stream bytes.
4. `mygo.webview` / `ui.WebView` for application chrome — response bodies only.
5. One giant `model` file — keep the split above so changes stay reviewable.
6. Using the JS engine for app logic (theming, config parsing, validation).
