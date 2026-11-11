# PROXY

**Agent:** zed-agent-1 (took over from `antigravity` to close B-011)
**Scope:** `internal/core/engine/proxy/**`
**State:** InReview

Proxy resolution for four modes (off / manual / system / PAC), SOCKS5, and the
per-request override.

## What was already here

`bypass.go`, `config.go` and `resolver.go` covered off/manual/system, NO_PROXY
matching, scheme and domain auth, and IPv4/IPv6 CIDR handling. `Resolve`
implements the precedence chain Request > Collection > App > Environment.

## B-011: PAC was a constant and a field, with no implementation

`model.ProxyPAC` existed and `ProxySettings.PACURL` was declared, but `Resolve`
fell straight through the `ProxyPAC` case to the environment, and nothing
anywhere evaluated `FindProxyForURL`. The acceptance criterion in
`packages.md` was dropped, not deferred.

### `pac.go`

**Sandbox.** PAC is evaluated in a purpose-built goja runtime, not the scripting
runtime. That is the central decision: a PAC file can be fetched over the
network and is untrusted input, and `internal/core/scripting` deliberately grants
`require`, timers, `fetch` and a console. Reusing it would have been less code
and a worse security posture.

Installed: `dnsDomainIs`, `shExpMatch`, `isPlainHostName`, `isInNet`,
`dnsResolve`, `myIpAddress`, `localHostOrDomainIs`, `dnsDomainLevels`,
`weekdayRange`, `dateRange`, `timeRange`, `alert`.

`TestPACSandboxHasNoScriptingPower` asserts `require`, `fetch`, `setTimeout`,
`setInterval`, `console`, `XMLHttpRequest` and `process` are all `undefined`.

**Calling convention.** The first argument is the URL **string**, not a context
object. The spec calls the parameter `url` and its own examples slice it
(`url.substring(0, 5) == "http:"`); real PAC files depend on that. There is a
test for it.

**Globals fail soft.** `dnsResolve` returns `""` and `isInNet` returns `false`
on a lookup failure rather than throwing, because PAC scripts are written to
fall through when a name does not resolve.

`shExpMatch` uses a purpose-written glob matcher rather than `path.Match`: these
patterns are hosts and URLs, where `*` must cross `/`, and `path.Match`
deliberately will not.

**Time predicates** compare seconds-since-midnight so an hour-only range covers
the whole hour. `GMT` is accepted and ignored for the date and time predicates;
honouring it would change which day selects a proxy, which is a surprising
effect for a feature PAC files rarely use. The tests compute expectations from
`time.Now()` rather than hard-coding a weekday.

**Deadline.** goja has no context support, so a watchdog interrupts the runtime
when the context expires and the result is `ErrPACTimeout`. Without it a PAC
file containing `while (true)` would hang request setup (`TestPACTimeout`).

### `pac_resolve.go`

`PACSource` is an interface so a test can hand in a fixed result and production
can hand in the cached `PACResolver`. `resolveFromPAC` applies an explicit
`Bypass` entry **before** running the script — a user's per-app bypass should
not be overridable by a network policy — and returns `ErrPACNoSource` when PAC
mode is on with nothing configured, rather than silently going direct.

### B-020: the fallback list collapses to one attempt

`FindProxyForURL` returns a _ranked list_: `"PROXY a; PROXY b; DIRECT"` means
try each in turn. `Resolve` returns one `*url.URL`, so only the first attempt is
used and a trailing `DIRECT` is dropped.

This is not a resolver limitation — `PACResult.Attempts` carries the whole list.
Consuming more than one attempt means retrying across transports, which belongs
to `TRANSPORT`. Logged as B-020 with the proposed `ProxySelector` shape rather
than papered over.

## Verification

```
go test -race -cover ./internal/core/engine/proxy/   ok  91.5 %
go build ./...                                      clean
go vet ./...                                        clean
gofmt -l internal/core/engine/proxy/                empty
sh scripts/check-deps.sh                            dependency-policy: ok
go test ./...                                       whole tree green
```

Coverage rose from 82.0 % to 91.5 %: three new test files (37 top-level tests)
plus the two pre-existing gaps the sweep exposed (`FromModelSettings`'s nil and
password paths, and `resolveFromEnvironment`'s per-scheme selection).

## Golden vectors

The Netscape "Proxy Auto-Config" specification's two canonical example
documents are the golden tests. Where the repo has been bitten before by
reconstructed fixtures (the AWS SigV4 SDK cage), the lesson applied here: use the
specification's own published text.
