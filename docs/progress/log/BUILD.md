# BUILD — Request preparation and synthesis

Owner history: antigravity (claimed 2026-11-10T23:59Z), antigravity-2 (fixes 2026-11-11T01:30Z)

## Entries

### 2026-11-10T23:59Z — antigravity

- Did: Claimed `BUILD`. Reviewed `flows/request-lifecycle.md`, `packages.md` acceptance criteria, and `internal/core/model/request.go`.
- Tests: not run yet.
- Notes: Implementing `internal/core/engine/build/` request builder: variable interpolation across URL, query/path parameters, headers, and body; URL synthesis with path param expansion and query serialization; body rendering with automatic Content-Type and Content-Length determination across all body modes (`none`, `json`, `text`, `xml`, `sparql`, `formUrlEncoded`, `multipartForm`, `graphql`, `file`); and live header emission with override semantics.

### 2026-11-11T00:15Z — antigravity

- Did: Completed implementation of `internal/core/engine/build/`:
  - `types.go`: Core builder interfaces and types (`BuildInput`, `BuiltRequest`, `HeaderEntry`, `Builder`).
  - `decomment.go`: FSM single-pass comment stripper for JSON bodies (strips `//` and `/* */` comments without touching string literals or URLs).
  - `merge.go`: Tree inheritance merger resolving collection -> parent folders -> request headers, auth, and settings.
  - `url.go`: URL synthesis with path parameter interpolation (`:param` and `:key` OData syntax), scheme fallback (`http://`), query merging, and RFC 3986 encoding.
  - `body.go`: Body rendering across all modes (`none`, `json`, `text`, `xml`, `sparql`, `graphql`, `formUrlEncoded`, `multipartForm`, `file`) with Content-Type calculation (explicit header wins) and Content-Length determination.
  - `build.go`: Request assembly pipeline with variable substitution (`vars.Interpolate`), unresolved variable tracking, auth headers (Basic, Bearer, APIKey header/query, WSSE), and `*http.Request` creation.
- Tests: `go test -race -v -cover ./internal/core/engine/build` passes at 84.6% statement coverage. `go vet` and `./scripts/check-deps.sh` pass cleanly.
- Ready for review.

### 2026-11-11T01:30Z — antigravity-2

- Did: Reviewed and fixed architectural & lifecycle defects in `internal/core/engine/build/`:
  - `merge.go`: Included collection root config (`col.Config.Headers` and `col.Config.Auth`) in the inheritance chain.
  - `merge.go`: Implemented UQ-B1 header deletion semantics (a child header with explicit empty string `""` deletes the inherited header from ancestor folders/collection).
  - `url.go`: Preserved duplicate query parameters and original declaration order without suppression or sorting.
  - `url.go`: Allowed URLs starting with unresolved template variables (e.g. `{{baseUrl}}`) without premature format failure.
  - `body.go`: Replaced alphabetical `url.Values.Encode()` key sorting in `BodyFormURLEncoded` with strict declaration-order serialization.
  - `build.go`: Correctly populated `httpReq.Host` from `Host` header.
  - `build.go`: Maintained auth preparation and query placement for prepare-only schemes (Basic, Bearer, APIKey, WSSE).
  - `build_test.go`: Added test suites for UQ-B1 deletion, collection root inheritance, query duplicates & order preservation, form urlencoded order, and host header propagation.
- Tests: `go test -race -v ./internal/core/engine/build/...` passes with 86.7% statement coverage. `go test -race ./internal/core/engine` passes. `go test -race ./internal/app/commands/...` passes. `go vet` and `./scripts/check-deps.sh` pass cleanly.
