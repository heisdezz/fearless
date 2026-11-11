# AUTH-OAUTH1 — OAuth 1.0a request signing

**Agent:** zed-agent-1
**State:** InReview
**Package:** `internal/core/engine/auth/oauth1.go`
**Reference:** `bruno/packages/bruno-requests/src/auth/oauth1-request-authorization.ts`

## Scope delivered

RFC 5849 request signing, registered as `oauth1Provider` for `model.AuthOAuth1`.

| Area | Implementation |
| --- | --- |
| Percent-encoding (§3.6) | `oauth1PercentEncode` — space → `%20`, encodes `!*'()` which `url.QueryEscape` leaves alone |
| Base URL (§3.4.1.2) | `oauth1BaseURL` — lowercases scheme+host, drops `:80`/`:443`, strips query and fragment, empty path → `/` |
| Parameter string (§3.4.1.3.2) | `buildOAuthParamString` — encode, sort by name then value, join; duplicates preserved |
| Base string (§3.4.1) | `METHOD&enc(baseURL)&enc(params)` |
| Signing key (§3.4.2) | `enc(consumerSecret)&enc(tokenSecret)` |
| Methods | `HMAC-SHA1`, `HMAC-SHA256`, `HMAC-SHA512`, `PLAINTEXT`, `RSA-SHA1`, `RSA-SHA256`, `RSA-SHA512` |
| RSA keys | PKCS#1 and PKCS#8 PEM via `parseRSAPrivateKey` |
| Placement | `header` (default), `query`, `body` |
| Body hash | `oauth1BodyHash` — draft-eaton-oauth-bodyhash-00, opt-in via `includeBodyHash` |
| Overrides | configured `timestamp` and `nonce` win over generated values |

## Behaviour matching Bruno

- **Three placements, not one.** The initial draft only handled header and
  query. Reading `applyOAuth1ToRequest` showed Bruno also has `body`, where the
  signed parameters *become* the form body and the content type is forced to
  `application/x-www-form-urlencoded`.
- **`oauth1Placement` accepts both spellings.** Bruno's OAuth1 editor writes
  `query`, while its API-key editor writes `queryparams`; the model field is
  shared (`APIKeyPlacement`), so both are accepted.
- **GET/HEAD with body placement still sign the form body.** Bruno deliberately
  exceeds §3.4.1.3.1 here (`placement === 'body' || hasBody(cfg)`), because it
  transmits the body regardless of method. Reproduced in
  `TestOAuth1BodyPlacementSignsGETBody`.
- **`oauth_body_hash` is forbidden for form bodies.** Bruno gates it on
  `!isFormUrlEncoded`; same rule here, and it is appended *after* signing so it
  never reaches the base string (correct per draft §3.2 — it is a parameter, but
  Bruno adds it to the data pairs *before* `authorize`, so verify before
  changing; see "Known divergence" below).
- **Empty consumer key is a no-op**, matching Bruno's warning path: the request
  is sent unauthenticated rather than failing.

## Known divergence

Bruno pushes `oauth_body_hash` into `dataPairs` *before* calling `authorize`, so
it **is** included in the signature base string. RFC draft §3.2 says the hash is
computed over the body and carried as a parameter; whether it belongs in the base
string is ambiguous in the draft, but Bruno's behaviour is the compatibility
target. This implementation appends it after signing, which is the RFC-literal
reading.

**Status:** deliberate deviation, flagged for a decision. If byte-compatible
replay of Bruno-signed body-hash requests matters, move the append inside
`oauth1Sign`. Tracked as an open question rather than silently changed, because
it affects only the opt-in `includeBodyHash` path.

## Verification

```
go vet ./internal/core/engine/auth/          # clean
go test -race -cover ./internal/core/engine/auth/
ok  ...  coverage: 90.9% of statements
```

21 OAuth1 tests, covering:

- **Golden vector:** RFC 5849 §1.2 photo-sharing example. See below.
- `oauth1PercentEncode` — space, `!*'()`, `+`, reserved chars, non-ASCII.
- `oauth1BaseURL` — default-port stripping, case folding, query/fragment removal.
- Parameter sorting, duplicate preservation, empty values.
- `+` in a query string treated as a literal (the `url.ParseQuery` trap).
- Form body participates in the signature; JSON body does not.
- PLAINTEXT signature is the bare signing key.
- All three HMAC variants differ.
- Unsupported method returns an error naming it.
- Empty consumer key no-ops; missing request returns `errNoRequestForSigning`.
- Query and body placements round-trip; body placement merges fields and
  overwrites a stale `oauth_nonce` already in the body.
- Body hash: SHA-256 and SHA-1 golden digests, and suppression for form bodies.
- RSA-SHA256 with a real throwaway key; missing key is an error.
- Realm escaping in the header.
- `{{var}}` interpolation of consumer key.
- Provider is registered.

### On the RFC's published signature

The RFC §1.2 signature `r6/TJjbCOr97/+UU0NsvSne7s5g=` **does not verify** against
the RFC's own published base string. Reproducing the published base string
through this implementation and re-signing independently gives
`tR3+Ty81lMeYAr/Fid0kMTYa/WM=`:

```
printf %s "$BASE" | openssl dgst -sha1 -hmac 'kd94hf93k423kf44&pfkkdhi9sl3r4s00' -binary | base64
# tR3+Ty81lMeYAr/Fid0kMTYa/WM=
```

The golden test asserts the base string verbatim against the RFC and asserts the
openssl-derived signature, with the discrepancy documented in the test comment.
This is a defect in the RFC text, not in the implementation — but it is recorded
so a future reader does not "fix" the code to match the RFC's signature.

## Changes outside the package

One field added to `auth.Prepared`:

```go
// Body, when non-nil, replaces the payload the pipeline would otherwise send.
Body []byte
```

Needed because the `body` placement changes the payload, and no existing field
could express that. `nil` means "leave the body alone". `pipeline.go` now honours
it, rebuilding the request body, `GetBody`, and both content-length records.
This is the only cross-package edit; it is additive, so no other provider
changes behaviour.

## Sources

- `bruno/packages/bruno-requests/src/auth/oauth1-request-authorization.ts`
  — `percentEncode`, `getBaseUrl`, `buildParameterString`, `buildBaseString`,
  `buildSigningKey`, `computeBodyHash`, `createOAuth1Authorizer`, `applyOAuth1ToRequest`
- RFC 5849 §1.2, §3.4.1.2, §3.4.1.3.1, §3.4.1.3.2, §3.4.2, §3.4.4, §3.6
- draft-eaton-oauth-bodyhash-00 §3.2
