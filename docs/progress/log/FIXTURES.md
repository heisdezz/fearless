# FIXTURES — Test corpora

Owner history: kilo (claimed 2026-11-10T20:26Z)

## Entries

### 2026-11-10T20:26Z — kilo
- Did: Claimed `FIXTURES`. Surveying Bruno clone for `.bru`, collection/env/yml, interpolation/auth/script fixtures and existing helper patterns.
- Tests: not run yet.
- Notes: Will create pinned corpus under `testdata/` with source metadata and add `internal/testutil` helpers (temp dir, golden, httptest) without network requirements.

### 2026-11-10T20:28Z — kilo
- Did: Added `testdata/SOURCE.md` (pinned Bruno commit 60d7517d0545ef82c032c10dcbb6a3c815d246cf). Created offline corpus under `testdata/corpus/`: `bru/basic` (bruno.json, requests, folder), `environments` (.bru), `yml/basic` (opencollection.yml, requests, folders), `interpolation` cases, `auth/digest`, `script` fixtures; each with `MANIFEST.txt` listing upstream source paths. Added `internal/testutil/testutil.go` + tests (TempDir, WriteFile/ReadFile, Golden, NewHTTPTestServer/NewHTTPSTestServer).
- Tests: `CGO_ENABLED=0 go test ./internal/testutil` -> ok.
- Notes: Corpus is copied from local `bruno/` clone at pinned commit; no network required. Additional corpora can be appended within FIXTURES ownership as needed.
