# CODEGEN — Code snippet generators

Owner history: antigravity (claimed 2026-11-11T01:30Z)

## Entries

### 2026-11-11T01:30Z — antigravity

- Did: Claimed `CODEGEN`. Reviewed requirements in `docs/progress/packages.md`, `ownership.md`, and `docs/logic/import-export.md` §5.
- Tests: not run yet.
- Notes: Implementing code generators for cURL, Go (net/http), JavaScript (fetch and axios), and Python (requests) in `internal/convert/codegen/`:
  - Pure functions taking `*model.Request` (or built request) and optional parameters/variables.
  - Generating idiomatic code snippets matching Bruno's snippet generator capabilities.
  - Byte-for-byte fidelity against target HTTP servers.

### 2026-11-11T01:35Z — antigravity

- Did: Completed implementation of `CODEGEN`:
  - `internal/convert/codegen/types.go`:
    - Target registry (`TargetCurl`, `TargetGo`, `TargetJSFetch`, `TargetJSAxios`, `TargetPythonRequests`).
    - `Generate(target Target, req *model.Request) (string, error)` and `ListTargets() []Target`.
    - Query parameter synthesizer handling encoded values and raw template braces (`{{...}}`).
  - `internal/convert/codegen/curl.go`:
    - `GenerateCurl` producing runnable cURL command snippets matching the built request.
  - `internal/convert/codegen/golang.go`:
    - `GenerateGo` producing compilable `net/http` Go code snippets with dynamic imports (`bytes`, `fmt`, `io`, `net/http`, `net/url`, `os`, `strings`), body modes (JSON, form-urlencoded, text, XML, Sparql, file binary), headers, and authentication.
  - `internal/convert/codegen/javascript.go`:
    - `GenerateJSFetch` producing standard Fetch API snippets with `URLSearchParams`, JSON payloads, custom headers, and authentication.
    - `GenerateJSAxios` producing Node.js / browser Axios configuration snippets with method, URL, headers, basic auth config object, and data payload.
  - `internal/convert/codegen/python.go`:
    - `GeneratePython` producing `requests.request` snippets with synthesized URL, headers dictionary, payload formatting, and `auth=('user', 'pass')` tuples.
  - Unit tests in `internal/convert/codegen/codegen_test.go`:
    - Tests for registry, targets list, cURL generation with re-import roundtrip fidelity, Go generator across all body modes, JS Fetch and Axios generators, Python requests generator, and error handling for unsupported targets / nil requests.
- Tests: `go test -race -cover ./internal/convert/codegen/...` passes with 87.5% statement coverage. `go vet`, `gofmt`, and `./scripts/check-deps.sh` pass cleanly.
- Ready for review.
