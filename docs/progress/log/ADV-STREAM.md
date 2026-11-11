# ADV-STREAM — SSE parsing, progress, cancellation

**Agent:** zed-agent-1
**State:** InReview (core half delivered; UI half blocked)
**Package:** `internal/core/engine/stream.go`
**Reference:** WHATWG HTML §9.2, `docs/logic/http-engine.md`

## Scope split

The package row covers "incremental SSE rendering, large-body streaming, progress"
across `internal/core/engine/stream*.go` **and**
`internal/ui/screens/response/stream*.go`. Only the core half is delivered:
the UI half depends on `UI-SCREEN-RESP`, which is unimplemented, and on a
presenters layer that does not exist yet.

What this half provides is exactly what that screen needs: a parser that emits
discrete events as fragments arrive, a progress tracker with a cheap snapshot,
and a pump with cancellation. Rendering is a `switch` on `Event.Kind` over data
this package already produces.

## Why this is not in `transport`

Transport forwards raw bytes through `Sink.Chunk`. It has no business knowing
what SSE is: a chunk may be a JPEG, a gzip tail, or an event. The split is
transport = bytes, this file = meaning.

## Parser

A single-use parser, one per stream, holding only an incomplete **line** — never
the body — so memory is constant however long the stream runs.

### Spec behaviours implemented, each with a test

| Behaviour | Why it matters |
| --- | --- |
| CR, LF and CRLF are all line endings | Providers differ; a CR-only stream is rare but legal |
| A CR at a fragment boundary followed by LF is one break | Without the `sawCR` flag this yields a phantom blank line and a stray empty event |
| Multiple `data:` lines join with `\n`, final newline removed | A JSON payload split across lines is the common case |
| `event:` defaults to `message` | The field is optional |
| `id:` persists to later messages | It is the `Last-Event-ID` the client resends on reconnect |
| `id:` containing NUL is ignored | Spec rule; a NUL cannot survive a header round trip |
| `retry:` is whole milliseconds; invalid values ignored | A non-numeric retry must not become a 0 ms reconnect storm |
| `:` prefix is a comment; a comment alone dispatches nothing | Keep-alives must not create empty events |
| A leading BOM is stripped | Some servers prefix one |
| `data:x` with no space is legal | The space is optional |
| `data` with no colon is an empty data line | It still counts as data, so it dispatches |
| Unknown fields are ignored | Forward compatibility |
| A stream ending without a blank line still dispatches its event | `Close` exists for exactly this |
| An empty line with no data does not dispatch | Paired with the comment rule above |

### Bounds

`MaxLineBytes` (1 MiB) and `MaxMessageBytes` (8 MiB) turn a malformed or hostile
stream into one `KindError` instead of unbounded growth. The first exceeded bound
sets `failed` and every later write is ignored, so a runaway response produces one
error, not thousands.

## Progress

`Snapshot` separates three cases that are easy to conflate:

- **No Content-Length:** `Expected == 0`, `Fraction == -1`. Not 0, which would
  render as "nothing downloaded".
- **Normal:** `Fraction` in [0,1).
- **Understated Content-Length:** `Fraction > 1`. It is **not** clamped, because a
  bar that reads 100 % while the body is still arriving is a lie. The screen can
  clamp for display; the data does not pre-empt that decision.

## Cancellation

`Pump` checks the context before every read and uses a 32 KiB buffer, so a
cancelled stream stops within one fragment. `TestPumpCancelsMidStream` runs a
real goroutine against a blocking reader, cancels after two fragments, and
asserts the third was never consumed.

## Bugs found while testing

Two, both in my own code, both caught by the tests written for the spec table:

1. **`data` with no colon dispatched nothing.** The dispatch guard checked
   `dataBuf.Len() == 0`, but an empty data value leaves the buffer empty, so the
   message vanished. A separate `hasData` flag now distinguishes "no data lines"
   from "a data line with an empty value".
2. **The message bound re-emitted on every line.** Once exceeded, `failed` was set
   but the loop kept calling `fail` for each remaining line — 17 error events for
   one oversized message. `Write` now stops feeding the parser once `failed` is
   set.

## Verification

```
go vet ./internal/core/engine/               # clean
go test -race -cover ./internal/core/engine/
ok  ...  coverage: 84.9% of statements
```

32 new tests: the 14-row spec table above; byte-at-a-time feeding (the strongest
fragmentation case); a CRLF split across fragments; both parse bounds including
the single-error rule; open/done bracketing; byte accounting with
Content-Length; dispatch-on-terminator rather than at end of stream; pump
incremental delivery; pump cancellation mid-stream; pump read errors; an
already-cancelled context; and six progress cases including the concurrent read
under `-race`.

## Not delivered

- **The response-screen rendering** (`internal/ui/screens/response/stream*.go`),
  blocked on `UI-SCREEN-RESP`.
- **Auto-reconnect.** The parser reports `retry:` and persists `id:` so a
  reconnecting client has what it needs, but reconnection policy (when to give
  up, what `Last-Event-ID` to send) belongs with the screen or a future
  keep-alive layer. Bruno does not reconnect either.
- **Bridging to `events.ResponseChunk`.** That type carries a `Seq` and a UID
  and is emitted by the pipeline; wiring this parser's output onto it needs the
  engine's execution UID, which the caller has and this package does not.

## Sources

- WHATWG HTML §9.2, "Server-sent events" — the event-stream grammar and dispatch
  rules
- `internal/core/engine/transport/transport.go` — `Sink.Chunk`, the byte source
  this layer sits above
