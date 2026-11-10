# ENGINE — Request execution pipeline

Owner history: antigravity (claimed 2026-11-11T00:20Z)

## Entries

### 2026-11-11T00:20Z — antigravity

- Did: Claimed `ENGINE`. Reviewed `docs/flows/request-lifecycle.md`, `docs/implementation/go-architecture.md`, `docs/implementation/interface-design.md`, and dependencies (`BUILD`, `TRANSPORT`, `AUTH-CORE`, `COOKIES`, `EVENTS`).
- Tests: not run yet.
- Notes: Implementing the request lifecycle pipeline orchestrating:
  - Phase ordering strictly conforming to `flows/request-lifecycle.md`:
    1. Pre-request hooks/scripts
    2. Request build (`build.Build`: tree inheritance, variables interpolation, body rendering, auth headers)
    3. Configure request (proxy resolution, TLS options, cookie jar attach, transport options, timeouts)
    4. Auth provider apply / transport wrapping
    5. Transmission via `transport.Execute` with sink streaming and timeline recording
    6. Cookie persistence & update event
    7. Post-response pipeline (post-response vars, post scripts, assertions, tests)
  - Core interfaces and types: `Engine`, `Input`, `Result`, `EventSink`, `HookSet` (script hooks, assert hooks, etc.).
  - Exactly one `Result` or error per `Do`.
  - Zero `ui/*` and `image` imports.

### 2026-11-11T00:39Z — antigravity

- Did: Completed implementation of `ENGINE` package:
  - `sink.go`: Defined `EventSink` interface (`Queued`, `Chunk`, `Phase`, `ScriptOutput`, `Unresolved`), `NopSink`, `RecordingSink`, and `transportSinkBridge` connecting engine events to `transport.Sink`.
  - `hooks.go`: Defined `HookSet` and adapters for `PreRequestHook`, `PostResponseHook`, `AssertHook`, and `TestHook`.
  - `result.go`: Implemented `TemplateResponseFromResult` for declarative variable block post-processing and `$res` mapping.
  - `engine.go`: Defined `Engine` interface, `Input` struct, engine options (`WithDefaultUserAgent`, `WithDefaultSpooler`, `WithDefaultJar`), and `New` constructor.
  - `pipeline.go`: Implemented `runPipeline` strictly orchestrating all phases of `flows/request-lifecycle.md`:
    1. Pre-request declarative variables (`template.PreRequest`).
    2. Pre-request hooks (`in.Hooks.PreRequest`).
    3. Build request synthesis (`build.Build`).
    4. Configure request and apply auth (`auth.ChainFor`, `auth.Apply`, timeouts, redirect configuration).
    5. Transmission (`in.Sink.Queued(sent)`, `transport.Execute` streaming chunks and timeline phases).
    6. Cookie jar updates.
    7. Post-response pipeline in strict order (`template.PostResponse`, post hooks, assertions, tests).
    8. Guaranteed invariant: exactly one of `(*model.Result, nil)` or `(nil, error)` is returned via `ensureOne`.
  - `engine_test.go`: Comprehensive test suite verifying phase ordering with recording fakes, single-result-or-error guarantee, variable scoping, timeline/chunk delivery to sink, and absence of GUI/image dependencies.
- Tests: `go test -race -v -cover ./internal/core/engine` passes at 86.8% coverage. `go vet`, `gofmt`, and `./scripts/check-deps.sh` pass cleanly.
- Ready for review.
