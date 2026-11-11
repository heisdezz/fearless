# AUTH-NTLM — NTLM handshake (NTLMv2)

**Agent:** zed-agent-1
**State:** InReview (with a known integration blocker, see below)
**Package:** `internal/core/engine/auth/ntlm.go`
**Reference:** `bruno/packages/bruno-requests/src/auth/ntlm.ts` (partial), `axios-ntlm` (external)

## What Bruno actually does

Worth stating plainly, because it changes what "parity" means here: **Bruno
does not implement NTLM.** `bruno-electron/src/ipc/network/index.js:178-181`
constructs `NtlmClient(request.ntlmConfig, {})` from the `axios-ntlm` npm
package and swaps it in as axios's adapter. The repository's own `ntlm.ts`
contains only `isNtlmAuthHeader` and `handleNtlmRedirect` — header detection and
one redirect-clearing helper, nothing about message construction.

So there is no in-repo algorithm to port. The implementation follows **MS-NLMP**
directly, which is what `axios-ntlm` implements, and is verified against that
specification's published test vectors.

## Scope delivered

| Area | Implementation |
| --- | --- |
| Type 1 (negotiate) | `ntlmNegotiateMessage` — 40 bytes, curl's NTLMv2 flag set |
| Type 2 parsing | `ntlmParseChallenge`, `findNTLMChallenge` |
| Type 3 (authenticate) | `ntlmCredentials.authenticateMessage` |
| Proof | `ntlmV2Response` (NTLMv2, `[MS-NLMP]` §3.3.1) |
| Key derivation | `ntlmNTowfv1` (MD4), `ntlmNTowfv2` (HMAC-MD5) |
| Transport | `ntlmTransport` — 401 → type 2 → type 3, via `Deps.Rewind` |
| Registration | `Register(ntlmProvider{})` |

### Decisions worth knowing

- **NTLMv2 only.** NTLMv1 is cryptographically broken and its response is
  computed differently; supporting it would add a code path nobody should
  select. A type 2 message without a target info block cannot produce an NTLMv2
  proof, so it is refused and the server's 401 stands — rather than sending an
  unverifiable message that would only produce a more confusing error.
- **No key exchange.** `ntlmFlagNegotiateKeyExch` is explicitly *not* requested
  and is cleared from the server's echoed flags. Without it no session key is
  transmitted, so a server cannot decrypt traffic it did not originate, and there
  is no key to derive wrong. This matches curl's default.
- **`NegotiateNTLM` is force-set on the type 3 message** even if the server did
  not echo it. A type 3 without that flag is rejected, and some servers echo a
  reduced flag word.
- **The 401 body is drained before the retry.** NTLM authenticates a *connection*,
  so the type 3 message must travel on the same socket. `closeAndDrain` (shared
  with the Digest transport) makes connection reuse likely.
- **Empty user name is a no-op**, consistent with the other providers: the
  server's own 401 is a better explanation than a locally-generated failure.

## Verification

```
go vet ./internal/core/engine/auth/          # clean
go test -race -cover ./internal/core/engine/auth/
ok  ...  coverage: 90.7% of statements
```

Golden vectors are MS-NLMP §4.2 (User / Domain / Password):

| Value | Expected | Source |
| --- | --- | --- |
| NTOWFv1 | `A4F49C406510BDCAB6824EE7C30FD852` | §4.2.2 |
| NTOWFv2 | `0C868A403BFD7A93A3001EF22EF02E3F` | §4.2.4 |
| NTProofStr | `C2606A75B2A0947CFD475D653F30C598` | §4.2.4.2.2 |
| Blob | `01010000 00000000 008E1AB736D39000 FFFFFF0011223344 00000000 <targetinfo> 00000000` | §4.2.4.1.1 |

All four match exactly. The blob comparison is byte-for-byte over the whole
structure, not a prefix, so a mistake in any field's position or the terminator
fails the test.

Other coverage:
- NTOWFv2 is case-insensitive in the user name and sensitive to the domain.
- The proof changes with the client nonce (no replay).
- Type 1 layout: flags, absent key exchange, security-buffer offsets past the
  message.
- Type 2 parsing: malformed input (empty, short, bad signature, wrong type) errors
  instead of panicking; an out-of-range target info block is ignored rather than
  sliced out of bounds.
- Type 3 layout: every security buffer points inside the message, no two overlap,
  and the LM field is empty.
- **End-to-end handshake** against an `httptest` server that independently
  recomputes the proof the way a server would — a wrong message fails the request
  rather than passing quietly.
- Pass-through when the server does not challenge; a Digest challenge is left
  alone; a missing `Rewind` returns the server's 401 instead of hanging.
- `{{var}}` interpolation of credentials.

## ⚠️ Integration blocker — B-006

**Digest and NTLM both silently no-op in a real execution.**

`auth.Prepared.Transport` is populated by both providers, but nothing consumes
it: `transport.Options` has no `RoundTripper` hook and `pipeline.go` never reads
`authPrep.Transport`. The unit tests above pass because they drive the transport
through an `http.Client` directly; the pipeline path is untested because the hook
does not exist.

Logged as `B-006` in `blockers.md` for the `TRANSPORT` owner (`antigravity`).
The required change is small — a `WrapTransport func(http.RoundTripper) http.RoundTripper`
field applied in `transport.newClient`, plus one line in `pipeline.go` — but it
is in a file this package does not own.

**No workaround exists.** Do not mark Digest or NTLM as integration-complete
until B-006 is closed.

## Sources

- `bruno/packages/bruno-electron/src/ipc/network/index.js:178-181` — NtlmClient wiring
- `bruno/packages/bruno-electron/package.json:55` — `axios-ntlm` dependency
- `bruno/packages/bruno-requests/src/auth/ntlm.ts` — header detection and redirect handling only
- MS-NLMP §2.2.1.1, §2.2.1.2, §2.2.2.5, §2.2.2.7, §3.3.1, §4.2
- MS-NTHT — NTLM over HTTP
