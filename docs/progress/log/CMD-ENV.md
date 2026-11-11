# CMD-ENV — Environment Commands

Owner history: antigravity-2 (claimed 2026-11-11T00:52Z, moved to InReview 2026-11-11T00:58Z)

## Entries

### 2026-11-11T00:52Z — antigravity-2

- Did: Claimed `CMD-ENV`. Reviewed `packages.md` acceptance criteria, `ownership.md` owned paths (`internal/app/commands/environment*.go`, `select_environment.go`, `save_environment.go`), `interface-design.md` §2 (`EnvSelector`), `docs/data/environment-format.md`, and `internal/core/model/collection.go`.
- Tests: not run yet.
- Notes: Scope covers selecting the active environment, creating new environment files (`.bru` and `.yml`), updating/saving environments, deleting environments, and publishing typed events (`CollectionChanged` with Modified/Added/Removed) while keeping workspace state synchronized and idempotent.

### 2026-11-11T00:58Z — antigravity-2

- Did: Implemented environment command handlers in `internal/app/commands/`:
  - `environment.go`: Defined consumer interfaces from `interface-design.md` §2 (`EnvSelector`, `EnvSaver`, `EnvCreator`, `EnvDeleter`) and command payloads (`SelectEnvironmentCmd`, `SaveEnvironmentCmd`, `NewEnvironmentCmd`, `DeleteEnvironmentCmd`).
  - Implemented `NewEnvironment`: Creates new `.bru` or `.yml` (OpenCollection) environment files on disk, computes stable path UIDs, inserts into collection state, and publishes `events.CollectionChanged` (Self, Added). Duplicate environment names rejected.
  - Implemented `DeleteEnvironment`: Removes environment file from disk with self-write suppression, removes environment from `col.Environments`, unselects it if active, and publishes `events.CollectionChanged` (Self, Removed). Idempotent when already removed.
  - `select_environment.go`: `SelectEnvironment` switching active environment by name or clearing it (when name is `""`), validating environment existence, updating `targetCol.ActiveEnvironment`, and publishing `events.CollectionChanged` (Self, Modified). Idempotent on already active environment.
  - `save_environment.go`: `SaveEnvironment` persisting modified environments to disk atomically via `collection.SaveEnvironment` with self-write suppression, updating environment variables in workspace state, and publishing `events.CollectionChanged` (Self, Modified).
  - `environment_test.go`: 5 test suites covering selecting active environments, creating environments with typed variables, updating and atomic saving, deleting environments with active environment cleanup, YAML collection environment handling, and validation error cases.
- Tests: `go test -count=1 -race -v -cover ./internal/app/commands` -> PASS (**88.5% statement coverage** across commands package). `go vet ./internal/app/commands/...` -> clean. `gofmt -l` -> clean. `./scripts/check-deps.sh` -> dependency-policy: ok.
- Notes: Ready for review. Meets all acceptance criteria in `packages.md`.
