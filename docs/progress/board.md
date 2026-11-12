# Fearless — Progress Board

State machine: `Backlog` -> `Claimed` -> `InProgress` -> `InReview` -> `Done`
Rules: one agent per package; update timestamp on every transition; claim before editing.
Phase dependencies: Phase 1 blocks Phase 2/3; Phase 3 blocks Phase 4/5/6/7.

## Phase 1

| Package   | State | Agent       | Claimed (UTC)     | Updated (UTC)     | Notes                                                                             |
| --------- | ----- | ----------- | ----------------- | ----------------- | --------------------------------------------------------------------------------- |
| `INFRA`   | Done  | kilo        | 2026-11-10T22:30Z | 2026-11-10T22:45Z | Confirmed 2026-11-11T03:00Z: module tidy, pre-commit hooked, directories in place |
| `MODEL`   | Done  | zed-agent-1 | 2026-11-10T22:30Z | 2026-11-11T04:35Z | Confirmed: 94.6 % cov, all 13 core domain types implemented and tested            |
| `EVENTS`  | Done  | zed-agent-1 | 2026-11-10T22:30Z | 2026-11-11T04:35Z | Confirmed: 95.8 % cov, all 17 event types implemented and tested                  |
| `BUS`     | Done  | zed-agent-1 | 2026-11-10T22:30Z | 2026-11-11T04:35Z | Confirmed: 96.2 % cov, synchronous event bus with typed subscriptions             |
| `PERSIST` | Done  | antigravity | 2026-11-10T22:35Z | 2026-11-11T04:35Z | Confirmed: 91.7 % cov, B-004 resolved, spool + sync-on-idle + wal pass            |

## Phase 2

| Package           | State | Agent        | Claimed (UTC)     | Updated (UTC)     | Notes                                                                                                |
| ----------------- | ----- | ------------ | ----------------- | ----------------- | ---------------------------------------------------------------------------------------------------- |
| `UI-THEME`        | Done  | mimo-flash-3 | 2026-11-11T23:16Z | 2026-11-12T02:20Z | D-017/D-023 world: type system, shared tab strip (D-025), placeholder ink past 4.5:1              |
| `UI-MAP`          | Done  | zed-design   | 2026-11-11T16:30Z | 2026-11-12T01:30Z | Bruno UI reconstruction spec in `ui-map/` (D-020); 1577 citations verified, checker clean (D-021)    |
| `UI-LAYOUT`       | Done  | mimo-flash-3 | 2026-11-11T23:16Z | 2026-11-12T02:20Z | D-017 first-viewport composition: 30 DIP title strip, 248 px sidebar, env+run move up, drawn icons  |
| `UI-WIDGETS`      | Done  | antigravity  | 2026-11-11T03:10Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 94.2 % cov, all 8 widgets + CodeEditor; review sweep                    |
| `UI-PRESENT-REQ`  | Done  | antigravity  | 2026-11-11T01:55Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 95.6 % cov (request/response/tabs); review sweep                        |
| `UI-PRESENT-NAV`  | Done  | antigravity  | 2026-11-11T02:10Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 92.4 % cov (sidebar/collection/environment); review sweep               |
| `UI-PRESENT-SET`  | Done  | antigravity  | 2026-11-11T02:25Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 97.7 % cov (preferences/settings); review sweep                         |
| `UI-SCREEN-REQ`   | Done  | antigravity  | 2026-11-11T03:47Z | 2026-11-12T01:20Z | Restyled with DataGroup monospace URL, PanelHeader section titles, Send button alignment; tests pass |
| `UI-SCREEN-RESP`  | Done  | mimo-flash-3 | 2026-11-11T23:36Z | 2026-11-12T02:20Z | D-023: status seam `200 OK · 412 ms · 2.4 kB`, shared strip, mono data; B-026 open on this package    |
| `UI-SCREEN-NAV`   | Done  | mimo-flash-3 | 2026-11-11T23:36Z | 2026-11-12T02:20Z | D-023: owns `screens/navigation/**`; shared strip, drawn rail icons, quiet actions                  |
| `UI-SCREEN-SHELL` | Done  | mimo-flash-3 | 2026-11-11T23:36Z | 2026-11-12T02:20Z | D-023: title strip takes env selector + run/stop; region accessor per D-024                         |
| `UI-SCREEN-SET`   | Done  | antigravity  | 2026-11-11T04:09Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 95.0 % cov, race-clean; review sweep                                    |
| `UI-DIALOGS`      | Done  | zed-agent-1  | 2026-11-11T12:10Z | 2026-11-11T04:35Z | Confirmed: 91.6 % cov, queue close-on-success, prompt vars, import flows pass                        |
| `UI-APP`          | Done  | mimo-flash-3 | 2026-11-11T23:36Z | 2026-11-12T02:20Z | D-023: first-viewport wiring; finish review shipped, DESIGN.md and capture provenance written        |
| `PLATFORM`        | Done  | zed-agent-1  | 2026-11-11T09:10Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 89.7 % cov, six OS interfaces + fakes; review sweep                     |

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

| Package       | State | Agent       | Claimed (UTC)     | Updated (UTC)     | Notes                                                                           |
| ------------- | ----- | ----------- | ----------------- | ----------------- | ------------------------------------------------------------------------------- |
| `FORMAT-YAML` | Done  | zed-agent-1 | 2026-11-11T08:35Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 95.8 % cov, single-doc reader/writer; review sweep |
| `FORMAT-BRU`  | Done  | zed-agent-1 | 2026-11-11T09:00Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 94.7 % cov, tokeniser + AST parser; review sweep   |
| `COLLECTION`  | Done  | zed-agent-1 | 2026-11-11T09:25Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 92.1 % cov, walker/tree builder; review sweep      |
| `IMPORT-PM`   | Done  | zed-agent-1 | 2026-11-11T09:50Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 93.4 % cov, v2.1 import; review sweep              |
| `IMPORT-INS`  | Done  | zed-agent-1 | 2026-11-11T10:15Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 92.8 % cov, v4 import; review sweep                |
| `IMPORT-OAS`  | Done  | zed-agent-1 | 2026-11-11T10:40Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 91.5 % cov, OpenAPI 3.0/3.1 import; review sweep   |
| `IMPORT-CURL` | Done  | zed-agent-1 | 2026-11-11T11:05Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 94.0 % cov, POSIX shlex curl parser; review sweep  |
| `EXPORT-CODE` | Done  | zed-agent-1 | 2026-11-11T11:30Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 92.3 % cov, 10 languages code gen; review sweep    |

## Phase 7

| Package       | State | Agent                     | Claimed (UTC)     | Updated (UTC)     | Notes                                                                             |
| ------------- | ----- | ------------------------- | ----------------- | ----------------- | --------------------------------------------------------------------------------- |
| `ADV-STREAM`  | Done  | zed-agent-1 / antigravity | 2026-11-11T12:35Z | 2026-11-12T02:00Z | Engine SSE parser + progress/cancel; UI SSEStreamPane, message cards, auto-scroll |
| `ADV-RUNNER`  | Done  | antigravity               | 2026-11-11T22:30Z | 2026-11-12T00:50Z | Confirmed JUnit/HTML reports exist in `internal/app/reporter/**`; review sweep    |
| `ADV-HISTORY` | Done  | antigravity               | 2026-11-12T00:50Z | 2026-11-12T00:50Z | Full UI implemented with snapshots/actions, badges, drawer, replay/delete/clear   |
| `ADV-EXT`     | Done  | zed-agent-1               | 2026-11-11T12:35Z | 2026-11-11T15:00Z | Confirmed 2026-11-11T15:00Z: 90.5 % cov, plugin hooks + wasm loader; review sweep |
