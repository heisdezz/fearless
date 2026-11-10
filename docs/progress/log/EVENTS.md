# Log — `EVENTS`

Owner history: zed-agent-1 (claimed 2026-11-10T22:00Z, moved to InReview 2026-11-10T22:20Z)

Package: `internal/core/events/**`. Depends only on `internal/core/model`.

---

## Entries

### 2026-11-10T22:20Z — zed-agent-1

- **Did:** Implemented the typed event contract in `internal/core/events/`:
  - `kind.go` — `Event` interface, `Kind` enum (13 kinds, zero value invalid), `AllKinds`, `Valid()`, `String()`, plus `IsTerminal()` and `IsDroppable()` which encode the bus's back-pressure rules as data rather than commentary.
  - `request.go` — `RequestQueued`, `RequestSent` (with attempt/resend-reason for redirects and auth challenges), `ResponseChunk`, `ResponseReceived`, `UnresolvedVariables`, `ScriptedRequest`, `ScriptOutput`.
  - `results.go` — `TestResults` (+ `Summary()`), `AssertionResults` (+ `AllPassed()`), `DomainCookies`/`CookiesUpdated`, `CredentialsUpdated`, `TreeChange`/`CollectionChanged`.
  - `error.go` — `ErrorOccurred` with scope/class/message/details, `NewError`, `Classify`, and `With*` builders.
- **Tests:** `events_test.go` — 18 test functions. `go test -race` passes, **100 % statement coverage**, `go vet` clean, `gofmt` clean.
- **Notes:**
  - **`ErrorOccurred` is a value type, not a pointer.** `go-architecture.md` writes it as a value with `Err error`; `interface-design.md` §4 sketches all events as values. Making it a pointer would force presenters to special-case one event in an otherwise uniform `switch`. Changing it later is a breaking API change, so this is recorded as D-008.
  - `CredentialsUpdated` deliberately has **no** refresh-token field. A test documents the intent so adding one becomes a conscious act.
  - `ResponseChunk.Data` is documented as not-to-be-retained: the producer may reuse the slice. Consumers needing the full body read `Result.Body` (a `BodyRef`) from `ResponseReceived` instead of concatenating fragments. This is what makes chunk dropping safe.
  - `CookiesUpdated` carries a **full snapshot** per changed domain, not a delta, so consumers can replace state wholesale.
  - Added `model.Cookie` (in `model/cookie.go`) because no cookie value type existed and `CookiesUpdated` needs one. The jar and its policy still belong to `COOKIES`; only the value type lives in the model. Noted in that file's doc comment.
- **Spec gaps found** (both recorded in `decisions.md`):
  - `interface-design.md` §6 references `model.Draft` in `StateActor.Apply`, but no such type exists — `model.Item` has `Draft *Request`. This blocks `BUS`, not `EVENTS`. Recorded as D-007.
  - `Model` had no cookie type at all; added.
- **Unrelated breakage noticed:** `internal/testutil/testutil.go` (owned by `FIXTURES`) did not compile during this work (`net/http` missing while `httptest` was imported). Reported as B-003; resolved by `kilo` before this package finished. Not touched by me.
- **Open:** awaiting review. `BUS` is now unblocked once D-007 is settled.
