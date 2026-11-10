# PROXY — HTTP and SOCKS proxy resolution

Owner history: antigravity (claimed 2026-11-10T23:30Z, moved to InReview 2026-11-10T23:55Z)

## Entries

### 2026-11-10T23:30Z — antigravity

- Did: Claimed `PROXY`. Reviewing `packages.md` acceptance criteria, `flows/request-lifecycle.md` §3, and `logic/http-engine.md`.
- Tests: not run yet.
- Notes: Implementing proxy resolution with precedence (request > collection > app preferences > env), bypass CIDR/domain pattern matching with curl semantics, and SOCKS5/HTTP/HTTPS upstream support in `internal/core/engine/proxy/`.

### 2026-11-10T23:55Z — antigravity

- Did: Implemented proxy configuration, bypass evaluation, resolution precedence, and transport binding:
  - `config.go`: Defined `Config` (`Protocol`, `Hostname`, `Port`, `Username`, `Password`, `Bypass`, `Disabled`), `CollectionProxy` (`Inherit`, `Disabled`, `Config`), and `FromModelSettings` mapping.
  - `bypass.go`: `MatchBypass` implementing curl-compatible NO_PROXY semantics: wildcard `*`, domain suffix (`.example.com`, `example.com`), exact IPv4/IPv6, CIDR blocks (`10.0.0.0/8`, `192.168.0.0/16`, `fe80::/10`), port-qualified hosts, and `localhost` alias (`127.0.0.1`, `::1`).
  - `resolver.go`: `Resolve` adhering strictly to precedence `Request > Collection > App Preferences > Environment`. Supports explicit disabled override at higher scopes preventing leakage to lower scopes. `ConfigureTransport` configures `http.Transport` for HTTP/HTTPS (via `http.ProxyURL`) and SOCKS5/SOCKS5h (via `golang.org/x/net/proxy` context dialer).
  - `proxy_test.go`: 26 test cases verifying config URL generation, comprehensive curl-matching bypass rules, 10-step precedence matrix, transport configuration, and live HTTP proxy roundtrip through mock servers.
- Tests: `go test -race -v -cover ./internal/core/engine/proxy` -> PASS (82.0% statement coverage). `go vet ./internal/core/engine/proxy` -> clean. `gofmt -l` -> clean.
- Notes: Ready for review. `TRANSPORT` and `ENGINE` can directly invoke `proxy.Resolve` and `proxy.ConfigureTransport` to configure outbound HTTP/HTTPS/SOCKS5 connections.
