# Progress Board

**Edit only your own package's row.** Do not re-sort, re-number or reformat this table — the stable row order is what keeps concurrent edits conflict-free.

States: `Ready` · `Claimed` · `InProgress` · `Blocked` · `InReview` · `Done` · `Deferred`

`Deferred` means deliberately out of v1 scope (see `decisions.md` D-011), not unblocked work. Do not claim a deferred package without a new decision entry superseding it.

`Claimed (UTC)` and `Updated (UTC)` use `YYYY-MM-DDTHH:MMZ`. Fill them yourself.

Package definitions: [`packages.md`](packages.md). File ownership: [`ownership.md`](ownership.md).

---

## Phase 0

| Package    | State | Agent | Claimed (UTC)     | Updated (UTC)     | Notes                                         |
| ---------- | ----- | ----- | ----------------- | ----------------- | --------------------------------------------- |
| `INFRA`    | Done  | kilo  | 2026-11-10T20:16Z | 2026-11-10T22:30Z | Confirmed: CI, lint, dep-guard, version clean |
| `FIXTURES` | Done  | kilo  | 2026-11-10T20:26Z | 2026-11-10T22:30Z | Confirmed: corpus, manifests, testutil pass   |

## Phase 1

| Package          | State | Agent         | Claimed (UTC)     | Updated (UTC)     | Notes                                                                                               |
| ---------------- | ----- | ------------- | ----------------- | ----------------- | --------------------------------------------------------------------------------------------------- |
| `MODEL`          | Done  | zed-agent-1   | 2026-11-10T21:35Z | 2026-11-10T22:30Z | Confirmed: builds, vet-clean, 72.6 % coverage, race-clean                                           |
| `EVENTS`         | Done  | zed-agent-1   | 2026-11-10T22:00Z | 2026-11-10T22:30Z | Confirmed: builds, vet-clean, 100 % coverage, race-clean                                            |
| `BUS`            | Done  | zed-agent-1   | 2026-11-10T22:45Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 77.1 % cov, race-clean; review sweep                                   |
| `FORMAT-CODE`    | Done  | antigravity   | 2026-11-10T22:30Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 98.4 % cov, race-clean; review sweep                                   |
| `FORMAT-BRU`     | Done  | zed-agent-1   | 2026-11-10T23:15Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 77.2 % cov, 357/357 parse, 335/335 round-trip; review sweep            |
| `FORMAT-YAML`    | Done  | zed-agent-1   | 2026-11-11T00:30Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 86.5 % cov, 64/64 fixtures; review sweep                               |
| `VARS`           | Done  | kilo          | 2026-11-10T21:08Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 84.6 % cov, race-clean; B-014 closed                                   |
| `TPL`            | Done  | zed-agent-1   | 2026-11-11T02:25Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 89.5 % cov, zero goja dep; review sweep                                |
| `COOKIES`        | Done  | zed-agent-1   | 2026-11-10T21:06Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 83.4 % cov; committed under B-021                                      |
| `PERSIST`        | Done  | antigravity   | 2026-11-10T22:45Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 80.5 % cov, race-clean; review sweep                                   |
| `COLLECTION`     | Done  | antigravity   | 2026-11-10T23:28Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 84.1 % cov; review sweep                                               |
| `BUILD`          | Done  | zed-agent-1   | 2026-11-10T23:59Z | 2026-11-11T15:25Z | Duplicate Authorization fixed: auth headers replace same-named user headers; 87.0 % cov             |
| `PROXY`          | Done  | zed-agent-1   | 2026-11-11T13:45Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 91.5 % cov; B-011 closed, PAC implemented                              |
| `TLS`            | Done  | antigravity   | 2026-11-10T23:13Z | 2026-11-11T01:17Z | Confirmed: 4-way precedence + live mTLS; gates green; 85.4 % cov                                    |
| `AUTH-CORE`      | Done  | zed-agent-1   | 2026-11-11T01:10Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 89.9 % cov (package 90.9 %); review sweep                              |
| `TRANSPORT`      | Done  | zed-agent-1   | 2026-11-11T01:45Z | 2026-11-11T02:52Z | Confirmed: B-012 fixed; phase+cap tests green, gates green; 86.1 % cov                              |
| `ENGINE`         | Done  | antigravity-2 | 2026-11-11T00:20Z | 2026-11-11T02:58Z | Confirmed: B-013 fixed, E2E CA green; race-clean; 81.8 % cov                                        |
| `CMD-SEND`       | Done  | antigravity   | 2026-11-11T00:45Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 83.4 % cov, terminal invariant verified; review sweep                  |
| `CMD-COLLECTION` | Done  | antigravity-2 | 2026-11-11T00:05Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 85.7 % cov. Nit: move_item.go:79 ignores SaveItem error                |
| `CMD-ENV`        | Done  | antigravity-2 | 2026-11-11T00:52Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 88.5 % cov. Open item withdrawn: SuppressPath present (save.go:94)     |
| `CMD-SETTINGS`   | Done  | antigravity-2 | 2026-11-11T01:02Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 85.9 % cov. Nit: future schema version downgraded (preferences.go:149) |
| `CMD-RUNNER`     | Done  | antigravity   | 2026-11-11T01:08Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 83.5 % cov, tags/control/bail verified; review sweep                   |
| `CLI`            | Done  | antigravity   | 2026-11-11T00:55Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 82.8 % cov, zero GUI deps; review sweep                                |

## Phase 2

| Package           | State   | Agent         | Claimed (UTC)     | Updated (UTC)     | Notes                                                                                                |
| ----------------- | ------- | ------------- | ----------------- | ----------------- | ---------------------------------------------------------------------------------------------------- |
| `UI-THEME`        | Claimed | zed-design    | 2026-11-11T04:24Z | 2026-11-11T04:24Z | D-017: Graphite Console world — repaint tokens; fix dark-mode token plumbing under live frame themes |
| `UI-MAP`          | Claimed | zed-design    | 2026-11-11T16:30Z | 2026-11-11T19:05Z | Bruno UI reconstruction spec in `ui-map/` (D-020); ownership + checker (D-021)                       |
| `UI-LAYOUT`       | Done    | mimo-flash-2  | 2026-11-11T00:19Z | 2026-11-11T04:35Z | Confirmed: 97.1 % cov, DPI 100-200 % verified, View normalizes, B-017 resolved, race-clean           |
| `UI-WIDGETS`      | Done    | antigravity   | 2026-11-11T03:10Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 94.2 % cov, all 8 widgets + CodeEditor; review sweep                    |
| `UI-PRESENT-REQ`  | Done    | antigravity   | 2026-11-11T01:55Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 95.6 % cov (request/response/tabs); review sweep                        |
| `UI-PRESENT-NAV`  | Done    | antigravity   | 2026-11-11T02:10Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 92.4 % cov (sidebar/collection/environment); review sweep               |
| `UI-PRESENT-SET`  | Done    | antigravity   | 2026-11-11T02:25Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 97.7 % cov (preferences/settings); review sweep                         |
| `UI-SCREEN-REQ`   | Done    | antigravity   | 2026-11-11T03:47Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 87.0 % cov, race-clean; review sweep                                    |
| `UI-SCREEN-RESP`  | Done    | zed-agent-1   | 2026-11-11T11:25Z | 2026-11-11T04:35Z | Confirmed: 89.8 % cov, pure view of ResponsePresenter, 4 panes verified, race-clean                  |
| `UI-SCREEN-NAV`   | Done    | antigravity-2 | 2026-11-11T04:05Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 96.0 % cov, race-clean; review sweep                                    |
| `UI-SCREEN-SHELL` | Done    | mimo-flash-2  | 2026-11-11T02:43Z | 2026-11-11T04:35Z | Confirmed: 100.0 % cov, race×5 clean, pure view + Actions, goldens pass                              |
| `UI-SCREEN-SET`   | Done    | antigravity   | 2026-11-11T04:09Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 95.0 % cov, race-clean; review sweep                                    |
| `UI-DIALOGS`      | Done    | zed-agent-1   | 2026-11-11T12:10Z | 2026-11-11T04:35Z | Confirmed: 91.6 % cov, queue close-on-success, prompt vars, import flows pass                        |
| `UI-APP`          | Done    | antigravity   | 2026-11-11T04:30Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 83.4 % cov, race-clean; review sweep                                    |
| `PLATFORM`        | Done    | zed-agent-1   | 2026-11-11T09:10Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 89.7 % cov, six OS interfaces + fakes; review sweep                     |

## Phase 4

| Package         | State | Agent       | Claimed (UTC)     | Updated (UTC)     | Notes                                                                              |
| --------------- | ----- | ----------- | ----------------- | ----------------- | ---------------------------------------------------------------------------------- |
| `AUTH-DIGEST`   | Done  | zed-agent-1 | 2026-11-11T02:55Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 85.7 % cov, e2e challenge; review sweep               |
| `AUTH-NTLM`     | Done  | zed-agent-1 | 2026-11-11T05:15Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 89.7 % cov, MS-NLMP vectors exact; B-006 closed       |
| `AUTH-OAUTH1`   | Done  | zed-agent-1 | 2026-11-11T04:20Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 91.8 % cov, RFC 5849; review sweep                    |
| `AUTH-OAUTH2`   | Done  | zed-agent-1 | 2026-11-11T09:40Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 89.0 % cov, 4 grants + PKCE + cache; review sweep     |
| `AUTH-AWSV4`    | Done  | zed-agent-1 | 2026-11-11T03:40Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 91.5 % cov, AWS documented vector exact; review sweep |
| `AUTH-EDGEGRID` | Done  | zed-agent-1 | 2026-11-11T06:25Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 92.1 % cov, EG1-HMAC-SHA256; review sweep             |
| `ENV-FILES`     | Done  | zed-agent-1 | 2026-11-11T05:50Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 91.2 % cov, extends chains + cycles; review sweep     |

## Phase 5

| Package          | State | Agent       | Claimed (UTC)     | Updated (UTC)     | Notes                                                                            |
| ---------------- | ----- | ----------- | ----------------- | ----------------- | -------------------------------------------------------------------------------- |
| `SCRIPT-RUNTIME` | Done  | zed-agent-1 | 2026-11-11T06:55Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 84.3 % cov, interrupt + promise await; review sweep |
| `SCRIPT-BRU-API` | Done  | zed-agent-1 | 2026-11-11T07:20Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 84.3 % cov, bru/req/res/test/expect; review sweep   |
| `SCRIPT-POST`    | Done  | zed-agent-1 | 2026-11-11T08:10Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 96.5 % cov, 28 operators; review sweep              |

## Phase 6

| Package         | State | Agent       | Claimed (UTC)     | Updated (UTC)     | Notes                                                                           |
| --------------- | ----- | ----------- | ----------------- | ----------------- | ------------------------------------------------------------------------------- |
| `CONV-POSTMAN`  | Done  | antigravity | 2026-11-11T01:38Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 82.2 % cov, v2/v2.1; review sweep                  |
| `CONV-INSOMNIA` | Done  | antigravity | 2026-11-11T01:45Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 83.8 % cov, v4/v5 JSON/YAML; review sweep          |
| `CONV-OPENAPI`  | Done  | antigravity | 2026-11-11T01:50Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 94.1 % cov, OpenAPI 3/Swagger 2; review sweep      |
| `CONV-CURL`     | Done  | antigravity | 2026-11-11T01:18Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 87.6 % cov, import+export+round-trip; review sweep |
| `CODEGEN`       | Done  | antigravity | 2026-11-11T01:30Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 87.5 % cov, cURL/Go/JS/Python; review sweep        |

## Phase 7

| Package          | State    | Agent       | Claimed (UTC)     | Updated (UTC)     | Notes                                                                               |
| ---------------- | -------- | ----------- | ----------------- | ----------------- | ----------------------------------------------------------------------------------- |
| `ADV-STREAM`     | Done     | zed-agent-1 | 2026-11-11T10:50Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 84.9 % cov; B-018 closed, post-cancel fragment dropped |
| `ADV-RUNNER`     | InReview | antigravity | 2026-11-11T02:35Z | 2026-11-11T15:00Z | Held 2026-11-11T15:00Z: reviewed now (not owner-only). JUnit/HTML reports absent    |
| `ADV-GRPC`       | Deferred | —           | —                 | —                 | Out of v1 (D-011)                                                                   |
| `ADV-WS`         | Deferred | —           | —                 | —                 | Out of v1 (D-011)                                                                   |
| `ADV-HISTORY`    | InReview | zed-agent-1 | 2026-11-11T08:50Z | 2026-11-11T15:00Z | Held 2026-11-11T15:00Z: store done (20 tests, D-011); UI half not written           |
| `ADV-EXTENSIONS` | Done     | antigravity | 2026-11-11T02:50Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 83.3 % cov; each registry has an internal impl         |

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
