# CMD-RUNNER — Folder runner and control commands

Owner history: antigravity (claimed 2026-11-11T01:08Z)

## Entries

### 2026-11-11T01:08Z — antigravity

- Did: Claimed `CMD-RUNNER`. Reviewed requirements in `docs/progress/packages.md`, `ownership.md`, and `interface-design.md`.
- Tests: not run yet.
- Notes: Scope covers `internal/app/commands/runner*.go` and `run_folder.go`. Needs folder runner, recursive execution, tag filtering, runner control (`setNextRequest`, `skipRequest`, `stopExecution`), delay between requests, bail-on-failure, cancellation tracking, and event emission.

### 2026-11-11T01:13Z — antigravity

- Did: Completed implementation of `CMD-RUNNER`:
  - `internal/app/commands/runner.go`:
    - `FolderRunner` interface (`RunFolder`, `RunFolderSync`, `CancelRunner`, `InFlight`, `ActiveCount`).
    - `RunnerController` for dynamic control: `SetNextRequest(target string)`, `SkipRequest()`, `StopExecution()`, `GetNextRequest() (string, bool)`, `ShouldSkip() bool`, `ShouldStop() bool`, and `Reset()`.
    - `RunFolderCmd` specifying `CollectionUID`, `FolderUID`, `Environment`, `Recursive`, `DelayMs`, `Bail`, `Tags`, `RuntimeVars`, and `PromptVars`.
    - `RunnerSummary` reporting totals for run, passed, failed, skipped, duration, and individual request results.
    - `RunnerService` tracking active runner contexts by UID, cancellable via `CancelRunner`.
  - `internal/app/commands/run_folder.go`:
    - `RunFolder` execution pipeline: collection resolution, folder traversal (recursive or shallow), tag matching via `hasAnyTag`.
    - Dynamic loop control reacting to `RunnerController`: target jump via `SetNextRequest` (finding item index by name or UID), skip logic via `SkipRequest`, and immediate halt via `StopExecution`.
    - Configurable inter-request delay (`DelayMs`), bail-on-first-failure (`Bail`), and cancellation propagation (`ScopeRunner`).
    - Bus event emissions: `RunnerStarted`, `RunnerItemStarted`, `RunnerItemCompleted`, `RunnerCompleted`, and terminal `ErrorOccurred` on cancellation.
  - Comprehensive unit tests in `internal/app/commands/runner_test.go`:
    - `TestRunFolderSuccess`: validates standard run, order, delay, summary, and lifecycle events.
    - `TestRunFolderTagFiltering`: validates selective inclusion of requests based on tags.
    - `TestRunFolderCancellation`: validates active runner cancellation with `ScopeRunner` and proper termination.
    - `TestRunFolderRunnerControl`: validates `SetNextRequest` jumping across items dynamically.
    - `TestRunFolderBailOnFailure`: validates immediate loop termination on request failure when `Bail: true`.
- Tests: `go test -race -cover ./internal/app/commands` passes with 83.5% statement coverage. `go vet` and `./scripts/check-deps.sh` pass cleanly.
- Ready for review.
