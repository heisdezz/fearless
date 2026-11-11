# CONV-INSOMNIA — Insomnia export importer and converter

Owner history: antigravity (claimed 2026-11-11T01:45Z)

## Entries

### 2026-11-11T01:45Z — antigravity

- Did: Claimed `CONV-INSOMNIA`. Reviewed requirements in `docs/progress/packages.md`, `ownership.md`, and `docs/logic/import-export.md`.
- Tests: not run yet.
- Notes: Implementing Insomnia v4/v5 export importer in `internal/convert/insomnia/`:
  - Parsing Insomnia JSON and YAML exports (`__export_format` 4 or 5).
  - Graph/tree reconstruction linking parentId to workspace roots and request groups (folders).
  - Extracting requests, methods, URLs, query parameters, headers, auth (basic, bearer, apikey, digest, ntlm, awsv4, oauth2).
  - Extracting request bodies (raw json/xml/text, form-urlencoded, multipart formdata with file/text parts, graphql, binary file).
  - Extracting base and sub-environments into `model.Environment` and collection variables.
  - Exporting collections back into Insomnia v4 export JSON.
  - Reporting dropped or unmapped constructs.

### 2026-11-11T01:48Z — antigravity

- Did: Completed implementation of `CONV-INSOMNIA`:
  - `internal/convert/insomnia/schema.go`:
    - Full schema mappings with JSON and YAML struct tags for Insomnia export format v4 and v5.
    - Resource models: `workspace`, `request_group` (folders), `request`, `environment`, and `cookie_jar`.
  - `internal/convert/insomnia/import.go`:
    - `Import` supporting both JSON and YAML Insomnia exports.
    - Graph reconstruction: indexing resources by `_id` and `parentId`, identifying workspace root, extracting base environment into collection variables, and creating sub-environments with secret flags for private envs.
    - Tree assembly: recursive creation of folders (`request_group`) and request items (`request`) with sequential ordering.
    - URL parsing with parameter decomposition, query string separation, and structured parameter merging.
    - Full body mode support: JSON, GraphQL, form-urlencoded, multipart form-data (text and `@file(...)` parts), XML, binary files, and plain text.
    - Authentication mapping: Basic, Bearer, APIKey (header/query), Digest, NTLM, AWS IAM / SigV4, OAuth2, and warnings for unsupported types.
    - Settings mapping: `settingFollowRedirects` ("on", "off", "global").
    - Dropped constructs reporting for unsupported resources (e.g. `cookie_jar`).
  - `internal/convert/insomnia/export.go`:
    - `Export` converting `model.Collection` and `model.Environment` trees back to Insomnia v4 export JSON.
  - Unit tests in `internal/convert/insomnia/insomnia_test.go`:
    - Validates JSON import covering workspace, base env, sub-environments, folders, nested requests, all body modes, and auth types.
    - Validates YAML export import.
    - Validates authentication variants.
    - Validates round-trip export and re-import fidelity.
- Tests: `go test -race -cover ./internal/convert/insomnia/...` passes with 83.8% statement coverage. `go vet`, `gofmt`, and `./scripts/check-deps.sh` pass cleanly.
- Ready for review.
