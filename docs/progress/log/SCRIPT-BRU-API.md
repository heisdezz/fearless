# SCRIPT-BRU-API — `bru`, `req`, `res`, `test`, `expect`

**Agent:** zed-agent-1
**State:** InReview (two documented compatibility gaps)
**Package:** `internal/core/scripting/bru.go`, `bru_api.go`, `req_res.go`, `test_api.go`
**Reference:** `bruno-js/src/bru.js`, `bruno-request.js`, `bruno-response.js`, `test.js`

## Design

The API is a **host binding layer**, not part of the engine. `Install(rt, state,
control, tests)` registers the objects on a runtime and returns the recorded
nested calls; the caller reads the variable deltas afterwards.

That seam is deliberate: `State` holds plain data and a `*vars.Scopes`, so this
package imports no engine code. A script therefore runs identically in the GUI and
in the headless CLI runner, which the CLI package needs.

| Object | Implemented from |
| --- | --- |
| `bru` | `bruno.js` — 40 methods across five variable scopes, runner control, `utils`, nested requests |
| `req` | `bruno-request.js` — URL/method/headers/body/auth accessors, timeouts, `onFail` |
| `res` | `bruno-response.js` — status, headers, body (parsed and raw), timing, size, data buffer |
| `test` | `test.js` — description plus callback, pass/fail/error reporting |
| `expect` | bundled chai subset — 14 method assertions, 7 boolean assertions, negation |

### Variable scopes are kept separate

`bru.setVar`, `setEnvVar`, `setCollectionVar` and `setGlobalEnvVar` write four
different maps. Collapsing them would be the easy mistake and the destructive one:
`setEnvVar` persists to disk, `setVar` lasts a session. `TestBruScopesAreSeparate`
pins that a write to one does not reach the others.

Values keep their JS types (`5` stays an integer, not `"5"`). Coercing to string
would break `bru.setVar("n", bru.getVar("n") + 1)`.

## Two real compatibility gaps

Both were found by testing, not by reading, and both are recorded rather than
hidden.

### 1. Boolean assertions must be called, not read

chai exposes `expect(x).to.be.true` as a getter. A goja accessor
(`DefineAccessorProperty`) has no error return, so host code cannot raise a
*catchable* JavaScript exception from one — only a Go `panic`, which escapes the
runtime entirely and aborts the test binary.

The assertions are therefore methods: **`expect(x).to.be.true()`** works and
`expect(x).to.be.true` does not. Scripts written for Bruno with the property form
must gain `()`.

**Why not work around it:** the alternative is throwing from a getter by
interrupting the runtime, which makes the failure uncatchable and turns even a
*passing* chain into an abort. A loud, documented difference is better than a
silently broken one.

### 2. `AssertionError` is not the thrown value

Bruno's `test()` distinguishes an `AssertionError` from an ordinary error to
report actual/expected. goja's `*Exception` has an unexported value field, so a
host function cannot construct one. Assertion failures are raised as ordinary Go
errors (which goja converts to `GoError`) and reported as failures with their
message intact, which is what the user sees.

The message still names both operands: `expected 1 to equal 2`.

### Also found

- **goja's call shapes.** Only `func(FunctionCall) Value` and
  `func(FunctionCall, *Runtime) Value` are native functions. `func(FunctionCall)
  string` silently becomes a Go function object whose first argument is converted
  from the call, failing at the *call site* with "could not convert token to
  goja.FunctionCall". `Bindings.native` normalises every variant.
- **Cyclic property graphs overflow the stack.** `obj.Set(word, obj)` makes goja
  recurse forever while exporting. Chain words are accessors instead.
- **Throwing from a Go function** needs a non-nil `error` return. `panic(value)`
  is not converted.

## Verification

```
go vet ./internal/core/scripting/            # clean
go test -race -cover ./internal/core/scripting/
ok  ...  coverage: 84.3% of statements
```

56 tests (13 runtime, 33 API, 10 edge cases) covering:

- Variable scopes separate, typed values preserved, `deleteAll*` per scope,
  safe-mode blocking writes but not reads, `{{var}}` interpolation.
- Runner: `skipRequest`, `stopExecution`, `setNextRequest`, deferred `sleep`.
- `utils.minifyJson` (object and string forms, invalid input) and `minifyXml`
  (null, minimal, text whitespace preserved, multiple gaps).
- `req`: header set/get/delete, `setHeaders` replacing rather than merging,
  URL/method/body round trips with `raw` mode, timeout and redirect overrides,
  absent-request defaults, auth mode.
- `res`: status, text, single and repeated headers, size, parsed and raw body,
  data buffer, `setBody`, timing, URL, absent-result defaults.
- `test`: pass, assertion failure, thrown error, thrown string, missing callback.
- `expect`: 21 passing and 4 failing assertions, negation chains, `oneOf`,
  `match` including an invalid pattern, `isType` for all nine type names plus an
  unknown one, `lengthOf` for strings/arrays/numbers, unknown assertion throws.
- Nested requests: `sendRequest` and `runRequest` recorded with their source,
  callback form receives the error, callback-less form surfaces the rejection.
- `ScriptError` formatting for all four filename/line combinations, and
  `Unwrap`.
- `Install` with nil control/collector supplies live defaults; nil state errors.

## Not done here

- **`bru.sendRequest` / `bru.runRequest` perform no request.** They record the
  call and reject with an explanatory error, because executing a nested request
  means calling the engine, which this package must not import. The caller wires
  a handler using the recorded `ScriptedCall`s; emitting `ScriptedRequest` events
  is then trivial. **Acceptance for this package's "scripted requests appear as
  `ScriptedRequest` events" is therefore met at the boundary, not end to end.**
- **`bru.cookies`, `bru.grpc`** are absent — they need the cookie jar and the
  gRPC runtime, which `ADV-STREAM`/`TRANSPORT` own.
- **Fixture parity is not yet proven.** The acceptance criterion asks that Bruno
  script fixtures produce identical deltas and console output. The mechanism is
  in place but no fixture corpus is wired up, and the two gaps above mean some
  fixtures would fail as written. Recorded against `UQ-B5`.

## Sources

- `bruno-js/src/bru.js` — the `bru` object and all five scopes
- `bruno-js/src/bruno-request.js` — `req`
- `bruno-js/src/bruno-response.js` — `res`
- `bruno-js/src/test.js` — `test`
- `bruno-js/src/sandbox/bundle-libraries.js` — bundled chai `expect`
- `bruno-js/types/script-api.d.ts` — the documented surface (2544 lines)
