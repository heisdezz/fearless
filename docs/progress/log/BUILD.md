# BUILD — Request preparation and synthesis

Owner history: antigravity (claimed 2026-11-10T23:59Z)

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
