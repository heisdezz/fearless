# Log — `AUTH-DIGEST`

Owner history: zed-agent-1 (claimed 2026-11-11T02:55Z, moved to InReview 2026-11-11T03:05Z)

Files: `internal/core/engine/auth/digest.go` and the digest half of `basic.go`. `AUTH-CORE` owns `auth.go`, the registry and the other providers.

---

## Entries

### 2026-11-11T03:05Z — zed-agent-1

- **Did:** Completed the digest wiring. The transport itself was written during `AUTH-CORE`; this pass closed the gap between it and `Apply`.
  - `digestProvider.Prepare` now interpolates the credentials and returns a `TransportProvider` wrapping `digestTransport`, instead of returning nothing.
  - `Deps.Rewind` added: challenge-response schemes consume the body, so a retry needs a fresh request built by the pipeline.
  - `digestTransport` returns the original 401 when no `rewind` is supplied, rather than panicking on a nil function.
  - `closeAndDrain` factored out of the retry path.
- **Tests:** 3 new end-to-end tests plus the 5 transport tests from `AUTH-CORE`. `go test -race` passes; **88.2 % statement coverage** for the package; `gofmt`/`go vet` clean.
- **Acceptance criterion:** *succeeds against a challenge server using the retry RoundTripper* — `TestDigestProviderSuppliesWorkingTransport` drives the full path: `Apply` → `Prepared.Transport.Wrap` → `http.Client.Do` against an `httptest` server that verifies the digest server-side. It fails if the provider merely records config, which the previous version did.

### Why this was a real gap, not paperwork

Before this pass, `digestConfigProvider.Prepare` returned an empty `Prepared`. A request configured with Digest auth would have gone out **unauthenticated**, and the user would have seen a bare 401 with no indication that auth was ignored. It resolved and registered cleanly, so nothing failed — the failure mode was silence. The end-to-end test is now the guard against that class of bug: it asserts the server observes a correct digest, not merely that a header exists.

### Behaviour decisions

| Case | Behaviour | Why |
| --- | --- | --- |
| No `Rewind` supplied | Return the original 401 | The pipeline could not rebuild the body; reporting the server's answer is honest, and inventing an error would hide it |
| Unsupported algorithm (e.g. SHA-256) | Send no `Authorization` header | An MD5 header against a SHA-256 challenge would trip server-side intrusion detection; the 401 stands either way |
| `qop=auth-int` only | Fall back to the RFC 2069 form | `auth-int` needs the body bytes, which the transport does not own |
| Credentials containing `{{var}}` | Interpolated before hashing | Consistent with D-009 (WSSE) |

### Not in this package

- **NTLM** — `AUTH-NTLM`. Connection-oriented, so it needs a different wrapper that pins a single connection.
- **`auth-int`**, SHA-256 digest, and `nc`/`cnonce` incrementing across challenges. One challenge allows one retry, so a fixed `nc` is correct here.

- **Open:** awaiting review. `ENGINE` must supply `Deps.Rewind` when building the request, or digest degrades to the honest 401 path.
