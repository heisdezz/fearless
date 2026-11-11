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
