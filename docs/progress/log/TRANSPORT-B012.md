# TRANSPORT — B-012 review fixes

**Agent:** zed-agent-1
**State:** InReview (re-review requested)
**Package:** `internal/core/engine/transport/`
**Reference:** B-012 (review), `docs/logic/http-engine.md`

## Scope of this pass

The package shipped earlier; this pass addresses the two defects B-012 reported
plus one the review could not have seen, and closes B-006.

## Defect 1 — duplicate `waiting` phase

`timeline.end` fabricated a start time when the phase was not open:

```go
start, ok := t.open[phase]
if !ok {
    start = time.Now()   // ← every stray end produced a zero-duration entry
}
```

`GotFirstResponseByte` ended `waiting`, and `doHop` ended it again after `Do`
returned. The second call found no open phase, invented one, and recorded a
duplicate. Two changes:

- **`end` ignores an unopened phase.** A phase is recorded once, by whoever opened
  it.
- **One owner per phase.** The trace hooks own `request-sent` and `waiting`.
  `doHop` calls `endIfOpen` afterwards, which is a no-op when a hook fired and a
  safety net for a pooled connection where a hook legitimately does not.

## Defect 2 — `request-sent` measured the whole round trip

It was opened before `client.Do` and closed after, so it spanned the entire
exchange and appeared before the trailing `waiting`. The phases are now split
where the transitions actually occur:

| Phase | Opens | Closes |
| --- | --- | --- |
| `request-sent` | `doHop`, before `Do` | `WroteRequest` hook |
| `waiting` | `WroteRequest` hook | `GotFirstResponseByte` hook |
| `download` | `doHop`, after headers | after the body |

`TestRequestSentMeasuresTheWrite` uses a 120 ms server delay and asserts it lands
in `waiting`, not `request-sent`.

## Defect 3 — found while writing the tests

**Every duration was wrong.** `drain` ran after `Do` returned, and each event was
stamped with `time.Now()` *there*, so begin and end shared a timestamp and every
phase collapsed to microseconds regardless of what the hooks reported.

This is why fixing defects 1 and 2 initially looked correct at the end of the
request and was still wrong: `waiting` existed, was unique, and lasted 700 ns
even though the server had taken 120 ms. `traceEvent` now carries `at`, captured
on the connection goroutine in the hook, and `begin`/`end` use it.

The lesson is worth recording because it cost the most time: a phase that exists,
is unique, and is in the right order can still be meaningless if its timestamps
are taken at the wrong point.

## Defect 4 — the body cap was not observable

`spoolBody`'s no-spooler path read `io.LimitReader(body, MaxResponseBytes)` and
built a `BodyRef` with `Truncated` left false. Reading exactly the cap cannot
distinguish "the body ended here" from "it was cut", so a truncated body looked
complete. It now reads one byte past the cap and sets the flag.

Also, `TestStreamingChunks` was asserting chunks it could not get: the server
flushed three parts with no pause, so the client's first read returned all three.
The test now paces the writes — the behaviour was correct, the test was not
testing it.

## Acceptance criterion 4

The criterion (body cap with spill through `PERSIST`'s `BodyStore`) now has
transport-level tests rather than relying on the spool's own suite:

- `TestBodyOverCapSpillsToDisk` — 4 KiB through a 16-byte inline threshold, then
  reads the spooled file back and compares byte for byte.
- `TestBodyOverCapTruncatesAtTheLimit` — the cap flag and size.
- `TestSmallBodyStaysInline` — a small body touches no disk.

## B-006 closed

`Options` gained `WrapTransport func(http.RoundTripper) http.RoundTripper`,
applied **once** in `newClient` around the whole client, not per hop — a per-hop
wrap would run the handshake twice. `pipeline.go` passes
`authPrep.Transport.Wrap`, so Digest and NTLM finally reach the wire.

`Options.TLSConfig *tls.Config` was also added for the engine to pass a resolved
configuration (B-013), taking precedence over the verification switches.

## Verification

```
go vet ./internal/core/engine/transport/     # clean
go test -race -cover ./internal/core/engine/transport/
ok  ...  coverage: 86.1% of statements
```

New tests: phase uniqueness, chronological order, write-vs-wait split, per-hop
timeline across a redirect, a failed connection leaving no duplicate, spill to
disk with round-trip, the truncation flag, and the inline case.

`dns` is deliberately not asserted: `httptest` binds `127.0.0.1`, so no lookup
happens and no DNS hook fires. Asserting it would make the test depend on a
resolver the address bypasses.

## Process note

While working, `read_file` twice returned stale content for `client.go` and
`hop.go`, and edits built on it produced a file that did not compile
(`TLSConfig` without the `crypto/tls` import, a duplicated `Transport:` field).
The reliable check was `go build` after every edit rather than trusting the read
back. Both were caught and fixed before committing, but the pattern is worth
knowing: verify with the compiler, not with a re-read.
