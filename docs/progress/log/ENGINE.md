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
