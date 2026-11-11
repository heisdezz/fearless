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
