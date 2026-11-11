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

| Package          | State    | Agent         | Claimed (UTC)     | Updated (UTC)     | Notes                                                                      |
| ---------------- | -------- | ------------- | ----------------- | ----------------- | -------------------------------------------------------------------------- |
| `MODEL`          | Done     | zed-agent-1   | 2026-11-10T21:35Z | 2026-11-10T22:30Z | Confirmed: builds, vet-clean, 72.6 % coverage, race-clean                  |
| `EVENTS`         | Done     | zed-agent-1   | 2026-11-10T22:00Z | 2026-11-10T22:30Z | Confirmed: builds, vet-clean, 100 % coverage, race-clean                   |
| `BUS`            | InReview | zed-agent-1   | 2026-11-10T22:45Z | 2026-11-10T23:10Z | Builds, vet-clean, 78 % coverage, race-clean (-count=5)                    |
| `FORMAT-CODE`    | InReview | antigravity   | 2026-11-10T22:30Z | 2026-11-10T22:40Z | Builds, vet-clean, race-clean, 98.4 % coverage                             |
| `FORMAT-BRU`     | InReview | zed-agent-1   | 2026-11-10T23:15Z | 2026-11-11T00:25Z | Parser+mapper+writer; 357/357 parse, 335/335 round-trip, 77 % cov          |
| `FORMAT-YAML`    | InReview | zed-agent-1   | 2026-11-11T00:30Z | 2026-11-11T01:05Z | 64/64 yml fixtures; 86.5 % coverage; three body shapes handled             |
| `VARS`           | InReview | kilo          | 2026-11-10T21:08Z | 2026-11-11T02:30Z | B-014 resolved: gofmt applied to scopes.go/vars_test.go; tests pass        |
| `TPL`            | InReview | zed-agent-1   | 2026-11-11T02:25Z | 2026-11-11T02:50Z | vars blocks natively; 89.5 % cov; zero goja dependency                     |
| `COOKIES`        | InReview | mimo-flash    | 2026-11-10T21:06Z | 2026-11-11T02:18Z | Reviewed: 83.3 % cov. Open: Secure over http dropped; no cookie cap        |
| `PERSIST`        | InReview | antigravity   | 2026-11-10T22:45Z | 2026-11-10T23:20Z | Builds, vet-clean, race-clean, 80.5 % coverage                             |
| `COLLECTION`     | InReview | antigravity   | 2026-11-10T23:28Z | 2026-11-10T23:55Z | Builds, vet-clean, race-clean, 83.2 % cov, 1000 reqs in 22ms               |
| `BUILD`          | InReview | antigravity-2 | 2026-11-10T23:59Z | 2026-11-11T02:18Z | Reviewed: 86.7 % cov. Open: user Authorization precedes auth's             |
| `PROXY`          | Blocked  | antigravity   | 2026-11-10T23:30Z | 2026-11-11T02:30Z | Blocked on B-011: PAC acceptance criterion, owner notified                 |
| `TLS`            | Done     | antigravity   | 2026-11-10T23:13Z | 2026-11-11T01:17Z | Confirmed: 4-way precedence + live mTLS; gates green; 85.4 % cov           |
| `AUTH-CORE`      | InReview | zed-agent-1   | 2026-11-11T01:10Z | 2026-11-11T01:40Z | Registry+none/basic/bearer/apikey/wsse+digest; 86 % cov                    |
| `TRANSPORT`      | InReview | zed-agent-1   | 2026-11-11T01:45Z | 2026-11-11T02:30Z | B-012 fixed: unique ordered phases + body-cap tests; 86.1 % cov            |
| `ENGINE`         | InReview | antigravity-2 | 2026-11-11T00:20Z | 2026-11-11T02:35Z | B-013 fixed: tls.Resolve wired, E2E CA verified, race-clean, 81.8% cov     |
| `CMD-SEND`       | InReview | antigravity   | 2026-11-11T00:45Z | 2026-11-11T00:55Z | Builds, vet-clean, race-clean, 87.4 % cov, terminal invariant verified     |
| `CMD-COLLECTION` | InReview | antigravity-2 | 2026-11-11T00:05Z | 2026-11-11T02:18Z | Reviewed: 85.7 % cov. Open: move_item.go:79 drops SaveItem error           |
| `CMD-ENV`        | InReview | antigravity-2 | 2026-11-11T00:52Z | 2026-11-11T02:18Z | Reviewed: 88.5 % cov. Open: save_environment lacks self-suppression        |
| `CMD-SETTINGS`   | InReview | antigravity-2 | 2026-11-11T01:02Z | 2026-11-11T02:18Z | Reviewed: 85.9 % cov. Open: future schema version silently downgraded      |
| `CMD-RUNNER`     | InReview | antigravity   | 2026-11-11T01:08Z | 2026-11-11T01:13Z | Builds, vet-clean, race-clean, 83.5 % cov, folder run, tags, control, bail |
| `CLI`            | InReview | antigravity   | 2026-11-11T00:55Z | 2026-11-11T01:06Z | Builds, vet-clean, race-clean, 82.8 % cov, zero GUI deps, runs corpus      |

## Phase 2

| Package           | State    | Agent        | Claimed (UTC)     | Updated (UTC)     | Notes                                                                                      |
| ----------------- | -------- | ------------ | ----------------- | ----------------- | ------------------------------------------------------------------------------------------ |
| `UI-THEME`        | InReview | mimo-flash-2 | 2026-11-11T01:10Z | 2026-11-11T02:18Z | Reviewed: 97.3 % cov, 7-verb x light/dark table. Open: mode match is case-sensitive        |
| `UI-LAYOUT`       | InReview | mimo-flash-2 | 2026-11-11T00:19Z | 2026-11-11T02:40Z | Open: B-017 vet %d-vs-float32 in layout_test.go, plus Normalize                            |
| `UI-WIDGETS`      | InReview | antigravity  | 2026-11-11T03:10Z | 2026-11-11T03:38Z | Custom widgets (all 8 + CodeEditor); 94.2% cov, race-clean, dep-clean                      |
| `UI-PRESENT-REQ`  | InReview | antigravity  | 2026-11-11T01:55Z | 2026-11-11T02:02Z | Pure Go view-models for request, response, tabs; 95.6% cov, race-clean, dep-clean          |
| `UI-PRESENT-NAV`  | InReview | antigravity  | 2026-11-11T02:10Z | 2026-11-11T02:18Z | Pure Go view-models for sidebar, collection, environment; 93.8% cov, race-clean, dep-clean |
| `UI-PRESENT-SET`  | InReview | antigravity  | 2026-11-11T02:25Z | 2026-11-11T02:27Z | Pure Go view-models for preferences and settings; 94.8% cov, race-clean, dep-clean         |
| `UI-SCREEN-REQ`   | Ready    | —            | —                 | —                 |                                                                                            |
| `UI-SCREEN-RESP`  | Ready    | —            | —                 | —                 |                                                                                            |
| `UI-SCREEN-NAV`   | Ready    | —            | —                 | —                 |                                                                                            |
| `UI-SCREEN-SHELL` | Ready    | —            | —                 | —                 |                                                                                            |
| `UI-SCREEN-SET`   | Ready    | —            | —                 | —                 |                                                                                            |
| `UI-DIALOGS`      | Ready    | —            | —                 | —                 |                                                                                            |
| `UI-APP`          | Ready    | —            | —                 | —                 |                                                                                            |
| `PLATFORM`        | InReview | zed-agent-1  | 2026-11-11T09:10Z | 2026-11-11T09:35Z | Six OS interfaces + fakes, real loopback listener; 89.7 % cov                              |

## Phase 4

| Package         | State      | Agent       | Claimed (UTC)     | Updated (UTC)     | Notes                                                                                       |
| --------------- | ---------- | ----------- | ----------------- | ----------------- | ------------------------------------------------------------------------------------------- |
| `AUTH-DIGEST`   | InReview   | zed-agent-1 | 2026-11-11T02:55Z | 2026-11-11T03:05Z | Wired into Apply; end-to-end challenge test                                                 |
| `AUTH-NTLM`     | InReview   | zed-agent-1 | 2026-11-11T05:15Z | 2026-11-11T05:45Z | NTLMv2 per MS-NLMP; §4.2 vectors exact; e2e blocked on B-006                                |
| `AUTH-OAUTH1`   | InReview   | zed-agent-1 | 2026-11-11T04:20Z | 2026-11-11T05:10Z | RFC 5849 signing; openssl-verified base string; header/query/body placement, body hash, RSA |
| `AUTH-OAUTH2`   | InReview   | zed-agent-1 | 2026-11-11T09:40Z | 2026-11-11T10:20Z | 4 grants, PKCE, cache+refresh, placement; 3 bugs found; 89 % cov                            |
| `AUTH-AWSV4`    | InProgress | zed-agent-1 | 2026-11-11T03:40Z | 2026-11-11T03:40Z | SigV4 against AWS golden vectors                                                            |
| `AUTH-EDGEGRID` | InReview   | zed-agent-1 | 2026-11-11T06:25Z | 2026-11-11T06:50Z | EG1-HMAC-SHA256; all intermediates recomputed independently; no live-gateway vector         |
| `ENV-FILES`     | InReview   | zed-agent-1 | 2026-11-11T05:50Z | 2026-11-11T06:20Z | `extends` chains w/ cycle detection, `.env`, path-safe loader; 91.2 % cov                   |

## Phase 5

| Package          | State    | Agent       | Claimed (UTC)     | Updated (UTC)     | Notes                                                                 |
| ---------------- | -------- | ----------- | ----------------- | ----------------- | --------------------------------------------------------------------- |
| `SCRIPT-RUNTIME` | InReview | zed-agent-1 | 2026-11-11T06:55Z | 2026-11-11T07:15Z | goja sandbox; interrupt+promise await; inbuilt-lib gap logged (UQ-B5) |
| `SCRIPT-BRU-API` | InReview | zed-agent-1 | 2026-11-11T07:20Z | 2026-11-11T08:05Z | bru/req/res/test/expect; 2 chai gaps logged; 84.3 % cov               |
| `SCRIPT-POST`    | InReview | zed-agent-1 | 2026-11-11T08:10Z | 2026-11-11T08:45Z | 28-operator assertions, partial results; 96.5 % cov                   |

## Phase 6

| Package         | State    | Agent       | Claimed (UTC)     | Updated (UTC)     | Notes                                                                        |
| --------------- | -------- | ----------- | ----------------- | ----------------- | ---------------------------------------------------------------------------- |
| `CONV-POSTMAN`  | InReview | antigravity | 2026-11-11T01:38Z | 2026-11-11T01:43Z | Builds, vet-clean, race-clean, 82.2 % cov, v2/v2.1 import+export+env         |
| `CONV-INSOMNIA` | InReview | antigravity | 2026-11-11T01:45Z | 2026-11-11T01:48Z | Builds, vet-clean, race-clean, 83.8 % cov, v4/v5 JSON/YAML import+export     |
| `CONV-OPENAPI`  | InReview | antigravity | 2026-11-11T01:50Z | 2026-11-11T01:52Z | Builds, vet-clean, race-clean, 94.1 % cov, OpenAPI 3/Swagger 2 import+export |
| `CONV-CURL`     | InReview | antigravity | 2026-11-11T01:18Z | 2026-11-11T01:25Z | Builds, vet-clean, race-clean, 87.6 % cov, import+export+round-trip          |
| `CODEGEN`       | InReview | antigravity | 2026-11-11T01:30Z | 2026-11-11T01:35Z | Builds, vet-clean, race-clean, 87.5 % cov, cURL/Go/JS/Python generators      |

## Phase 7

| Package          | State      | Agent       | Claimed (UTC)     | Updated (UTC)     | Notes                                                                |
| ---------------- | ---------- | ----------- | ----------------- | ----------------- | -------------------------------------------------------------------- |
| `ADV-STREAM`     | InProgress | zed-agent-1 | 2026-11-11T10:50Z | 2026-11-11T02:40Z | Open: B-016 wrong package clause                                     |
| `ADV-RUNNER`     | InReview   | antigravity | 2026-11-11T02:35Z | 2026-11-11T02:18Z | Owner-reported only; not reviewed                                    |
| `ADV-GRPC`       | Deferred   | —           | —                 | —                 | Out of v1 (D-011)                                                    |
| `ADV-WS`         | Deferred   | —           | —                 | —                 | Out of v1 (D-011)                                                    |
| `ADV-HISTORY`    | InReview   | zed-agent-1 | 2026-11-11T08:50Z | 2026-11-11T09:05Z | JSONL store per D-011; UI half blocked on UI-SCREEN-RESP; 87.5 % cov |
| `ADV-EXTENSIONS` | InReview   | antigravity | 2026-11-11T02:50Z | 2026-11-11T02:18Z | Reviewed: 83.3 % cov. Open: unwired                                  |

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
