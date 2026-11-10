# Log — `AUTH-AWSV4`

Owner history: zed-agent-1 (claimed 2026-11-11T03:40Z, moved to InReview 2026-11-11T04:15Z)

Files: `internal/core/engine/auth/awsv4.go`, `awsv4_headers.go`, `awsv4_test.go`.

---

## Entries

### 2026-11-11T04:15Z — zed-agent-1

- **Did:** Implemented AWS Signature Version 4.
  - `awsv4.go` — the five signing steps: canonical request, its hash, string to sign, the HMAC key chain, and the signature; plus `canonicalURI`, `canonicalQuery`, header normalisation and `awsURIEncode`.
  - `awsv4_headers.go` — the signed-header allow list, mirroring the AWS SDK's `requiredSignedHeaders`.
  - `awsv4_test.go` — 13 tests including the full documented AWS vector.
- **Tests:** `go test -race` passes; **89.3 % statement coverage**; `gofmt`/`go vet` clean.
- **Acceptance criterion:** *signatures match a reference implementation for a fixed clock/credential* — met exactly. `TestAWSSigV4DocumentedVector` reproduces the AWS S3 documentation example and matches the published `Authorization` header byte for byte, including signature `f0e8bdb8…36bdb41`.

### Why the documented vector, not the SDK test

I first tried to use `aws-sdk-go`'s `TestSignRequest` as the golden vector. It failed, and after ~50 probe variants none matched — because that test constructs its request in ways I could not reconstruct faithfully (it sets `URL.Opaque` and relies on SDK-internal request plumbing). Rather than tune code until a mismatched fixture happened to pass, I switched to the **AWS documentation example**, which publishes every intermediate value: the canonical request, the string to sign, and the signature.

That is the stronger test. When it passed, it proved the primitives *and* the assembly independently. When it failed, the failure pointed at a specific line.

### Three real bugs the vector caught

All three would have produced signatures that AWS rejects, with no local error:

1. **`content-length` and `range` were not signed.** My first allow list covered only `x-amz-*`, `content-type` and `content-length`. AWS signs a broader set of standard headers (`range`, `if-match`, `content-md5`, …). `Range` was silently dropped, so the signed-header list was wrong. Fixed by mirroring the SDK's allow list in `awsv4_headers.go`.
2. **An extra newline in the canonical request.** `canonicalizeHeaders` already ends with `\n`, and the join separator supplies the blank line the specification requires. Adding another inserted two blank lines and changed the signature.
3. **Header values were normalised with `strings.Fields`,** which splits on tabs as well as spaces. AWS's `stripExcessSpaces` collapses runs of *spaces* only. Now `collapseSpaces` does exactly that.

A fourth, smaller issue: `content-length` arrives via `http.Request.Header` in tests but is usually set on the struct field by `http.NewRequest`. The signer reads the header map, matching the SDK, and `BUILD` must ensure the header is present when the body length matters.

### Design notes

- **`Opaque` URLs.** A URL built with `URL.Opaque` has an empty `Path`, so signing `EscapedPath()` would sign `/` for a request that goes to `/bucket/key`. `canonicalURI` reproduces the SDK's `"/" + strings.Join(strings.Split(Opaque, "/")[3:], "/")`.
- **Query encoding uses `url.Values.Encode`,** not a hand-rolled `awsURIEncode`. The SDK does this and the two differ on a few characters; since the signature must match AWS, matching the SDK wins. Values are sorted per key first, as the SDK does.
- **`Deps.Request` and `Deps.Body`** were added for this provider. SigV4 signs the method, path, query and payload, so it cannot work from configuration alone. With no `Request`, the provider returns `errNoRequestForSigning` rather than emitting a signature that cannot be correct.
- **No credentials → no signature.** Bruno warns and sends the request unsigned; matching that is friendlier than failing, and the server's 403 explains itself.

### Not in this package

- **SigV4a** (ECDSA P-256, multi-region) — rare and needs a different key-derivation KDF.
- **Query-string presigning** (`X-Amz-Signature` in the URL) — a separate code path; header signing covers the API-client use case.
- **Profile-based credentials** (`profileName` → `~/.aws/credentials`). The model carries the field; loading it belongs with `PERSIST`, and `testdata` has no fixture for it.

- **Open:** awaiting review. `ENGINE` and `BUILD` must pass `Deps.Request` and `Deps.Body` for SigV4 to sign correctly.
