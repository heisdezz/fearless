# CMD-SEND — Send and cancel commands

Owner history: antigravity (claimed 2026-11-11T00:45Z)

## Entries

### 2026-11-11T00:45Z — antigravity

- Did: Claimed `CMD-SEND`. Reviewed `docs/progress/packages.md`, `docs/implementation/go-architecture.md`, `docs/implementation/interface-design.md`, and dependencies (`ENGINE`, `BUS`, `EVENTS`).
- Tests: not run yet.
- Notes: Implementing `SendRequest` and `CancelRequest` command handlers in `internal/app/commands/`:
  - Request registry tracking in-flight requests by UID and cancelling prior in-flight requests for the same item UID.
  - Context cancellation wiring (`context.WithCancel`).
  - Terminal event publication invariant: every `RequestQueued` yields exactly one terminal event (`ResponseReceived` or `ErrorOccurred`, including on cancel emitted as `ErrorOccurred{Err: context.Canceled}` with `ScopeSend`).
  - Event sink adapter forwarding chunk, timeline phase, script output, and unresolved variables directly to the bus.

### 2026-11-11T00:53Z — antigravity

- Did: Completed implementation of `CMD-SEND` package:
  - `internal/app/commands/send_request.go`:
    - `RequestSender` interface (`Send`, `SendSync`).
    - `SendRequestCmd` payload supporting collection/item resolution from `store` or direct parameters, environment selection, runtime and prompt variables.
    - `RequestService`: in-flight tracking registry, context cancellation wiring, and bus event publishing.
    - Automatic pre-emption: if a request is already in-flight for an item, sending that item again cancels the prior request before queueing.
    - `busSink`: EventSink adapter publishing `RequestSent`, `ResponseChunk`, `ScriptOutput`, and `UnresolvedVariables` to the event bus.
    - Strict invariant: every `RequestQueued` yields exactly one terminal event (`ResponseReceived` or `ErrorOccurred`).
  - `internal/app/commands/cancel_request.go`:
    - `RequestCanceller` interface (`Cancel`, `InFlight`, `ActiveCount`).
    - `Cancel` by `RequestUID` or by `ItemUID`.
    - Cancellation yields `ErrorOccurred{Err: context.Canceled, Scope: ScopeSend, Class: ErrorCancelled}`.
  - Unit tests in `send_request_test.go` and `cancel_request_test.go`:
    - Validated normal send lifecycle (RequestQueued -> RequestSent -> ResponseReceived).
    - Verified cancellation by request UID and item UID producing exactly one terminal `ErrorOccurred` with `context.Canceled` and `ScopeSend`.
    - Verified in-flight replacement cancels prior execution.
    - Verified draft edits take precedence over saved requests.
    - Verified resolution of environment, collection, folder, runtime, and prompt variables into scopes.
- Tests: `go test -race -v ./internal/app/commands` passes 100% of tests (87.4% statement coverage). `go vet`, `gofmt`, and `./scripts/check-deps.sh` pass cleanly.
- Ready for review.
