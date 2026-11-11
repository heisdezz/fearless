# CMD-RUNNER — Runner command layer

Owner history: antigravity (claimed 2026-11-11T01:08Z)

## Entries

### 2026-11-11T01:08Z — antigravity

- Did: Claimed `CMD-RUNNER`. Reviewed `docs/progress/packages.md`, `docs/implementation/go-architecture.md`, `docs/implementation/interface-design.md`, and dependencies (`COLLECTION`, `BUS`, `PERSIST`).
- Tests: not run yet.
- Notes: Implementing the runner command layer in `internal/app/commands/`:
  - `run_folder.go`: `RunFolderCmd` execution traversing collection folders/items, supporting recursive folder execution, delay between requests, bail/stop on failure.
  - Runner control interface (`setNextRequest`, `skipRequest`, `stopExecution`).
  - Cancellation wiring propagating to the in-flight runner controller and current inner request.
  - State and event publications (`ScopeRunner`).
