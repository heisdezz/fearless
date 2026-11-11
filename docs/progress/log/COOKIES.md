# COOKIES — Cookie jar

Owner history: mimo-flash (claimed 2026-11-10T21:06Z)

## Entries

### 2026-11-10T21:06Z — mimo-flash
- Did: Claimed `COOKIES`. `FORMAT-CODE` was taken by `antigravity` between my review and my claim, so taking this Wave 2 package instead (deps: `MODEL`, which is `Done`).
- Tests: not run yet.
- Notes: Scope per `packages.md` — `net/http/cookiejar` wrapper, trustworthy-origin policy, export/import, persistence hook. Reference: `docs/logic/cookies.md` (single global jar, tough-cookie semantics). Persistence must be injected, not imported (`PERSIST` owns stores). Related type already exists: `model.Cookie` (defined in MODEL, referenced by `events.CookiesUpdated`).

### 2026-11-10T22:30Z — mimo-flash
- Did: Finished design research against Go 1.27 `net/http/cookiejar` source and the Bruno reference (`bruno-requests/src/cookies`, `bruno-electron/src/ipc/network/index.js`). Key findings that shaped the design:
  - `jar.Cookies()` returns **only Name/Value/Quoted** — no Path/Domain/expiry — so the jar cannot back enumeration. Architecture: **own index is the read authority** (CookiesFor/HeaderFor/Domains/matching per `logic/cookies.md` §2–§6); the jar is the **write-time validation oracle** (round-trip probe after `SetCookies` → whatever the jar rejects, we reject with an error). This also makes `logic/cookies.md` §6 (jar wins over explicit `Cookie` header) implementable — verified against Bruno source `{...parseCookies(existing), ...parseCookies(jar)}`. Note: `flows/request-lifecycle.md:278` says "user values win", which contradicts both the VERIFIED logic doc and the source — following `logic/cookies.md` + source (jar wins).
  - Go jar details mirrored: `defaultPath`, MaxAge<0/past-Expires → removal, MaxAge>0 → now+MaxAge, loopback secure-match (`secureMatch` checks the *cookie's* domain), RFC 6265 §5.4 read order (longest path, then creation).
  - Scheme gate: jar only accepts http/https; `wss`→https, `ws`→http mapping implements §3 trustworthy-origin; `file`/other schemes have no jar participation (documented).
  - `__Host-` enforced manually (§4): Secure + Path=/ + no Domain.
  - Settings gap: `model.RequestDefaults` has no `sendCookies`/`storeCookies` (MODEL owns `settings.go`; see `docs/ui/settings.md` lines 39–40). Will append a `decisions.md` request; jar exposes `SetSendCookies`/`SetStoreCookies` (default true) meanwhile.
- Tests: not run yet (writing package now).
- Notes: `golang.org/x/net/publicsuffix` used for `cookiejar.Options.PublicSuffixList` (already in go.mod as indirect; no go.mod edit needed).

### 2026-11-10T23:00Z — mimo-flash
- Did: Implemented `internal/core/cookies/**` (3 source files + tests):
  - `cookies.go` — package doc, injected `Persistence` (§10), `New` with load-time expiry filtering (§7), 5 s debounce with a synchronous test mode, `Persist`/`Close`/`SaveErr` (async save failures recorded and retried, never lost), §7 toggles defaulting to Bruno's `true`, expiry helpers mirroring `wireExpiry`/`modelExpiry` precedence rules.
  - `store.go` — `AddRaw` (strict `http.ParseSetCookie` with a `{loose:true}`-style fallback, `__Host-` matrix per §4, host/Domain validation mirroring `domainAndType` incl. the IP-host rule, write-time round-trip probe through the wrapped jar → `ErrRejected` with nothing stored on refusal, expired lines delete), `AddForDomain` (group-domain fallback, count of stored, `MaxAge<0` deletes), `Delete`/`DeleteForDomain` (idempotent), jar helpers (`acceptJar`, `removeFromJar`).
  - `read.go` — `CookiesFor`/`HeaderFor` from the index with RFC 6265 domain/path/secure rules byte-for-byte from `cookiejar` (`secureMatch` checks the cookie's domain: loopback/`*.localhost` may carry Secure over http), §3 scheme mapping (`wss→https`, `ws→http`, `file`/others out of the jar — documented), `MergeHeaderFor` per §6 (jar wins per name, Bruno's object-spread order), `Domains()` in Bruno's `getDomainsWithCookies` shape (sorted domains, expired filtered), RFC 6265 §5.4 wire order.
- Tests: `gofmt` clean; `go vet ./...` clean repo-wide; `go test -race ./internal/core/cookies/` **ok, 83.3 % coverage**; `go test ./...` whole repo ok; `./scripts/check-deps.sh` ok. 24 tests: domain/path/secure rules, trustworthy origins (loopback, localhost, wss), scheme participation, expiry/removal (`Max-Age=-1/0`, past `Expires`), `__Host-` matrix, PSL + domain-mismatch + IP-mismatch rejections, IP host-only port-independence, merge precedence + toggle passthrough, `Domains` shape, delete idempotency, `AddForDomain` (fallback/count/`__Host-` skip/delete), toggles, loose parse, RFC wire order, `CreatedAt` preservation, persistence lifecycle (load filter, synchronous save, `Close` flush, explicit `Persist`, load error), nil-URL errors, concurrent access (race).
- Notes: Settings gap raised as **D-010** (`RequestDefaults` lacks `sendCookies`/`storeCookies`). Second doc discrepancy for the record: `flows/request-lifecycle.md:278` ("user values win") contradicts `logic/cookies.md` §6 and the Bruno source (jar wins) — implemented jar wins; the flow doc is its owner's to fix. No files outside `internal/core/cookies/**` + this log + board row + D-010 were touched.
- Board: → `InReview`.

### 2026-11-11T03:31Z — mimo-flash

- Did: Answered the two open review items from the 02:18Z row note — **no code change**, both verified correct/intentional:
  - **"Secure over http dropped"** — intended and spec-mandated. `cookieMatches` (`read.go:212`) sends a Secure cookie when the scheme is https/wss **or** the cookie's own domain is loopback/`*.localhost`, and drops it on plain http otherwise. That is byte-for-byte Go's cookiejar `secureMatch` (GOROOT `net/http/cookiejar/jar.go:158`) and, for any domain-matching cookie, equivalent to Bruno's URL-based `isPotentiallyTrustworthyOrigin(url)` (`logic/cookies.md:38`) — a cookie that matches a trustworthy host must itself be that host or a localhost/loopback suffix, so both formulations agree. Evidence: `TestSecureAndTrustworthyOrigins` (https/wss/loopback/localhost send; plain http and ws drop); the write path stores Secure-over-http unconditionally (the round-trip probe reads back over an https URL, `store.go:107`); live probe this session: `AddRaw` + `HeaderFor` round-trips a Secure cookie on `http://[::1]:8080`, so IPv6 loopback is included. If a specific failing URL was seen, a repro would pin it down — nothing in the read/write paths drops a cookie this rule should keep.
  - **"No cookie cap"** — matches the reference. Bruno/tough has no cookie cap anywhere (`grep -rn "max.*cookie|cookieLimit|too many cookies" bruno/packages/` is empty). RFC 6265 §6.1's 4096-byte and 50-per-domain figures are floors an implementation must *support*, not limits to enforce. Adding a cap would be a deliberate divergence from Bruno parity; if one is ever wanted it belongs in a `decisions.md` entry, not a silent change here.
- Tests: unchanged code — `go test -race ./internal/core/cookies/` ok, 83.3 % coverage; `gofmt`/`go vet` clean.
- Board: row note updated; state stays `InReview` awaiting re-confirmation.
