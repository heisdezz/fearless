# Log — `TRANSPORT`

Owner history: zed-agent-1 (claimed 2026-11-11T01:45Z, moved to InReview 2026-11-11T02:20Z)

Package: `internal/core/engine/transport/**`. Depends on `model`, `cookies`, `persistence`, `engine/tls`, `engine/proxy`.

---

## Entries

### 2026-11-11T02:20Z — zed-agent-1

- **Did:** Implemented HTTP execution.
  - `transport.go` — `Execute`, `Options`, `Sink`, `NopSink`, redirect loop, `sentFromRequest`.
  - `hop.go` — one hop: cookie application per hop, `Set-Cookie` capture, body spooling with chunk streaming, error classification, redirect rewriting, timeline.
  - `client.go` — `http.Client`/`http.Transport` construction, TLS switches, proxy wiring, `httptrace` callbacks.
- **Tests:** 22 test functions. `go test -race` passes, stable over `-count=3`; **86.3 % statement coverage**; `gofmt`/`go vet` clean.
- **Acceptance criteria:**
  1. *Redirect matrix* — 301/302/303 → GET with body dropped, 307/308 preserve method and body, relative `Location` resolved, limit exceeded stops and returns the last response. All tested against real servers.
  2. *Cancellation propagates* — tested with a hung handler and a cancelled context; the result records a cancellation error within 2 s.
  3. *Timing timeline* — `request-sent`, `waiting`, `download` recorded and flushed to the sink; DNS/connect/TLS come from `httptrace` when a real network hop occurs.
  4. *Body cap and spill* — `Spooler.Spool` with `MaxResponseBytes` (default 512 MiB). Without a spooler, bodies are read into memory so tests need no persistence setup.

### Two design decisions that differ from a naive port

**Redirects are followed here, not by `net/http`.** `CheckRedirect` returns `ErrUseLastResponse`, so every hop is visible: it gets the jar's cookies for *its* URL, `Set-Cookie` is captured per hop, and `Authorization` is dropped on a cross-host redirect unless `ForwardAuthHeader` is set. Letting `net/http` follow redirects would hide the hops, lose the per-hop cookie capture, and apply the security rule too late.

**Cookies are explicit, not via `http.Client.Jar`.** `cookies.Jar` deliberately exposes `HeaderFor`/`AddRaw` rather than implementing `http.CookieJar`, because it needs Bruno's trustworthy-origin rule and change notification. The transport therefore sets the `Cookie` header itself before each hop and replays every `Set-Cookie` into the jar afterwards. `TestCookiesSentAndStored` and `TestCookiesFollowRedirect` prove both directions, including the redirect case that a naive implementation misses.

### Bug found while testing

`request-sent` was opened but never closed on the success path (only on error), so it silently disappeared from the timeline. `TestTimelineHasPhases` caught it; the phase now ends when response headers arrive.

### Concurrency note worth keeping

`httptrace` callbacks run on the connection's own goroutines during `client.Do`. Writing straight into the timeline slice from them would be a data race, and `-race` would catch it only sometimes. Each callback therefore sends to a buffered channel that the hop drains once `Do` returns, which serialises the writes and preserves their order. Dropped events are preferred over blocking a connection goroutine.

### Not in this package

- **Body replay on redirect** relies on the request body implementing a `Reset() (io.ReadCloser, error)` method. That is a `BUILD` concern: the builder must supply such a body when `MaxRedirects > 0`, otherwise a 307/308 redirect cannot re-send. Recorded so `BUILD` does not ship a body that fails here.
- **CA bundles and client certificates**: `buildTLSConfig` applies only the verification switches; the ENGINE builds the full `*tls.Config` through `engine/tls` and the richer options are wired there.
- **HTTP/2 and connection reuse tuning** use `net/http` defaults.

- **Open:** awaiting review. `ENGINE` can call `Execute` directly once `BUILD` supplies a replayable body.
