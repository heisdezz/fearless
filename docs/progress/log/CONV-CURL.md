# CONV-CURL — cURL command importer and exporter

Owner history: antigravity (claimed 2026-11-11T01:18Z)

## Entries

### 2026-11-11T01:18Z — antigravity

- Did: Claimed `CONV-CURL`. Reviewed requirements in `docs/progress/packages.md`, `ownership.md`, and `docs/logic/import-export.md`.
- Tests: not run yet.
- Notes: Implementing cURL importer and exporter in `internal/convert/curl/`:
  - Shell command tokenization handling single quotes, double quotes, escapes, backslash line breaks.
  - Method extraction (`-X`, `--request`, `-I`/`--head`, defaulting to POST on `--data` or GET).
  - Header extraction (`-H`, `--header`), User-Agent (`-A`), Referer (`-e`), Cookie (`-b`).
  - Auth extraction (`-u`, `--user` -> basic auth, Bearer auth headers).
  - Query parameters parsing and extraction into `model.QueryParam`.
  - Body extraction:
    - Raw text, JSON, XML (`-d`, `--data`, `--data-raw`, `--data-ascii`, `--data-binary`).
    - Form urlencoded (`--data-urlencode`).
    - Multipart form-data (`-F`, `--form` including `@filename` / `name=value`).
  - Dropped constructs / warnings reporting for flags that don't map directly to the request model (e.g. `--insecure`, `--compressed`, `--proxy`, `--max-time`, etc.).
  - Exporter (`ExportCurl`): converts `model.Request` into a formatted, reproducible cURL command string.

### 2026-11-11T01:25Z — antigravity

- Did: Completed implementation of `CONV-CURL`:
  - `internal/convert/curl/tokenize.go`:
    - Shell tokenizer handling POSIX quoting rules: single quotes `'...'`, double quotes `"..."` with escape sequences (`\"`, `\\`, `\$`), unquoted backslash escapes, line continuations (`\` before newline), comments (`#`), and strips leading curl executable names.
  - `internal/convert/curl/parser.go`:
    - Command parser supporting attached short options (e.g. `-XPOST`, `-H'Foo: Bar'`) and long options with `=` (e.g. `--request=POST`).
    - URL parsing, fragment isolation, query parameters extraction into `model.Param` rows (`ParamQuery`), and item name inference.
    - HTTP method detection defaulting to GET or POST (when payload present).
    - Headers parsing (`-H`, `--header`, empty headers `Key;`, `-A` User-Agent, `-e` Referer, `-b` Cookie).
    - Authentication extraction: Basic auth (`-u user:pass` and Authorization header base64), Bearer auth (`--bearer` and Authorization header), Digest (`--digest`), NTLM (`--ntlm`), AWS SigV4 (`--aws-sigv4`).
    - Body mode handling: JSON (with GraphQL auto-detection), XML, FormURLEncoded (`--data-urlencode` and multiple `-d`), MultipartForm (`-F` with `@file(...)` and custom Content-Type), and File (`--data-binary @...`).
    - `-G` flag support converting data payloads to URL query parameters.
    - Comprehensive dropped constructs / warnings reporting for unmapped flags (`--compressed`, `-k`/`--insecure`, `--proxy`, `--max-time`, `--cacert`, etc.).
  - `internal/convert/curl/exporter.go`:
    - `Export` and `ExportWithOptions` converting `model.Request` into reproducible, POSIX-escaped cURL commands with multiline (` \`) or single-line formatting.
    - Exports URL with synthesized query parameters, live headers, auth (`-u`, Bearer headers, `--digest`, `--ntlm`, `--aws-sigv4`), body payloads (`--data-raw`, `--data-urlencode`, `-F`, `--data-binary`), and follow-redirects (`-L`).
  - `internal/convert/curl/curl.go`:
    - Clean public API: `Import`, `ImportRequest`, `ImportItem`, `Export`, `ExportWithOptions`.
  - Comprehensive test suite in `internal/convert/curl/curl_test.go`:
    - Tokenization edge cases, error conditions, methods, query parameters, headers, all auth types, all body modes, dropped constructs reporting, and end-to-end round-trip test.
- Tests: `go test -race -cover ./internal/convert/curl` passes with 87.6% statement coverage. `go vet`, `gofmt`, and `./scripts/check-deps.sh` pass cleanly.
- Ready for review.
