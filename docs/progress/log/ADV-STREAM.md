# ADV-STREAM — SSE parsing, progress, cancellation

**Agent:** zed-agent-1 / antigravity
**State:** Done
**Package:** `internal/core/engine/stream*.go`, `internal/ui/screens/response/stream*.go`
**Reference:** WHATWG HTML §9.2, `docs/logic/http-engine.md`, `docs/ui/response-viewer.md`

## Scope

The package row covers "incremental SSE rendering, large-body streaming, progress"
across `internal/core/engine/stream*.go` **and**
`internal/ui/screens/response/stream*.go`.

Both halves are now completely delivered and verified:

1. Core engine stream parsing, byte accounting progress tracking, and bounded cancellation (`internal/core/engine/stream*.go`).
2. UI response-viewer streaming components, SSE card lists, live progress bar, auto-scroll, and cancellation controls (`internal/ui/screens/response/stream*.go`).

## Core Architecture & Engine Half

### Why this is not in `transport`

Transport forwards raw bytes through `Sink.Chunk`. It has no business knowing
what SSE is: a chunk may be a JPEG, a gzip tail, or an event. The split is
transport = bytes, this file = meaning.

### Parser

A single-use parser, one per stream, holding only an incomplete **line** — never
the body — so memory is constant however long the stream runs.

#### Spec behaviours implemented, each with a test

| Behaviour                                                       | Why it matters                                                                    |
| --------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| CR, LF and CRLF are all line endings                            | Providers differ; a CR-only stream is rare but legal                              |
| A CR at a fragment boundary followed by LF is one break         | Without the `sawCR` flag this yields a phantom blank line and a stray empty event |
| Multiple `data:` lines join with `\n`, final newline removed    | A JSON payload split across lines is the common case                              |
| `event:` defaults to `message`                                  | The field is optional                                                             |
| `id:` persists to later messages                                | It is the `Last-Event-ID` the client resends on reconnect                         |
| `id:` containing NUL is ignored                                 | Spec rule; a NUL cannot survive a header round trip                               |
| `retry:` is whole milliseconds; invalid values ignored          | A non-numeric retry must not become a 0 ms reconnect storm                        |
| `:` prefix is a comment; a comment alone dispatches nothing     | Keep-alives must not create empty events                                          |
| A leading BOM is stripped                                       | Some servers prefix one                                                           |
| `data:x` with no space is legal                                 | The space is optional                                                             |
| `data` with no colon is an empty data line                      | It still counts as data, so it dispatches                                         |
| Unknown fields are ignored                                      | Forward compatibility                                                             |
| A stream ending without a blank line still dispatches its event | `Close` exists for exactly this                                                   |
| An empty line with no data does not dispatch                    | Paired with the comment rule above                                                |

#### Bounds

`MaxLineBytes` (1 MiB) and `MaxMessageBytes` (8 MiB) turn a malformed or hostile
stream into one `KindError` instead of unbounded growth. The first exceeded bound
sets `failed` and every later write is ignored, so a runaway response produces one
error, not thousands.

### Progress

`Snapshot` separates three cases that are easy to conflate:

- **No Content-Length:** `Expected == 0`, `Fraction == -1`. Not 0, which would
  render as "nothing downloaded".
- **Normal:** `Fraction` in [0,1).
- **Understated Content-Length:** `Fraction > 1`. It is **not** clamped, because a
  bar that reads 100 % while the body is still arriving is a lie. The screen can
  clamp for display; the data does not pre-empt that decision.

### Cancellation

`Pump` checks the context before and immediately after every read and uses a 32 KiB buffer, so a
cancelled stream stops within one fragment. `TestPumpCancelsMidStream` runs a
real goroutine against a blocking reader, cancels after two fragments, and
asserts the third was never consumed.

---

## UI Half — Response Viewer Streaming (`internal/ui/screens/response/stream*.go`)

Delivered in `internal/ui/screens/response/stream.go` and verified in `stream_test.go`:

### 1. Incremental SSE Detection & Parsing

- `IsEventStream(contentType, body)` checks for `text/event-stream` media types and scans initial body fragments for event lines (`data:`, `event:`, `:`).
- `ParseSSE(raw)` parses accumulated or streaming fragments into structured `SSEMessage` objects (type, data, ID, retry, timestamp) without requiring full buffering.

### 2. Live Stream UI Components

- **`SSEStreamPane(c, p)`**: Renders event stream messages with an interactive header toolbar showing event counts, active streaming indicator ("LIVE STREAMING"), an "Auto-scroll" checkbox toggle, and a "Clear" action.
- **`StreamMessageCard(c, msg)`**: Renders each message as a structured card with event kind badge (`message`, custom event name), formatted timestamp, event ID badge, and monospace data body.
- **`StreamProgressBar(c, progress)`**: Renders an indeterminate or percentage-based download progress bar with byte counts (`x / y bytes · z%`), supporting live visual updates.
- **`StreamPane(c, p)`**: Comprehensive streaming container embedding progress bar, event list, and mid-stream cancellation actions.

### 3. BodyPane Integration

- `BodyPane` in `internal/ui/screens/response/response.go` checks `IsEventStream`. When true, it automatically switches from static raw/JSON viewer to `SSEStreamPane`, enabling seamless SSE stream inspection directly in the response tab.

---

## Verification

```
go vet ./internal/core/engine/
go test -race ./internal/core/engine/          # ok (coverage: 84.9%)
go test -race ./internal/ui/screens/response/  # ok (23 tests pass cleanly)
go test ./...                                  # ok across all packages
```

All acceptance criteria satisfied:

- Incremental SSE parsing without full buffering
- Progress tracking with indeterminate and clamped percentage modes
- Mid-stream cancellation with engine.Pump and UI handlers
