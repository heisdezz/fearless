# Ownership Map — File Paths per Work Package

**This file is the anti-collision contract.** Each row owns a disjoint set of paths under `module fearless` .

Rules:

1. If a path is listed in a package's row, **only that package's owner writes it**.
2. Paths are disjoint by construction — verify before you start; do not "just add one file" to another owner's tree.
3. `cmd/` and `internal/ui/` rows are split finely on purpose (see §4) because they are the busiest areas.
4. Test files belong to the package that owns the code under test: `foo_test.go` → the owner of `foo.go`.
5. Shared fixtures are owned by exactly one package; others **read** them and must not modify them.
6. Root files (`go.mod`, `go.sum`, `.golangci.yml`, CI workflows) are owned by `INFRA`.

---

## 1. Legend

| Symbol | Meaning                                               |
| ------ | ----------------------------------------------------- |
| ✔      | Package owns this path outright                       |
| 👁     | Package may **read** this path but must not modify it |
| —      | Not this package's business                           |

---

## 2. Foundation and shared code (Phase 1)

| Package          | Owned paths                                                                                                                                                         |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `INFRA`          | `go.mod`, `go.sum`, `go.work`, `.golangci.yml`, `Makefile`, `Taskfile.yml`, `.github/**`, `.editorconfig`, `scripts/**`, `LICENSE`, `NOTICE`, `internal/version/**` |
| `MODEL`          | `internal/core/model/**`                                                                                                                                            |
| `EVENTS`         | `internal/core/events/**`                                                                                                                                           |
| `BUS`            | `internal/app/bus/**`                                                                                                                                               |
| `FORMAT-BRU`     | `internal/core/format/bru/**`                                                                                                                                       |
| `FORMAT-CODE`    | `internal/core/format/codec.go`, `internal/core/format/registry.go`, `internal/core/format/detect*.go`                                                              |
| `FORMAT-YAML`    | `internal/core/format/yaml/**`                                                                                                                                      |
| `VARS`           | `internal/core/vars/**`                                                                                                                                             |
| `TPL`            | `internal/core/template/**`                                                                                                                                         |
| `BUILD`          | `internal/core/engine/build/**`                                                                                                                                     |
| `TRANSPORT`      | `internal/core/engine/transport/**`                                                                                                                                 |
| `PROXY`          | `internal/core/engine/proxy/**`                                                                                                                                     |
| `TLS`            | `internal/core/engine/tls/**`                                                                                                                                       |
| `COOKIES`        | `internal/core/cookies/**`                                                                                                                                          |
| `AUTH-CORE`      | `internal/core/engine/auth/registry.go`, `auth/types.go`, `auth/none.go`, `auth/basic.go`, `auth/bearer.go`, `auth/apikey.go`, `auth/wsse.go`                       |
| `ENGINE`         | `internal/core/engine/engine.go`, `engine/result.go`, `engine/sink.go`, `engine/pipeline*.go`                                                                       |
| `PERSIST`        | `internal/core/persistence/**`                                                                                                                                      |
| `COLLECTION`     | `internal/core/collection/**`                                                                                                                                       |
| `CMD-SEND`       | `internal/app/commands/send_request.go`, `cancel_request.go`, `send_request_test.go`, `cancel_request_test.go`                                                      |
| `CMD-COLLECTION` | `internal/app/commands/collection*.go`, `save_item.go`, `delete_item.go`, `move_item.go`, `new_item.go`, `reload_collection.go`, `load_collection.go`               |
| `CMD-ENV`        | `internal/app/commands/environment*.go`, `select_environment.go`, `save_environment.go`                                                                             |
| `CMD-SETTINGS`   | `internal/app/commands/preferences*.go`, `save_preferences.go`, `oauth_cache.go`                                                                                    |
| `CMD-RUNNER`     | `internal/app/commands/runner*.go`, `run_folder.go`                                                                                                                 |
| `CLI`            | `cmd/fearless-cli/**`                                                                                                                                               |
| `FIXTURES`       | `testdata/**`, `internal/testdata/**`, `internal/testutil/**`                                                                                                       |

---

## 3. GUI layer (Phase 2+)

Split so that no two UI packages share a file. The rule of thumb: **one screen package per screen**, plus shared widget and presenter trees.

| Package           | Owned paths                                                                                                             |
| ----------------- | ----------------------------------------------------------------------------------------------------------------------- |
| `UI-THEME`        | `internal/ui/theme/**`                                                                                                  |
| `UI-LAYOUT`       | `internal/ui/layout/**`                                                                                                 |
| `UI-WIDGETS`      | `internal/ui/widgets/**` (MethodBadge, key-value grid cells, tree rows, variable input, tab bar)                        |
| `UI-APP`          | `internal/ui/app/**` (window, menu bar, keybinding router, root view)                                                   |
| `UI-PRESENT-REQ`  | `internal/ui/presenters/request*.go`, `internal/ui/presenters/response*.go`, `internal/ui/presenters/tabs*.go`          |
| `UI-PRESENT-NAV`  | `internal/ui/presenters/sidebar*.go`, `internal/ui/presenters/collection*.go`, `internal/ui/presenters/environment*.go` |
| `UI-PRESENT-SET`  | `internal/ui/presenters/preferences*.go`, `internal/ui/presenters/settings*.go`                                         |
| `UI-SCREEN-SHELL` | `internal/ui/screens/shell/**`                                                                                          |
| `UI-SCREEN-REQ`   | `internal/ui/screens/request/**`                                                                                        |
| `UI-SCREEN-RESP`  | `internal/ui/screens/response/**`                                                                                       |
| `UI-SCREEN-NAV`   | `internal/ui/screens/navigation/**` (D-023: replaces old unmapped `screens/sidebar` and `screens/collection`)           |
| `UI-SCREEN-SET`   | `internal/ui/screens/settings/**`                                                                                       |
| `UI-DIALOGS`      | `internal/ui/dialogs/**`                                                                                                |
| `PLATFORM`        | `internal/platform/**` (`dialog/`, `secrets/`, `browser/`, `watcher/`, `windowstate/`)                                  |
| `UI-MAP`          | `ui-map/**`, `refs/**`, `scripts/check-ownership.sh`                                                                    |

---

## 4. Why `cmd/` and `internal/ui/` are split so finely

Both are high-churn and would otherwise become the collision point for every agent. The splits follow real boundaries:

- `cmd/` is split by **command type** (send / collection / env / settings / runner) because these share no state and evolve independently.
- `internal/ui/presenters` is split by **region** (request, navigation, settings) because each region has its own view-model.
- `internal/ui/screens` is split by **screen**, one package each.

`internal/ui/widgets` stays a single package on purpose: shared widgets need one owner. If it becomes a bottleneck, split it along the same lines as §3's screens — do that via `blockers.md`, not unilaterally.

---

## 5. Later-phase packages (Phases 4–7)

Each provider, converter and advanced feature lives in its own directory so these packages are parallel-safe by construction.

| Package          | Owned paths                                                                                         |
| ---------------- | --------------------------------------------------------------------------------------------------- |
| `AUTH-DIGEST`    | `internal/core/engine/auth/digest*.go`                                                              |
| `AUTH-NTLM`      | `internal/core/engine/auth/ntlm*.go`                                                                |
| `AUTH-OAUTH1`    | `internal/core/engine/auth/oauth1*.go`                                                              |
| `AUTH-OAUTH2`    | `internal/core/engine/auth/oauth2*.go`, `internal/core/oauth2/**`                                   |
| `AUTH-AWSV4`     | `internal/core/engine/auth/awsv4*.go`                                                               |
| `AUTH-EDGEGRID`  | `internal/core/engine/auth/edgegrid*.go`                                                            |
| `ENV-FILES`      | `internal/core/environment/**`                                                                      |
| `SCRIPT-RUNTIME` | `internal/core/scripting/runtime*.go`, `scripting/sandbox*.go`, `scripting/modules/**`              |
| `SCRIPT-BRU-API` | `internal/core/scripting/bru*.go`, `scripting/req*.go`, `scripting/res*.go`, `scripting/expect*.go` |
| `SCRIPT-POST`    | `internal/core/engine/post/**`                                                                      |
| `CONV-POSTMAN`   | `internal/convert/postman/**`                                                                       |
| `CONV-INSOMNIA`  | `internal/convert/insomnia/**`                                                                      |
| `CONV-OPENAPI`   | `internal/convert/openapi/**`                                                                       |
| `CONV-CURL`      | `internal/convert/curl/**`                                                                          |
| `CODEGEN`        | `internal/convert/codegen/**`                                                                       |
| `ADV-STREAM`     | `internal/core/engine/stream*.go`, `internal/ui/screens/response/stream*.go`                        |
| `ADV-RUNNER`     | `internal/runner/**`, `internal/app/reporter/**`                                                    |
| `ADV-GRPC`       | `internal/protocol/grpc/**`                                                                         |
| `ADV-WS`         | `internal/protocol/ws/**`                                                                           |
| `ADV-HISTORY`    | `internal/history/**`, `internal/ui/screens/history/**`                                             |
| `ADV-EXTENSIONS` | `internal/extension/**`                                                                             |

Note on `AUTH-CORE` vs the provider packages: `AUTH-CORE` owns `auth/registry.go`, `auth/types.go` and the five base providers. The Phase-4 packages own **only** their own `<mode>*.go` file in the same directory. This is deliberate — the registry is a shared interface, so adding a provider never edits a neighbour's file.

Similarly `SCRIPT-RUNTIME` and `SCRIPT-BRU-API` share a directory but not a file: the runtime owns the engine and sandbox, the API package owns the host functions.

---

## 6. Read-only paths

These are read by everyone and written by no one during implementation:

| Path                                                      | Owner to approach for changes                                          |
| --------------------------------------------------------- | ---------------------------------------------------------------------- |
| `docs/**` (all design documentation)                      | Nobody — these are the specification. Amend only by explicit agreement |
| `docs/progress/ownership.md`, `docs/progress/packages.md` | Nobody — the contract itself                                           |
| Generated files (`*.gen.go`)                              | The package that owns the generator                                    |

---

## 7. Verification

Before opening a PR or reporting completion, list the files you changed and confirm every one appears in your package's row:

```sh
git --no-pager diff --name-only HEAD~1 | while read -r f; do echo "$f"; done
```

Compare that list against your row above. A file that is not in your row is a contract violation — revert it and use §5 of the protocol.

---

## 8. Cross-package interfaces

The following interfaces are **defined once** and consumed by several packages. Only the defining owner changes the definition:

| Interface                                           | Defined in                               | Consumers                                       |
| --------------------------------------------------- | ---------------------------------------- | ----------------------------------------------- |
| `model.*` types                                     | `internal/core/model` (MODEL)            | everyone                                        |
| `events.Event`                                      | `internal/core/events` (EVENTS)          | BUS, presenters, commands                       |
| `engine.Engine`, `engine.Input`, `engine.EventSink` | `internal/core/engine` (ENGINE)          | CMD-SEND, CLI                                   |
| `vars.Scopes`, `vars.Interpolate`                   | `internal/core/vars` (VARS)              | BUILD, ENGINE, AUTH-CORE                        |
| `format.Codec`                                      | `internal/core/format` (FORMAT-CODE)     | COLLECTION, FORMAT-BRU, FORMAT-YAML, converters |
| `persistence.Secrets`, `BodyStore`                  | `internal/core/persistence` (PERSIST)    | ENGINE, PLATFORM                                |
| `auth.AuthProvider`                                 | `internal/core/engine/auth` (AUTH-CORE)  | AUTH-ADV, ENGINE                                |
| Presenter ports (`RequestSender`, `ItemSaver`, …)   | `internal/ui/presenters` (UI-PRESENT-\*) | UI-WIDGETS, UI-SCREEN-\*                        |

Changing one of these requires an entry in `decisions.md` naming every consumer.
