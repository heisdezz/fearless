# Log — `BUS`

Owner history: zed-agent-1 (claimed 2026-11-10T22:45Z, moved to InReview 2026-11-10T23:10Z)

Package: `internal/app/bus/**`. Depends on `internal/core/events` and `internal/core/model`.

---

## Entries

### 2026-11-10T23:10Z — zed-agent-1

- **Did:** Implemented the event bus and state actor in `internal/app/bus/`:
  - `bus.go` — `Bus`: fan-out with per-subscriber buffered queues, non-blocking `Publish`, `Subscribe`/cancel, `Close` (idempotent), and `Stats` counters for dropped/disconnected.
  - `store.go` — `Store`: single-owner actor. `Apply`/`ApplyContext` serialize mutators; `Snapshot` publishes immutable states; `Subscribe`/`Publish` delegate to the bus.
  - `state.go` — deep-copy helpers (`cloneState` and friends) that make copy-on-write real.
  - `actor.go` — the `StateActor` interface commands and presenters depend on, with compile-time conformance assertions.
- **Tests:** `bus_test.go` (15 tests) and `store_test.go` (13 tests). `go test -race` passes, stable over `-count=5`, **78 % statement coverage**, `go vet` clean, `gofmt` clean.
- **Notes — back-pressure design:** `Publish` never blocks. A droppable event (chunk, console line) is discarded when a queue is full and counted in `Stats.Dropped`. A **non-droppable** event that cannot be queued disconnects that subscriber (closing its channel) rather than being silently lost or blocking the producer. Rationale: a presenter that misses a state transition would render stale data forever, and a closed channel is a loud, debuggable failure while a silent drop is not.
- **Notes — snapshot semantics:** `Snapshot()` returns a **published, immutable** value (copy-on-write), not a defensive copy per read. `go-architecture.md` §3 describes exactly this, and a deep copy per frame would be wasteful. Readers must treat it as read-only; a comment on both the method and the interface says so.
- **Bug found and fixed during implementation:** `ApplyContext` selected on `quit`, which is closed by `Close` — but `Close` had already returned, so a waiter could hang forever on `op.done`. Fixed by waiting on `done` (actor exited) with a fail-fast check before sending and a re-check on the wait path. A test (`TestApplyAfterCloseReturnsError`) covers it; the original hang was caught by `go test` timing out at 600 s.
- **Data race found and fixed:** the first `subscription` design checked a closed-channel flag then sent, leaving a window where a concurrent `shutdown` closed `s.ch` and the send panicked. Replaced with an `RWMutex` held only across a non-blocking send, so the pair is atomic with respect to closing. `-race` is clean.
- **Open:** awaiting review. `CMD-*` packages can now depend on `StateActor`; `UI-PRESENT-*` can depend on `Subscribe`.

### Test coverage by concern

| Concern | Tests |
| --- | --- |
| Fan-out and ordering | `TestPublishReachesEverySubscriber`, `TestOrderingIsFIFOPerSubscriber` |
| Back-pressure | `TestPublishNeverBlocks`, `TestDroppableEventsAreDiscardedSilently`, `TestTerminalEventsAreNeverDropped`, `TestSlowSubscriberDoesNotAffectFastSubscriber` |
| Lifecycle | `TestCancelIsIdempotentAndClosesChannel`, `TestPublishAfterCancelDoesNotPanic`, `TestCloseClosesAllSubscribers`, `TestPublishAfterCloseIsNoop`, `TestSubscribeAfterCloseYieldsClosedChannel` |
| Concurrency | `TestConcurrentPublishAndSubscribe` |
| State | `TestStoreStartsFromInitialState`, `TestSnapshotIsCopyOnWrite`, `TestDraftAndSavedAreIndependent`, `TestApplyIsSerialised`, `TestCloneResultCopiesInlineBody` |
| Lifecycle (store) | `TestApplyAfterCloseReturnsError`, `TestCloseIsIdempotentAndDrains`, `TestSnapshotWorksAfterClose` |
