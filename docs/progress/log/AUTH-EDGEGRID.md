# AUTH-EDGEGRID — Akamai EdgeGrid EG1-HMAC-SHA256

**Agent:** zed-agent-1
**State:** InReview
**Package:** `internal/core/engine/auth/edgegrid.go`
**Reference:** `bruno-requests/src/auth/edgegrid-helper.js`, `bruno-common/src/utils/edgegrid.ts`

## Algorithm

Akamai's signature is a tab-joined string plus a double HMAC:

```
dataToSign = METHOD \t scheme \t host \t path+query \t signedHeaders \t contentHash \t authHeader
signingKey = base64( HMAC-SHA256( timestamp, clientSecret ) )
signature  = base64( HMAC-SHA256( dataToSign, signingKey ) )     ← key is the base64 TEXT
Authorization: authHeader + "signature=" + signature
```

where `authHeader` is also the final field of `dataToSign`, so it appears twice.

### The three traps, and how each is pinned by a test

| Trap | Consequence | Test |
| --- | --- | --- |
| Using the **raw** digest of the first HMAC as the second key, instead of its base64 text | every signature wrong | `TestEdgeGridSigningKeyIsBase64String` computes the wrong variant too and asserts they differ |
| Sorting signed headers alphabetically | looks right, is rejected by Akamai — **configuration order is required** | `TestEdgeGridSignedHeaderOrderIsConfigOrder` configures `Zebra,Alpha` and expects `zebra:z\talpha:a` |
| Signing `request.url` when a base URL is set, or mis-parsing a scheme-less base | silent mismatch: `url.Parse("localhost:6000")` reads `localhost:` as a scheme and leaves the host empty | `TestEdgeGridBaseURLOverridesAuthority` covers full, scheme-less and path-bearing bases |

## Behaviour matching Bruno

- **All three credentials required.** A partial config contributes nothing. The
  literal string `"undefined"` counts as absent — Bruno's `isStrPresent` exists
  precisely because an unresolved template can stringify to that, and signing it
  would produce a header no server accepts.
- **Timestamp `YYYYMMDDTHH:MM:SS+0000`**, not RFC 3339. Akamai expects the
  literal `+0000` offset, so the layout is `20060102T15:04:05-0700` applied to a
  UTC time rather than `...Z`.
- **Only POST bodies are hashed**, truncated to 128 KB. A GET with a body gets an
  empty content hash. This is Akamai's rule, not an oversight, and it is easy to
  "fix" by accident.
- **Configured timestamp and nonce override generation**, for reproducible
  captures.
- **Base URL replaces scheme and host only**; the request's own path and query
  are signed.

### Extension beyond Bruno

Bruno's `canonicalizeHeaders` lowercases header names into a plain object, so two
spellings of one header (`Accept` and `accept`) silently collide and one wins by
assignment order. Here values are joined with `", "`, matching HTTP's own
equivalent-value rule, so neither is dropped. For a single spelling the output is
identical.

Also, a multi-valued header (`X-Multi: one`, `X-Multi: two`) is joined as
`one, two`. Bruno receives axios's already-flattened value, so this reproduces
what it would see.

## Verification

```
go vet ./internal/core/engine/auth/          # clean
go test -race -cover ./internal/core/engine/auth/
ok  ...  coverage: 90.9% of statements
```

Independent recomputation rather than a pasted digest: no public EdgeGrid test
vector was verified here, so the tests recompute every intermediate
(`signingKey`, `dataToSign`, `signature`) with `crypto/hmac` directly and compare
against the implementation. That proves the implementation is *self-consistent
with the documented construction*, not that the construction matches Akamai —
see the open question below.

Coverage:

- `dataToSign` layout: exact field order, exact tab joining, auth header last.
- Signing key is the base64 text; the raw-digest variant is computed and asserted
  to differ.
- POST + body + signed headers: full independent recomputation.
- Configuration order preserved for signed headers (not alphabetical).
- Canonicalisation rules: lowercase, multi-value join, absent header skipped,
  empty value kept, blank config entry ignored, case-insensitive lookup.
- Content hash: GET/PUT hashed empty, POST hashed, empty body → empty hash,
  case-insensitive method name, truncation to the window, zero window falls back.
- Base URL: full, scheme-less, path-bearing; URL with no host rejected; empty path
  becomes `/`.
- Provider: registration, partial credentials no-op, `"undefined"` no-op, header
  shape, generated timestamp format, UUID v4 nonce (version and variant nibbles),
  missing request errors, `{{var}}` interpolation, configured max body size.

## Open question (logged, not blocking)

**No external golden vector has been verified.** Akamai publishes no static
example, and the only sources of truth are the reference client and the live
gateway. The construction matches two independent Bruno implementations
(`edgegrid-helper.js` runtime and `har/edgegrid.ts` Generate Code), which agree
with each other and with Akamai's `AkamaiOPEN-edgegrid-node`.

The remaining risk is a field we both read the same way but neither of us
verified against the service — most plausibly the `+0000` timestamp rendering or
whether a multi-valued header should be space-joined rather than comma-joined.
**Confirmation requires one request against a real Akamai collection.** Recorded
as an unresolved question rather than claimed as verified.

## Sources

- `bruno-requests/src/auth/edgegrid-helper.js` — `signEdgeGridRequest`, `addEdgeGridInterceptor`
- `bruno-common/src/utils/edgegrid.ts` — `canonicalizeHeaders`, `makeContentHash`,
  `makeEdgeGridTimestamp`, `makeEdgeGridNonce`, `base64HmacSha256`, `isStrPresent`
- `bruno-common/src/generate-code/har/edgegrid.ts` — second implementation, cross-checked
- Akamai: https://techdocs.akamai.com/developer/docs/authenticate-with-edgegrid
