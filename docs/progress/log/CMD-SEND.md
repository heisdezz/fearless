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
