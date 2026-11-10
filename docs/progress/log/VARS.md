# VARS — Scopes, precedence, interpolation

Owner history: kilo (claimed 2026-11-10T21:08Z)

## Entries

### 2026-11-10T21:08Z — kilo
- Did: Claimed `VARS` (deps MODEL, FIXTURES Done). Reviewing interpolation/resolution specs and existing fixtures.
- Tests: not run yet.
- Notes: Scope limited to `internal/core/vars/**` plus own board/log updates.

### 2026-11-10T21:22Z — kilo
- Did: Implemented initial `internal/core/vars` package: `Interpolate` with {{$mock}} expansion, dotted/bracket `Lookup`, recursive resolution with cycle guard+memoization, JSON-string escaping mode, `Scopes` merge/prelude precedence helper, `UnresolvedCollector`, and prompt variable extraction (`{{?Label}}`) with deterministic map traversal. Added `vars_test.go` covering basic interpolation, missing/unresolved, recursion/cycle, object stringify, JSON escaping, lookup paths, prompt extraction, scope precedence/process.env, and mock timestamp.
- Tests: `CGO_ENABLED=0 go test ./internal/core/vars -count=1` -> ok.
- Notes: Partial implementation toward full Bruno `interpolate.spec.ts` parity; further cases/faker breadth and env pre-interpolation helpers may be needed. Kept changes within VARS ownership.
