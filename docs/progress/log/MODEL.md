# MODEL — Domain types

Owner history: kilo (claimed 2026-11-10T20:33Z)

## Entries

### 2026-11-10T20:33Z — kilo
- Did: Claimed `MODEL`. Inspecting existing `internal/core/model/**` (enums.go present, vet reports redeclared field) and package requirements.
- Tests: not run yet.
- Notes: Must define/align domain types (Collection, Item, Request, Body union, Auth, Result, Timeline, BodyRef, Settings, Environment, TestResult, AssertionResult), JSON round-trip, pure data (no engine imports).
