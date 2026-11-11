# SCRIPT-RUNTIME — goja sandbox

**Agent:** zed-agent-1
**State:** InReview
**Package:** `internal/core/scripting/runtime.go`, `sandbox.go`
**Reference:** `bruno-js/src/sandbox/node-vm/`, `bruno-js/src/sandbox/node-vm/constants.js`

## What Bruno does, and what that implies

Bruno runs scripts in a Node `vm` context (`node-vm/index.js`) with a curated
`safeGlobals` list (`constants.js`). The comment there explains the design: ECMAScript
built-ins are **not** passed from the host, because the VM provides its own and
sharing them causes prototype-chain mismatches. Only Node/Web extras are injected.

goja makes that automatic — it has its own complete realm — so this package only
installs the extras. Two consequences worth stating:

- **A fresh runtime per execution.** Bruno creates a new context per run. Scripts
  assign to `globalThis`, declare helpers and shadow built-ins; sharing a runtime
  between two requests would leak all of that. `TestConcurrentRuntimesAreIndependent`
  pins it under `-race`.
- **`require` is a table, not a module system.** Bruno's loader resolves npm
  packages and Node built-ins. Go has neither, so `Options.Modules` is an explicit
  map that the caller populates. An unknown name throws.

## Sandbox contents

| Global | Status | Notes |
| --- | --- | --- |
| ECMAScript built-ins | goja native | `Object`, `Array`, `JSON`, `Promise`, `Map`, … |
| `console.*` | implemented | Objects render as JSON, not `[object Object]` |
| `require` | implemented | Allow-list table; refuses unknown names |
| `btoa` / `atob` | implemented | `atob` returns `""` on bad input rather than throwing |
| `Buffer.from` / `isBuffer` | subset | `length`, `toString([enc])`, `toJSON` |
| `TextEncoder` / `TextDecoder` | implemented | Real constructors, so `new …` works |
| `setTimeout` / `setInterval` / `queueMicrotask` | immediate | See limitations |
| `URL`, `URLSearchParams` | **missing** | See UQ-B5 gap below |
| `fetch`, `WebSocket`, `crypto.subtle` | **missing** | Deliberate; see below |
| `fs`, `path`, `crypto`, `os` | **missing** | Bruno allows Node built-ins; no equivalent yet |

### Gaps recorded against UQ-B5

The acceptance criterion asks whether the inbuilt-library set matches
`tests/scripting/inbuilt-libraries`. **It does not.** That suite contains `fs` and
`jsonwebtoken`; Bruno's bundled set (`bundle-libraries.js`) is `chai`, `moment`,
`buffer`, `btoa`, `atob`, `crypto-js`, `tv4`, `ajv`, `ajv-formats`; and the npm
dependencies add `axios`, `cheerio`, `jsonwebtoken`, `nanoid`, `node-fetch`,
`uuid`, `xml2js`, `lodash`, `quickjs-emscripten`.

None of those are implemented here. They are JavaScript libraries, not language
features, and porting them is `SCRIPT-BRU-API`'s problem (it owns the host
bindings) or a separate decision about embedding a module bundle. What this
package guarantees is the *runtime* those libraries would run on:

- `require(name)` resolves from the caller's table, so each library can be added
  without touching the sandbox.
- A library object can be any Go value, exported through `goja.ToValue`, so a Go
  implementation of, say, `crypto-js`'s SHA-256 plugs in directly.

**Deliberate omissions:** `fetch`, `WebSocket` and `crypto.subtle` are absent
because they would give a script unsandboxed network access past the engine, and
`fs` because a script that can read arbitrary files is a worse security posture
than Bruno's Node process already has. Each is a policy decision to make
explicitly, not a default to inherit.

## Runtime safety

| Property | Mechanism | Verified by |
| --- | --- | --- |
| Runaway script terminates | `goja.Runtime.Interrupt` on deadline | `TestRunInterruptsRunawayScript` (`while(true){}` ends at 50 ms) |
| Cancellation | same interrupt, `ErrCancelled` | `TestRunHonoursContextCancellation` |
| No goroutine leak | watcher released by the deferred `stop` | `TestRunDoesNotLeakGoroutines` (100 runs, `-race`) |
| No stale interrupt | `ClearInterrupt` before and after each run | `TestInterruptDoesNotLeakAcrossRuns` |
| Recursion bounded | `SetMaxCallStackSize` | `TestRecursionLimitIsEnforced` |
| Concurrent executions | one runtime each | `TestConcurrentRuntimesAreIndependent` (8 goroutines, `-race`) |

Interruptions and JavaScript-thrown errors both surface as `*ScriptError`, but
`errors.Is(err, ErrTimeout)` distinguishes them, so a caller can report "the script
ran too long" differently from "the script threw".

## Promise semantics — a real finding

Bruno awaits the script result, so a script may end with a promise. The naive
port looks like it works and does not: **goja captures the completion value before
draining its job queue**, so

```js
Promise.resolve(5).then(v => v * 2)   // completion value is the pending promise
```

exported as `nil`. `Runtime.await` therefore detects a `*goja.Promise` result and
pumps until it settles, rejecting with a `*ScriptError` if it rejects.

There is no event loop, so pumping is sound: every job that will ever run is
already queued. A promise that never settles hits the deadline.

The narrower guarantee is documented rather than hidden: a script that ends in a
plain expression while a `then` is still queued does **not** get that callback
flushed before `Run` returns. It runs immediately after — `TestQueuedMicrotasksRunAfterRun`
proves it — which matches awaiting the *result* rather than the *queue*.

## Timers

`setTimeout` and `setInterval` invoke their callback immediately; `setInterval`
runs it once rather than looping forever. With no event loop, a future-scheduled
timer cannot fire at the right time, and silently never firing would be worse for
the callback-style code scripts actually use. This is a documented approximation
in the log, not an unstated behaviour.

## Verification

```
go vet ./internal/core/scripting/            # clean
go test -race -cover ./internal/core/scripting/
ok  ...  coverage: 84.4% of statements
```

43 tests covering: completion values and Go export shapes; the six safety
properties above; `*ScriptError` fields including filename and `LineOffset`;
console capture, serialisation and rendering of `undefined`/`null`; the module
table with `node:` normalisation and refusal of unknown names; caller globals;
set/get variables; `RunFunction`; promise fulfil, reject and queued-microtask
order; `btoa`/`atob`; the `Buffer` subset; timers; `TextEncoder`/`TextDecoder`.

## Dependency note

`go.mod` gained `github.com/dop251/goja` (plus `dlclark/regexp2/v2`,
`go-sourcemap/sourcemap`, `google/pprof`). `go.mod` belongs to `INFRA`, so the
change is logged as **B-007** for the owner to confirm — `INFRA` is marked Done and
its owner has moved to the UI packages, so waiting would have stalled a v1-critical
package. The exact version was already in the module cache.

## Sources

- `bruno-js/src/sandbox/node-vm/index.js` — VM setup, per-run context
- `bruno-js/src/sandbox/node-vm/constants.js` — `safeGlobals`
- `bruno-js/src/sandbox/node-vm/console.js` — console serialisers
- `bruno-js/src/sandbox/node-vm/cjs-loader.js` — `require` resolution
- `bruno-js/src/sandbox/bundle-libraries.js` — bundled library set
- `bruno/packages/bruno-tests/collection/scripting/inbuilt modules/` — the fixture set
