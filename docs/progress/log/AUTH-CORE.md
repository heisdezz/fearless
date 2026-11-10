# Log — `AUTH-CORE`

Owner history: zed-agent-1 (claimed 2026-11-11T01:10Z, moved to InReview 2026-11-11T01:40Z)

Package: `internal/core/engine/auth/**` (registry, base providers, digest transport). Advanced providers live in `AUTH-DIGEST`, `AUTH-NTLM`, `AUTH-OAUTH1`, `AUTH-OAUTH2`, `AUTH-AWSV4`, `AUTH-EDGEGRID` and own only their own `<mode>*.go`.

---

## Entries

### 2026-11-11T01:40Z — zed-agent-1

- **Did:** Implemented the auth registry and base providers.
  - `auth.go` — `Provider` interface, `Prepared` output, `Deps` injection, `Register`/`Get`/`Modes`, `Apply`, `Resolve`.
  - `basic.go` — `basic`, `digest` (config side), `wsse` providers.
  - `bearer.go` — `bearer`, `apikey`, `none` providers.
  - `digest.go` — `digestTransport`: full 401 challenge → retry handshake, MD5 and MD5-sess, `stale` handling, `auth` qop selection.
  - `helpers.go` — injected-clock/random helpers, WSSE digest composition, header quoting, auth-param parser.
  - `registry.go` — `RegisterBase` wiring and `ChainFor` for inheritance.
- **Tests:** 30 test functions. `go test -race` passes; **86.2 % statement coverage**; `gofmt`/`go vet` clean.
- **Acceptance criteria:**
  1. *Adding a provider is one file plus one `Register` call* — satisfied structurally: each provider implements one interface (`Mode` + `Prepare`) and lives in its own file (61–73 lines each). Nothing else changes.
  2. *Inherit resolution works through the merge chain* — `Resolve`/`ChainFor` tested including collection→folder→request precedence and the `none` opt-out.
  3. *Each provider has a table test asserting the exact header/query effect* — done for basic, bearer, apikey (both placements), wsse and none.

### Design decisions worth recording

| Decision | Why |
| --- | --- |
| `Prepared` is returned, not applied to an `*http.Request` | Auth must run before the body is serialised and, for api-key query params, before the URL is finalised. Returning a description keeps the merge order in one place. |
| `Deps` injects clock, randomness, interpolation and token access | WSSE needs time and random bytes; without injection its tests could only assert shape, not values. `TestWSSEDigestGolden` asserts an exact digest because of this. |
| `digest` registers a **config-only** provider here | Digest cannot produce a header before the server supplies a nonce, so its `Prepare` is a no-op and the transport half (`digestTransport`) does the work. Registering it here keeps `Apply` from reporting "no provider" for a mode that is implemented. |
| Unsupported digest algorithms return an **empty header** | A SHA-256 challenge answered with an MD5 header would be worse than no header: the 401 stands either way, but the wrong header can trip server-side intrusion detection. |
| nc/cnonce are fixed, not random | One challenge allows one retry, so replay protection has nothing to protect; fixed values make the handshake deterministic and testable. |

### Bugs found and fixed during implementation

- **nil `next` in the digest transport.** `Wrap` returned the receiver with `next` set, but a caller constructing the transport directly got a nil `next` and RoundTrip panicked. `RoundTrip` now falls back to `http.DefaultTransport`, and `Wrap` returns a copy rather than mutating the original.
- **`Resolve` aliased stored configuration.** Copying the `Auth` struct is insufficient: its string fields are pointers, so a provider writing through one reached the caller's original. Now deep-copies all 40 pointer fields. `TestResolveReturnsCopy` pins it.

### Divergence from Bruno

**WSSE interpolates before hashing** (D-009). Bruno hashes the raw config, so `password: {{pw}}` produces a digest of the literal braces. That is a defect; `docs/logic/authentication.md` §5 proposes the same fix. Pinned by a test.

### Not in this package

- `ntlm`, `oauth1`, `oauth2`, `awsv4`, `akamai-edgegrid` — their own packages. `Apply` reports `ErrNoProvider` naming the mode until they land, which is deliberate: silently sending an unauthenticated request would look like a server bug.
- `auth-int` qop (needs the body bytes) and SHA-256 digest variants.

- **Open:** awaiting review. `BUILD` can depend on `Apply`/`Resolve`/`ChainFor`.
