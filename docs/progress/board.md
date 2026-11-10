# Progress Board

**Edit only your own package's row.** Do not re-sort, re-number or reformat this table — the stable row order is what keeps concurrent edits conflict-free.

States: `Ready` · `Claimed` · `InProgress` · `Blocked` · `InReview` · `Done`

`Claimed (UTC)` and `Updated (UTC)` use `YYYY-MM-DDTHH:MMZ`. Fill them yourself.

Package definitions: [`packages.md`](packages.md). File ownership: [`ownership.md`](ownership.md).

---

## Phase 0

| Package    | State | Agent | Claimed (UTC)     | Updated (UTC)     | Notes                                         |
| ---------- | ----- | ----- | ----------------- | ----------------- | --------------------------------------------- |
| `INFRA`    | Done  | kilo  | 2026-11-10T20:16Z | 2026-11-10T22:30Z | Confirmed: CI, lint, dep-guard, version clean |
| `FIXTURES` | Done  | kilo  | 2026-11-10T20:26Z | 2026-11-10T22:30Z | Confirmed: corpus, manifests, testutil pass   |

## Phase 1

| Package          | State      | Agent       | Claimed (UTC)     | Updated (UTC)     | Notes                                                     |
| ---------------- | ---------- | ----------- | ----------------- | ----------------- | --------------------------------------------------------- |
| `MODEL`          | Done       | zed-agent-1 | 2026-11-10T21:35Z | 2026-11-10T22:30Z | Confirmed: builds, vet-clean, 72.6 % coverage, race-clean |
| `EVENTS`         | Done       | zed-agent-1 | 2026-11-10T22:00Z | 2026-11-10T22:30Z | Confirmed: builds, vet-clean, 100 % coverage, race-clean  |
| `BUS`            | InReview   | zed-agent-1 | 2026-11-10T22:45Z | 2026-11-10T23:10Z | Builds, vet-clean, 78 % coverage, race-clean (-count=5)   |
| `FORMAT-CODE`    | InReview   | antigravity | 2026-11-10T22:30Z | 2026-11-10T22:40Z | Builds, vet-clean, race-clean, 98.4 % coverage            |
| `FORMAT-BRU`     | InProgress | zed-agent-1 | 2026-11-10T23:15Z | 2026-11-10T23:15Z | Claimed; parser + writer for the .bru dialect             |
| `FORMAT-YAML`    | Ready      | —           | —                 | —                 |                                                           |
| `VARS`           | InProgress | kilo        | 2026-11-10T21:08Z | 2026-11-10T21:08Z | Scopes, precedence, interpolation                         |
| `TPL`            | Ready      | —           | —                 | —                 | Optional; confirm need first                              |
| `COOKIES`        | Claimed    | mimo-flash  | 2026-11-10T21:06Z | 2026-11-10T21:06Z | Cookie jar, trustworthy-origin policy, persistence hook   |
| `PERSIST`        | InReview   | antigravity | 2026-11-10T22:45Z | 2026-11-10T23:20Z | Builds, vet-clean, race-clean, 80.5 % coverage            |
| `COLLECTION`     | Ready      | —           | —                 | —                 |                                                           |
| `BUILD`          | Ready      | —           | —                 | —                 |                                                           |
| `PROXY`          | InReview   | antigravity | 2026-11-10T23:30Z | 2026-11-10T23:55Z | Builds, vet-clean, race-clean, 82.0 % coverage            |
| `TLS`            | Ready      | —           | —                 | —                 |                                                           |
| `AUTH-CORE`      | Ready      | —           | —                 | —                 |                                                           |
| `TRANSPORT`      | Ready      | —           | —                 | —                 |                                                           |
| `ENGINE`         | Ready      | —           | —                 | —                 |                                                           |
| `CMD-SEND`       | Ready      | —           | —                 | —                 |                                                           |
| `CMD-COLLECTION` | Ready      | —           | —                 | —                 |                                                           |
| `CMD-ENV`        | Ready      | —           | —                 | —                 |                                                           |
| `CMD-SETTINGS`   | Ready      | —           | —                 | —                 |                                                           |
| `CMD-RUNNER`     | Ready      | —           | —                 | —                 | Command layer for the runner; `ADV-RUNNER` builds on it   |
| `CLI`            | Ready      | —           | —                 | —                 | Proves the core headlessly                                |

## Phase 2

| Package           | State | Agent | Claimed (UTC) | Updated (UTC) | Notes                            |
| ----------------- | ----- | ----- | ------------- | ------------- | -------------------------------- |
| `UI-THEME`        | Ready | —     | —             | —             |                                  |
| `UI-LAYOUT`       | Ready | —     | —             | —             |                                  |
| `UI-WIDGETS`      | Ready | —     | —             | —             | Single owner; may need splitting |
| `UI-PRESENT-REQ`  | Ready | —     | —             | —             |                                  |
| `UI-PRESENT-NAV`  | Ready | —     | —             | —             |                                  |
| `UI-PRESENT-SET`  | Ready | —     | —             | —             |                                  |
| `UI-SCREEN-REQ`   | Ready | —     | —             | —             |                                  |
| `UI-SCREEN-RESP`  | Ready | —     | —             | —             |                                  |
| `UI-SCREEN-NAV`   | Ready | —     | —             | —             |                                  |
| `UI-SCREEN-SHELL` | Ready | —     | —             | —             |                                  |
| `UI-SCREEN-SET`   | Ready | —     | —             | —             |                                  |
| `UI-DIALOGS`      | Ready | —     | —             | —             |                                  |
| `UI-APP`          | Ready | —     | —             | —             |                                  |
| `PLATFORM`        | Ready | —     | —             | —             |                                  |

## Phase 4

| Package         | State | Agent | Claimed (UTC) | Updated (UTC) | Notes            |
| --------------- | ----- | ----- | ------------- | ------------- | ---------------- |
| `AUTH-DIGEST`   | Ready | —     | —             | —             |                  |
| `AUTH-NTLM`     | Ready | —     | —             | —             |                  |
| `AUTH-OAUTH1`   | Ready | —     | —             | —             |                  |
| `AUTH-OAUTH2`   | Ready | —     | —             | —             | Needs `PLATFORM` |
| `AUTH-AWSV4`    | Ready | —     | —             | —             |                  |
| `AUTH-EDGEGRID` | Ready | —     | —             | —             |                  |
| `ENV-FILES`     | Ready | —     | —             | —             |                  |

## Phase 5

| Package          | State | Agent | Claimed (UTC) | Updated (UTC) | Notes |
| ---------------- | ----- | ----- | ------------- | ------------- | ----- |
| `SCRIPT-RUNTIME` | Ready | —     | —             | —             |       |
| `SCRIPT-BRU-API` | Ready | —     | —             | —             |       |
| `SCRIPT-POST`    | Ready | —     | —             | —             |       |

## Phase 6

| Package         | State | Agent | Claimed (UTC) | Updated (UTC) | Notes |
| --------------- | ----- | ----- | ------------- | ------------- | ----- |
| `CONV-POSTMAN`  | Ready | —     | —             | —             |       |
| `CONV-INSOMNIA` | Ready | —     | —             | —             |       |
| `CONV-OPENAPI`  | Ready | —     | —             | —             |       |
| `CONV-CURL`     | Ready | —     | —             | —             |       |
| `CODEGEN`       | Ready | —     | —             | —             |       |

## Phase 7

| Package          | State | Agent | Claimed (UTC) | Updated (UTC) | Notes                       |
| ---------------- | ----- | ----- | ------------- | ------------- | --------------------------- |
| `ADV-STREAM`     | Ready | —     | —             | —             |                             |
| `ADV-RUNNER`     | Ready | —     | —             | —             |                             |
| `ADV-GRPC`       | Ready | —     | —             | —             |                             |
| `ADV-WS`         | Ready | —     | —             | —             |                             |
| `ADV-HISTORY`    | Ready | —     | —             | —             | Blocked on `UQ-D3` decision |
| `ADV-EXTENSIONS` | Ready | —     | —             | —             |                             |

---

## Parallel-safe starting sets

These groups contain **no shared file paths and no unmet dependencies**, so every package in a group can be claimed at the same moment.

**Wave 0** (immediately): `INFRA`, `FIXTURES`

**Wave 1**: `MODEL`

**Wave 2** (after `MODEL`): `EVENTS`, `FORMAT-CODE`, `VARS`, `COOKIES`, `PERSIST`, `PROXY`, `TLS`

**Wave 3**: `BUS`, `FORMAT-BRU`, `FORMAT-YAML`, `VARS`·consumers, `COLLECTION`, `AUTH-CORE`, `TPL`

**Wave 4**: `BUILD`, `TRANSPORT`, `ENGINE`

**Wave 5**: `CMD-SEND`, `CMD-COLLECTION`, `CMD-ENV`, `CMD-SETTINGS`, `CLI`

**Wave 6** (GUI, can start as soon as `MODEL` is `Done`): `UI-THEME` → `UI-LAYOUT` → `UI-WIDGETS`, in that order; the three `UI-PRESENT-*` packages can start right after `EVENTS`+`BUS` and run **in parallel with all of Phase 1's remaining work**, because they touch no core files.

**Wave 7** (after GUI screens exist): `UI-SCREEN-*` in parallel, then `UI-APP`.

**Wave 8**: `AUTH-*` providers (five packages, fully parallel), `ENV-FILES`, `SCRIPT-*` (sequential among themselves).

**Wave 9**: `CONV-*` in parallel, `CODEGEN`, then `ADV-*`.
