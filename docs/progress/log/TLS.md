# TLS — TLS and certificates

Owner history: antigravity (claimed 2026-11-10T23:13Z, moved to InReview 2026-11-10T23:25Z)

## Entries

### 2026-11-10T23:13Z — antigravity

- Did: Claimed `TLS`. Reviewed `docs/logic/tls-and-certificates.md`, `docs/progress/packages.md`, and upstream Bruno tests `bruno/tests/ssl/client-certs/`.
- Tests: not run yet.
- Notes: Implementing CA bundle overrides (system + custom + NODE_EXTRA_CA_CERTS / SSL_CERT_FILE / SSL_CERT_DIR), client certificate loading for both PEM (cert + key) and PKCS#12 (.p12 / .pfx with passphrase), domain matching regex with wildcards, four-way precedence resolution (request / collection / global preferences), and tls.Config generation in `internal/core/engine/tls/`.

### 2026-11-10T23:25Z — antigravity

- Did: Implemented TLS configuration, CA bundle loader, domain matcher, certificate loaders (PEM + PKCS#12), and four-way precedence resolver:
  - `config.go`: Defined `ClientCertConfig`, `ClientCertificates`, `CAConfig`, `CACertCounts`, `BuildOptions`, `ResolutionResult`, and conversions `FromModelClientCertRef` / `FromModelClientCertificates`.
  - `ca.go`: `LoadCACertPool` supporting system trust pool, user-provided custom CA PEMs/bytes, extra CAs from env (`NODE_EXTRA_CA_CERTS`/`SSL_CERT_FILE`) or path, de-duplication, and diagnostic certificate counts (`System`, `Root`, `Custom`, `Extra`).
  - `matcher.go`: `MatchDomain` supporting upstream Bruno regex rules (`^(https:\/\/|grpc:\/\/|grpcs:\/\/|ws:\/\/|wss:\/\/)?` + escaped domain with `*` as `.*`), host wildcard expansion, exact hostname matches, and port handling.
  - `loader.go`: `LoadCertificate` decoding both PEM cert/key pairs (with legacy PEM passphrase decryption) and PKCS#12 (.p12 / .pfx) bundles via `software.sslmate.com/src/go-pkcs12` with complete AES/SHA-256 and legacy PBES support, resolving relative paths against collection base path.
  - `resolver.go`: `Resolve` adhering strictly to verification toggle rules (skipping CA pool when verification is disabled) and four-way client certificate precedence (Collection > Global > None, respecting `Disabled` flags). Configures `MinVersion` (default TLS 1.2), SNI hostname, session resumption caching (`tls.NewLRUClientSessionCache(64)`), and dynamic `GetClientCertificate`.
  - `tls_test.go`: 10 comprehensive test functions asserting domain matcher rules, verification precedence, four-way precedence matrix, live mTLS client-authenticated HTTP test server roundtrips (PEM and PFX), handshake rejection on untrusted client certs, encrypted PEM key decryption, relative path resolution, and error propagation.
- Tests: `go test -race -v -cover ./internal/core/engine/tls` -> PASS (85.4% statement coverage). `go vet ./internal/core/engine/tls/...` -> clean. `gofmt -l` -> clean. `./scripts/check-deps.sh` -> dependency-policy: ok.
- Notes: Ready for review. `TRANSPORT` and `ENGINE` can directly use `tls.Resolve` to configure outbound TLS clients.
